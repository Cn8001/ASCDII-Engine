package dragonfall

import (
	"errors"
	"fmt"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/fx"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const (
	LanternPrefab = "prefabs/lantern.prefab.json"
	StartScene    = "scenes/dragonfall.scene.json"
	CrystalScript = "scripts/crystal.lua"
	DragonScript  = "scripts/dragon.lua"
)

const crystalLua = `-- The moon crystal greets the knight and glows brighter up close.
-- Fields show up in the Inspector; drop the hero there or leave it empty.
fields = {
  hero = object(),
  reach = range(6, 1, 20),
  message = "the moon remembers",
}

function update(self, dt)
  local hero = self.hero or find("hero")
  if not hero then return end
  local glow = self:component("blob")
  if self:distance(hero) < self.reach then
    say(self.message, 0.5)
    glow.glow = 1
  else
    glow.glow = 0.6
  end
end
`

const dragonLua = `-- A hint while the dragon sleeps and the knight is close.
fields = {
  reach = range(14, 2, 40),
  hint = "press E",
}

function update(self, dt)
  local hero = find("hero")
  local dragon = self:component("df.dragon")
  if hero and not dragon.awake and self:distance(hero) < self.reach then
    say(self.hint, 0.4)
  end
end
`

func (b *builder) script(file string) scene.Component {
	beh, err := b.reg.New("script")
	if err == nil {
		err = beh.Set("file", file)
	}
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("build dragonfall: %w", err))
	}
	return beh
}

// TemplateGUID is the fixed id of a template asset; projects copied from the
// template keep it, so their scenes reference assets by the same id.
func TemplateGUID(path string) string {
	return guid.Stable(time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC), "dragonfall:"+path)
}

type builder struct {
	reg  *scene.Registry
	errs []error
}

func (b *builder) node(kind string) scene.Component {
	n, err := b.reg.New(kind)
	if err != nil {
		b.errs = append(b.errs, fmt.Errorf("build dragonfall: %w", err))
	}
	return n
}

// Lantern is the prefab placed along the road: a post, a small flame and a light.
func Lantern() *scene.GameObject {
	s := scene.New("lantern")
	s.UseFIDs(scene.CountingFIDs())
	root := s.Add("lantern", components.NewCylinder(.14, 3, color.RGB(92, 70, 52))).Blocking()
	flame := components.NewEmitter()
	flame.FX, _ = fx.Preset("fire")
	flame.FX.Rate, flame.FX.Life, flame.FX.Speed, flame.FX.Spread, flame.FX.Gravity = 18, .6, 1.1, .3, -1.5
	flame.FX.Shape, flame.FX.Size, flame.FX.SizeEnd, flame.FX.Seed, flame.FX.Turbulence = fx.Point, .5, .2, 11, 0
	flame.FX.Start, flame.FX.End = color.RGB(255, 220, 140), color.RGB(200, 70, 30)
	cage := components.NewBox(geom.V3(.55, .55, .7), color.RGB(255, 196, 110))
	cage.Emissive = .85
	root.Add("cage", cage).At(0, 0, 2.6)
	root.Add("flame", flame).At(0, 0, 3.3)
	glow := components.NewLight(9, color.RGB(255, 176, 96))
	glow.Intensity, glow.Flicker = 1.5, .25
	root.Add("glow", glow).At(0, 0, 3.4)
	return root
}

// BuildScene authors the Dragonfall scene; the generator writes it to the template.
func BuildScene(reg *scene.Registry) (*scene.Scene, error) {
	b := &builder{reg: reg}
	s := scene.New("Dragonfall")
	s.UseFIDs(scene.CountingFIDs())
	s.Camera.Target, s.Camera.Span, s.Camera.Pitch, s.Camera.AnchorY = geom.V3(114, 79.5, 0), 38, 52, .6
	s.Add("terrain", b.node("df.terrain")).Blocking()
	s.Add("forest", b.node("df.forest")).Blocking()
	s.Add("grass", b.node("df.grass"))
	s.Add("bridge", b.node("df.bridge")).At(122, 84, 0)
	s.Add("castle", b.node("df.castle")).At(104, 70, 0).Blocking()
	dragon := s.Add("dragon", b.node("df.dragon")).At(143, 73, 0).Blocking().With(components.NewToggle("e", "", "awake"), b.script(TemplateGUID(DragonScript)))
	dragon.Local.Flip = true
	s.Add("hero", b.node("df.hero")).At(114, 79.5, 0).With(components.NewPlayer())
	road := s.Add("road", nil)
	for i, p := range []geom.Vec2{{X: 100, Y: 76.3}, {X: 110, Y: 78}, {X: 118, Y: 79.2}, {X: 130, Y: 79.4}} {
		l := road.Add(fmt.Sprintf("lantern %d", i+1), nil).At(p.X, p.Y, 0)
		l.Prefab, l.Solid, l.Name = TemplateGUID(LanternPrefab), true, fmt.Sprintf("lantern %d", i+1)
	}
	crystal := components.NewBlob(.9, color.RGB(140, 214, 232))
	crystal.Glow = .8
	cr := s.Add("crystal", crystal).At(110, 106, .6).With(b.script(TemplateGUID(CrystalScript)))
	moon := components.NewLight(12, color.RGB(120, 200, 255))
	moon.Flicker = .05
	cr.Add("shine", moon).At(0, 0, 1.2)
	bob := anim.Tween("pos.z", .6, 1.6, 2.2, anim.InOut).Bouncing()
	bob.Name = "bob"
	s.Animate("crystal", bob)
	s.Add("hud", b.node("df.hud")).With(components.NewToggle("m", "", "showMap"), components.NewToggle("h", "", "showHelp"))
	s.Add("rules", nil).With(components.NewDayNight(), &components.Quit{Press: "q"})
	return s, errors.Join(b.errs...)
}
