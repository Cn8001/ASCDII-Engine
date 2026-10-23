package vegetation

import (
	"math"
	"testing"
)

func TestUnit_DisturbedGrassSpringsBack(t *testing.T) {
	// Arrange
	f := New()
	f.Disturb(5, 5, 1)

	// Act
	f.Step(.05)
	pushed := f.BendAt(5, 5, 0) - f.BendAt(50, 50, 0)
	for i := 0; i < 400; i++ {
		f.Step(.02)
	}

	// Assert
	if math.Abs(pushed) < .01 || len(f.patches) != 0 {
		t.Fatalf("pushed %f patches %d", pushed, len(f.patches))
	}
}
