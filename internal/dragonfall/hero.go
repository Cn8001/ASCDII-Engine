package dragonfall

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const heroScale = .057 // 73 art pixels make a 4.2 unit tall knight

func v(x, y float64) geom.Vec2 { return geom.V2(x, y) }

func rgb(r, g, b uint8) draw.Material { return draw.M(color.RGB(r, g, b)) }

func heroRig() anim.Rig {
	return anim.Rig{Bones: []anim.Bone{
		{Name: "torso", Pivot: v(0, -25)},
		{Name: "legL", Pivot: v(-6, -24)},
		{Name: "legR", Pivot: v(6, -24)},
		{Name: "cape", Parent: "torso", Pivot: v(0, -50)},
		{Name: "head", Parent: "torso", Pivot: v(0, -50)},
		{Name: "shield", Parent: "torso", Pivot: v(-13, -42)},
		{Name: "sword", Parent: "torso", Pivot: v(13, -42)},
	}}
}

func heroIdle() anim.Motion {
	rest := anim.Pose{}
	breathe := anim.Pose{"torso": {DY: .9}, "cape": {Rot: 3}, "sword": {Rot: -3}, "head": {Rot: 1.5}}
	return anim.Motion{Name: "idle", Loop: true, Keys: []anim.PoseKey{{T: 0, Pose: rest}, {T: 1.2, Pose: breathe}, {T: 2.4, Pose: rest}}}
}

func heroWalk() anim.Motion {
	a := anim.Pose{"legL": {Rot: 24}, "legR": {Rot: -24}, "shield": {Rot: -9}, "sword": {Rot: 11}, "cape": {Rot: 7}}
	b := anim.Pose{"legL": {Rot: -24}, "legR": {Rot: 24}, "shield": {Rot: 9}, "sword": {Rot: -11}, "cape": {Rot: 4}}
	up := anim.Pose{"torso": {DY: -1.4}, "cape": {Rot: 5}}
	return anim.Motion{Name: "walk", Loop: true, Keys: []anim.PoseKey{{T: 0, Pose: a}, {T: .15, Pose: up}, {T: .3, Pose: b}, {T: .45, Pose: up}, {T: .6, Pose: a}}}
}

// Hero is the knight from Volume V, rigged: legs swing from the hips, arms and
// cape follow the torso, and walking blends in instead of snapping.
type Hero struct {
	Walking bool
	Speed   float64 // walk cycle speed multiplier
	weight  float64
	rig     anim.Rig
	idle    anim.Motion
	walk    anim.Motion
}

func NewHero() *Hero {
	return &Hero{Speed: 1, rig: heroRig(), idle: heroIdle(), walk: heroWalk()}
}

func (*Hero) Kind() string { return "df.hero" }
func (h *Hero) bind() *prop.Binder {
	return prop.Bind().Bool("walking", &h.Walking).Float("speed", &h.Speed, .1, 4)
}
func (h *Hero) Props() []prop.Prop        { return h.bind().Props() }
func (h *Hero) Set(n string, v any) error { return h.bind().Set(n, v) }
func (h *Hero) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(geom.V3(t.Pos.X, t.Pos.Y, 0), t.Size(geom.V3(1.6, 1, 4.3)))
}

func (h *Hero) Tick(_ *scene.Ctx, _ *scene.GameObject, dt float64) {
	target := 0.0
	if h.Walking {
		target = 1
	}
	h.weight += (target - h.weight) * math.Min(1, dt*8)
}

func (h *Hero) pose(t float64, airborne bool) anim.Pose {
	p := anim.Blend(h.idle.Sample(t), h.walk.Sample(t*h.Speed), h.weight)
	if airborne {
		p = anim.Blend(p, anim.Pose{"legL": {Rot: 32}, "legR": {Rot: -14}, "cape": {Rot: 14}, "shield": {Rot: -14}}, .8)
	}
	return p
}

