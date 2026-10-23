package scene

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
)

// Geometry draws every visible node whose bounds touch the screen, tagging dots
// with the entity id for picking.
type Geometry struct{ Scene func() *Scene }

func (*Geometry) Name() string          { return "scene" }
func (*Geometry) Stage() pipeline.Stage { return pipeline.Geometry }
func (g *Geometry) Run(f *pipeline.Frame) {
	s := g.Scene()
	s.Walk(func(e *GameObject, visible bool) bool {
		if !visible {
			return false
		}
		t := e.World()
		f.Buf.SetTag(uint32(e.ID))
		for _, c := range e.Components {
			r, ok := c.(Renderer)
			if !ok || !e.Enabled(c) {
				continue
			}
			if b, ok := c.(Bounder); ok {
				if bb := b.Bounds(t); !bb.Empty() && !f.View.Visible(bb) {
					continue
				}
			}
			r.Draw(f.Ctx, t)
		}
		return true
	})
	f.Buf.SetTag(0)
}

// Painted applies paint layers to the dots their objects drew.
type Painted struct{ Scene func() *Scene }

func (*Painted) Name() string          { return "paint" }
func (*Painted) Stage() pipeline.Stage { return pipeline.Geometry }
func (p *Painted) Run(f *pipeline.Frame) {
	p.Scene().Walk(func(o *GameObject, visible bool) bool {
		if !visible {
			return false
		}
		for _, c := range o.Components {
			if d, ok := c.(DotPainter); ok && o.Enabled(c) {
				d.PaintDots(f, o, o.World())
			}
		}
		return true
	})
}

type Cells struct{ Scene func() *Scene }

func (*Cells) Name() string          { return "cells" }
func (*Cells) Stage() pipeline.Stage { return pipeline.Overlay }
func (c *Cells) Run(f *pipeline.Frame) {
	c.Scene().Walk(func(e *GameObject, visible bool) bool {
		if !visible {
			return false
		}
		for _, c := range e.Components {
			if d, ok := c.(CellDrawer); ok && e.Enabled(c) {
				d.DrawCells(f, e.World())
			}
		}
		return true
	})
}

// Render binds the game camera, time and environment, then runs the
// pipeline; a scene without a camera shows a notice instead.
func Render(p *pipeline.Pipeline, f *pipeline.Frame, s *Scene) {
	c, ok := s.View()
	if !ok {
		noCamera(f)
		return
	}
	f.Begin(c, s.Time, s.Env)
	p.Render(f)
}

func noCamera(f *pipeline.Frame) {
	bg := color.RGB(14, 14, 14)
	f.Buf.Clear(bg)
	f.Cells.Clear(cell.Cell{Rune: ' ', FG: bg, BG: bg})
	msg := "No camera"
	f.Cells.Text((f.Cells.W-len(msg))/2, f.Cells.H/2, msg, color.RGB(150, 146, 140), bg)
}

// Pick maps a cell to the entity drawn nearest there.
func Pick(f *pipeline.Frame, s *Scene, col, row int) (*GameObject, error) {
	tag := f.Buf.Pick(col, row)
	return s.ByID(uint64(tag))
}

// RenderWith draws the scene through another camera, the editor's free view.
func RenderWith(p *pipeline.Pipeline, f *pipeline.Frame, s *Scene, cam view.Camera) {
	f.Begin(cam, s.Time, s.Env)
	p.Render(f)
}
