package raster

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
)

type Point struct{ X, Y int }

// Vertex carries per corner depth, world position and an optional color for
// interpolated triangles.
type Vertex struct {
	X, Y  float64
	Depth float64
	WX    float64
	WY    float64
	WZ    float64
	R     float64
	G     float64
	B     float64
	U     float64 // texture coordinates, read when the frag has a texture
	V     float64
}

func (b *Buffer) Rect(x, y, w, h int, f Frag) {
	if w <= 0 || h <= 0 {
		return
	}
	x0, y0 := max(0, x), max(0, y)
	x1, y1 := min(b.W, x+w), min(b.H, y+h)
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			b.Plot(xx, yy, f)
		}
	}
}

// Line is a clipped Bresenham stroke; width is in dots.
func (b *Buffer) Line(x0, y0, x1, y1, width int, f Frag) {
	width = max(1, width)
	r := width/2 + 1
	if max(x0, x1) < -r || min(x0, x1) >= b.W+r || max(y0, y1) < -r || min(y0, y1) >= b.H+r {
		return
	}
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if width == 1 {
			b.Plot(x0, y0, f)
		} else {
			b.Disc(x0, y0, width/2, f)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (b *Buffer) Disc(cx, cy, r int, f Frag) {
	if r <= 0 {
		b.Plot(cx, cy, f)
		return
	}
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				b.Plot(cx+dx, cy+dy, f)
			}
		}
	}
}

func (b *Buffer) Ellipse(cx, cy, rx, ry int, f Frag) {
	if rx < 1 || ry < 1 {
		b.Plot(cx, cy, f)
		return
	}
	for y := max(0, cy-ry); y <= min(b.H-1, cy+ry); y++ {
		ny := float64(y-cy) / float64(ry)
		span := int(float64(rx) * math.Sqrt(math.Max(0, 1-ny*ny)))
		b.Rect(cx-span, y, span*2+1, 1, f)
	}
}

// Polygon is an even-odd scanline fill, so concave silhouettes work.
func (b *Buffer) Polygon(pts []Point, f Frag) {
	if len(pts) < 3 {
		return
	}
	lo, hi := pts[0].Y, pts[0].Y
	for _, p := range pts {
		lo, hi = min(lo, p.Y), max(hi, p.Y)
	}
	xs := make([]float64, 0, len(pts))
	for y := max(0, lo); y <= min(b.H-1, hi); y++ {
		xs = xs[:0]
		for i := range pts {
			a, n := pts[i], pts[(i+1)%len(pts)]
			if (a.Y <= y && n.Y > y) || (n.Y <= y && a.Y > y) {
				xs = append(xs, float64(a.X)+float64(y-a.Y)*float64(n.X-a.X)/float64(n.Y-a.Y))
			}
		}
		for i := 1; i < len(xs); i++ {
			for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
				xs[j], xs[j-1] = xs[j-1], xs[j]
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			l, r := int(math.Ceil(xs[i])), int(math.Floor(xs[i+1]))
			b.Rect(l, y, r-l+1, 1, f)
		}
	}
}

func (b *Buffer) Polyline(pts []Point, width int, f Frag) {
	for i := 1; i < len(pts); i++ {
		b.Line(pts[i-1].X, pts[i-1].Y, pts[i].X, pts[i].Y, width, f)
	}
}

func (b *Buffer) QuadBezier(a, c, e Point, segments, width int, f Frag) {
	segments = max(1, segments)
	last := a
	for i := 1; i <= segments; i++ {
		t := float64(i) / float64(segments)
		u := 1 - t
		p := Point{
			X: int(math.Round(u*u*float64(a.X) + 2*u*t*float64(c.X) + t*t*float64(e.X))),
			Y: int(math.Round(u*u*float64(a.Y) + 2*u*t*float64(c.Y) + t*t*float64(e.Y))),
		}
		b.Line(last.X, last.Y, p.X, p.Y, width, f)
		last = p
	}
}

// Triangle samples dot centers with edge functions and interpolates depth and
// world position, so faces of 3D solids occlude each other per dot.
func (b *Buffer) Triangle(v0, v1, v2 Vertex, f Frag) { b.triangle(v0, v1, v2, f, false) }

// TriangleShaded also blends the vertex colors, for smooth ground without
// visible tile edges.
func (b *Buffer) TriangleShaded(v0, v1, v2 Vertex, f Frag) { b.triangle(v0, v1, v2, f, true) }

// plane holds an attribute as a linear function of dot position, so the
// inner loop only adds a step per dot instead of solving barycentrics.
type plane struct{ at0, dx, dy float64 }

func newPlane(v0, v1, v2 Vertex, a0, a1, a2, inv float64) plane {
	return plane{
		at0: a0,
		dx:  ((a1-a0)*(v2.Y-v0.Y) - (a2-a0)*(v1.Y-v0.Y)) * inv,
		dy:  ((a2-a0)*(v1.X-v0.X) - (a1-a0)*(v2.X-v0.X)) * inv,
	}
}

