package prop

import (
	"errors"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
)

type sample struct {
	speed float64
	tint  color.Color
	pos   geom.Vec3
	mode  string
}

func (s *sample) bind() *Binder {
	return Bind().Float("speed", &s.speed, 0, 10).Color("tint", &s.tint).Vec3("pos", &s.pos).Enum("mode", &s.mode, "a", "b")
}
func (s *sample) Props() []Prop                { return s.bind().Props() }
func (s *sample) Set(name string, v any) error { return s.bind().Set(name, v) }

func TestUnit_BinderSetsFromJSONForms(t *testing.T) {
	// Arrange
	s := &sample{mode: "a"}

	// Act
	errs := []error{
		s.Set("speed", 42.0),
		s.Set("tint", "#102030"),
		s.Set("pos", []any{1.0, 2.0, 3.0}),
		s.Set("mode", "b"),
	}

	// Assert
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if s.speed != 10 || s.tint != color.RGB(16, 32, 48) || s.pos != geom.V3(1, 2, 3) || s.mode != "b" {
		t.Fatalf("got %+v", s)
	}
}

func TestUnit_BinderSetsSingleAxis(t *testing.T) {
	// Arrange
	s := &sample{pos: geom.V3(1, 2, 3)}

	// Act
	err := s.Set("pos.z", 9)

	// Assert
	p, ok := Get(s, "pos.z")
	if err != nil || s.pos != geom.V3(1, 2, 9) || !ok || p.Value.(float64) != 9 {
		t.Fatalf("err %v pos %+v", err, s.pos)
	}
}

func TestUnit_BinderRejectsUnknownAndInvalid(t *testing.T) {
	// Arrange
	s := &sample{mode: "a"}

	// Act
	unknown := s.Set("nope", 1)
	invalid := s.Set("mode", "z")

	// Assert
	if !errors.Is(unknown, ErrUnknownProp) || !errors.Is(invalid, ErrInvalidValue) {
		t.Fatalf("unknown %v invalid %v", unknown, invalid)
	}
}

func TestUnit_EncodeUsesJSONFriendlyForms(t *testing.T) {
	// Arrange
	s := &sample{tint: color.RGB(255, 0, 0), pos: geom.V3(1, 2, 3)}

	// Act
	props := s.Props()

	// Assert
	if Encode(props[1]) != "#ff0000" {
		t.Fatalf("color %v", Encode(props[1]))
	}
	if v := Encode(props[2]).([]float64); v[2] != 3 {
		t.Fatalf("vec %v", v)
	}
}
