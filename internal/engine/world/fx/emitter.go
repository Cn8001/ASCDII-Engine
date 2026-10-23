// Package fx is a stateless particle system: each particle is derived from its
// index and the current time, so effects are deterministic and scrubbable.
package fx

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/noise"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

type Emitter struct {
	Rate     float64     `json:"rate"` // particles per second
	Life     float64     `json:"life"` // seconds, each particle varies by +-30%
	Speed    float64     `json:"speed"`
	Dir      geom.Vec3   `json:"dir"`
	Spread   float64     `json:"spread"` // 0 is a beam, 1 is a full sphere
	Gravity  float64     `json:"gravity"`
	Size     float64     `json:"size"`
	Start    color.Color `json:"start"`
	End      color.Color `json:"end"`
	Emissive float64     `json:"emissive"`
	Seed     int         `json:"seed"`
}

type Particle struct {
	Pos   geom.Vec3
	Age   float64 // 0..1 of its life
	Color color.Color
	Size  float64
}

func Fire() Emitter {
	return Emitter{Rate: 40, Life: .7, Speed: 7, Dir: geom.V3(1, 0, .15), Spread: .25, Gravity: -3, Size: .5,
		Start: color.RGB(255, 226, 140), End: color.RGB(150, 40, 20), Emissive: 1, Seed: 7}
}

func Embers() Emitter {
	return Emitter{Rate: 6, Life: 2.4, Speed: .8, Dir: geom.V3(0, 0, 1), Spread: .6, Gravity: -.4, Size: .25,
		Start: color.RGB(255, 196, 96), End: color.RGB(120, 40, 20), Emissive: 1, Seed: 3}
}

func (e *Emitter) Bind(b *prop.Binder) *prop.Binder {
	return b.Float("rate", &e.Rate, 0, 400).
		Float("life", &e.Life, .05, 20).
		Float("speed", &e.Speed, 0, 60).
		Vec3("dir", &e.Dir).
		Float("spread", &e.Spread, 0, 1).
		Float("gravity", &e.Gravity, -40, 40).
		Float("size", &e.Size, .05, 4).
		Color("start", &e.Start).
		Color("end", &e.End).
		Float("emissive", &e.Emissive, 0, 1).
		Int("seed", &e.Seed, 0, 9999)
}

// Each visits particles alive at time t; active gates new births so effects can
// be switched on and off while live particles finish their life.
func (e Emitter) Each(t float64, origin geom.Vec3, active func(birth float64) bool, fn func(Particle)) {
	if e.Rate <= 0 || e.Life <= 0 {
		return
	}
	maxLife := e.Life * 1.3
	first := int(math.Floor((t - maxLife) * e.Rate))
	last := int(math.Floor(t * e.Rate))
	dir := e.Dir.Norm()
	if dir == (geom.Vec3{}) {
		dir = geom.V3(0, 0, 1)
	}
	for i := max(0, first); i <= last; i++ {
		birth := float64(i) / e.Rate
		if active != nil && !active(birth) {
			continue
		}
		h := noise.Hash(i, e.Seed, 977)
		life := e.Life * (.7 + .6*noise.Unit(h))
		age := t - birth
		if age < 0 || age > life {
			continue
		}
		jitter := geom.V3(noise.Unit(h>>3)-.5, noise.Unit(h>>7)-.5, noise.Unit(h>>11)-.5).Scale(2 * e.Spread)
		v := dir.Add(jitter).Norm().Scale(e.Speed * (.75 + .5*noise.Unit(h>>15)))
		pos := origin.Add(v.Scale(age)).Add(geom.V3(0, 0, -.5*e.Gravity*age*age))
		k := age / life
		fn(Particle{Pos: pos, Age: k, Color: color.Mix(e.Start, e.End, k), Size: e.Size * (1 - .6*k)})
	}
}
