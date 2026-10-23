package scene

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

// Geometry draws every visible node whose bounds touch the screen, tagging dots
// with the entity id for picking.
type Geometry struct{ Scene func() *Scene }

func (*Geometry) Name() string          { return "scene" }
func (*Geometry) Stage() pipeline.Stage { return pipeline.Geometry }
func (g *Geometry) Run(f *pipeline.Frame) {
	s := g.Scene()
	s.Walk(func(e *Entity, visible bool) bool {
		if !visible {
			return false
		}
		if e.Node == nil {
			return true
		}
		t := e.World()
		if b := e.Node.Bounds(t); !b.Empty() && !f.View.Visible(b) {
			return true
		}
		f.Buf.SetTag(uint32(e.ID))
		e.Node.Draw(f.Ctx, t)
		return true
	})
	f.Buf.SetTag(0)
}

type Cells struct{ Scene func() *Scene }

func (*Cells) Name() string          { return "cells" }
func (*Cells) Stage() pipeline.Stage { return pipeline.Overlay }
func (c *Cells) Run(f *pipeline.Frame) {
	c.Scene().Walk(func(e *Entity, visible bool) bool {
		if !visible {
			return false
		}
		if d, ok := e.Node.(CellDrawer); ok {
			d.DrawCells(f, e.World())
		}
		return true
	})
}

// Render binds the scene camera, time and environment, then runs the pipeline.
func Render(p *pipeline.Pipeline, f *pipeline.Frame, s *Scene) {
	f.Begin(s.Camera, s.Time, s.Env)
	p.Render(f)
}

// Pick maps a cell to the entity drawn nearest there.
func Pick(f *pipeline.Frame, s *Scene, col, row int) (*Entity, error) {
	tag := f.Buf.Pick(col, row)
	return s.ByID(uint64(tag))
}

// RenderWith draws the scene through another camera, the editor's free view.
func RenderWith(p *pipeline.Pipeline, f *pipeline.Frame, s *Scene, cam view.Camera) {
	f.Begin(cam, s.Time, s.Env)
	p.Render(f)
}
