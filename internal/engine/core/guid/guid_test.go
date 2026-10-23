package guid

import (
	"bytes"
	"testing"
	"time"
)

func TestUnit_NewIsVersion7AndCarriesTime(t *testing.T) {
	// Arrange
	at := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	s := Source{Now: func() time.Time { return at }, Rand: bytes.NewReader(bytes.Repeat([]byte{0xab}, 16))}

	// Act
	id, err := s.New()
	back, errTime := Time(id)

	// Assert
	if err != nil || errTime != nil || !Valid(id) || !back.Equal(at) {
		t.Fatalf("id %s err %v %v time %v", id, err, errTime, back)
	}
}

func TestUnit_IdsSortByCreation(t *testing.T) {
	// Arrange
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(at time.Time) string {
		id, _ := Source{Now: func() time.Time { return at }, Rand: bytes.NewReader(make([]byte, 16))}.New()
		return id
	}

	// Act
	a, b := mk(base), mk(base.Add(time.Millisecond))

	// Assert
	if !(a < b) {
		t.Fatalf("%s !< %s", a, b)
	}
}

func TestUnit_StableIsDeterministic(t *testing.T) {
	// Arrange
	at := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

	// Act
	a, b, c := Stable(at, "art/pine.json"), Stable(at, "art/pine.json"), Stable(at, "art/hero.json")

	// Assert
	if a != b || a == c || !Valid(a) {
		t.Fatalf("a %s b %s c %s", a, b, c)
	}
}

func TestUnit_ValidRejectsOtherVersions(t *testing.T) {
	// Arrange
	v4 := "550e8400-e29b-41d4-a716-446655440000"

	// Act
	ok := Valid(v4)

	// Assert
	if ok || Valid("prefabs/x.prefab.json") {
		t.Fatal("accepted a non v7 value")
	}
}
