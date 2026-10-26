// Package pixels turns a frame rendered with the Pixels pass into what a
// pixel display shows: an image with one pixel per dot, and the text glyphs
// of overlay cells, which the display draws on top at screen resolution so
// they stay sharp when the image is scaled up.
package pixels

import (
	"image"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/braille"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// Glyph is a character an overlay wrote, at cell position Col, Row.
type Glyph struct {
	Col, Row int
	Rune     rune
	FG       color.Color
}

// Compose writes the frame's dots into dst, resized to the dot buffer, and
// appends the glyphs left to draw to glyphs. A cell with rune 0 shows its
// dots. An overlay cell fills its dots with its background unless that is
// the cell average, which an overlay reading the cell beneath copied, so
// text without a panel keeps the scene behind it. Braille and half block
// glyph art becomes dots, as a terminal would draw it.
func Compose(dst *image.RGBA, f platform.Frame, glyphs []Glyph) (*image.RGBA, []Glyph) {
	b := f.Dots
	if dst == nil || dst.Rect.Dx() != b.W || dst.Rect.Dy() != b.H {
		dst = image.NewRGBA(image.Rect(0, 0, b.W, b.H))
	}
	for i, d := range b.Dots() {
		put(dst, i, d.Color)
	}
	g := f.Cells
	for row := range min(g.H, b.Rows) {
		for col := range min(g.W, b.Cols) {
			c, _ := g.At(col, row)
			if c.Rune == 0 {
				continue
			}
			mask, glyph := uint8(0), false
			switch {
			case c.Rune >= 0x2800 && c.Rune <= 0x28ff:
				mask = uint8(c.Rune - 0x2800)
			case c.Rune == braille.UpperHalf:
				mask = 0x1b // the top two dot rows
			case c.Rune != ' ':
				glyph = true
			}
			if mask == 0 && c.BG == b.CellAverage(col, row) {
				if glyph {
					glyphs = append(glyphs, Glyph{col, row, c.Rune, c.FG})
				}
				continue
			}
			for dy := range raster.SubH {
				for dx := range raster.SubW {
					x, y := col*raster.SubW+dx, row*raster.SubH+dy
					if mask&braille.Bit(dx, dy) != 0 {
						put(dst, y*b.W+x, c.FG)
					} else {
						put(dst, y*b.W+x, c.BG)
					}
				}
			}
			if glyph {
				glyphs = append(glyphs, Glyph{col, row, c.Rune, c.FG})
			}
		}
	}
	return dst, glyphs
}

func put(dst *image.RGBA, i int, c color.Color) {
	p := dst.Pix[i*4 : i*4+4 : i*4+4]
	p[0], p[1], p[2], p[3] = c.R, c.G, c.B, 255
}
