package headless

import (
	"bytes"
	"context"
	"image/png"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
)

func TestUnit_PNGHasCellPixels(t *testing.T) {
	// Arrange
	g := cell.New(3, 2)
	g.Set(1, 0, cell.Cell{Rune: 0x2801, FG: color.RGB(255, 0, 0), BG: color.RGB(0, 0, 255)})
	var out bytes.Buffer
	d := &Display{Out: &out, Format: PNG, Cols: 3, Rows: 2}

	// Act
	err := d.Present(context.Background(), platform.Frame{Cells: g})
	img, errDecode := png.Decode(&out)

	// Assert
	if err != nil || errDecode != nil || img.Bounds().Dx() != 24 || img.Bounds().Dy() != 32 {
		t.Fatalf("err %v %v bounds %v", err, errDecode, img.Bounds())
	}
	if r, _, b, _ := img.At(9, 1).RGBA(); r>>8 != 255 || b>>8 != 0 {
		t.Fatalf("dot pixel %v", img.At(9, 1))
	}
}

func TestUnit_PlainPreviewWritesText(t *testing.T) {
	// Arrange
	g := cell.New(2, 1)
	g.Text(0, 0, "ok", color.Color{}, color.Color{})
	var out bytes.Buffer
	d := &Display{Out: &out, Format: Plain}

	// Act
	err := d.Present(context.Background(), platform.Frame{Cells: g})

	// Assert
	if err != nil || out.String() != "ok\n" {
		t.Fatalf("got %q", out.String())
	}
}
