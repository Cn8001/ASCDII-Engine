package session

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/kit"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

const (
	SceneView = "scene"
	GameView  = "game"
)

var (
	ErrNotPlaying = errors.New("not in play mode")
	ErrBadAxis    = errors.New("unknown axis")
)

func (s *Session) Resize(cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.frame.Resize(max(10, min(cols, 480)), max(6, min(rows, 240))) {
		s.touch()
	}
}

// SetView switches between the free editor camera and the game camera.
func (s *Session) SetView(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if mode == GameView {
		s.viewMode = GameView
	} else {
		s.viewMode = SceneView
	}
	s.touch()
}

// camera is what navigation moves: the editor camera in the scene view, the
// game camera otherwise.
func (s *Session) camera() *view.Camera {
	if s.viewMode == SceneView {
		return &s.cam
	}
	return &s.current().Camera
}

func (s *Session) view() view.View { return s.camera().View(s.frame.Buf.W, s.frame.Buf.H) }

// ground is the world point at a cell position on the plane at height z.
func ground(v view.View, col, row, z float64) geom.Vec3 {
	return v.Unproject(col*raster.SubW, row*raster.SubH, z)
}

func (s *Session) Pan(dcol, drow float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.camera()
	v := s.view()
	d := ground(v, dcol, drow, c.Target.Z).Sub(ground(v, 0, 0, c.Target.Z))
	c.Target = c.Target.Sub(d)
	s.touch()
}

func (s *Session) Orbit(dyaw, dpitch float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.camera()
	c.Yaw = math.Mod(c.Yaw+dyaw+540, 360) - 180
	c.Pitch = geom.Clamp(c.Pitch+dpitch, 15, 89)
	s.touch()
}

func (s *Session) Zoom(k float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.camera()
	c.Span = geom.Clamp(c.Span*k, 6, 400)
	s.touch()
}

func (s *Session) Focus(id uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.current().ByID(id)
	if err != nil {
		return err
	}
	p := e.World().Pos
	s.camera().Target = geom.V3(p.X, p.Y, 0)
	s.touch()
	return nil
}

// AlignGame copies the editor view into the game camera, like Unity's
// "align with view".
func (s *Session) AlignGame() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.play == nil {
		s.checkpoint("")
	}
	s.current().Camera = s.cam
	s.changed()
}

// Pick selects whatever is drawn nearest at a cell; empty ground clears selection.
func (s *Session) Pick(col, row int) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sel = uint64(s.frame.Buf.Pick(col, row))
	s.touch()
	return s.sel
}

func (s *Session) Select(id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sel = id
	s.touch()
}

func (s *Session) Selected() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sel
}

// Drag moves an entity between two cell positions; axis is "free" on its
// ground plane, "x" or "y" along one world axis, or "z" for height.
func (s *Session) Drag(id uint64, axis string, fromCol, fromRow, toCol, toRow float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.play == nil {
		s.checkpoint(fmt.Sprintf("drag:%d", id))
	}
	e, err := s.current().ByID(id)
	if err != nil {
		return err
	}
	if e.Inherited() {
		return fmt.Errorf("drag %s: %w", e.Name, scene.ErrPrefabLocked)
	}
	w := e.World()
	v := s.view()
	d := ground(v, toCol, toRow, w.Pos.Z).Sub(ground(v, fromCol, fromRow, w.Pos.Z))
	switch axis {
	case "free":
		d.Z = 0
	case "x":
		d = geom.V3(d.X, 0, 0)
	case "y":
		d = geom.V3(0, d.Y, 0)
	case "z":
		_, cp := v.Pitch()
		d = geom.V3(0, 0, -(toRow-fromRow)*raster.SubH/(v.Zoom*cp))
	default:
		return fmt.Errorf("drag %s: %w: %s", e.Name, ErrBadAxis, axis)
	}
	if p := e.Parent(); p != nil {
		pw := p.World()
		k := pw.Scale
		if k == 0 {
			k = 1
		}
		d = d.Scale(1 / k)
		if pw.Flip {
			d.X = -d.X
		}
	}
	e.Local.Pos = e.Local.Pos.Add(d)
	s.changed()
	return nil
}

// Drop places an asset or node kind on the ground under a cell, which is how
// assets are dragged into the viewport. Kinds are "prefab:<path>",
// "art:<path>" or a registered node kind.
func (s *Session) Drop(kind string, col, row float64) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	at := ground(s.view(), col, row, 0)
	var e *scene.Entity
	switch {
	case strings.HasPrefix(kind, "prefab:"):
		p := strings.TrimPrefix(kind, "prefab:")
		e = scene.NewEntity("", nil)
		e.Prefab = s.lib.Ref(p)
	case strings.HasPrefix(kind, "art:"):
		p := strings.TrimPrefix(kind, "art:")
		if _, err := s.lib.Art(p); err != nil {
			return 0, err
		}
		e = scene.NewEntity(strings.TrimSuffix(filepath.Base(s.lib.Resolve(p)), filepath.Ext(s.lib.Resolve(p))), kit.NewSprite(s.lib, s.lib.Ref(p)))
	default:
		n, err := s.reg.NewNode(kind)
		if err != nil {
			return 0, err
		}
		e = scene.NewEntity(kind, n)
	}
	if s.play == nil {
		s.checkpoint("")
	}
	e.Local.Pos = geom.V3(math.Round(at.X*4)/4, math.Round(at.Y*4)/4, 0)
	got, err := sc.Graft(nil, e, -1)
	if err != nil {
		return 0, err
	}
	s.sel = got.ID
	s.changed()
	return got.ID, nil
}

// Key forwards a key to the running game, named like "w", "space" or "up".
func (s *Session) Key(name string) error {
	k, err := input.ParseName(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode != Play {
		return ErrNotPlaying
	}
	s.keys = append(s.keys, k)
	return nil
}
