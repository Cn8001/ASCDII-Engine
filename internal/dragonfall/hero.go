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

const heroScale = 1.8

func NewHero() *Hero { return &Hero{Speed: 1} }

func (*Hero) Kind() string { return "df.hero" }
func (h *Hero) bind() *prop.Binder {
	return prop.Bind().Bool("walking", &h.Walking).Float("speed", &h.Speed, .1, 4)
}
func (h *Hero) Props() []prop.Prop        { return h.bind().Props() }
func (h *Hero) Set(n string, v any) error { return h.bind().Set(n, v) }
func (h *Hero) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(geom.V3(t.Pos.X, t.Pos.Y, 0), t.Size(geom.V3(1.8, 1.1, 5.8)))
}

func (h *Hero) Tick(_ *scene.Ctx, _ *scene.GameObject, dt float64) {
	target := 0.0
	if h.Walking {
		target = 1
	}
	h.weight += (target - h.weight) * math.Min(1, dt*8)
}

func (h *Hero) Draw(c *draw.Context, t scene.Transform) {
	// Chunky, big headed proportions: at the game's distance and pitch a
	// realistic build collapses into a few columns of dots.
	k := t.Uniform() * heroScale
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
	step := math.Sin(ph) * w
	bob := math.Abs(step)*.06 + .025*math.Sin(c.Time*2.6)*(1-w)
	tuck := 0.0
	if t.Pos.Z > .05 {
		tuck, step = .2, 0
	}

	// Albedos stay dark: sun, lanterns and glow add up, and bright metal
	// would wash out to white.
	steel, dark, gold := rgb(66, 76, 90), rgb(36, 40, 50), rgb(170, 122, 50)
	leather, teal, red := rgb(70, 44, 30), rgb(22, 92, 100), rgb(150, 36, 34)
	visor := rgb(10, 12, 16)
	ground := geom.V3(t.Pos.X, t.Pos.Y, 0)
	c.Shadow(ground, 1.1*k*(1-math.Min(.5, t.Pos.Z*.08)), .75*k, .65)

	// He faces the camera three quarters and marches: legs lift in turn
	// while the whole figure moves along X.
	for i, x := range [2]float64{-.24, .24} {
		lift := math.Max(0, step*float64(1-2*i)) * .22
		c.Cylinder(at(x, 0, lift+tuck), .2*k, (.9-tuck)*k, dark, dark, 10)
		box(x, .08, lift+tuck, .34, .5, .26, leather)
	}
	z := .88 + bob

	sway := .1*math.Sin(c.Time*2.2) + .45*w + tuck
	cape := rgb(26, 76, 84)
	top0, top1 := at(-.42, -.3, z+.95), at(.42, -.3, z+.95)
	low0, low1 := at(-.55-sway, -.5, .3+tuck), at(.55-sway, -.5, .3+tuck)
	c.Tri(top0, top1, low1, cape)
	c.Tri(top0, low1, low0, cape)

	c.Cylinder(at(0, 0, z), .46*k, 1.0*k, steel.Tint(1.05), steel, 16)
	box(0, 0, z, .98, .98, .2, leather)
	c.Point(at(-.25, .46, z+.1), .14*k, gold.Glow(.25))
	box(0, .44, z+.15, .64, .08, .9, teal)
	box(0, .49, z+.5, .1, .04, .44, gold)
	box(0, .49, z+.66, .34, .04, .1, gold)
	for _, x := range [2]float64{-.5, .5} {
		c.Blob(at(x, 0, z+.94), .27*k, steel.Tint(1.15))
	}

	// A closed great helm, larger than life so it reads as the head.
	hz := z + 1.5
	c.Blob(at(0, 0, hz), .44*k, steel.Tint(1.2))
	box(0, 0, hz-.08, .9, .9, .1, gold)
	box(0, .4, hz+.02, .56, .1, .12, visor)
	box(0, .4, hz-.24, .1, .1, .3, visor)
	c.Cone(at(0, -.05, hz+.4), .1*k, .62*k, red, 8, geom.V2(-(.24+.26*w)*dir*k+.05*math.Sin(c.Time*3), 0))

	// Shield on the leading arm, facing the camera; sword in the other.
	sy := .5 + .04*step
	box(.62, sy, z+.05, .7, .1, .95, gold)
	box(.62, sy+.04, z+.1, .58, .06, .83, red)
	box(.62, sy+.08, z+.45, .5, .04, .1, gold)
	box(.62, sy+.08, z+.18, .1, .04, .66, gold)
	hx := -.62
	c.Blob(at(hx, .2, z+.38), .15*k, leather)
	box(hx, .2, z+.48, .1, .4, .08, gold)
	box(hx, .2, z+.56, .1, .1, 1.35, rgb(176, 192, 200).Glow(.12))
}
