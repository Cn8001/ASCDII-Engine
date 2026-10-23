package fx

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

type shift geom.Vec3

func (s shift) Apply(p geom.Vec3) geom.Vec3 { return p.Add(geom.Vec3(s)) }
func (s shift) Dir(d geom.Vec3) geom.Vec3   { return d }

func run(s *Sim, sys System, f Frame, secs float64) {
	for t := 0.0; t < secs; t += 1.0 / 30 {
		s.Step(sys, f, 1.0/30)
	}
}

func TestUnit_SimEmitsAtRateAndCapsAtMax(t *testing.T) {
	// Arrange
	sys, _ := Preset("fire")
	sys.Prewarm, sys.Life, sys.Rate, sys.Max = false, 100, 30, 40
	s := NewSim(1)

	// Act
	s.Play(sys, shift{})
	run(s, sys, shift{}, 1)
	early := s.Count()
	run(s, sys, shift{}, 3)

	// Assert
	if early < 27 || early > 31 || s.Count() != 40 {
		t.Fatalf("after 1s %d, after 4s %d", early, s.Count())
	}
}

func TestUnit_SimBurstOnceWithoutLoop(t *testing.T) {
	// Arrange
	sys, _ := Preset("explosion")
	s := NewSim(1)

	// Act
	s.Play(sys, shift{})
	run(s, sys, shift{}, .1)
	burst := s.Count()
	run(s, sys, shift{}, 3)

	// Assert
	if burst != 120 || s.Alive() {
		t.Fatalf("burst %d alive %v count %d", burst, s.Alive(), s.Count())
	}
}

func TestUnit_WorldSpaceParticlesStayWhenEmitterMoves(t *testing.T) {
	// Arrange
	sys := base()
	sys.Speed, sys.Spread, sys.Gravity = 0, 0, 0
	world, local := NewSim(1), NewSim(1)
	lsys := sys
	lsys.Space = Local

	// Act
	world.Emit(sys, shift{}, 1)
	local.Emit(lsys, shift{}, 1)
	var w, l Particle
	world.Each(sys, shift{X: 10}, func(p Particle) { w = p })
	local.Each(lsys, shift{X: 10}, func(p Particle) { l = p })

	// Assert
	if w.Pos.X != 0 || l.Pos.X != 10 {
		t.Fatalf("world %+v local %+v", w.Pos, l.Pos)
	}
}

func TestUnit_CollidingParticlesBounceOnGround(t *testing.T) {
	// Arrange
	sys := base()
	sys.Dir, sys.Spread, sys.Speed, sys.Gravity, sys.Collide, sys.Bounce, sys.Life = geom.V3(0, 0, -1), 0, 5, 10, true, .5, 10
	s := NewSim(1)
	s.Emit(sys, shift{Z: 1}, 1)

	// Act
	low := 1.0
	for range 120 {
		s.Step(sys, shift{}, 1.0/60)
		s.Each(sys, shift{}, func(p Particle) { low = min(low, p.Pos.Z) })
	}

	// Assert
	if s.Count() != 1 || low < 0 {
		t.Fatalf("count %d lowest %f", s.Count(), low)
	}
}

func TestUnit_EveryPresetExists(t *testing.T) {
	// Arrange
	names := Presets()

	// Act
	_, unknown := Preset("nope")

	// Assert
	for _, n := range names {
		if _, ok := Preset(n); !ok {
			t.Fatalf("preset %s missing", n)
		}
	}
	if unknown {
		t.Fatal("unknown preset found")
	}
}
