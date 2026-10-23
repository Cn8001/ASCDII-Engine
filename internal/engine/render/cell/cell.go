// Package cell is the backend independent character framebuffer. The terminal
// presenter and the editor both consume it.
package cell

import "gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"

type Cell struct {
	Rune   rune
	FG, BG color.Color
}

type Grid struct {
	W, H  int
	cells []Cell
}

// New clamps to one cell so callers never need to guard a zero sized viewport.
func New(w, h int) *Grid {
	g := &Grid{}
	g.Resize(w, h)
	return g
}

func (g *Grid) Resize(w, h int) bool {
	w, h = max(1, w), max(1, h)
	if g.W == w && g.H == h && g.cells != nil {
		return false
	}
	g.W, g.H = w, h
	g.cells = make([]Cell, w*h)
	g.Clear(Cell{Rune: ' '})
	return true
}

func (g *Grid) Clear(c Cell) {
	for i := range g.cells {
		g.cells[i] = c
	}
}

func (g *Grid) Inside(x, y int) bool { return x >= 0 && y >= 0 && x < g.W && y < g.H }

func (g *Grid) Set(x, y int, c Cell) {
	if !g.Inside(x, y) {
		return
	}
	if c.Rune == 0 {
		c.Rune = ' '
	}
	g.cells[y*g.W+x] = c
}

func (g *Grid) At(x, y int) (Cell, bool) {
	if !g.Inside(x, y) {
		return Cell{}, false
	}
	return g.cells[y*g.W+x], true
}

// Text writes a string and returns the column after the last rune.
func (g *Grid) Text(x, y int, s string, fg, bg color.Color) int {
	for _, r := range s {
		g.Set(x, y, Cell{Rune: r, FG: fg, BG: bg})
		x++
	}
	return x
}

func (g *Grid) Fill(x, y, w, h int, c Cell) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			g.Set(xx, yy, c)
		}
	}
}

// Cells exposes the backing slice row by row for encoders; callers must not resize it.
func (g *Grid) Cells() []Cell { return g.cells }
