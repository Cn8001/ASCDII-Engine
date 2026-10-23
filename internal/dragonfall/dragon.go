package dragonfall

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/fx"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

const dragonScale = .1

func dragonRig() anim.Rig {
	return anim.Rig{Bones: []anim.Bone{
		{Name: "body", Pivot: v(0, -23)},
		{Name: "tail", Parent: "body", Pivot: v(11, -13)},
		{Name: "wingL", Parent: "body", Pivot: v(-13, -43)},
		{Name: "wingR", Parent: "body", Pivot: v(10, -42)},
		{Name: "neck", Parent: "body", Pivot: v(12, -45)},
	}}
}

func dragonSleep() anim.Motion {
	rest := anim.Pose{}
	in := anim.Pose{"body": {DY: 1.4}, "wingL": {Rot: -3}, "wingR": {Rot: 3}, "neck": {Rot: 5}, "tail": {Rot: -3}}
	return anim.Motion{Name: "sleep", Loop: true, Keys: []anim.PoseKey{{T: 0, Pose: rest}, {T: 2, Pose: in}, {T: 4, Pose: rest}}}
}

func dragonFlap() anim.Motion {
	up := anim.Pose{"wingL": {Rot: -24}, "wingR": {Rot: 24}, "body": {DY: -2.5}, "neck": {Rot: -4}, "tail": {Rot: 4}}
	down := anim.Pose{"wingL": {Rot: 18}, "wingR": {Rot: -18}, "body": {DY: 1}, "neck": {Rot: 3}, "tail": {Rot: -4}}
	return anim.Motion{Name: "flap", Loop: true, Keys: []anim.PoseKey{{T: 0, Pose: up}, {T: .45, Pose: down}, {T: .9, Pose: up}}}
}

// Dragon sleeps until Awake is set, then blends into flight and breathes fire
// that lights the scene.
type Dragon struct {
	Awake bool
	Fire  bool
	wake  float64
	rig   anim.Rig
	sleep anim.Motion
	flap  anim.Motion
	flame fx.Emitter
}

func NewDragon() *Dragon {
	f := fx.Fire()
	f.Size, f.Speed, f.Rate = .7, 9, 50
	return &Dragon{Fire: true, rig: dragonRig(), sleep: dragonSleep(), flap: dragonFlap(), flame: f}
}

func (*Dragon) Kind() string { return "df.dragon" }
func (d *Dragon) bind() *prop.Binder {
	return prop.Bind().Bool("awake", &d.Awake).Bool("fire", &d.Fire)
}
func (d *Dragon) Props() []prop.Prop        { return d.bind().Props() }
func (d *Dragon) Set(n string, v any) error { return d.bind().Set(n, v) }
func (d *Dragon) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos, geom.V3(9, 4, 10).Scale(scaleOf(t)))
}

func (d *Dragon) Tick(_ *scene.Ctx, _ *scene.Entity, dt float64) {
	target := 0.0
	if d.Awake {
		target = 1
	}
	d.wake += (target - d.wake) * math.Min(1, dt*2.5)
}

