// Package draw is the content API. Everything here takes world units and
// materials; projection, depth and normals are derived, never hand written.
package draw

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

type Material struct {
	Albedo   color.Color `json:"albedo"`
	Emissive float64     `json:"emissive"` // 0..1, light the surface gives off itself
	Unlit    bool        `json:"unlit"`
	Water    bool        `json:"water"`
}

func M(c color.Color) Material { return Material{Albedo: c} }

func (m Material) Glow(k float64) Material {
	m.Emissive = k
	return m
}

func (m Material) Tint(k float64) Material {
	m.Albedo = color.Scale(m.Albedo, k)
	return m
}

// Caster is a soft contact shadow resolved later against whatever ground was drawn.
type Caster struct {
	At       geom.Vec3
	RX, RY   float64
	Strength float64
}

type Context struct {
	Buf     *raster.Buffer
	View    view.View
	Time    float64
	Lights  []light.Point
	Shadows []Caster
}

func (c *Context) Light(p light.Point) { c.Lights = append(c.Lights, p) }

func (c *Context) Shadow(at geom.Vec3, rx, ry, strength float64) {
	c.Shadows = append(c.Shadows, Caster{At: at, RX: rx, RY: ry, Strength: strength})
}

func (c *Context) frag(m Material, n geom.Vec3) raster.Frag {
	f := raster.Frag{Albedo: m.Albedo, Normal: raster.PackNormal(n)}
	if m.Emissive > 0 {
		f.Emissive = uint8(geom.Clamp(m.Emissive, 0, 1) * 255)
	}
	if m.Unlit {
		f.Flags |= raster.FlagUnlit
	}
	if m.Water {
		f.Flags |= raster.FlagWater
	}
	return f
}

func (c *Context) vertex(p geom.Vec3) raster.Vertex {
	x, y, d := c.View.Project(p)
	return raster.Vertex{X: x, Y: y, Depth: d, WX: p.X, WY: p.Y, WZ: p.Z}
}

// Face draws a convex polygon with a known outward normal, skipping it when it
// faces away from the camera.
func (c *Context) Face(pts []geom.Vec3, n geom.Vec3, m Material) {
	if len(pts) < 3 || n.Dot(c.View.ToCam) <= 1e-6 {
		return
	}
	f := c.frag(m, n)
	v0 := c.vertex(pts[0])
	prev := c.vertex(pts[1])
	for i := 2; i < len(pts); i++ {
		cur := c.vertex(pts[i])
		c.Buf.Triangle(v0, prev, cur, f)
		prev = cur
	}
}

// FaceShaded draws a convex face whose color blends between its corners.
func (c *Context) FaceShaded(pts []geom.Vec3, cols []color.Color, n geom.Vec3, m Material) {
	if len(pts) < 3 || len(cols) != len(pts) || n.Dot(c.View.ToCam) <= 1e-6 {
		return
	}
	f := c.frag(m, n)
	v := func(i int) raster.Vertex {
		out := c.vertex(pts[i])
		out.R, out.G, out.B = cols[i].Float()
		return out
	}
	v0, prev := v(0), v(1)
	for i := 2; i < len(pts); i++ {
		cur := v(i)
		c.Buf.TriangleShaded(v0, prev, cur, f)
		prev = cur
	}
}

// Tri is two sided; the normal is flipped toward the camera when needed.
func (c *Context) Tri(a, b, d geom.Vec3, m Material) {
	n := b.Sub(a).Cross(d.Sub(a)).Norm()
	if n.Dot(c.View.ToCam) < 0 {
		n = n.Scale(-1)
	}
	c.Buf.Triangle(c.vertex(a), c.vertex(b), c.vertex(d), c.frag(m, n))
}
