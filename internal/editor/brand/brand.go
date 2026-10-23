// Package brand draws the ASCDII Engine mark: one Braille cell, 2x4 dots, the
// unit every frame the engine renders is made of. Lit dots form a rising step,
// a nod to the 2.5D view.
package brand

import (
	"image"
	"image/color"
	"math"
)

// Lit lists the lit dots as column, row pairs of the 2x4 cell.
func Lit() [][2]int { return [][2]int{{1, 0}, {0, 1}, {1, 1}, {0, 2}, {0, 3}, {1, 3}} }

func lit(c, r int) bool {
	for _, d := range Lit() {
		if d[0] == c && d[1] == r {
			return true
		}
	}
	return false
}

// Program picks the badge that tells the programs apart; they share the mark.
// Launcher is ascdii, which opens the editor and is the CLI.
type Program int

const (
	Launcher Program = iota
	Editor
	Player
)

func (p Program) String() string {
	return [...]string{"launcher", "editor", "player"}[p]
}

// Programs lists every program that has an icon.
func Programs() []Program { return []Program{Launcher, Editor, Player} }

// Icon is the editor's icon, the one its window shows.
func Icon(size int) *image.RGBA { return Draw(Editor, size) }

// Draw renders the mark with p's badge at size px, with 4x4 supersampling
// for clean edges.
func Draw(p Program, size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	s := float64(size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var acc [4]float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+.5)/ss
					py := float64(y) + (float64(sy)+.5)/ss
					col, ok := sample(p, px/s, py/s)
					if !ok {
						continue
					}
					acc[0], acc[1], acc[2], acc[3] = acc[0]+col[0], acc[1]+col[1], acc[2]+col[2], acc[3]+1
				}
			}
			if acc[3] == 0 {
				continue
			}
			a := acc[3] / (ss * ss)
			img.SetRGBA(x, y, color.RGBA{uint8(acc[0] / acc[3] * a), uint8(acc[1] / acc[3] * a), uint8(acc[2] / acc[3] * a), uint8(a * 255)})
		}
	}
	return img
}

var (
	amber = [3]float64{232, 176, 75}
	dim   = [3]float64{46, 44, 41}
	deep  = [3]float64{107, 72, 28}
	ink   = [3]float64{16, 15, 14}
)

// accent is each badge's color; the launcher shares the mark's amber.
func (p Program) accent() [3]float64 {
	return [...][3]float64{{240, 128, 88}, {84, 188, 200}, {112, 200, 120}}[p]
}

// sample is the color at x, y in units of the icon size, false outside it.
func sample(p Program, x, y float64) ([3]float64, bool) {
	const radius = .22
	if !inRounded(x, y, 1, radius) {
		return [3]float64{}, false
	}
	col := [3]float64{geomLerp(26, 12, y), geomLerp(25, 12, y), geomLerp(24, 12, y)}
	const dotR, off = .074, .028 // off is the extrusion that gives each lit dot depth
	cx := []float64{.37, .6}
	cy := []float64{.21, .4, .59, .78}
	for c := 0; c < 2; c++ {
		for r := 0; r < 4; r++ {
			d := math.Hypot(x-cx[c], y-cy[r])
			if lit(c, r) {
				if g := 1 - d/(dotR*2.4); g > 0 {
					col = mix(col, amber, g*g*.3)
				}
				if math.Hypot(x-cx[c]-off, y-cy[r]-off) < dotR && d >= dotR {
					col = deep
				}
			}
			if d < dotR {
				if lit(c, r) {
					col = mix(amber, [3]float64{255, 226, 160}, math.Max(0, 1-d/dotR)*.35)
				} else {
					col = dim
				}
			}
		}
	}
	// The badge sits in the lower right corner, cut out of the mark by a ring
	// of the background so it reads at 16 px.
	const bx, by, br, ring = .77, .77, .19, .035
	d := math.Hypot(x-bx, y-by)
	switch {
	case d < br:
		col = p.accent()
		if glyph(p, (x-bx)/br, (y-by)/br) {
			col = ink
		}
	case d < br+ring:
		col = [3]float64{14, 14, 13}
	}
	return col, true
}

// glyph reports whether u, v (badge units, -1..1, y down) is on p's symbol:
// a rising arrow to start things, a pencil, a play triangle.
func glyph(p Program, u, v float64) bool {
	const w = .14
	switch p {
	case Launcher:
		return segment(u, v, -.38, .38, .38, -.38, w) || segment(u, v, .38, -.38, -.06, -.38, w) || segment(u, v, .38, -.38, .38, .06, w)
	case Editor:
		along, across := (u-v)/math.Sqrt2, math.Abs(u+v)/math.Sqrt2
		const half = .17
		body := along > -.2 && along < .52 && across < half && (along < .26 || along > .33)
		tip := along > -.56 && along <= -.2 && across < half*(along+.56)/.36
		return body || tip
	case Player:
		return u > -.3 && math.Abs(v) < (.42-u)*.66 && u < .5
	}
	return false
}

// segment is a capsule of half width w around the line from a to b.
func segment(u, v, ax, ay, bx, by, w float64) bool {
	dx, dy := bx-ax, by-ay
	t := math.Max(0, math.Min(1, ((u-ax)*dx+(v-ay)*dy)/(dx*dx+dy*dy)))
	return math.Hypot(u-ax-t*dx, v-ay-t*dy) < w
}

func inRounded(x, y, s, r float64) bool {
	qx := math.Max(math.Abs(x-s/2)-(s/2-r), 0)
	qy := math.Max(math.Abs(y-s/2)-(s/2-r), 0)
	return qx*qx+qy*qy <= r*r
}

func mix(a, b [3]float64, t float64) [3]float64 {
	t = math.Max(0, math.Min(1, t))
	return [3]float64{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t, a[2] + (b[2]-a[2])*t}
}

func geomLerp(a, b, t float64) float64 { return a + (b-a)*t }
