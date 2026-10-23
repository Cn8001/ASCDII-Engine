package cell

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
)

func TestUnit_SetIgnoresOutOfBounds(t *testing.T) {
	// Arrange
	g := New(3, 2)

	// Act
	g.Set(-1, 0, Cell{Rune: 'x'})
	g.Set(3, 1, Cell{Rune: 'x'})
	g.Set(2, 1, Cell{Rune: 'y'})

	// Assert
	c, ok := g.At(2, 1)
	if !ok || c.Rune != 'y' {
		t.Fatalf("got %+v", c)
	}
	if _, ok := g.At(3, 1); ok {
		t.Fatal("expected clipping")
	}
}

func TestUnit_TextAdvancesColumn(t *testing.T) {
	// Arrange
	g := New(10, 1)
	fg := color.RGB(1, 2, 3)

	// Act
	next := g.Text(2, 0, "hé!", fg, color.Color{})

	// Assert
	c, _ := g.At(3, 0)
	if next != 5 || c.Rune != 'é' || c.FG != fg {
		t.Fatalf("next %d cell %+v", next, c)
	}
}

func TestUnit_ResizeClampsToOneCell(t *testing.T) {
	// Arrange
	g := New(4, 4)

	// Act
	changed := g.Resize(0, -3)

	// Assert
	if !changed || g.W != 1 || g.H != 1 || len(g.Cells()) != 1 {
		t.Fatalf("got %dx%d", g.W, g.H)
	}
}
