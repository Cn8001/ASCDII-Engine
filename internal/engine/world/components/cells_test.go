package components

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func row(f *pipeline.Frame, y int) string {
	out := []rune{}
	for x := 0; x < f.Cells.W; x++ {
		c, _ := f.Cells.At(x, y)
		if c.Rune == 0 {
			c.Rune = ' '
		}
		out = append(out, c.Rune)
	}
	return string(out)
}

func TestUnit_LabelLinesStackAtTheirAnchor(t *testing.T) {
	// Arrange
	f := pipeline.NewFrame(20, 8)
	top, bottom := NewLabel("ab\ncdef"), NewLabel("xy\nz")
	top.Anchor, bottom.Anchor = "top-left", "bottom"

	// Act
	top.DrawCells(f, scene.Transform{})
	bottom.DrawCells(f, scene.Transform{})

	// Assert
	if row(f, 0)[:4] != "ab  " || row(f, 1)[:4] != "cdef" {
		t.Fatalf("top rows %q %q", row(f, 0), row(f, 1))
	}
	if got := row(f, 6)[9:11]; got != "xy" {
		t.Fatalf("bottom first line %q", row(f, 6))
	}
	if got := row(f, 7)[9:10]; got != "z" {
		t.Fatalf("bottom last line %q", row(f, 7))
	}
}
