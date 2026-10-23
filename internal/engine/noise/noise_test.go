package noise

import "testing"

func TestUnit_HashIsDeterministicAndVaries(t *testing.T) {
	// Arrange
	a1, a2, b := Hash(3, 4, 7), Hash(3, 4, 7), Hash(4, 3, 7)

	// Act
	same, differs := a1 == a2, a1 != b

	// Assert
	if !same || !differs {
		t.Fatalf("a1 %d a2 %d b %d", a1, a2, b)
	}
}

func TestUnit_ValueStaysInRange(t *testing.T) {
	// Arrange
	lo, hi := 1.0, 0.0

	// Act
	for i := 0; i < 500; i++ {
		v := Value(float64(i)*.37, float64(i)*.11, 9)
		lo, hi = min(lo, v), max(hi, v)
	}

	// Assert
	if lo < 0 || hi >= 1 || hi-lo < .3 {
		t.Fatalf("range %f..%f", lo, hi)
	}
}
