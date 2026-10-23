package realm

import "testing"

func TestUnit_RealmIsDeterministic(t *testing.T) {
	// Arrange
	a, b := New(260, 140, 2026), New(260, 140, 2026)

	// Act
	same := true
	for y := 0; y < 140; y += 7 {
		for x := 0; x < 260; x += 5 {
			if a.At(x, y) != b.At(x, y) {
				same = false
			}
		}
	}

	// Assert
	if !same {
		t.Fatal("realm differs for the same seed")
	}
}

func TestUnit_RiverHasBridgeAndOutsideIsCliff(t *testing.T) {
	// Arrange
	r := New(260, 140, 2026)

	// Act
	by := 0
	for y := 70; y < 100; y++ {
		if BridgeAt(122.5, float64(y)+.5) {
			by = y
		}
	}
	river, bridge, outside := r.At(100, 88), r.At(122, by), r.At(-1, 10)

	// Assert
	if river.Kind != River || bridge.Kind != Bridge || outside.Kind != Crag {
		t.Fatalf("river %v bridge %v outside %v", river.Kind, bridge.Kind, outside.Kind)
	}
}

func TestUnit_CacheSharesInstance(t *testing.T) {
	// Arrange
	c := NewCache()

	// Act
	a, b := c.Get(1), c.Get(1)

	// Assert
	if a != b {
		t.Fatal("cache returned different realms")
	}
}
