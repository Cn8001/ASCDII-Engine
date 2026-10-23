package session

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/physics"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// colliders outlines the selected object's colliders in the Scene view, at
// their bottom and top, over everything else.
func (s *Session) colliders(f *pipeline.Frame) {
	if s.viewMode != SceneView || s.play != nil || s.sel == 0 {
		return
	}
	o, err := s.edit.ByID(s.sel)
	if err != nil {
		return
	}
	t := o.World()
	for _, c := range o.Components {
		col, ok := c.(scene.Collider)
		if !ok || !o.Enabled(c) {
			continue
		}
		ring := outline(col.Shape(o, t))
		z0, z1 := col.Span(o, t)
		tint := color.RGB(120, 220, 120)
		if col.IsTrigger() {
			tint = color.RGB(120, 190, 240)
		}
		frag := raster.Frag{Albedo: tint, Flags: raster.FlagUnlit, Depth: math.MaxFloat32}
		line := func(a, b geom.Vec3) {
			ax, ay, _ := f.View.Project(a)
			bx, by, _ := f.View.Project(b)
			f.Buf.Line(int(ax), int(ay), int(bx), int(by), 1, frag)
		}
		for i, p := range ring {
			q := ring[(i+1)%len(ring)]
			line(geom.V3(p.X, p.Y, z0), geom.V3(q.X, q.Y, z0))
			line(geom.V3(p.X, p.Y, z1), geom.V3(q.X, q.Y, z1))
			if i%max(1, len(ring)/4) == 0 {
				line(geom.V3(p.X, p.Y, z0), geom.V3(p.X, p.Y, z1))
			}
		}
	}
}

// outline is a shape's border; circles become 24 sided rings.
func outline(s physics.Shape) []geom.Vec2 {
	switch v := s.(type) {
	case physics.Poly:
		return v.Pts
	case physics.Circle:
		out := make([]geom.Vec2, 24)
		for i := range out {
			a := float64(i) / 24 * 2 * math.Pi
			out[i] = v.C.Add(geom.V2(math.Cos(a), math.Sin(a)).Scale(v.R))
		}
		return out
	}
	return nil
}
