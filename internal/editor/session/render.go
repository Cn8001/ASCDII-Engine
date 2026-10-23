package session

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"math"
	"strings"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

const FrameEvent = "frame"

type Gizmo struct {
	ID      uint64       `json:"id"`
	Corners [][2]float64 `json:"corners"` // 8 box corners in cell units, bit 0 x, bit 1 y, bit 2 z
	Anchor  [2]float64   `json:"anchor"`
	Axes    [][2]float64 `json:"axes"` // screen tips of the x, y and z move handles
}

type Frame struct {
	Cols   int             `json:"cols"`
	Rows   int             `json:"rows"`
	Cells  string          `json:"cells,omitempty"` // base64 of Raw's cell bytes when frames ride in events
	Seq    uint64          `json:"seq"`
	Time   float64         `json:"time"`
	Mode   Mode            `json:"mode"`
	Millis float64         `json:"millis"`
	Passes []pipeline.Info `json:"passes"`
	Gizmo  *Gizmo          `json:"gizmo,omitempty"`
	View   string          `json:"view"`
}

// cellBytes is the wire format: 10 bytes per cell, rune u32 LE, fg rgb, bg rgb.
func cellBytes(g *cell.Grid) []byte {
	cells := g.Cells()
	b := make([]byte, len(cells)*10)
	for i, c := range cells {
		o := i * 10
		binary.LittleEndian.PutUint32(b[o:], uint32(c.Rune))
		b[o+4], b[o+5], b[o+6] = c.FG.R, c.FG.G, c.FG.B
		b[o+7], b[o+8], b[o+9] = c.BG.R, c.BG.G, c.BG.B
	}
	return b
}

func EncodeCells(g *cell.Grid) string {
	b := cellBytes(g)
	var out strings.Builder
	out.Grow(base64.StdEncoding.EncodedLen(len(b)))
	enc := base64.NewEncoder(base64.StdEncoding, &out)
	_, _ = enc.Write(b) // a strings.Builder never fails a write
	_ = enc.Close()
	return out.String()
}

func toCell(x, y float64) [2]float64 { return [2]float64{x / raster.SubW, y / raster.SubH} }

func (s *Session) gizmo(sc *scene.Scene) *Gizmo {
	e, err := sc.ByID(s.sel)
	if err != nil {
		return nil
	}
	g := &Gizmo{ID: e.ID}
	v := s.frame.View
	x, y, _ := v.Project(e.World().Pos)
	g.Anchor = toCell(x, y)
	reach := 30 / v.Zoom // handles stay about 30 dots long at any zoom
	for _, d := range []geom.Vec3{geom.V3(reach, 0, 0), geom.V3(0, reach, 0), geom.V3(0, 0, reach)} {
		ax, ay, _ := v.Project(e.World().Pos.Add(d))
		g.Axes = append(g.Axes, toCell(ax, ay))
	}
	b := e.Bounds()
	if b.Empty() {
		b = geom.Footprint(e.World().Pos, geom.V3(1, 1, 1))
	}
	for _, c := range b.Corners() {
		x, y, _ := v.Project(c)
		g.Corners = append(g.Corners, toCell(x, y))
	}
	return g
}

// Tick advances play or live preview by dt and renders when something changed.
// It returns nil when there is nothing new to show.
func (s *Session) Tick(dt float64) (*Frame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	switch {
	case s.mode == Play || s.step > 0:
		if s.step > 0 {
			dt, s.step = s.step, 0
		}
		s.sctx.Scene, s.sctx.Quit = s.play, false
		err = s.play.Update(&s.sctx, dt, s.keys)
		s.keys = s.keys[:0]
		if s.sctx.Quit {
			s.stop()
		}
		s.touch()
	case s.mode == Edit && s.live:
		s.sctx.Scene = s.edit
		err = s.edit.Preview(&s.sctx, dt)
		s.touch()
	}
	if !s.dirty {
		return nil, err
	}
	return s.render(), err
}

func (s *Session) render() *Frame {
	s.dirty = false
	sc := s.current()
	if s.viewMode == SceneView {
		scene.RenderWith(s.pipe, s.frame, sc, s.cam)
	} else {
		scene.Render(s.pipe, s.frame, sc)
	}
	var gz *Gizmo
	if s.viewMode == SceneView {
		gz = s.gizmo(sc)
	}
	s.seq++
	body := cellBytes(s.frame.Cells)
	s.raw = make([]byte, 12+len(body))
	binary.LittleEndian.PutUint32(s.raw[0:], uint32(s.frame.Cells.W))
	binary.LittleEndian.PutUint32(s.raw[4:], uint32(s.frame.Cells.H))
	binary.LittleEndian.PutUint32(s.raw[8:], uint32(s.seq))
	copy(s.raw[12:], body)
	cells := ""
	if !s.d.Binary {
		cells = EncodeCells(s.frame.Cells)
	}
	return &Frame{
		Seq: s.seq, Cols: s.frame.Cells.W, Rows: s.frame.Cells.H, Cells: cells,
		Time: sc.Time, Mode: s.mode, Millis: float64(s.pipe.Total().Microseconds()) / 1000,
		Passes: s.pipe.Infos(), Gizmo: gz, View: s.viewMode,
	}
}

// Snapshot renders now regardless of the dirty flag, for tests and first paint.
func (s *Session) Snapshot() *Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.render()
}

// Run streams frames until ctx ends.
func (s *Session) Run(ctx context.Context, fps int) error {
	t := time.NewTicker(time.Second / time.Duration(max(1, fps)))
	defer t.Stop()
	last := s.d.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			now := s.d.Now()
			dt := math.Min(now.Sub(last).Seconds(), .1)
			last = now
			f, err := s.Tick(dt)
			if err != nil && s.d.Emit != nil {
				s.d.Emit.Emit(ctx, "error", err.Error())
			}
			if f != nil && s.d.Emit != nil {
				s.d.Emit.Emit(ctx, FrameEvent, f)
			}
		}
	}
}

// Raw is the latest frame: cols, rows and sequence as u32 LE, then the cells.
// The editor serves it as a binary response, avoiding base64 and JSON.
func (s *Session) Raw() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.raw
}
