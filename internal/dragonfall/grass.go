package dragonfall

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/realm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/vegetation"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/noise"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const bladeScale = .06 // world units per blade art pixel, matching the hero

// Grass draws wind blown blades rooted in world tiles; whoever has a player
// behaviour pushes them aside.
type Grass struct {
	Seed    int
	Density float64
	Detail  bool
	cache   *realm.Cache
	field   *vegetation.Field
}

func (*Grass) Kind() string { return "df.grass" }
func (g *Grass) bind() *prop.Binder {
	return prop.Bind().Int("seed", &g.Seed, 0, 99999).Float("density", &g.Density, 0, 1).Bool("detail", &g.Detail)
}
func (g *Grass) Props() []prop.Prop        { return g.bind().Props() }
func (g *Grass) Set(n string, v any) error { return g.bind().Set(n, v) }

func (g *Grass) Tick(c *scene.Ctx, _ *scene.GameObject, dt float64) {
	g.field.Step(dt)
	c.Scene.Walk(func(e *scene.GameObject, _ bool) bool {
		p, ok := e.Component("player").(*components.Player)
		if ok && p.Moving() && e.Local.Pos.Z < .3 {
			dx := 1.0
			if e.Local.Flip {
				dx = -1
			}
			g.field.Disturb(int(e.Local.Pos.X), int(e.Local.Pos.Y), dx*dt*20)
		}
		return true
	})
}

func (g *Grass) Draw(c *draw.Context, _ scene.Transform) {
	r := g.cache.Get(uint32(g.Seed))
	lo, hi := c.View.GroundRect(1)
	step := 1
	if (hi.X-lo.X)*(hi.Y-lo.Y) > 9000 {
		step = 2 // zoomed far out, blades are below a dot anyway
	}
	s := c.Sprite(geom.Vec3{}, bladeScale, false)
	for wy := int(lo.Y); wy <= int(hi.Y); wy += step {
		for wx := int(lo.X); wx <= int(hi.X); wx += step {
			t := r.At(wx, wy)
			switch t.Kind {
			case realm.River, realm.Bridge, realm.Path, realm.Crag, realm.Ruin:
				continue
			}
			seed := noise.Hash(wx, wy, 8814)
			if seed%7 < 2 || noise.Unit(seed>>9) > g.Density {
				continue
			}
			n := 2
			if g.Detail && seed%3 == 0 {
				n = 3
			}
			bend := g.field.BendAt(wx, wy, c.Time)
			for i := 0; i < n; i++ {
				g.blade(c, s, wx, wy, i, bend, t.Kind)
			}
		}
	}
}

func (g *Grass) blade(c *draw.Context, s *draw.Sprite, wx, wy, i int, bend float64, kind realm.Kind) {
	h := noise.Hash(wx*9+i*17, wy*13+i*7, 9879)
	ox := float64(int((h>>2)%5)-2) * .2
	oy := float64(int((h>>6)%4)-1) * .25
	s.Move(geom.V3(float64(wx)+.5+ox, float64(wy)+.5+oy, 0), bladeScale, false)
	if !s.Visible(20) {
		return
	}
	height := 4 + float64((h>>10)%7)
	// Slow, small sway: fast motion of one dot wide blades reads as flicker.
	lean := bend*.55 + .8*math.Sin(c.Time*1.1+float64(wx)*.21+float64(wy)*.07)
	tip := math.Round(lean*height/9*2) / 2
	// Blades stay close to the ground tone so they texture rather than speckle.
	shade := float64((h>>17)%24) / 24
	col := color.Mix(color.RGB(70, 108, 58), color.RGB(98, 138, 70), shade)
	if kind == realm.Grove {
		col = color.Mix(color.RGB(70, 104, 104), color.RGB(92, 132, 128), shade)
	}
	if h%97 == 0 {
		col = color.RGB(176, 158, 98)
	}
	s.Color(draw.M(col)).Curve(geom.V2(0, 0), geom.V2(lean*.4, -height/2), geom.V2(tip, -height), 1)
}
