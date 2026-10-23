// Package color is the 24-bit RGB model shared by every engine layer.
package color

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var ErrInvalidHex = errors.New("invalid hex color")

type Color struct{ R, G, B uint8 }

func RGB(r, g, b uint8) Color { return Color{R: r, G: g, B: b} }

// Hex is meant for literals in content code, where a typo should fail loudly.
func Hex(s string) Color {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

func ParseHex(s string) (Color, error) {
	v := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(v) != 6 {
		return Color{}, fmt.Errorf("%w: %q", ErrInvalidHex, s)
	}
	n, err := strconv.ParseUint(v, 16, 24)
	if err != nil {
		return Color{}, fmt.Errorf("%w: %q", ErrInvalidHex, s)
	}
	return Color{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}, nil
}

func (c Color) Hex() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func (c Color) MarshalJSON() ([]byte, error) { return json.Marshal(c.Hex()) }

func (c *Color) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("decode color: %w", err)
	}
	v, err := ParseHex(s)
	if err != nil {
		return err
	}
	*c = v
	return nil
}

func Mix(a, b Color, t float64) Color {
	t = math.Max(0, math.Min(1, t))
	lerp := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-t) + float64(y)*t)) }
	return Color{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B)}
}

func Scale(c Color, k float64) Color {
	k = math.Max(0, k)
	return Color{clamp(float64(c.R) * k), clamp(float64(c.G) * k), clamp(float64(c.B) * k)}
}

// Mul modulates like a light filter: white leaves a color unchanged.
func Mul(a, b Color) Color {
	return Color{
		uint8(uint16(a.R) * uint16(b.R) / 255),
		uint8(uint16(a.G) * uint16(b.G) / 255),
		uint8(uint16(a.B) * uint16(b.B) / 255),
	}
}

func Add(a, b Color) Color {
	return Color{clamp(float64(a.R) + float64(b.R)), clamp(float64(a.G) + float64(b.G)), clamp(float64(a.B) + float64(b.B))}
}

// Luma uses Rec. 601 weights; the encoder and outline pass only need relative brightness.
func (c Color) Luma() float64 {
	return (.299*float64(c.R) + .587*float64(c.G) + .114*float64(c.B)) / 255
}

// Float returns channels in 0..1 for lighting math that must not round per step.
func (c Color) Float() (r, g, b float64) {
	return float64(c.R) / 255, float64(c.G) / 255, float64(c.B) / 255
}

func FromFloat(r, g, b float64) Color { return Color{clamp(r * 255), clamp(g * 255), clamp(b * 255)} }

func clamp(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}
