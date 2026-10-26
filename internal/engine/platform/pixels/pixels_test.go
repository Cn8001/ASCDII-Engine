package pixels

import (
	"image"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

func at(img *image.RGBA, x, y int) color.Color {
	c := img.RGBAAt(x, y)
	return color.RGB(c.R, c.G, c.B)
}

func TestUnit_ComposeCopiesDotsAndDrawsOverlay(t *testing.T) {
	// Arrange
	sky, red, panel, ink := color.RGB(20, 40, 60), color.RGB(255, 0, 0), color.RGB(9, 9, 9), color.RGB(250, 250, 250)
	f := pipeline.NewFrame(4, 1)
	f.Buf.Clear(sky)
	f.Buf.Plot(1, 2, raster.Frag{Albedo: red, Depth: 1})
	(&pipeline.Pixels{}).Run(f)
	under, _ := f.Cells.At(0, 0)
	f.Cells.Set(0, 0, cell.Cell{Rune: 'a', FG: ink, BG: under.BG})
	f.Cells.Set(1, 0, cell.Cell{Rune: 'b', FG: ink, BG: panel})
	f.Cells.Set(2, 0, cell.Cell{Rune: 0x2801, FG: red, BG: panel})

	// Act
	img, glyphs := Compose(nil, platform.Frame{Cells: f.Cells, Dots: f.Buf}, nil)

	// Assert
	if img.Rect.Dx() != 8 || img.Rect.Dy() != 4 {
		t.Fatalf("image %v, want one pixel per dot", img.Rect)
	}
	if at(img, 1, 2) != red || at(img, 0, 0) != sky {
		t.Fatalf("a label without a panel must keep the dots: %v %v", at(img, 1, 2), at(img, 0, 0))
	}
	if at(img, 3, 3) != panel {
		t.Fatalf("a label with a panel fills its cell: %v", at(img, 3, 3))
	}
	if at(img, 4, 0) != red || at(img, 5, 0) != panel {
		t.Fatalf("braille art becomes dots: %v %v", at(img, 4, 0), at(img, 5, 0))
	}
	if at(img, 6, 0) != sky {
		t.Fatalf("an untouched cell shows its dots: %v", at(img, 6, 0))
	}
	if len(glyphs) != 2 || glyphs[0] != (Glyph{0, 0, 'a', ink}) || glyphs[1] != (Glyph{1, 0, 'b', ink}) {
		t.Fatalf("glyphs %+v", glyphs)
	}
}

func TestUnit_ComposeReusesImageOfSameSize(t *testing.T) {
	// Arrange
	f := pipeline.NewFrame(2, 2)
	(&pipeline.Pixels{}).Run(f)
	fr := platform.Frame{Cells: f.Cells, Dots: f.Buf}
	first, _ := Compose(nil, fr, nil)

	// Act
	second, _ := Compose(first, fr, nil)
	f.Resize(3, 2)
	(&pipeline.Pixels{}).Run(f)
	third, _ := Compose(second, fr, nil)

	// Assert
	if second != first || third == first || third.Rect.Dx() != 6 {
		t.Fatalf("reuse %v %v %v", second == first, third == first, third.Rect)
	}
}
