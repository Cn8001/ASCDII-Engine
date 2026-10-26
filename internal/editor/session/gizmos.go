package session

import (
	"fmt"
	"math"
	"slices"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/physics"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// colliders outlines every collider in the Scene view, over everything
// else: bright on the selected object, dim elsewhere, blue for triggers.
func (s *Session) colliders(f *pipeline.Frame) {
	if s.viewMode != SceneView || s.play != nil || !s.gizmos {
		return
	}
	selected := s.selection()
	s.edit.Walk(func(o *scene.GameObject, visible bool) bool {
		if !visible {
			return false
		}
		t := o.World()
		for _, c := range o.Components {
			col, ok := c.(scene.Collider)
			if !ok || !o.Enabled(c) {
				continue
			}
			tint := color.RGB(120, 220, 120)
			if col.IsTrigger() {
				tint = color.RGB(120, 190, 240)
			}
			if !slices.Contains(selected, o.ID) {
				tint = color.Scale(tint, .55)
			}
			ring := outline(col.Shape(o, t))
			z0, z1 := col.Span(o, t)
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
		return true
	})
}

// colliderHandles are the draggable points of a collider in its object's
// local space; colliders without editable size have none.
func colliderHandles(c scene.Component) map[string]geom.Vec3 {
	switch v := c.(type) {
	case *components.BoxCollider:
		o, hx, hy := v.Offset, v.Size.X/2, v.Size.Y/2
		return map[string]geom.Vec3{
			"x+": o.Add(geom.V3(hx, 0, 0)), "x-": o.Add(geom.V3(-hx, 0, 0)),
			"y+": o.Add(geom.V3(0, hy, 0)), "y-": o.Add(geom.V3(0, -hy, 0)),
			"z+": o.Add(geom.V3(0, 0, v.Size.Z)),
		}
	case *components.CircleCollider:
		return map[string]geom.Vec3{"r": v.Offset.Add(geom.V3(v.Radius, 0, 0)), "z+": v.Offset.Add(geom.V3(0, 0, v.Height))}
	}
	return nil
}

// DragCollider moves one handle of a collider between two cell positions.
// Sides move on their own, so the opposite side stays where it is.
func (s *Session) DragCollider(id uint64, key, handle string, fromCol, fromRow, toCol, toRow float64) error {
	// The collider of id alone changes; the rest of the selection is left as is.
	none := func(*scene.GameObject) {}
	return s.transformDrag(id, "collider", func(e *scene.GameObject, v view.View) (func(*scene.GameObject), error) {
		c := e.ComponentByKey(key)
		pts := colliderHandles(c)
		at, ok := pts[handle]
		if !ok {
			return nil, fmt.Errorf("drag %s: %w: %s", key, ErrBadAxis, handle)
		}
		t := e.World()
		z := t.Apply(at).Z
		d := t.Inward(ground(v, toCol, toRow, z).Sub(ground(v, fromCol, fromRow, z)))
		_, cp := v.Pitch()
		up := -(toRow - fromRow) * raster.SubH / (v.Zoom * cp) / t.Size(geom.V3(1, 1, 1)).Z
		switch col := c.(type) {
		case *components.BoxCollider:
			switch handle {
			case "x+", "x-":
				sign := map[string]float64{"x+": 1, "x-": -1}[handle]
				col.Size.X, col.Offset.X = math.Max(.1, col.Size.X+sign*d.X), col.Offset.X+d.X/2
			case "y+", "y-":
				sign := map[string]float64{"y+": 1, "y-": -1}[handle]
				col.Size.Y, col.Offset.Y = math.Max(.1, col.Size.Y+sign*d.Y), col.Offset.Y+d.Y/2
			case "z+":
				col.Size.Z = math.Max(0, col.Size.Z+up)
			}
		case *components.CircleCollider:
			switch handle {
			case "r":
				p := t.Inward(ground(v, toCol, toRow, z).Sub(t.Pos)).Sub(col.Offset)
				col.Radius = math.Max(.05, math.Hypot(p.X, p.Y))
			case "z+":
				col.Height = math.Max(0, col.Height+up)
			}
		}
		return none, nil
	})
}

// cameraFrame outlines on the ground what the game camera shows, in the
// Scene view only.
func (s *Session) cameraFrame(f *pipeline.Frame) {
	if s.viewMode != SceneView || s.play != nil || !s.gizmos {
		return
	}
	o, v := s.edit.MainCamera()
	if o == nil {
		return
	}
	game := v.View(o.World()).View(f.Buf.W, f.Buf.H)
	tint := color.RGB(232, 176, 75)
	if o.ID != s.sel {
		tint = color.Scale(tint, .55)
	}
	frag := raster.Frag{Albedo: tint, Flags: raster.FlagUnlit, Depth: math.MaxFloat32}
	corners := [4][2]float64{{0, 0}, {float64(f.Buf.W), 0}, {float64(f.Buf.W), float64(f.Buf.H)}, {0, float64(f.Buf.H)}}
	for i, a := range corners {
		b := corners[(i+1)%4]
		ax, ay, _ := f.View.Project(game.Unproject(a[0], a[1], 0))
		bx, by, _ := f.View.Project(game.Unproject(b[0], b[1], 0))
		f.Buf.Line(int(ax), int(ay), int(bx), int(by), 1, frag)
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
