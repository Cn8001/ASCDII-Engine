// Package braille reduces 2x4 colored dots to one Unicode Braille glyph with a
// single foreground and background, the only colors a terminal cell can show.
package braille

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/par"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// Bit follows the Unicode dot numbering, where the bottom row was added last.
func Bit(x, y int) uint8 {
	if x < 0 || x > 1 || y < 0 || y > 3 {
		return 0
	}
	if y == 3 {
		return 0x40 << x
	}
	return 1 << (y + 3*x)
}

// distance weights green highest because the eye separates it best.
func distance(a, b color.Color) int {
	r := int(a.R) - int(b.R)
	g := int(a.G) - int(b.G)
	bl := int(a.B) - int(b.B)
	return 2*r*r + 4*g*g + 3*bl*bl
}

func mean(samples *[8]color.Color, groups *[8]bool, selected bool) color.Color {
	r, g, b, n := 0, 0, 0, 0
	for i, c := range samples {
		if groups[i] == selected {
			r, g, b, n = r+int(c.R), g+int(c.G), b+int(c.B), n+1
		}
	}
	if n == 0 {
		return samples[0]
	}
	return color.Color{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n)}
}

// Encode runs a deterministic 2-means pass. Unlike a brightness threshold this
// keeps chromatic edges such as dark stone against bright water.
func Encode(samples [8]color.Color) cell.Cell { return EncodeFlat(samples, 0) }

// EncodeFlat merges a cell to one flat color when its two clusters are closer
// than flat. Texture noise then stops turning into random dot patterns, which
// is what makes a Braille image tiring to look at; real edges stay sharp.
func EncodeFlat(samples [8]color.Color, flat int) cell.Cell {
	same := true
	for i := 1; i < 8; i++ {
		if samples[i] != samples[0] {
			same = false
			break
		}
	}
	if same {
		return cell.Cell{Rune: ' ', FG: samples[0], BG: samples[0]}
	}
	first, furthest, far := samples[0], samples[0], -1
	for _, c := range samples {
		if d := distance(first, c); d > far {
			far, furthest = d, c
		}
	}
	fg, bg := furthest, first
	var groups [8]bool
	for iter := 0; iter < 4; iter++ {
		fgN := 0
		for i, c := range samples {
			groups[i] = distance(c, fg) < distance(c, bg)
			if groups[i] {
				fgN++
			}
		}
		if fgN == 0 || fgN == 8 {
			break
		}
		fg, bg = mean(&samples, &groups, true), mean(&samples, &groups, false)
	}
	if flat > 0 && distance(fg, bg) < flat {
		m := mean(&samples, &[8]bool{}, false)
		return cell.Cell{Rune: ' ', FG: m, BG: m}
	}
	var mask uint8
	fgN := 0
	for i, c := range samples {
		if distance(c, fg) < distance(c, bg) {
			mask |= Bit(i%2, i/2)
			fgN++
		}
	}
	switch mask {
	case 0:
		return cell.Cell{Rune: ' ', FG: bg, BG: bg}
	case 255:
		return cell.Cell{Rune: ' ', FG: fg, BG: fg}
	}
	if fgN > 4 {
		mask, fg, bg = ^mask, bg, fg
	}
	return cell.Cell{Rune: rune(0x2800) + rune(mask), FG: fg, BG: bg}
}

// Resolve encodes the final dot colors of buf into g, resizing g to match.
func Resolve(buf *raster.Buffer, g *cell.Grid, flat int) {
	g.Resize(buf.Cols, buf.Rows)
	dots := buf.Dots()
	par.Rows(buf.Rows, func(lo, hi int) {
		var samples [8]color.Color
		for cy := lo; cy < hi; cy++ {
			for cx := 0; cx < buf.Cols; cx++ {
				for dy := 0; dy < raster.SubH; dy++ {
					row := (cy*raster.SubH + dy) * buf.W
					for dx := 0; dx < raster.SubW; dx++ {
						samples[dy*2+dx] = dots[row+cx*raster.SubW+dx].Color
					}
				}
				g.Set(cx, cy, EncodeFlat(samples, flat))
			}
		}
	})
}
