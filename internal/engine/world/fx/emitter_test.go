package fx

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

func collect(e Emitter, t float64, active func(float64) bool) []Particle {
	var out []Particle
	e.Each(t, geom.Vec3{}, active, func(p Particle) { out = append(out, p) })
	return out
}

func TestUnit_EmitterIsDeterministic(t *testing.T) {
	// Arrange
	e := Fire()

	// Act
	a, b := collect(e, 3.2, nil), collect(e, 3.2, nil)

	// Assert
	if len(a) == 0 || len(a) != len(b) || a[0] != b[0] {
		t.Fatalf("len %d vs %d", len(a), len(b))
	}
}

func TestUnit_InactiveEmitterStopsBirths(t *testing.T) {
	// Arrange
	e := Fire()
	off := func(birth float64) bool { return birth < 1 }

	// Act
	late := collect(e, 5, off)

	// Assert
	if len(late) != 0 {
		t.Fatalf("expected no particles, got %d", len(late))
	}
}

func TestUnit_ParticlesAgeWithinLife(t *testing.T) {
	// Arrange
	e := Embers()

	// Act
	ps := collect(e, 10, nil)

	// Assert
	for _, p := range ps {
		if p.Age < 0 || p.Age > 1 {
			t.Fatalf("age %f", p.Age)
		}
	}
}
