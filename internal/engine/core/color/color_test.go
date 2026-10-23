package color

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestUnit_ParseHexRoundTrips(t *testing.T) {
	// Arrange
	in := "#e8b04b"

	// Act
	c, err := ParseHex(in)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if c != (Color{232, 176, 75}) || c.Hex() != in {
		t.Fatalf("got %+v %s", c, c.Hex())
	}
}

func TestUnit_ParseHexRejectsShortInput(t *testing.T) {
	// Arrange
	in := "#fff"

	// Act
	_, err := ParseHex(in)

	// Assert
	if !errors.Is(err, ErrInvalidHex) {
		t.Fatalf("want ErrInvalidHex, got %v", err)
	}
}

func TestUnit_MixClampsFactor(t *testing.T) {
	// Arrange
	a, b := RGB(0, 0, 0), RGB(200, 100, 50)

	// Act
	over := Mix(a, b, 2)
	half := Mix(a, b, .5)

	// Assert
	if over != b || half != RGB(100, 50, 25) {
		t.Fatalf("over %+v half %+v", over, half)
	}
}

func TestUnit_MulWithWhiteIsIdentity(t *testing.T) {
	// Arrange
	c := RGB(12, 140, 250)

	// Act
	got := Mul(c, RGB(255, 255, 255))

	// Assert
	if got != c {
		t.Fatalf("got %+v", got)
	}
}

func TestUnit_ColorJSONUsesHexString(t *testing.T) {
	// Arrange
	c := RGB(1, 2, 3)

	// Act
	b, err := json.Marshal(c)
	var back Color
	errBack := json.Unmarshal(b, &back)

	// Assert
	if err != nil || errBack != nil || string(b) != `"#010203"` || back != c {
		t.Fatalf("got %s %+v %v %v", b, back, err, errBack)
	}
}
