package components

import (
	"fmt"
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

// Emitter is the particle system component. Switching active off stops
// births and lets live particles finish their life.
type Emitter struct {
	FX     fx.System
	Active bool
	Glow   float64 // point light radius, 0 for none
	sim    *fx.Sim
	was    bool
}

func NewEmitter() *Emitter {
	sys, _ := fx.Preset("embers")
	return &Emitter{FX: sys, Active: true}
}

func (*Emitter) Kind() string { return "emitter" }
func (e *Emitter) bind() *prop.Binder {
	return e.FX.Bind(prop.Bind().Bool("active", &e.Active).Float("glow", &e.Glow, 0, 60))
}
func (e *Emitter) Props() []prop.Prop        { return e.bind().Props() }
func (e *Emitter) Set(n string, v any) error { return e.bind().Set(n, v) }
func (e *Emitter) Bounds(t scene.Transform) geom.AABB {
	r := math.Max(2, e.FX.Speed*e.FX.Life*1.3+e.FX.Radius)
	return geom.Footprint(t.Pos.Sub(geom.V3(0, 0, r)), geom.V3(r*2, r*2, r*2))
}

func (e *Emitter) state() *fx.Sim {
	if e.sim == nil {
		e.sim = fx.NewSim(e.FX.Seed)
	}
	return e.sim
}

func (e *Emitter) Tick(_ *scene.Ctx, o *scene.GameObject, dt float64) {
	sim, w := e.state(), o.World()
	if e.Active != e.was {
		e.was = e.Active
		if e.Active {
			sim.Play(e.FX, w)
		} else {
			sim.Stop()
		}
	}
	sim.Step(e.FX, w, dt)
}

func (e *Emitter) Methods() []string { return []string{"play", "stop", "emit", "clear", "count"} }

func (e *Emitter) Call(_ *scene.Ctx, o *scene.GameObject, name string, args []any) (any, error) {
	sim := e.state()
	switch name {
	case "play":
		e.Active, e.was = true, true
		sim.Play(e.FX, o.World())
	case "stop":
		e.Active, e.was = false, false
		sim.Stop()
	case "emit":
		n := e.FX.Burst
		if len(args) > 0 {
			f, err := prop.ToFloat(args[0])
			if err != nil {
				return nil, err
			}
			n = int(f)
		}
		sim.Emit(e.FX, o.World(), n)
	case "clear":
		sim.Clear()
	case "count":
		return float64(sim.Count()), nil
	default:
		return nil, fmt.Errorf("no method %s", name)
	}
	return nil, nil
}

func (e *Emitter) Draw(c *draw.Context, t scene.Transform) {
	// The editor draws without ticking unless Live is on; starting here keeps
	// prewarmed effects such as flames visible there, frozen.
	if e.Active && !e.was {
		e.was = true
		e.state().Play(e.FX, t)
	}
	if e.sim == nil {
		return
	}
	n := 0
	streak := e.FX.Render == fx.Streak
	e.sim.Each(e.FX, t, func(p fx.Particle) {
		n++
		m := draw.Material{Albedo: p.Color, Emissive: e.FX.Emissive * (1 - p.Age*.5)}
		c.Point(p.Pos, p.Size, m)
		if streak {
			for i := 1.0; i <= 3; i++ {
				c.Point(p.Pos.Sub(p.Vel.Scale(.02*i)), p.Size*(1-.2*i), m)
			}
		}
	})
	if e.Glow > 0 && n > 0 {
		c.Light(lightAt(t.Pos.Add(e.FX.Dir.Norm().Scale(e.FX.Speed*e.FX.Life*.3)), e.Glow, e.FX.Start, 1.4, .35))
	}
}