// row returns the value at the first sample of a row, given the origin vertex.
func (p plane) row(v0 Vertex, px, py float64) float64 {
	return p.at0 + p.dx*(px-v0.X) + p.dy*(py-v0.Y)
}

func channel(v float64) uint8 {
	v = v*255 + .5
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}

func (b *Buffer) triangle(v0, v1, v2 Vertex, f Frag, shaded bool) {
	area := edge(v0, v1, v2.X, v2.Y)
	if math.Abs(area) < 1e-9 {
		return
	}
	x0 := max(0, int(math.Floor(min(v0.X, v1.X, v2.X))))
	x1 := min(b.W-1, int(math.Ceil(max(v0.X, v1.X, v2.X))))
	y0 := max(0, int(math.Floor(min(v0.Y, v1.Y, v2.Y))))
	y1 := min(b.H-1, int(math.Ceil(max(v0.Y, v1.Y, v2.Y))))
	if x0 > x1 || y0 > y1 {
		return
	}
	inv := 1 / area
	depth := newPlane(v0, v1, v2, v0.Depth, v1.Depth, v2.Depth, inv)
	wx := newPlane(v0, v1, v2, v0.WX, v1.WX, v2.WX, inv)
	wy := newPlane(v0, v1, v2, v0.WY, v1.WY, v2.WY, inv)
	wz := newPlane(v0, v1, v2, v0.WZ, v1.WZ, v2.WZ, inv)
	var cr, cg, cb, pu, pv plane
	if shaded {
		cr = newPlane(v0, v1, v2, v0.R, v1.R, v2.R, inv)
		cg = newPlane(v0, v1, v2, v0.G, v1.G, v2.G, inv)
		cb = newPlane(v0, v1, v2, v0.B, v1.B, v2.B, inv)
	}
	tex, tint, lod := f.Tex, f.Albedo, 0.0
	if tex != nil {
		pu = newPlane(v0, v1, v2, v0.U, v1.U, v2.U, inv)
		pv = newPlane(v0, v1, v2, v0.V, v1.V, v2.V, inv)
		lod = level(tex, pu, pv)
	}
	// Edge functions grow linearly in x; their per dot steps are constant.
	s0, s1 := -(v2.Y-v1.Y)*inv, -(v0.Y-v2.Y)*inv
	for y := y0; y <= y1; y++ {
		px, py := float64(x0)+.5, float64(y)+.5
		w0 := edge(v1, v2, px, py) * inv
		w1 := edge(v2, v0, px, py) * inv
		d, ex, ey, ez := depth.row(v0, px, py), wx.row(v0, px, py), wy.row(v0, px, py), wz.row(v0, px, py)
		var r, g, bl, u, v float64
		if shaded {
			r, g, bl = cr.row(v0, px, py), cg.row(v0, px, py), cb.row(v0, px, py)
		}
		if tex != nil {
			u, v = pu.row(v0, px, py), pv.row(v0, px, py)
		}
		for x := x0; x <= x1; x++ {
			if w0 >= -1e-6 && w1 >= -1e-6 && 1-w0-w1 >= -1e-6 {
				f.Depth, f.World.X, f.World.Y, f.World.Z = d, ex, ey, ez
				if shaded {
					f.Albedo = color.Color{R: channel(r), G: channel(g), B: channel(bl)}
				}
				if tex == nil {
					b.Plot(x, y, f)
				} else if c, solid := tex.Sample(lod, u, v); solid {
					base := tint
					if shaded {
						base = f.Albedo
					}
					f.Albedo = color.Mul(c, base)
					b.Plot(x, y, f)
				}
			}
			w0, w1 = w0+s0, w1+s1
			d, ex, ey, ez = d+depth.dx, ex+wx.dx, ey+wy.dx, ez+wz.dx
			if shaded {
				r, g, bl = r+cr.dx, g+cg.dx, bl+cb.dx
			}
			if tex != nil {
				u, v = u+pu.dx, v+pv.dx
			}
		}
	}
}

// Sampler is a texture as the rasterizer needs it.
type Sampler interface {
	Sample(lod, u, v float64) (color.Color, bool)
	Size() (int, int)
}

// lodBias picks a blurrier mip than the exact footprint; a Braille dot is
// coarse, and sharper mips shimmer as the camera moves.
const lodBias = .5

// level is the mip level matching how many texels one dot spans. UVs are
// linear across a triangle, so it is the same for every dot of it.
func level(t Sampler, u, v plane) float64 {
	w, h := t.Size()
	fw, fh := float64(w), float64(h)
	rho := math.Max(math.Hypot(u.dx*fw, v.dx*fh), math.Hypot(u.dy*fw, v.dy*fh))
	if rho <= 1 {
		return 0
	}
	return math.Log2(rho) + lodBias
}

func edge(a, b Vertex, x, y float64) float64 { return (b.X-a.X)*(y-a.Y) - (b.Y-a.Y)*(x-a.X) }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
