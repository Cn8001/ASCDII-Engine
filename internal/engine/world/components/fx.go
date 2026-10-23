package components

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/fx"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func lightAt(p geom.Vec3, r float64, c color.Color, k, flicker float64) light.Point {
	return light.Point{Pos: p, Radius: r, Color: c, Intensity: k, Flicker: flicker}
}

// Emitter switches births on and off while live particles finish naturally.
type Emitter struct {
	FX     fx.Emitter
	Active bool
	Glow   float64 // point light radius, 0 for none
	onAt   float64
	offAt  float64
	was    bool
}

func NewEmitter() *Emitter {
	return &Emitter{FX: fx.Embers(), Active: true, was: true, offAt: math.Inf(1)}
}

func (*Emitter) Kind() string { return "emitter" }
func (e *Emitter) bind() *prop.Binder {
	return e.FX.Bind(prop.Bind().Bool("active", &e.Active).Float("glow", &e.Glow, 0, 60))
}
func (e *Emitter) Props() []prop.Prop        { return e.bind().Props() }
func (e *Emitter) Set(n string, v any) error { return e.bind().Set(n, v) }
func (e *Emitter) Bounds(t scene.Transform) geom.AABB {
	r := math.Max(2, e.FX.Speed*e.FX.Life*1.3)
	return geom.Footprint(t.Pos.Sub(geom.V3(0, 0, r)), geom.V3(r*2, r*2, r*2))
}

func (e *Emitter) Tick(c *scene.Ctx, _ *scene.GameObject, _ float64) {
	if e.Active != e.was {
		if e.Active {
			e.onAt, e.offAt = c.Scene.Time, math.Inf(1)
		} else {
			e.offAt = c.Scene.Time
		}
		e.was = e.Active
	}
}

func (e *Emitter) alive(birth float64) bool {
	if e.Active {
		return birth >= e.onAt
	}
	return birth >= e.onAt && birth < e.offAt
}

func (e *Emitter) Draw(c *draw.Context, t scene.Transform) {
	n := 0
	e.FX.Each(c.Time, t.Pos, e.alive, func(p fx.Particle) {
		n++
		c.Point(p.Pos, p.Size, draw.Material{Albedo: p.Color, Emissive: e.FX.Emissive * (1 - p.Age*.5)})
	})
	if e.Glow > 0 && n > 0 {
		c.Light(lightAt(t.Pos.Add(e.FX.Dir.Norm().Scale(e.FX.Speed*e.FX.Life*.3)), e.Glow, e.FX.Start, 1.4, .35))
	}
}
