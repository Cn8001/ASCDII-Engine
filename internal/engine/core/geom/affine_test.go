package geom

import (
	"math"
	"testing"
)

func TestUnit_AffineMulAppliesRightFirst(t *testing.T) {
	// Arrange
	m := Translate(10, 0).Mul(Rotate(math.Pi / 2))

	// Act
	p := m.Apply(V2(1, 0))

	// Assert
	if math.Abs(p.X-10) > 1e-9 || math.Abs(p.Y-1) > 1e-9 {
		t.Fatalf("got %+v", p)
	}
}

func TestUnit_AffineScaleIgnoresRotation(t *testing.T) {
	// Arrange
	m := Rotate(1.1).Mul(ScaleXY(2, 2))

	// Act
	s := m.Scale()

	// Assert
	if math.Abs(s-2) > 1e-9 {
		t.Fatalf("got %f", s)
	}
}
