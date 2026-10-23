package dragonfall

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func v(x, y float64) geom.Vec2 { return geom.V2(x, y) }

func rgb(r, g, b uint8) draw.Material { return draw.M(color.RGB(r, g, b)) }

// Hero is the knight, built from lit solids so the sun, lanterns, shadows
// and fog shade him like the rest of the valley. Walking blends in: legs
// swing, the body bobs and the cape streams back.
type Hero struct {
	Walking bool
	Speed   float64 // walk cycle speed multiplier
	weight  float64
}

func NewHero() *Hero { return &Hero{Speed: 1} }

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

func (h *Hero) Draw(c *draw.Context, t scene.Transform) {
	k := t.Uniform()
	dir := 1.0 // facing +X; flipped sprites face -X
	if t.Flip {
		dir = -1
	}
	at := func(x, y, z float64) geom.Vec3 { return t.Pos.Add(geom.V3(x*dir*k, y*k, z*k)) }
	box := func(x, y, z, sx, sy, sz float64, m draw.Material) {
		c.Box(geom.Footprint(at(x, y, z), geom.V3(sx*k, sy*k, sz*k)), m)
	}
	w := h.weight
	ph := c.Time * 7 * h.Speed
	swing := math.Sin(ph) * .38 * w
	bob := math.Abs(math.Sin(ph))*.09*w + .03*math.Sin(c.Time*2.6)*(1-w)
	air := t.Pos.Z > .05
	tuck := 0.0
	if air {
		tuck, swing = .35, .25
	}

	// Albedos stay dark: sun, lanterns and glow add up, and bright metal
	// would wash out to white.
	steel, dark, gold := rgb(104, 118, 132), rgb(52, 60, 72), rgb(196, 150, 70)
	leather, teal, skin := rgb(84, 56, 40), rgb(30, 104, 110), rgb(176, 128, 100)
	ground := geom.V3(t.Pos.X, t.Pos.Y, 0)
	c.Shadow(ground, 1*k*(1-math.Min(.5, t.Pos.Z*.08)), .65*k, .65)

	// Legs swing along the facing, one in front of the other in depth.
	for _, leg := range [2]struct{ y, s float64 }{{-.17, -swing}, {.17, swing}} {
		c.Cylinder(at(leg.s, leg.y, tuck), .16*k, (1.45-tuck)*k, dark, dark, 10)
		box(leg.s+.1, leg.y, tuck, .42, .26, .24, leather)
	}
	z := 1.45 + bob
	box(0, 0, z, .66, .5, .2, leather)
	c.Point(at(.2, .27, z+.1), .12*k, gold.Glow(.2))
	c.Cylinder(at(0, 0, z+.18), .32*k, .95*k, steel.Tint(1.05), steel, 14)
	box(.02, .31, z+.3, .4, .05, .72, teal)
	c.Point(at(.02, .35, z+.78), .14*k, gold.Glow(.35))
	for _, y := range [2]float64{-.34, .34} {
		c.Blob(at(0, y, z+1.0), .17*k, steel)
	}

	// Head, helmet with a visor slit, and a plume that sways.
	c.Cylinder(at(0, 0, z+1.12), .1*k, .14*k, skin, skin, 8)
	c.Blob(at(.03, 0, z+1.28), .23*k, skin)
	c.Blob(at(0, 0, z+1.4), .26*k, steel.Tint(1.1))
	box(.1, .22, z+1.36, .3, .05, .06, rgb(20, 22, 28))
	c.Cone(at(-.04, 0, z+1.62), .07*k, .45*k, rgb(170, 52, 44), 8, geom.V2(-(.18+.22*w)*dir*k+.04*math.Sin(c.Time*3), 0))

	// Shield on the camera side, sword in the far hand.
	box(.42, .42, z+.2, .1, .52, .7, gold)
	box(.47, .42, z+.25, .06, .42, .6, teal)
	c.Point(at(.5, .42, z+.55), .15*k, gold.Glow(.3))
	sx := .38 - swing*.6
	c.Blob(at(sx, -.4, z+.42), .11*k, leather)
	box(sx, -.4, z+.5, .34, .1, .07, gold)
	box(sx, -.4, z+.57, .07, .07, 1.25, rgb(176, 192, 200).Glow(.1))

	// Cape: from the shoulders down and back, streaming further when walking.
	sway := .12*math.Sin(c.Time*2.2) + .45*w + tuck
	cape := rgb(26, 76, 84)
	top0, top1 := at(-.24, -.3, z+.98), at(-.24, .3, z+.98)
	low0, low1 := at(-.45-sway, -.36, .5+tuck), at(-.45-sway, .36, .5+tuck)
	c.Tri(top0, top1, low1, cape)
	c.Tri(top0, low1, low0, cape)
}
