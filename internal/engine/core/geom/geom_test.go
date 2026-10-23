package geom

import (
	"encoding/json"
	"math"
	"testing"
)

func TestUnit_FootprintStandsOnBase(t *testing.T) {
	// Arrange
	base, size := V3(10, 20, 1), V3(4, 2, 3)

	// Act
	b := Footprint(base, size)

	// Assert
	if b.Min != V3(8, 19, 1) || b.Max != V3(12, 21, 4) {
		t.Fatalf("got %+v", b)
	}
	if !b.Contains2D(V2(10, 20)) || b.Contains2D(V2(12, 20)) {
		t.Fatal("contains is not half open")
	}
}

func TestUnit_CrossFollowsRightHand(t *testing.T) {
	// Arrange
	x, y := V3(1, 0, 0), V3(0, 1, 0)

	// Act
	z := x.Cross(y)

	// Assert
	if z != V3(0, 0, 1) {
		t.Fatalf("got %+v", z)
	}
}

func TestUnit_Vec3JSONIsArray(t *testing.T) {
	// Arrange
	v := V3(1, 2.5, -3)

	// Act
	b, err := json.Marshal(v)
	var back Vec3
	errBack := json.Unmarshal(b, &back)

	// Assert
	if err != nil || errBack != nil || string(b) != "[1,2.5,-3]" || back != v {
		t.Fatalf("got %s %+v", b, back)
	}
}

func TestUnit_RotateQuarterTurn(t *testing.T) {
	// Arrange
	v := V2(1, 0)

	// Act
	r := v.Rotate(math.Pi / 2)

	// Assert
	if math.Abs(r.X) > 1e-9 || math.Abs(r.Y-1) > 1e-9 {
		t.Fatalf("got %+v", r)
	}
}

func TestUnit_RotateEulerAroundUp(t *testing.T) {
	// Arrange
	v := V3(1, 0, 0)

	// Act
	r := v.RotateEuler(V3(0, 0, 90))
	same := v.RotateEuler(Vec3{})

	// Assert
	if math.Abs(r.X) > 1e-9 || math.Abs(r.Y-1) > 1e-9 || same != v {
		t.Fatalf("got %+v", r)
	}
}

func TestUnit_UnrotateEulerInvertsRotation(t *testing.T) {
	// Arrange
	v, deg := V3(1, 2, 3), V3(30, -45, 120)

	// Act
	back := v.RotateEuler(deg).UnrotateEuler(deg)

	// Assert
	if back.Sub(v).Len() > 1e-9 {
		t.Fatalf("got %+v", back)
	}
}