func (h *Hero) Draw(c *draw.Context, t scene.Transform) {
	k := t.Uniform()
	ground := geom.V3(t.Pos.X, t.Pos.Y, 0)
	c.Shadow(ground, 1.2*k*(1-math.Min(.5, t.Pos.Z*.08)), .7*k, .65)
	s := c.Sprite(t.Pos, heroScale*k, t.Flip)
	if !s.Visible(90) {
		return
	}
	bones := h.rig.Solve(h.pose(c.Time, t.Pos.Z > .05))
	flap := math.Round(2.8*math.Sin(c.Time*2.2) + .9*math.Sin(c.Time*4.4+2))
	with := func(b string, fn func()) { s.Push(bones[b]); fn(); s.Pop() }
	with("cape", func() {
		s.Color(rgb(27, 110, 118)).Poly(v(-10, -51), v(10, -52), v(11, -35), v(16+flap, -9), v(-21+flap/2, -8), v(-11, -35))
		s.Color(rgb(31, 66, 82)).Poly(v(-8, -49), v(7, -47), v(8+flap/2, -17), v(-12+flap, -10))
		for i := 0.0; i < 4; i++ {
			sx := -11 + i*7
			s.Color(rgb(68, 171, 171)).Curve(v(sx, -45), v(sx+flap/2-5, -24), v(sx+flap, -10+math.Mod(i, 2)), 1)
		}
	})
	with("legL", func() {
		s.Color(rgb(91, 114, 125)).Poly(v(-10, -24), v(-2, -23), v(-3, -3), v(-13, -2))
		s.Color(rgb(52, 70, 82)).Line(-11, -8, -10, -1, 4)
		s.Color(rgb(171, 173, 167)).Line(-12, -2, -19, -1, 3)
	})
	with("legR", func() {
		s.Color(rgb(94, 118, 132)).Poly(v(2, -24), v(10, -23), v(13, -3), v(3, -2))
		s.Color(rgb(52, 70, 82)).Line(10, -8, 12, -1, 4)
		s.Color(rgb(171, 173, 167)).Line(12, -2, 19, -1, 3)
	})
	with("torso", func() {
		s.Color(rgb(91, 111, 129)).Poly(v(-11, -48), v(11, -48), v(10, -25), v(-11, -25))
		s.Color(rgb(166, 185, 194)).Poly(v(-7, -47), v(7, -47), v(6, -30), v(-7, -30))
		s.Color(rgb(234, 213, 142)).Line(0, -45, 0, -32, 2)
		s.Color(rgb(69, 191, 179)).Poly(v(0, -45), v(4, -39), v(0, -34), v(-4, -39))
		s.Color(rgb(67, 59, 61)).Line(-12, -28, 12, -28, 3)
		s.Color(rgb(225, 183, 100)).Line(-3, -28, 3, -28, 2)
		s.Color(rgb(173, 185, 194)).Ellipse(-12, -44, 7, 5)
		s.Color(rgb(185, 198, 205)).Ellipse(12, -44, 7, 5)
		s.Color(rgb(235, 222, 166)).Line(-15, -45, -9, -48, 2).Line(9, -48, 16, -45, 2)
	})
	with("shield", func() {
		s.Color(rgb(110, 136, 148)).Line(-13, -42, -19, -24, 5)
		s.Color(rgb(165, 190, 196)).Line(-19, -26, -23, -20, 4)
	})
	with("sword", func() {
		s.Color(rgb(120, 145, 151)).Line(13, -42, 20, -26, 5)
		s.Color(rgb(172, 190, 203)).Line(20, -27, 24, -22, 4)
	})
	with("head", func() {
		s.Color(rgb(195, 155, 127)).Ellipse(0, -54, 9, 10)
		s.Color(rgb(84, 95, 104)).Rect(-8, -57, 17, 4)
		s.Color(rgb(245, 224, 171)).Line(-6, -56, -3, -56, 1).Line(3, -56, 6, -56, 1)
		s.Color(rgb(132, 91, 79)).Line(0, -55, 1, -51, 1)
		s.Color(rgb(126, 155, 167)).Poly(v(-10, -59), v(-9, -66), v(-5, -71), v(6, -71), v(10, -66), v(10, -58))
		s.Color(rgb(241, 196, 103)).Poly(v(-5, -68), v(0, -73), v(4, -68))
		s.Color(rgb(207, 220, 219)).Line(-10, -59, 10, -59, 3)
		s.Color(rgb(59, 147, 161)).Line(0, -71, 0, -61, 2)
		s.Color(rgb(113, 137, 150)).Line(-10, -61, -14, -57, 2).Line(10, -61, 14, -57, 2)
	})
	with("shield", func() {
		s.Color(rgb(196, 159, 91)).Poly(v(-25, -40), v(-13, -36), v(-14, -20), v(-22, -13), v(-29, -27))
		s.Color(rgb(38, 117, 122)).Poly(v(-24, -38), v(-16, -35), v(-17, -21), v(-22, -17), v(-27, -27))
		s.Color(rgb(228, 217, 156)).Line(-22, -35, -22, -20, 2).Line(-26, -28, -18, -28, 1)
		s.Color(rgb(236, 199, 110)).Ellipse(-22, -28, 2, 3)
	})
	with("sword", func() {
		sw := math.Round(1.2 * math.Sin(c.Time*1.5))
		s.Color(rgb(191, 219, 222)).Line(24, -25, 30+sw, -62, 3)
		s.Color(rgb(248, 249, 224)).Line(25, -27, 31+sw, -61, 1)
		s.Color(rgb(224, 182, 90)).Line(19, -32, 34, -29, 3)
		s.Color(rgb(104, 76, 65)).Line(22, -27, 22, -18, 3)
		s.Color(rgb(244, 247, 231)).Dot(30+sw, -64)
	})
}
