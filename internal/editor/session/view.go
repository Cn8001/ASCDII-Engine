package session

import (
	"errors"
	"fmt"
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
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

// camera is what navigation looks through: the editor camera in the scene
// view, the game camera otherwise.
func (s *Session) camera() view.Camera {
	if s.viewMode == SceneView {
		return s.cam
	}
	if c, ok := s.current().View(); ok {
		return c
	}
	return s.cam // a scene without a camera shows a notice; picks still need a view
}

// startView is where the editor camera begins: at the game camera, or at the
// default view in scenes without one.
func startView(sc *scene.Scene) view.Camera {
	if c, ok := sc.View(); ok {
		return c
	}
	return view.Default()
}

// moveCamera changes what navigation looks through; in the game view that
// moves the scene's camera object.
func (s *Session) moveCamera(fn func(c *view.Camera)) {
	if s.viewMode == SceneView {
		fn(&s.cam)
		return
	}
	c, ok := s.current().View()
	if !ok {
		return
	}
	fn(&c)
	_ = s.current().SetView(c) // the camera was just found
}

func (s *Session) view() view.View { return s.camera().View(s.frame.Buf.W, s.frame.Buf.H) }

// ground is the world point at a cell position on the plane at height z.
func ground(v view.View, col, row, z float64) geom.Vec3 {
	return v.Unproject(col*raster.SubW, row*raster.SubH, z)
}

func (s *Session) Pan(dcol, drow float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.view()
	s.moveCamera(func(c *view.Camera) {
		d := ground(v, dcol, drow, c.Target.Z).Sub(ground(v, 0, 0, c.Target.Z))
		c.Target = c.Target.Sub(d)
	})
	s.touch()
}

func (s *Session) Orbit(dyaw, dpitch float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.moveCamera(func(c *view.Camera) {
		c.Yaw = math.Mod(c.Yaw+dyaw+540, 360) - 180
		c.Pitch = geom.Clamp(c.Pitch+dpitch, 15, 89)
	})
	s.touch()
}

func (s *Session) Zoom(k float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.moveCamera(func(c *view.Camera) { c.Span = geom.Clamp(c.Span*k, 6, 400) })
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
	s.moveCamera(func(c *view.Camera) { c.Target = geom.V3(p.X, p.Y, 0) })
	s.touch()
	return nil
}

// AlignGame copies the editor view into the game camera.
func (s *Session) AlignGame() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.play == nil {
		s.checkpoint("")
	}
	sc := s.current()
	if err := sc.SetView(s.cam); errors.Is(err, scene.ErrNoCamera) {
		if c, err := s.reg.New("camera"); err == nil {
			sc.Add("Main Camera", c)
			_ = sc.SetView(s.cam) // the camera was just added
		}
	}
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
		d = p.World().Inward(d)
	}
	e.Local.Pos = e.Local.Pos.Add(d)
	s.changed()
	return nil
}

// Turn rotates an entity by a drag: around "z" by the angle the pointer
// sweeps about it on the ground, around "x" or "y" by the sideways distance.
func (s *Session) Turn(id uint64, axis string, fromCol, fromRow, toCol, toRow float64) error {
	return s.transformDrag(id, "turn", func(e *scene.GameObject, v view.View) error {
		w := e.World()
		switch axis {
		case "z", "free":
			a, b := ground(v, fromCol, fromRow, w.Pos.Z).Sub(w.Pos), ground(v, toCol, toRow, w.Pos.Z).Sub(w.Pos)
			e.Local.Rot.Z += (math.Atan2(b.Y, b.X) - math.Atan2(a.Y, a.X)) * 180 / math.Pi
		case "x":
			e.Local.Rot.X += (toCol - fromCol) * degPerCell
		case "y":
			e.Local.Rot.Y += (toCol - fromCol) * degPerCell
		default:
			return fmt.Errorf("turn %s: %w: %s", e.Name, ErrBadAxis, axis)
		}
		return nil
	})
}

// Stretch scales an entity by a drag, right or up to grow; "free" scales
// evenly.
func (s *Session) Stretch(id uint64, axis string, fromCol, fromRow, toCol, toRow float64) error {
	return s.transformDrag(id, "stretch", func(e *scene.GameObject, _ view.View) error {
		k := math.Exp((toCol - fromCol - (toRow-fromRow)*2) * scalePerCell)
		sc := e.Local.Scale
		switch axis {
		case "free":
			sc = sc.Scale(k)
		case "x":
			sc.X *= k
		case "y":
			sc.Y *= k
		case "z":
			sc.Z *= k
		default:
			return fmt.Errorf("stretch %s: %w: %s", e.Name, ErrBadAxis, axis)
		}
		e.Local.Scale = sc
		return nil
	})
}

const (
	degPerCell   = 3
	scalePerCell = .03
)

func (s *Session) transformDrag(id uint64, kind string, fn func(e *scene.GameObject, v view.View) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.play == nil {
		s.checkpoint(fmt.Sprintf("%s:%d", kind, id))
	}
	e, err := s.current().ByID(id)
	if err != nil {
		return err
	}
	if e.Inherited() {
		return fmt.Errorf("%s %s: %w", kind, e.Name, scene.ErrPrefabLocked)
	}
	if err := fn(e, s.view()); err != nil {
		return err
	}
	s.changed()
	return nil
}

// Drop places an asset or node kind on the ground under a cell, which is how
// assets are dragged into the viewport; kinds are those of object.
func (s *Session) Drop(kind string, col, row float64) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	at := ground(s.view(), col, row, 0)
	e, err := s.object(kind)
	if err != nil {
		return 0, err
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
	s.sctx.Input.Apply(input.Event{Key: k, Down: true, Exact: true}, s.play.Time)
	return nil
}

// KeyUp releases a key; unlike terminals, the editor knows when keys go up.
func (s *Session) KeyUp(name string) error {
	k, err := input.ParseName(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.play != nil {
		s.sctx.Input.Apply(input.Event{Key: k, Exact: true}, s.play.Time)
	}
	return nil
}
