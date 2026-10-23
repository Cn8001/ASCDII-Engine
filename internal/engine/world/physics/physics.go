// Package physics is the 2D geometry of colliders: shapes on the ground
// plane and the overlap test that pushes them apart. Height is handled by
// the bodies themselves, which keeps this package small.
package physics

import (
	"math"
	"sort"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

// Shape is a Circle or a convex Poly.
type Shape interface {
	center() geom.Vec2
	project(axis geom.Vec2) (float64, float64)
}

type Circle struct {
	C geom.Vec2
	R float64
}

// Poly is convex; corners may wind either way.
type Poly struct{ Pts []geom.Vec2 }

func (c Circle) center() geom.Vec2 { return c.C }
func (c Circle) project(a geom.Vec2) (float64, float64) {
	d := c.C.Dot(a)
	return d - c.R, d + c.R
}

func (p Poly) center() geom.Vec2 {
	var s geom.Vec2
	for _, q := range p.Pts {
		s = s.Add(q)
	}
	return s.Scale(1 / float64(max(1, len(p.Pts))))
}

func (p Poly) project(a geom.Vec2) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, q := range p.Pts {
		d := q.Dot(a)
		lo, hi = math.Min(lo, d), math.Max(hi, d)
	}
	return lo, hi
}

// Box is a rectangle around center with half extents, turned by angle (radians).
func Box(center, half geom.Vec2, angle float64) Poly {
	corners := []geom.Vec2{{X: -half.X, Y: -half.Y}, {X: half.X, Y: -half.Y}, {X: half.X, Y: half.Y}, {X: -half.X, Y: half.Y}}
	for i, c := range corners {
		corners[i] = center.Add(c.Rotate(angle))
	}
	return Poly{Pts: corners}
}

// Hull is the convex hull of points, the footprint of a mesh collider.
func Hull(pts []geom.Vec2) Poly {
	p := append([]geom.Vec2(nil), pts...)
	sort.Slice(p, func(i, j int) bool { return p[i].X < p[j].X || (p[i].X == p[j].X && p[i].Y < p[j].Y) })
	if len(p) < 3 {
		return Poly{Pts: p}
	}
	cross := func(o, a, b geom.Vec2) float64 { return (a.X-o.X)*(b.Y-o.Y) - (a.Y-o.Y)*(b.X-o.X) }
	var h []geom.Vec2
	for pass := 0; pass < 2; pass++ {
		start := len(h)
		for _, q := range p {
			for len(h) >= start+2 && cross(h[len(h)-2], h[len(h)-1], q) <= 0 {
				h = h[:len(h)-1]
			}
			h = append(h, q)
		}
		h = h[:len(h)-1]
		for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
			p[i], p[j] = p[j], p[i]
		}
	}
	return Poly{Pts: h}
}

func axes(s Shape, other Shape) []geom.Vec2 {
	switch v := s.(type) {
	case Poly:
		out := make([]geom.Vec2, 0, len(v.Pts))
		for i, a := range v.Pts {
			b := v.Pts[(i+1)%len(v.Pts)]
			e := b.Sub(a)
			out = append(out, geom.V2(-e.Y, e.X).Norm())
		}
		return out
	case Circle:
		if p, ok := other.(Poly); ok {
			best, bd := geom.Vec2{}, math.Inf(1)
			for _, q := range p.Pts {
				if d := q.Sub(v.C).Len(); d < bd {
					best, bd = q, d
				}
			}
			return []geom.Vec2{best.Sub(v.C).Norm()}
		}
	}
	return nil
}

// Overlap reports whether a and b intersect and the shortest push that
// moves a out of b.
func Overlap(a, b Shape) (geom.Vec2, bool) {
	if ca, ok := a.(Circle); ok {
		if cb, ok := b.(Circle); ok {
			d := ca.C.Sub(cb.C)
			dist, r := d.Len(), ca.R+cb.R
			if dist >= r {
				return geom.Vec2{}, false
			}
			if dist == 0 {
				return geom.V2(r, 0), true
			}
			return d.Scale((r - dist) / dist), true
		}
	}
	best, depth := geom.Vec2{}, math.Inf(1)
	for _, ax := range append(axes(a, b), axes(b, a)...) {
		if ax == (geom.Vec2{}) {
			continue
		}
		alo, ahi := a.project(ax)
		blo, bhi := b.project(ax)
		o := math.Min(ahi, bhi) - math.Max(alo, blo)
		if o <= 0 {
			return geom.Vec2{}, false
		}
		if o < depth {
			best, depth = ax, o
		}
	}
	if math.IsInf(depth, 1) {
		return geom.Vec2{}, false
	}
	if a.center().Sub(b.center()).Dot(best) < 0 {
		best = best.Scale(-1)
	}
	return best.Scale(depth), true
}

// Contains reports whether p lies inside s, what blocking a walker needs.
func Contains(s Shape, p geom.Vec2) bool {
	_, ok := Overlap(Circle{C: p, R: 1e-6}, s)
	return ok
}
