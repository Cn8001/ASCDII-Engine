package launcher

import (
	"math"
	"unicode/utf8"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/braille"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
)

func palette() (bg, panel, text, dim, accent, bad color.Color) {
	return color.RGB(12, 12, 12), color.RGB(22, 22, 22), color.RGB(228, 224, 216), color.RGB(112, 108, 100), color.RGB(232, 176, 75), color.RGB(208, 106, 82)
}

// logo draws the brand mark in Braille: the 2x4 cell, each of its dots a disc,
// lighting up one by one during the first second.
func logo(g *cell.Grid, left, top int, t float64) {
	const cols, rows = 20, 10
	b := raster.New(cols, rows)
	bg, _, _, _, accent, _ := palette()
	b.Clear(bg)
	lit := map[[2]int]bool{{1, 0}: true, {0, 1}: true, {1, 1}: true, {0, 2}: true, {0, 3}: true, {1, 3}: true}
	w, h := float64(b.W), float64(b.H)
	n := 0
	for r := 0; r < 4; r++ {
		for c := 0; c < 2; c++ {
			x := int(math.Round(w * (.38 + .24*float64(c))))
			y := int(math.Round(h * (.13 + .245*float64(r))))
			col := color.RGB(46, 44, 41)
			if lit[[2]int{c, r}] {
				k := math.Max(0, math.Min(1, (t-float64(n)*.09)/.25))
				col = color.Mix(col, accent, k)
				n++
			}
			b.Disc(x, y, 4, raster.Frag{Albedo: col, Depth: 1})
		}
	}
	small := cell.New(cols, rows)
	braille.Resolve(b, small, 0)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			c, _ := small.At(x, y)
			g.Set(left+x, top+y, c)
		}
	}
}

func text(g *cell.Grid, x, y int, s string, fg, bg color.Color) {
	g.Text(x, y, s, fg, bg)
}

func center(g *cell.Grid, y int, s string, fg, bg color.Color) {
	text(g, (g.W-utf8.RuneCountInString(s))/2, y, s, fg, bg)
}

func spaced(s string) string {
	out := make([]rune, 0, len(s)*2)
	for i, r := range s {
		if i > 0 {
			out = append(out, ' ')
		}
		out = append(out, r)
	}
	return string(out)
}

// Render paints the whole launcher for one frame.
func Render(g *cell.Grid, m *Menu, version, status string, t float64) {
	bg, panel, fg, dim, accent, bad := palette()
	g.Clear(cell.Cell{Rune: ' ', BG: bg})
	if g.W < 40 || g.H < 18 {
		center(g, g.H/2, "enlarge the terminal", fg, bg)
		return
	}
	top := max(1, g.H/2-14)
	logo(g, (g.W-20)/2, top, t)
	center(g, top+11, spaced("ASCDII ENGINE"), fg, bg)
	center(g, top+12, version, dim, bg)
	w := min(60, g.W-6)
	left := (g.W - w) / 2
	y := top + 14
	g.Fill(left, y, w, 1, cell.Cell{Rune: ' ', BG: bg})
	text(g, left+1, y, m.Title, dim, bg)
	y += 2
	if m.Prompt {
		g.Fill(left, y, w, 1, cell.Cell{Rune: ' ', BG: panel})
		cursor := " "
		if int(t*2)%2 == 0 {
			cursor = "_"
		}
		shown := []rune(m.Text + cursor)
		if len(shown) > w-4 {
			shown = shown[len(shown)-(w-4):]
		}
		text(g, left+2, y, string(shown), fg, panel)
		y += 2
		text(g, left+1, y, "enter confirm   esc back", dim, bg)
	} else {
		visible := min(len(m.Items), max(3, g.H-y-4))
		first := max(0, min(m.Cursor-visible/2, len(m.Items)-visible))
		for i := first; i < first+visible; i++ {
			it := m.Items[i]
			row, ink, mark := bg, fg, "  "
			if i == m.Cursor {
				row, ink, mark = panel, accent, "> "
			}
			g.Fill(left, y, w, 1, cell.Cell{Rune: ' ', BG: row})
			text(g, left+1, y, mark+it.Label, ink, row)
			if it.Hint != "" {
				hint := []rune(it.Hint)
				room := w - utf8.RuneCountInString(it.Label) - 6
				if room > 6 {
					if len(hint) > room {
						hint = append([]rune{'.', '.'}, hint[len(hint)-room+2:]...)
					}
					text(g, left+w-len(hint)-1, y, string(hint), dim, row)
				}
			}
			y++
		}
		y++
		text(g, left+1, y, "up/down move   enter select   esc back   q quit", dim, bg)
	}
	if status != "" {
		center(g, g.H-2, status, bad, bg)
	}
}