func (d *Dragon) Draw(c *draw.Context, t scene.Transform) {
	k := scaleOf(t)
	c.Shadow(t.Pos, 6*k, 3*k, .7)
	s := c.Sprite(t.Pos, dragonScale*k, t.Flip)
	if !s.Visible(160) {
		return
	}
	w := d.wake
	if w < .01 && d.Awake {
		w = 1 // editor preview without ticks still shows the awake pose
	}
	bones := d.rig.Solve(anim.Blend(d.sleep.Sample(c.Time), d.flap.Sample(c.Time), w))
	with := func(b string, fn func()) { s.Push(bones[b]); fn(); s.Pop() }
	s.Push(geom.Translate(0, -8))
	with("tail", func() {
		s.Color(rgb(70, 52, 62)).Curve(v(11, -13), v(68, -3), v(81, -35), 8)
		s.Color(rgb(168, 104, 105)).Curve(v(14, -16), v(68, -6), v(81, -35), 3)
		s.Color(rgb(171, 110, 90)).Poly(v(74, -41), v(90, -38), v(79, -27))
	})
	with("wingL", func() {
		s.Color(rgb(45, 44, 61)).Poly(v(-13, -43), v(-59, -92), v(-69, -65), v(-59, -57), v(-48, -62), v(-39, -45), v(-29, -51))
		s.Color(rgb(104, 62, 77)).Poly(v(-16, -44), v(-57, -85), v(-57, -61), v(-38, -49))
		s.Color(rgb(182, 102, 104)).Line(-12, -45, -58, -91, 2).Line(-15, -44, -69, -65, 2).Line(-16, -44, -48, -62, 2)
	})
	with("wingR", func() {
		s.Color(rgb(45, 44, 61)).Poly(v(10, -42), v(54, -88), v(68, -63), v(57, -54), v(45, -62), v(35, -44), v(28, -53))
		s.Color(rgb(104, 62, 77)).Poly(v(14, -43), v(53, -82), v(53, -60), v(32, -47))
		s.Color(rgb(182, 102, 104)).Line(11, -44, 54, -88, 2).Line(14, -42, 68, -63, 2).Line(14, -42, 45, -62, 2)
	})
	with("body", func() {
		s.Color(rgb(122, 80, 83)).Ellipse(0, -23, 27, 27)
		s.Color(rgb(157, 118, 102)).Ellipse(1, -16, 17, 21)
		for yy := -29.0; yy < -4; yy += 7 {
			s.Color(rgb(204, 156, 119)).Line(-11, yy, 12, yy+2, 1)
		}
		for _, ox := range []float64{-18, 18} {
			s.Color(rgb(85, 56, 63)).Line(ox, -8, ox*2, 2, 6)
			for claw := 0.0; claw < 3; claw++ {
				s.Color(rgb(217, 187, 139)).Line(ox*2+claw*4-4, 1, ox*2+claw*4-6, 8, 2)
			}
		}
		for i := 0.0; i < 7; i++ {
			sx, sy := -23+i*7, -41-6*math.Sin(i*.6)
			s.Color(rgb(168, 108, 91)).Poly(v(sx-4, sy+5), v(sx, sy-7), v(sx+4, sy+5))
		}
	})
	with("neck", func() {
		s.Color(rgb(120, 77, 76)).Poly(v(10, -41), v(12, -62), v(24, -75), v(44, -72), v(48, -62), v(37, -48), v(20, -43))
		s.Color(rgb(151, 91, 83)).Poly(v(20, -71), v(24, -81), v(39, -84), v(51, -76), v(45, -67), v(33, -67))
		for _, tx := range []float64{25, 36} {
			s.Color(rgb(204, 170, 134)).Poly(v(tx-2, -79), v(tx, -93), v(tx+3, -79))
		}
		s.Color(draw.M(color.RGB(246, 195, 88)).Glow(.4+.6*w)).Ellipse(41, -76, 3, 3)
		s.Color(rgb(43, 28, 38)).Dot(42, -76)
		s.Color(rgb(70, 49, 59)).Poly(v(39, -68), v(54, -64), v(39, -61))
		for i := 0.0; i < 5; i++ {
			s.Color(rgb(225, 183, 144)).Line(43+i*3, -65, 46+i*3, -62, 1)
		}
		if w > .75 && d.Fire {
			d.breathe(c, s, t)
		}
	})
}

func (d *Dragon) breathe(c *draw.Context, s *draw.Sprite, t scene.Transform) {
	l := 25 + 6*math.Sin(c.Time*7)
	s.Color(draw.M(color.RGB(218, 102, 59)).Glow(1)).Poly(v(50, -63), v(50+l, -69), v(53+l, -59), v(56, -60))
	s.Color(draw.M(color.RGB(255, 196, 86)).Glow(1)).Curve(v(53, -63), v(66, -67), v(53+l, -61), 3)
	mouth := s.World(54, -63)
	dir := 1.0
	if t.Flip {
		dir = -1
	}
	e := d.flame
	e.Dir = c.View.Right.Scale(dir).Add(geom.V3(0, 0, -.12))
	e.Each(c.Time, s.World(53+l, -62), nil, func(p fx.Particle) {
		c.Point(p.Pos, p.Size, draw.Material{Albedo: p.Color, Emissive: 1 - p.Age*.4})
	})
	c.Light(light.Point{Pos: mouth.Add(e.Dir.Scale(3)), Radius: 16, Color: color.RGB(255, 140, 64), Intensity: 1.8, Flicker: .4})
}
