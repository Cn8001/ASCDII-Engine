package braille

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/par"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// UpperHalf is the half block glyph: its foreground fills the top of a cell
// and its background the bottom.
const UpperHalf = '▀'

// ResolveBlocks is the alternative to Resolve for terminals whose fonts draw
// Braille poorly: every cell becomes two solid halves, each the average of
// the four dots it covers. Half the dots per cell, but no gaps and no font.
func ResolveBlocks(buf *raster.Buffer, g *cell.Grid) {
	g.Resize(buf.Cols, buf.Rows)
	dots := buf.Dots()
	par.Rows(buf.Rows, func(lo, hi int) {
		for cy := lo; cy < hi; cy++ {
			for cx := 0; cx < buf.Cols; cx++ {
				var half [2]color.Color
				for h := range 2 {
					var r, gr, b float64
					for dy := h * 2; dy < h*2+2; dy++ {
						row := (cy*raster.SubH + dy) * buf.W
						for dx := 0; dx < raster.SubW; dx++ {
							cr, cg, cb := dots[row+cx*raster.SubW+dx].Color.Float()
							r, gr, b = r+cr, gr+cg, b+cb
						}
					}
					half[h] = color.FromFloat(r/4, gr/4, b/4)
				}
				if half[0] == half[1] {
					g.Set(cx, cy, cell.Cell{Rune: ' ', FG: half[0], BG: half[0]})
					continue
				}
				g.Set(cx, cy, cell.Cell{Rune: UpperHalf, FG: half[0], BG: half[1]})
			}
		}
	})
}
