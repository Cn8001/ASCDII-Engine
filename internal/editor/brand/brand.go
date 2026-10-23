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

// Icon renders the mark at size px with 4x4 supersampling for clean edges.
func Icon(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	const ss = 4
	s := float64(size)
	radius := s * .22
	amber := [3]float64{232, 176, 75}
	dim := [3]float64{46, 44, 41}
	deep := [3]float64{107, 72, 28}
	dotR := s * .074
	cx := []float64{s * .37, s * .6}
	cy := []float64{s * .21, s * .4, s * .59, s * .78}
	off := s * .028 // extrusion offset that gives each lit dot a little depth
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var acc [4]float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+.5)/ss
					py := float64(y) + (float64(sy)+.5)/ss
					if !inRounded(px, py, s, radius) {
						continue
					}
					t := py / s
					bg := [3]float64{geomLerp(26, 12, t), geomLerp(25, 12, t), geomLerp(24, 12, t)}
					col := bg
					for c := 0; c < 2; c++ {
						for r := 0; r < 4; r++ {
							d := math.Hypot(px-cx[c], py-cy[r])
							if lit(c, r) {
								if g := 1 - d/(dotR*2.4); g > 0 {
									col = mix(col, amber, g*g*.3)
								}
								if math.Hypot(px-cx[c]-off, py-cy[r]-off) < dotR && d >= dotR {
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
					acc[0], acc[1], acc[2], acc[3] = acc[0]+col[0], acc[1]+col[1], acc[2]+col[2], acc[3]+1
				}
			}
			n := float64(ss * ss)
			if acc[3] == 0 {
				continue
			}
			a := acc[3] / n
			img.SetRGBA(x, y, color.RGBA{uint8(acc[0] / acc[3] * a), uint8(acc[1] / acc[3] * a), uint8(acc[2] / acc[3] * a), uint8(a * 255)})
		}
	}
	return img
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
