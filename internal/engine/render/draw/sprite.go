package draw

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
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
	ss     *smooth
}

// smooth is a supersampled sprite in progress: parts land in buf at k times
// the frame's resolution, covering frame dots [x0,x1) by [y0,y1).
type smooth struct {
	buf            *raster.Buffer
	x0, y0, x1, y1 int
	k              int
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

// Smooth draws the sprite supersampled when its art is finer than the dots
// it lands on: End then gives each dot the share of art it covers and its
// average color, so small parts and edges read as shapes instead of
// flickering in and out. radius bounds the art around the anchor, in art
// pixels. A sprite drawn at least one dot per art pixel is left as it is.
func (s *Sprite) Smooth(radius float64) *Sprite {
	k := min(4, int(math.Ceil(1/s.unit)))
	if k < 2 {
		return s
	}
	r := radius * s.unit
	x0, y0 := max(0, int(math.Floor(s.ax-r))), max(0, int(math.Floor(s.ay-r)))
	x1, y1 := min(s.c.Buf.W, int(math.Ceil(s.ax+r))), min(s.c.Buf.H, int(math.Ceil(s.ay+r/4)))
	if x1 <= x0 || y1 <= y0 {
		return s
	}
	buf := s.c.Scratch
	if buf == nil {
		buf = raster.New(1, 1)
	}
	w, h := (x1-x0)*k, (y1-y0)*k
	buf.Grow(w, h)
	buf.ClearRect(0, 0, w, h)
	s.ss = &smooth{buf: buf, x0: x0, y0: y0, x1: x1, y1: y1, k: k}
	return s
}

// End resolves a smoothed sprite into the frame; other sprites need no End.
func (s *Sprite) End() {
	ss := s.ss
	if ss == nil {
		return
	}
	s.ss = nil
	k := ss.k
	need := int(math.Ceil(float64(k*k) * .4))
	for y := ss.y0; y < ss.y1; y++ {
		for x := ss.x0; x < ss.x1; x++ {
			var n, em int
			var r, g, b, wx, wy float64
			var top *raster.Dot
			for sy := 0; sy < k; sy++ {
				for sx := 0; sx < k; sx++ {
					d := ss.buf.At((x-ss.x0)*k+sx, (y-ss.y0)*k+sy)
					if d == nil || d.Flags&raster.FlagEmpty != 0 {
						continue
					}
					n++
					cr, cg, cb := d.Albedo.Float()
					r, g, b, em = r+cr, g+cg, b+cb, em+int(d.Emissive)
					wx, wy = wx+float64(d.X), wy+float64(d.Y)
					if top == nil || d.Depth > top.Depth {
						top = d
					}
				}
			}
			if n < need {
				continue
			}
			f := float64(n)
			s.c.Buf.Plot(x, y, raster.Frag{
				Albedo: color.FromFloat(r/f, g/f, b/f), Emissive: uint8(em / n), Flags: top.Flags, Normal: top.Normal,
				Depth: float64(top.Depth), World: geom.V3(wx/f, wy/f, float64(top.Z)),
			})
		}
	}
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

// target is where parts land: the frame, or a smoothed sprite's scratch.
func (s *Sprite) target() *raster.Buffer {
	if s.ss != nil {
		return s.ss.buf
	}
	return s.c.Buf
}

// res is target dots per frame dot.
func (s *Sprite) res() float64 {
	if s.ss != nil {
		return float64(s.ss.k)
	}
	return 1
}

func (s *Sprite) pt(x, y float64) raster.Point {
	p := s.top().Apply(geom.V2(x, y))
	if s.flip {
		p.X = -p.X
	}
	px, py := s.ax+p.X*s.unit, s.ay+p.Y*s.unit
	if s.ss != nil {
		k := float64(s.ss.k)
		return raster.Point{X: int(math.Round((px - float64(s.ss.x0)) * k)), Y: int(math.Round((py - float64(s.ss.y0)) * k))}
	}
	return raster.Point{X: int(math.Round(px)), Y: int(math.Round(py))}
}

func (s *Sprite) frag() raster.Frag {
	f := s.c.frag(s.mat, s.normal)
	f.Depth = s.depth + float64(s.layer)*spriteLayer
	f.World = s.anchor
	f.Row0 = int(math.Round(s.ay))
	f.RowDepth, f.RowZ = s.slopeD, s.slopeZ
	if s.ss != nil {
		k := float64(s.ss.k)
		f.Row0 = int(math.Round((s.ay - float64(s.ss.y0)) * k))
		f.RowDepth, f.RowZ = s.slopeD/k, s.slopeZ/k
	}
	s.layer++
	return f
}

// px is a stroke width in target dots; smoothed strokes keep a full frame
// dot so thin lines such as a sword stay visible.
func (s *Sprite) px(w float64) int {
	k := s.res()
	return max(int(k), int(math.Round(w*s.unit*s.top().Scale()*k)))
}

func (s *Sprite) Line(x0, y0, x1, y1, w float64) *Sprite {
	a, b := s.pt(x0, y0), s.pt(x1, y1)
	s.target().Line(a.X, a.Y, b.X, b.Y, s.px(w), s.frag())
	return s
}

func (s *Sprite) Poly(pts ...geom.Vec2) *Sprite {
	out := make([]raster.Point, len(pts))
	for i, p := range pts {
		out[i] = s.pt(p.X, p.Y)
	}
	s.target().Polygon(out, s.frag())
	return s
}

func (s *Sprite) Rect(x, y, w, h float64) *Sprite {
	return s.Poly(geom.V2(x, y), geom.V2(x+w, y), geom.V2(x+w, y+h), geom.V2(x, y+h))
}

func (s *Sprite) Curve(a, ctrl, e geom.Vec2, w float64) *Sprite {
	pa, pc, pe := s.pt(a.X, a.Y), s.pt(ctrl.X, ctrl.Y), s.pt(e.X, e.Y)
	seg := max(3, int(math.Hypot(float64(pe.X-pa.X), float64(pe.Y-pa.Y))/2))
	s.target().QuadBezier(pa, pc, pe, seg, s.px(w), s.frag())
	return s
}

func (s *Sprite) Ellipse(x, y, rx, ry float64) *Sprite {
	p := s.pt(x, y)
	k := s.unit * s.top().Scale() * s.res()
	s.target().Ellipse(p.X, p.Y, int(math.Round(rx*k)), int(math.Round(ry*k)), s.frag())
	return s
}

func (s *Sprite) Dot(x, y float64) *Sprite {
	p := s.pt(x, y)
	s.target().Plot(p.X, p.Y, s.frag())
	return s
}
