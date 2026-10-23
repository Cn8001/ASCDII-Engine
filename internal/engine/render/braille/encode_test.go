package braille

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

func TestUnit_BitMatchesUnicodeTable(t *testing.T) {
	// Arrange
	want := [2][4]uint8{{0x01, 0x02, 0x04, 0x40}, {0x08, 0x10, 0x20, 0x80}}
	var combined uint8

	// Act
	for x := 0; x < 2; x++ {
		for y := 0; y < 4; y++ {
			if Bit(x, y) != want[x][y] {
				t.Errorf("%d,%d got %02x", x, y, Bit(x, y))
			}
			combined |= Bit(x, y)
		}
	}

	// Assert
	if combined != 255 || Bit(2, 0) != 0 {
		t.Fatalf("combined %02x", combined)
	}
}

func TestUnit_ResolveSplitsTwoColors(t *testing.T) {
	// Arrange
	dark, light := color.RGB(10, 20, 30), color.RGB(230, 220, 200)
	b := raster.New(1, 1)
	b.Clear(dark)
	b.Plot(0, 0, raster.Frag{Albedo: light, Depth: 3})
	g := cell.New(1, 1)

	// Act
	Resolve(b, g, 0)

	// Assert
	c, _ := g.At(0, 0)
	if c.Rune != rune(0x2801) || c.FG != light || c.BG != dark {
		t.Fatalf("got %U %+v", c.Rune, c)
	}
}

func TestUnit_FlatCellIsSpace(t *testing.T) {
	// Arrange
	var s [8]color.Color
	for i := range s {
		s[i] = color.RGB(5, 6, 7)
	}

	// Act
	c := Encode(s)

	// Assert
	if c.Rune != ' ' || c.BG != s[0] {
		t.Fatalf("got %+v", c)
	}
}

func TestUnit_MajorityBecomesBackground(t *testing.T) {
	// Arrange
	var s [8]color.Color
	for i := range s {
		s[i] = color.RGB(200, 200, 200)
	}
	s[0] = color.RGB(0, 0, 0)

	// Act
	c := Encode(s)

	// Assert
	if c.BG != color.RGB(200, 200, 200) || c.Rune != rune(0x2801) {
		t.Fatalf("got %U %+v", c.Rune, c)
	}
}

func TestUnit_FlatMergesNearColors(t *testing.T) {
	// Arrange
	var s [8]color.Color
	for i := range s {
		s[i] = color.RGB(100, 120, 80)
	}
	s[3] = color.RGB(104, 124, 84)
	s[5] = color.RGB(30, 30, 30)

	// Act
	noisy := EncodeFlat([8]color.Color{s[0], s[0], s[0], s[3], s[0], s[0], s[0], s[3]}, 0)
	calm := EncodeFlat([8]color.Color{s[0], s[0], s[0], s[3], s[0], s[0], s[0], s[3]}, 800)
	edge := EncodeFlat(s, 800)

	// Assert
	if noisy.Rune == ' ' || calm.Rune != ' ' || edge.Rune == ' ' {
		t.Fatalf("noisy %U calm %U edge %U", noisy.Rune, calm.Rune, edge.Rune)
	}
}

func TestUnit_ResolveBlocksAveragesEachHalf(t *testing.T) {
	// Arrange
	b := raster.New(1, 1)
	b.Clear(color.RGB(0, 0, 0))
	for x := 0; x < 2; x++ {
		for y := 0; y < 2; y++ {
			b.Plot(x, y, raster.Frag{Albedo: color.RGB(200, 100, 0), Depth: 1})
		}
	}
	g := cell.New(1, 1)

	// Act
	ResolveBlocks(b, g)

	// Assert
	c, _ := g.At(0, 0)
	if c.Rune != UpperHalf || c.FG != color.RGB(200, 100, 0) || c.BG != color.RGB(0, 0, 0) {
		t.Fatalf("got %+v", c)
	}
}
