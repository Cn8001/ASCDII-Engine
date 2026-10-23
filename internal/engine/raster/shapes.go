package raster

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
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

func (b *Buffer) triangle(v0, v1, v2 Vertex, f Frag, shaded bool) {
	area := edge(v0, v1, v2.X, v2.Y)
	if math.Abs(area) < 1e-9 {
		return
	}
	x0 := max(0, int(math.Floor(min(v0.X, v1.X, v2.X))))
	x1 := min(b.W-1, int(math.Ceil(max(v0.X, v1.X, v2.X))))
	y0 := max(0, int(math.Floor(min(v0.Y, v1.Y, v2.Y))))
	y1 := min(b.H-1, int(math.Ceil(max(v0.Y, v1.Y, v2.Y))))
	inv := 1 / area
	for y := y0; y <= y1; y++ {
		py := float64(y) + .5
		for x := x0; x <= x1; x++ {
			px := float64(x) + .5
			w0 := edge(v1, v2, px, py) * inv
			w1 := edge(v2, v0, px, py) * inv
			w2 := 1 - w0 - w1
			if w0 < -1e-6 || w1 < -1e-6 || w2 < -1e-6 {
				continue
			}
			f.Depth = w0*v0.Depth + w1*v1.Depth + w2*v2.Depth
			f.World.X = w0*v0.WX + w1*v1.WX + w2*v2.WX
			f.World.Y = w0*v0.WY + w1*v1.WY + w2*v2.WY
			f.World.Z = w0*v0.WZ + w1*v1.WZ + w2*v2.WZ
			if shaded {
				f.Albedo = color.FromFloat(w0*v0.R+w1*v1.R+w2*v2.R, w0*v0.G+w1*v1.G+w2*v2.G, w0*v0.B+w1*v1.B+w2*v2.B)
			}
			b.Plot(x, y, f)
		}
	}
}

func edge(a, b Vertex, x, y float64) float64 { return (b.X-a.X)*(y-a.Y) - (b.Y-a.Y)*(x-a.X) }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
