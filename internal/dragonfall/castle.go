package dragonfall

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

// Castle is anchored at the middle of the gate. Every part below is in the
// same world units as its collision shape.
type Castle struct {
	Stone   color.Color
	Roof    color.Color
	Banner  color.Color
	Torches bool
}

func NewCastle() *Castle {
	return &Castle{Stone: color.RGB(118, 120, 124), Roof: color.RGB(62, 82, 102), Banner: color.RGB(176, 56, 62), Torches: true}
}

func (*Castle) Kind() string { return "df.castle" }
func (k *Castle) bind() *prop.Binder {
	return prop.Bind().Color("stone", &k.Stone).Color("roof", &k.Roof).Color("banner", &k.Banner).Bool("torches", &k.Torches)
}
func (k *Castle) Props() []prop.Prop        { return k.bind().Props() }
func (k *Castle) Set(n string, v any) error { return k.bind().Set(n, v) }

func scaleOf(t scene.Transform) float64 {
	if t.Scale == 0 {
		return 1
	}
	return t.Scale
}

func (k *Castle) Bounds(t scene.Transform) geom.AABB {
	s := scaleOf(t)
	return geom.AABB{Min: geom.V3(-21, -14, 0), Max: geom.V3(21, 1.5, 28.5)}.Scaled(s).Translate(t.Pos)
}

type part struct{ lo, hi geom.Vec3 }

func castleWalls() []part {
	return []part{
		{geom.V3(-15, -4, 0), geom.V3(15, 0, 8)},
		{geom.V3(-7, -13, 0), geom.V3(7, -5, 16)},
	}
}

func castleTowers() []geom.Vec3 { return []geom.Vec3{geom.V3(-17, -2, 0), geom.V3(17, -2, 0)} }

func (k *Castle) Draw(c *draw.Context, t scene.Transform) {
	s := scaleOf(t)
	at := func(v geom.Vec3) geom.Vec3 { return t.Pos.Add(v.Scale(s)) }
	box := func(lo, hi geom.Vec3) geom.AABB { return geom.AABB{Min: at(lo), Max: at(hi)} }
	stone, top := draw.M(k.Stone), draw.M(color.Scale(k.Stone, 1.15))
	for _, p := range castleWalls() {
		c.BoxFaces(box(p.lo, p.hi), top, stone)
	}
	for x := -14.5; x <= 14.6; x += 2 {
		c.BoxFaces(box(geom.V3(x-.5, -1, 8), geom.V3(x+.5, 0, 9.2)), top, stone)
		c.BoxFaces(box(geom.V3(x-.5, -4, 8), geom.V3(x+.5, -3, 9.2)), top, stone)
	}
	c.Gable(box(geom.V3(-7.6, -13.6, 16), geom.V3(7.6, -4.4, 22)), draw.M(k.Roof), stone)
	window := draw.M(color.RGB(250, 196, 110)).Glow(.9)
	for _, x := range []float64{-4, 0, 4} {
		c.Box(box(geom.V3(x-.5, -5.05, 11.5), geom.V3(x+.5, -4.95, 13.6)), window)
	}
	for i, tw := range castleTowers() {
		c.Cylinder(at(tw), 3.4*s, 13*s, top, draw.M(color.Scale(k.Stone, 1.05)), 16)
		c.Cone(at(tw.Add(geom.V3(0, 0, 13))), 4.1*s, 6.5*s, draw.M(k.Roof), 16, geom.Vec2{})
		k.pennant(c, at(tw.Add(geom.V3(0, 0, 19.3))), s, float64(i))
		slit := c.Sprite(at(tw.Add(geom.V3(0, 3.4, 0))), .1*s, false)
		slit.Color(window).Rect(-.6*10, -90, 12, 22).Rect(-.6*10, -50, 12, 18)
		c.Shadow(at(tw), 4*s, 3*s, .6)
	}
	c.Box(box(geom.V3(-2.6, -.1, 0), geom.V3(2.6, .08, 5)), draw.M(color.RGB(28, 26, 28)))
	gate := c.Sprite(at(geom.V3(0, .12, 0)), .1*s, false)
	gate.Color(draw.M(color.RGB(128, 132, 136)))
	for x := -20.0; x <= 20; x += 8 {
		gate.Line(x, 0, x, -46, 1)
	}
	for y := -40.0; y < 0; y += 10 {
		gate.Line(-22, y, 22, y, 1)
	}
	c.Shadow(at(geom.V3(0, -4, 0)), 17*s, 6*s, .55)
	if k.Torches {
		for _, x := range []float64{-4.2, 4.2} {
			k.torch(c, at(geom.V3(x, .7, 0)), s)
		}
	}
}

func (k *Castle) pennant(c *draw.Context, top geom.Vec3, s, phase float64) {
	p := c.Sprite(top, .1*s, false)
	wave := 4 * math.Sin(c.Time*2.4+phase*1.7)
	p.Color(draw.M(color.RGB(210, 198, 154))).Line(0, 0, 0, -26, 1)
	p.Color(draw.M(k.Banner)).Poly(geom.V2(0, -26), geom.V2(22+wave, -22+wave*.3), geom.V2(18+wave*.6, -18), geom.V2(0, -15))
}

func (k *Castle) torch(c *draw.Context, base geom.Vec3, s float64) {
	wood := draw.M(color.RGB(104, 76, 52))
	c.Cylinder(base, .18*s, 3.2*s, wood, wood, 6)
	flick := math.Sin(c.Time*9+base.X) * 1.5
	f := c.Sprite(base.Add(geom.V3(0, 0, 3.2*s)), .1*s, false)
	f.Color(draw.M(color.RGB(232, 120, 52)).Glow(1)).Ellipse(flick*.5, -6, 4, 7)
	f.Color(draw.M(color.RGB(255, 214, 120)).Glow(1)).Ellipse(flick*.3, -7, 2, 4)
	c.Light(light.Point{Pos: base.Add(geom.V3(0, 1, 3.8*s)), Radius: 11 * s, Color: color.RGB(255, 160, 84), Intensity: 1.5, Flicker: .3})
}

func (k *Castle) Blocks(t scene.Transform, p geom.Vec2) bool {
	s := scaleOf(t)
	l := p.Sub(t.Pos.XY()).Scale(1 / s)
	for _, w := range castleWalls() {
		if l.X >= w.lo.X && l.X < w.hi.X && l.Y >= w.lo.Y && l.Y < w.hi.Y {
			return true
		}
	}
	for _, tw := range castleTowers() {
		if l.Sub(tw.XY()).Len() < 3.4 {
			return true
		}
	}
	return false
}
