package draw

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

func up() geom.Vec3    { return geom.V3(0, 0, 1) }
func south() geom.Vec3 { return geom.V3(0, 1, 0) }
func north() geom.Vec3 { return geom.V3(0, -1, 0) }
func east() geom.Vec3  { return geom.V3(1, 0, 0) }
func west() geom.Vec3  { return geom.V3(-1, 0, 0) }

func (c *Context) Box(b geom.AABB, m Material) { c.BoxFaces(b, m, m) }

func (c *Context) BoxFaces(b geom.AABB, top, side Material) {
	x0, y0, z0, x1, y1, z1 := b.Min.X, b.Min.Y, b.Min.Z, b.Max.X, b.Max.Y, b.Max.Z
	v := geom.V3
	c.Face([]geom.Vec3{v(x0, y0, z1), v(x1, y0, z1), v(x1, y1, z1), v(x0, y1, z1)}, up(), top)
	c.Face([]geom.Vec3{v(x0, y1, z0), v(x1, y1, z0), v(x1, y1, z1), v(x0, y1, z1)}, south(), side)
	c.Face([]geom.Vec3{v(x0, y0, z0), v(x1, y0, z0), v(x1, y0, z1), v(x0, y0, z1)}, north(), side)
	c.Face([]geom.Vec3{v(x0, y0, z0), v(x0, y1, z0), v(x0, y1, z1), v(x0, y0, z1)}, west(), side)
	c.Face([]geom.Vec3{v(x1, y0, z0), v(x1, y1, z0), v(x1, y1, z1), v(x1, y0, z1)}, east(), side)
}

// Prism extrudes a convex footprint, the base for walls, bridges and plinths.
func (c *Context) Prism(poly []geom.Vec2, z0, z1 float64, top, side Material) {
	if len(poly) < 3 {
		return
	}
	area := 0.0
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		area += p.X*q.Y - q.X*p.Y
	}
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		d := q.Sub(p)
		n := geom.V3(d.Y, -d.X, 0)
		if area < 0 {
			n = n.Scale(-1)
		}
		c.Face([]geom.Vec3{geom.V3(p.X, p.Y, z0), geom.V3(q.X, q.Y, z0), geom.V3(q.X, q.Y, z1), geom.V3(p.X, p.Y, z1)}, n.Norm(), side)
	}
	cap := make([]geom.Vec3, len(poly))
	for i, p := range poly {
		cap[i] = geom.V3(p.X, p.Y, z1)
	}
	c.Face(cap, up(), top)
}

func ring(center geom.Vec3, r float64, seg int) []geom.Vec3 {
	out := make([]geom.Vec3, seg)
	for i := range out {
		a := float64(i) / float64(seg) * 2 * math.Pi
		out[i] = center.Add(geom.V3(math.Cos(a)*r, math.Sin(a)*r, 0))
	}
	return out
}

// Cylinder is flat shaded per facet, which reads well at Braille resolution.
// A texture wraps once around the side, bottom to top, and spans the cap.
func (c *Context) Cylinder(base geom.Vec3, r, h float64, top, side Material, seg int) {
	seg = max(6, seg)
	lo := ring(base, r, seg)
	hi := ring(base.Add(geom.V3(0, 0, h)), r, seg)
	for i := range lo {
		j := (i + 1) % seg
		mid := lo[i].Add(lo[j]).Scale(.5).Sub(base)
		n := geom.V3(mid.X, mid.Y, 0).Norm()
		if side.Texture == nil {
			c.Face([]geom.Vec3{lo[i], lo[j], hi[j], hi[i]}, n, side)
			continue
		}
		u0, u1 := float64(i)/float64(seg), float64(i+1)/float64(seg)
		c.FaceUV([]geom.Vec3{lo[i], lo[j], hi[j], hi[i]}, []geom.Vec2{{X: u0}, {X: u1}, {X: u1, Y: 1}, {X: u0, Y: 1}}, n, side)
	}
	if top.Texture == nil {
		c.Face(hi, up(), top)
		return
	}
	c.FaceUV(hi, discUV(hi, base, r), up(), top)
}

// discUV spans a texture across a circle of radius r around center.
func discUV(pts []geom.Vec3, center geom.Vec3, r float64) []geom.Vec2 {
	out := make([]geom.Vec2, len(pts))
	for i, p := range pts {
		out[i] = geom.V2(.5+(p.X-center.X)/(2*r), .5+(p.Y-center.Y)/(2*r))
	}
	return out
}

