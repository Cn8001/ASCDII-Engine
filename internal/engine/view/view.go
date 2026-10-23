// Package view owns the 2.5D camera. Ground y foreshortens by sin(pitch) and
// height z rises by cos(pitch), which gives the classic three quarter look
// while keeping every object in plain world units.
package view

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
)

type Camera struct {
	Target  geom.Vec3 `json:"target"`
	Span    float64   `json:"span"`  // world units across the viewport width
	Pitch   float64   `json:"pitch"` // degrees, 90 looks straight down
	Yaw     float64   `json:"yaw"`   // degrees around the up axis
	AnchorX float64   `json:"anchorX"`
	AnchorY float64   `json:"anchorY"`
}

func Default() Camera {
	return Camera{Span: 64, Pitch: 55, AnchorX: .5, AnchorY: .6}
}

func (c *Camera) bind() *prop.Binder {
	return prop.Bind().
		Vec3("target", &c.Target).
		Float("span", &c.Span, 4, 400).
		Float("pitch", &c.Pitch, 15, 89).
		Float("yaw", &c.Yaw, -180, 180).
		Float("anchorX", &c.AnchorX, 0, 1).
		Float("anchorY", &c.AnchorY, 0, 1)
}

func (c *Camera) Props() []prop.Prop           { return c.bind().Props() }
func (c *Camera) Set(name string, v any) error { return c.bind().Set(name, v) }

// Follow eases toward a target with frame rate independent damping.
func (c *Camera) Follow(target geom.Vec3, dt, rate float64) {
	k := 1 - math.Exp(-rate*dt)
	c.Target = c.Target.Lerp(target, k)
}

// View is a camera bound to a viewport size in dots, with trig precomputed.
type View struct {
	W, H   int
	Zoom   float64 // dots per world unit
	Target geom.Vec3
	Right  geom.Vec3
	Up     geom.Vec3 // world direction that moves up on screen
	ToCam  geom.Vec3
	ax, ay float64
	sp, cp float64
	sy, cy float64
}

func (c Camera) View(w, h int) View {
	span := c.Span
	if span <= 0 {
		span = Default().Span
	}
	pitch := geom.Clamp(c.Pitch, 15, 89) * math.Pi / 180
	yaw := c.Yaw * math.Pi / 180
	v := View{W: w, H: h, Zoom: float64(w) / span, Target: c.Target, ax: float64(w) * c.AnchorX, ay: float64(h) * c.AnchorY}
	v.sp, v.cp = math.Sincos(pitch)
	v.sy, v.cy = math.Sincos(yaw)
	ground := geom.V3(-v.sy, v.cy, 0)
	v.Right = geom.V3(v.cy, v.sy, 0)
	v.ToCam = ground.Scale(v.cp).Add(geom.V3(0, 0, v.sp))
	v.Up = geom.V3(0, 0, v.cp).Sub(ground.Scale(v.sp))
	return v
}

func (v View) Pitch() (sin, cos float64) { return v.sp, v.cp }

// Project returns dot coordinates and a depth that grows toward the camera.
func (v View) Project(p geom.Vec3) (x, y, depth float64) {
	d := p.Sub(v.Target)
	rx := d.X*v.cy + d.Y*v.sy
	ry := -d.X*v.sy + d.Y*v.cy
	return v.ax + rx*v.Zoom, v.ay + (ry*v.sp-d.Z*v.cp)*v.Zoom, ry*v.cp + d.Z*v.sp
}

// Unproject finds the world point on the horizontal plane at height z.
func (v View) Unproject(x, y, z float64) geom.Vec3 {
	dz := z - v.Target.Z
	rx := (x - v.ax) / v.Zoom
	ry := ((y-v.ay)/v.Zoom + dz*v.cp) / v.sp
	return geom.V3(v.Target.X+rx*v.cy-ry*v.sy, v.Target.Y+rx*v.sy+ry*v.cy, z)
}

// GroundRect is the world rectangle visible on the z=0 plane, grown by margin.
func (v View) GroundRect(margin float64) (lo, hi geom.Vec2) {
	lo, hi = geom.V2(math.Inf(1), math.Inf(1)), geom.V2(math.Inf(-1), math.Inf(-1))
	for _, c := range [][2]float64{{0, 0}, {float64(v.W), 0}, {0, float64(v.H)}, {float64(v.W), float64(v.H)}} {
		p := v.Unproject(c[0], c[1], 0)
		lo = geom.V2(math.Min(lo.X, p.X), math.Min(lo.Y, p.Y))
		hi = geom.V2(math.Max(hi.X, p.X), math.Max(hi.Y, p.Y))
	}
	return lo.Sub(geom.V2(margin, margin)), hi.Add(geom.V2(margin, margin))
}

// Visible reports whether any part of the box can land on screen.
func (v View) Visible(b geom.AABB) bool {
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range b.Corners() {
		x, y, _ := v.Project(c)
		x0, y0, x1, y1 = math.Min(x0, x), math.Min(y0, y), math.Max(x1, x), math.Max(y1, y)
	}
	return x1 >= 0 && y1 >= 0 && x0 < float64(v.W) && y0 < float64(v.H)
}
