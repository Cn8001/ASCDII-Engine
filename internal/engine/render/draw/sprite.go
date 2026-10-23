package draw

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

const (
	spriteBias  = .15  // keeps feet from sinking into the ground they stand on
	spriteLayer = 1e-3 // call order stacking inside one sprite
)

// Sprite is a camera facing vertical plane standing on an anchor. Coordinates
// are art pixels with y down and the origin at the feet; Scale converts them to
// world units. Later calls draw in front of earlier ones.
type Sprite struct {
	c      *Context
	anchor geom.Vec3
	ax, ay float64
	depth  float64
	unit   float64
	flip   bool
	mat    Material
	normal geom.Vec3
	stack  []geom.Affine
	layer  int
	slopeD float64
	slopeZ float64
}

func (c *Context) Sprite(anchor geom.Vec3, scale float64, flip bool) *Sprite {
	sp, cp := c.View.Pitch()
	perRow := 1 / (c.View.Zoom * cp)
	s := &Sprite{
		c: c, normal: c.View.ToCam.Add(geom.V3(0, 0, .6)).Norm(),
		stack:  []geom.Affine{geom.Identity()},
		slopeD: -perRow * sp, slopeZ: -perRow,
	}
	return s.Move(anchor, scale, flip)
}

// Move re-targets a sprite so dense content such as grass reuses one value.
func (s *Sprite) Move(anchor geom.Vec3, scale float64, flip bool) *Sprite {
	s.anchor, s.flip, s.layer = anchor, flip, 0
	s.ax, s.ay, s.depth = s.c.View.Project(anchor)
	s.depth += spriteBias
	s.unit = scale * s.c.View.Zoom
	s.stack = s.stack[:1]
	return s
}

// World returns the point on the sprite plane under art pixel (x, y), for
// attaching particles and lights to sprite features.
func (s *Sprite) World(x, y float64) geom.Vec3 {
	p := s.top().Apply(geom.V2(x, y))
	if s.flip {
		p.X = -p.X
	}
	_, cp := s.c.View.Pitch()
	k := s.unit / s.c.View.Zoom
	return s.anchor.Add(s.c.View.Right.Scale(p.X * k)).Add(geom.V3(0, 0, -p.Y*k/cp))
}

// Visible is a cheap screen test for an art space radius around the anchor.
func (s *Sprite) Visible(radius float64) bool {
	r := radius * s.unit
	return s.ax+r >= 0 && s.ay+r >= 0 && s.ax-r < float64(s.c.Buf.W) && s.ay-r*2 < float64(s.c.Buf.H)
}

func (s *Sprite) Color(c Material) *Sprite { s.mat = c; return s }

func (s *Sprite) Normal(n geom.Vec3) *Sprite { s.normal = n.Norm(); return s }

// Push composes a local transform, typically a rig bone.
func (s *Sprite) Push(a geom.Affine) *Sprite {
	s.stack = append(s.stack, s.stack[len(s.stack)-1].Mul(a))
	return s
}

func (s *Sprite) Pop() *Sprite {
	if len(s.stack) > 1 {
		s.stack = s.stack[:len(s.stack)-1]
	}
	return s
}

func (s *Sprite) top() geom.Affine { return s.stack[len(s.stack)-1] }

func (s *Sprite) pt(x, y float64) raster.Point {
	p := s.top().Apply(geom.V2(x, y))
	if s.flip {
		p.X = -p.X
	}
	return raster.Point{X: int(math.Round(s.ax + p.X*s.unit)), Y: int(math.Round(s.ay + p.Y*s.unit))}
}

func (s *Sprite) frag() raster.Frag {
	f := s.c.frag(s.mat, s.normal)
	f.Depth = s.depth + float64(s.layer)*spriteLayer
	f.World = s.anchor
	f.Row0 = int(math.Round(s.ay))
	f.RowDepth, f.RowZ = s.slopeD, s.slopeZ
	s.layer++
	return f
}

func (s *Sprite) px(w float64) int { return max(1, int(math.Round(w*s.unit*s.top().Scale()))) }

func (s *Sprite) Line(x0, y0, x1, y1, w float64) *Sprite {
	a, b := s.pt(x0, y0), s.pt(x1, y1)
	s.c.Buf.Line(a.X, a.Y, b.X, b.Y, s.px(w), s.frag())
	return s
}

func (s *Sprite) Poly(pts ...geom.Vec2) *Sprite {
	out := make([]raster.Point, len(pts))
	for i, p := range pts {
		out[i] = s.pt(p.X, p.Y)
	}
	s.c.Buf.Polygon(out, s.frag())
	return s
}

func (s *Sprite) Rect(x, y, w, h float64) *Sprite {
	return s.Poly(geom.V2(x, y), geom.V2(x+w, y), geom.V2(x+w, y+h), geom.V2(x, y+h))
}

func (s *Sprite) Curve(a, ctrl, e geom.Vec2, w float64) *Sprite {
	pa, pc, pe := s.pt(a.X, a.Y), s.pt(ctrl.X, ctrl.Y), s.pt(e.X, e.Y)
	seg := max(3, int(math.Hypot(float64(pe.X-pa.X), float64(pe.Y-pa.Y))/2))
	s.c.Buf.QuadBezier(pa, pc, pe, seg, s.px(w), s.frag())
	return s
}

func (s *Sprite) Ellipse(x, y, rx, ry float64) *Sprite {
	p := s.pt(x, y)
	k := s.unit * s.top().Scale()
	s.c.Buf.Ellipse(p.X, p.Y, int(math.Round(rx*k)), int(math.Round(ry*k)), s.frag())
	return s
}

func (s *Sprite) Dot(x, y float64) *Sprite {
	p := s.pt(x, y)
	s.c.Buf.Plot(p.X, p.Y, s.frag())
	return s
}