// Cone leans its tip by lean, which is how trees sway without a rig.
func (c *Context) Cone(base geom.Vec3, r, h float64, m Material, seg int, lean geom.Vec2) {
	seg = max(5, seg)
	rim := ring(base, r, seg)
	tip := base.Add(geom.V3(lean.X, lean.Y, h))
	for i := range rim {
		j := (i + 1) % seg
		n := rim[j].Sub(rim[i]).Cross(tip.Sub(rim[i])).Norm()
		mid := rim[i].Add(rim[j]).Scale(.5).Sub(base)
		if n.Dot(geom.V3(mid.X, mid.Y, 0)) < 0 {
			n = n.Scale(-1)
		}
		if m.Texture == nil {
			c.Face([]geom.Vec3{rim[i], rim[j], tip}, n, m)
			continue
		}
		u0, u1 := float64(i)/float64(seg), float64(i+1)/float64(seg)
		c.FaceUV([]geom.Vec3{rim[i], rim[j], tip}, []geom.Vec2{{X: u0}, {X: u1}, {X: (u0 + u1) / 2, Y: 1}}, n, m)
	}
}

// Gable is a pitched roof whose ridge runs along x.
func (c *Context) Gable(b geom.AABB, roof, gable Material) {
	x0, y0, x1, y1, z0, z1 := b.Min.X, b.Min.Y, b.Max.X, b.Max.Y, b.Min.Z, b.Max.Z
	cy := (y0 + y1) / 2
	v := geom.V3
	rise, run := z1-z0, (y1-y0)/2
	c.Face([]geom.Vec3{v(x0, y1, z0), v(x1, y1, z0), v(x1, cy, z1), v(x0, cy, z1)}, v(0, rise, run).Norm(), roof)
	c.Face([]geom.Vec3{v(x0, y0, z0), v(x1, y0, z0), v(x1, cy, z1), v(x0, cy, z1)}, v(0, -rise, run).Norm(), roof)
	c.Face([]geom.Vec3{v(x0, y0, z0), v(x0, y1, z0), v(x0, cy, z1)}, west(), gable)
	c.Face([]geom.Vec3{v(x1, y0, z0), v(x1, y1, z0), v(x1, cy, z1)}, east(), gable)
}

func (c *Context) Pyramid(b geom.AABB, m Material) {
	ctr := b.Center()
	tip := geom.V3(ctr.X, ctr.Y, b.Max.Z)
	z := b.Min.Z
	pts := []geom.Vec3{geom.V3(b.Min.X, b.Min.Y, z), geom.V3(b.Max.X, b.Min.Y, z), geom.V3(b.Max.X, b.Max.Y, z), geom.V3(b.Min.X, b.Max.Y, z)}
	for i := range pts {
		a, d := pts[i], pts[(i+1)%4]
		n := d.Sub(a).Cross(tip.Sub(a)).Norm()
		mid := a.Add(d).Scale(.5).Sub(ctr)
		if n.Dot(geom.V3(mid.X, mid.Y, 0)) < 0 {
			n = n.Scale(-1)
		}
		c.Face([]geom.Vec3{a, d, tip}, n, m)
	}
}

// Blob is a sphere shaded per dot, used for canopies, bushes and rocks. A
// texture wraps once around it, bottom to top.
func (c *Context) Blob(center geom.Vec3, r float64, m Material) {
	cx, cy, cd := c.View.Project(center)
	rd := r * c.View.Zoom
	if rd < .5 {
		return
	}
	f := c.frag(m, up())
	lod := 0.0
	if m.Texture != nil {
		w, _ := m.Texture.Size()
		if rho := float64(w) / (2 * math.Pi * rd); rho > 1 {
			lod = math.Log2(rho) + .5 // the bias triangles use, so spheres shimmer no more than faces
		}
	}
	x0, x1 := int(math.Floor(cx-rd)), int(math.Ceil(cx+rd))
	y0, y1 := int(math.Floor(cy-rd)), int(math.Ceil(cy+rd))
	for y := max(0, y0); y <= min(c.Buf.H-1, y1); y++ {
		for x := max(0, x0); x <= min(c.Buf.W-1, x1); x++ {
			dx, dy := (float64(x)+.5-cx)/rd, (float64(y)+.5-cy)/rd
			d2 := dx*dx + dy*dy
			if d2 > 1 {
				continue
			}
			nz := math.Sqrt(1 - d2)
			n := c.View.Right.Scale(dx).Add(c.View.Up.Scale(-dy)).Add(c.View.ToCam.Scale(nz))
			f.Normal = packed(n)
			f.Depth = cd + nz*r
			f.World = center.Add(n.Scale(r))
			if m.Texture != nil {
				uv := m.uv(geom.V2(math.Atan2(n.Y, n.X)/(2*math.Pi)+.5, math.Asin(geom.Clamp(n.Z, -1, 1))/math.Pi+.5))
				col, solid := m.Texture.Sample(lod, uv.X, uv.Y)
				if !solid {
					continue
				}
				f.Albedo = color.Mul(col, m.Albedo)
			}
			c.Buf.Plot(x, y, f)
		}
	}
}

// Point is a camera facing disc for particles and sparks.
func (c *Context) Point(p geom.Vec3, size float64, m Material) {
	x, y, d := c.View.Project(p)
	f := c.frag(m, c.View.ToCam)
	f.Depth, f.World = d+.2, p
	c.Buf.Disc(int(math.Round(x)), int(math.Round(y)), int(size*c.View.Zoom/2), f)
}
