package fx

import (
	"math"
	"math/rand/v2"
	"slices"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

// System describes a particle effect and Sim runs one. Unlike Emitter, a Sim
// keeps its particles, so they can stay in the world while the emitter
// moves, bounce on the ground, and be emitted in bursts from scripts.
type System struct {
	Duration   float64 // seconds per cycle
	Loop       bool
	Prewarm    bool // start as if it had been running for a while
	Rate       float64
	Burst      int // particles at the start of every cycle
	Max        int
	Shape      string // point, sphere, box or ring
	Radius     float64
	Box        geom.Vec3
	Life       float64 // seconds, each particle varies by +-30%
	Speed      float64
	Dir        geom.Vec3
	Spread     float64 // 0 is a beam, 1 is every direction
	Size       float64
	SizeEnd    float64
	Start      color.Color
	End        color.Color
	Emissive   float64
	Gravity    float64
	Drag       float64
	Wind       geom.Vec3
	Turbulence float64
	Collide    bool
	Bounce     float64
	Space      string // world or local
	Render     string // dot or streak
	Seed       int
}

const (
	Point  = "point"
	Sphere = "sphere"
	Box    = "box"
	Ring   = "ring"
	World  = "world"
	Local  = "local"
	Dot    = "dot"
	Streak = "streak"
)

func (s *System) Bind(b *prop.Binder) *prop.Binder {
	return b.Float("duration", &s.Duration, .1, 60).
		Bool("loop", &s.Loop).
		Bool("prewarm", &s.Prewarm).
		Float("rate", &s.Rate, 0, 400).
		Int("burst", &s.Burst, 0, 500).
		Int("max", &s.Max, 1, 4000).
		Enum("shape", &s.Shape, Point, Sphere, Box, Ring).
		Float("radius", &s.Radius, 0, 40).
		Vec3("box", &s.Box).
		Float("life", &s.Life, .05, 20).
		Float("speed", &s.Speed, 0, 60).
		Vec3("dir", &s.Dir).
		Float("spread", &s.Spread, 0, 1).
		Float("size", &s.Size, .05, 4).
		Float("sizeEnd", &s.SizeEnd, 0, 4).
		Color("start", &s.Start).
		Color("end", &s.End).
		Float("emissive", &s.Emissive, 0, 1).
		Float("gravity", &s.Gravity, -40, 40).
		Float("drag", &s.Drag, 0, 10).
		Vec3("wind", &s.Wind).
		Float("turbulence", &s.Turbulence, 0, 20).
		Bool("collide", &s.Collide).
		Float("bounce", &s.Bounce, 0, 1).
		Enum("space", &s.Space, World, Local).
		Enum("render", &s.Render, Dot, Streak).
		Int("seed", &s.Seed, 0, 9999)
}

func base() System {
	return System{Duration: 5, Loop: true, Max: 600, Shape: Point, Radius: 1, Box: geom.V3(2, 2, 2), Life: 1.5, Speed: 2,
		Dir: geom.V3(0, 0, 1), Spread: .3, Size: .4, SizeEnd: .15, Start: color.RGB(255, 255, 255), End: color.RGB(120, 120, 120),
		Space: World, Render: Dot, Bounce: .3}
}

// Presets are starting points the editor offers; every value stays editable.
var presets = []string{"embers", "fire", "smoke", "sparks", "rain", "snow", "magic", "fountain", "explosion"}

func Presets() []string { return slices.Clone(presets) }

func Preset(name string) (System, bool) {
	s := base()
	switch name {
	case "embers":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 6, 2.4, .8, .6, -.4, .25, .1
		s.Start, s.End, s.Emissive, s.Prewarm, s.Turbulence = color.RGB(255, 196, 96), color.RGB(120, 40, 20), 1, true, 1.5
	case "fire":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 30, .7, 1.4, .25, -2, .6, .15
		s.Shape, s.Radius = Sphere, .3
		s.Start, s.End, s.Emissive, s.Prewarm, s.Turbulence = color.RGB(255, 226, 140), color.RGB(170, 40, 20), 1, true, 2
	case "smoke":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 8, 3.5, .8, .3, -.3, .4, 1.2
		s.Start, s.End, s.Wind, s.Drag, s.Prewarm, s.Turbulence = color.RGB(110, 108, 104), color.RGB(60, 60, 64), geom.V3(.6, 0, 0), .3, true, 1
	case "sparks":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 25, .9, 6, .5, 14, .2, .1
		s.Start, s.End, s.Emissive, s.Collide, s.Bounce, s.Render = color.RGB(255, 236, 170), color.RGB(230, 90, 30), 1, true, .4, Streak
	case "rain":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 140, 1.2, 18, .02, 0, .2, .2
		s.Shape, s.Box, s.Dir, s.Render = Box, geom.V3(40, 30, 0), geom.V3(.1, 0, -1), Streak
		s.Start, s.End, s.Collide, s.Bounce, s.Max = color.RGB(150, 170, 200), color.RGB(110, 130, 170), true, 0, 2000
	case "snow":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 40, 7, 1.2, .1, 0, .3, .3
		s.Shape, s.Box, s.Dir, s.Turbulence, s.Prewarm = Box, geom.V3(40, 30, 0), geom.V3(0, 0, -1), 1.5, true
		s.Start, s.End, s.Collide, s.Bounce, s.Max = color.RGB(236, 240, 248), color.RGB(200, 210, 230), true, 0, 2000
	case "magic":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 20, 1.6, .6, 1, -.6, .3, .05
		s.Shape, s.Radius, s.Turbulence = Ring, 1.2, 3
		s.Start, s.End, s.Emissive = color.RGB(170, 140, 255), color.RGB(60, 200, 220), 1
	case "fountain":
		s.Rate, s.Life, s.Speed, s.Spread, s.Gravity, s.Size, s.SizeEnd = 60, 1.6, 9, .12, 14, .3, .2
		s.Start, s.End, s.Collide, s.Bounce, s.Prewarm = color.RGB(170, 210, 240), color.RGB(90, 140, 200), true, .2, true
	case "explosion":
		s.Rate, s.Burst, s.Loop, s.Duration, s.Life, s.Speed, s.Spread = 0, 120, false, 1, 1.1, 9, 1
		s.Gravity, s.Drag, s.Size, s.SizeEnd, s.Emissive = 6, 2.5, .7, .1, 1
		s.Start, s.End, s.Collide, s.Bounce = color.RGB(255, 230, 150), color.RGB(150, 40, 20), true, .3
	default:
		return System{}, false
	}
	return s, true
}

// Frame maps an emitter's local space into the world.
type Frame interface {
	Apply(p geom.Vec3) geom.Vec3
	Dir(d geom.Vec3) geom.Vec3
}

type particle struct {
	pos, vel  geom.Vec3
	age, life float64
	phase     float64
}

type Sim struct {
	ps       []particle
	clock    float64 // seconds into the current cycle
	owed     float64 // fractional births carried to the next step
	emitting bool
	burstDue bool
	rng      *rand.Rand
}

func NewSim(seed int) *Sim {
	return &Sim{rng: rand.New(rand.NewPCG(uint64(seed), 0x5eed))}
}

func (s *Sim) Play(sys System, f Frame) {
	s.emitting, s.burstDue, s.clock, s.owed = true, true, 0, 0
	if sys.Prewarm {
		for t := 0.0; t < sys.Life*1.3; t += 1.0 / 30 {
			s.Step(sys, f, 1.0/30)
		}
	}
}

func (s *Sim) Stop()          { s.emitting = false }
func (s *Sim) Clear()         { s.ps = s.ps[:0] }
func (s *Sim) Emitting() bool { return s.emitting }
func (s *Sim) Count() int     { return len(s.ps) }

// Alive is true while it emits or particles remain.
func (s *Sim) Alive() bool { return s.emitting || len(s.ps) > 0 }

// Emit spawns n particles at once, whether or not the system is playing.
func (s *Sim) Emit(sys System, f Frame, n int) {
	for range n {
		if sys.Max > 0 && len(s.ps) >= sys.Max {
			return
		}
		s.ps = append(s.ps, s.spawn(sys, f))
	}
}

func (s *Sim) Step(sys System, f Frame, dt float64) {
	if s.emitting {
		if s.burstDue {
			s.burstDue = false
			s.Emit(sys, f, sys.Burst)
		}
		s.owed += sys.Rate * dt
		n := int(s.owed)
		s.owed -= float64(n)
		s.Emit(sys, f, n)
		s.clock += dt
		if sys.Duration > 0 && s.clock >= sys.Duration {
			if sys.Loop {
				s.clock -= sys.Duration
				s.burstDue = true
			} else {
				s.emitting = false
			}
		}
	}
	keep := s.ps[:0]
	pull := geom.V3(0, 0, -sys.Gravity).Add(sys.Wind)
	slow := math.Max(0, 1-sys.Drag*dt)
	for _, p := range s.ps {
		p.age += dt
		if p.age >= p.life {
			continue
		}
		swirl := geom.V3(math.Sin(p.age*2.3+p.phase), math.Cos(p.age*1.9+p.phase*1.7), math.Sin(p.age*2.9+p.phase*.6))
		p.vel = p.vel.Add(pull.Add(swirl.Scale(sys.Turbulence)).Scale(dt)).Scale(slow)
		p.pos = p.pos.Add(p.vel.Scale(dt))
		if sys.Collide && p.pos.Z < 0 {
			if sys.Bounce <= 0 {
				continue
			}
			p.pos.Z = 0
			p.vel = geom.V3(p.vel.X*.8, p.vel.Y*.8, -p.vel.Z*sys.Bounce)
		}
		keep = append(keep, p)
	}
	s.ps = keep
}

func (s *Sim) spawn(sys System, f Frame) particle {
	r := s.rng.Float64
	var at geom.Vec3
	switch sys.Shape {
	case Sphere:
		at = s.unit().Scale(sys.Radius * math.Cbrt(r()))
	case Box:
		at = geom.V3((r()-.5)*sys.Box.X, (r()-.5)*sys.Box.Y, (r()-.5)*sys.Box.Z)
	case Ring:
		a := r() * 2 * math.Pi
		at = geom.V3(math.Cos(a)*sys.Radius, math.Sin(a)*sys.Radius, 0)
	}
	dir := sys.Dir.Norm()
	if dir == (geom.Vec3{}) {
		dir = geom.V3(0, 0, 1)
	}
	dir = dir.Add(s.unit().Scale(2 * sys.Spread)).Norm()
	vel := dir.Scale(sys.Speed * (.75 + .5*r()))
	if sys.Space != Local {
		at, vel = f.Apply(at), f.Dir(vel)
	}
	return particle{pos: at, vel: vel, life: sys.Life * (.7 + .6*r()), phase: r() * 2 * math.Pi}
}

// unit is a random direction.
func (s *Sim) unit() geom.Vec3 {
	z := s.rng.Float64()*2 - 1
	a := s.rng.Float64() * 2 * math.Pi
	k := math.Sqrt(1 - z*z)
	return geom.V3(k*math.Cos(a), k*math.Sin(a), z)
}

// Each visits the live particles in world space.
func (s *Sim) Each(sys System, f Frame, fn func(Particle)) {
	for _, p := range s.ps {
		pos, vel := p.pos, p.vel
		if sys.Space == Local {
			pos, vel = f.Apply(pos), f.Dir(vel)
		}
		k := p.age / p.life
		fn(Particle{Pos: pos, Vel: vel, Age: k, Color: color.Mix(sys.Start, sys.End, k), Size: sys.Size + (sys.SizeEnd-sys.Size)*k})
	}
}
