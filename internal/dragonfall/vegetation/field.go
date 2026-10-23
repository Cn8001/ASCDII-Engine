// Package vegetation is the grass spring field: blades bend from wind and
// from whoever walks through them, then spring back.
package vegetation

import "math"

type key struct{ X, Y int }

type spring struct{ Offset, Velocity float64 }

type Field struct {
	Time    float64
	patches map[key]spring
}

func New() *Field { return &Field{patches: map[key]spring{}} }

func (f *Field) Disturb(x, y int, dx float64) {
	if dx == 0 {
		return
	}
	for oy := -2; oy <= 2; oy++ {
		for ox := -3; ox <= 3; ox++ {
			d2 := float64(ox*ox + oy*oy)
			if d2 > 9 {
				continue
			}
			k := key{x + ox, y + oy}
			s := f.patches[k]
			s.Velocity += dx * (1 - d2/12) * 4
			f.patches[k] = s
		}
	}
}

// Step integrates damped springs with a clamped dt for stability.
func (f *Field) Step(dt float64) {
	dt = math.Max(0, math.Min(dt, .07))
	f.Time += dt
	for k, s := range f.patches {
		s.Velocity += (-24*s.Offset - 8*s.Velocity) * dt
		s.Offset = math.Max(-2.8, math.Min(2.8, s.Offset+s.Velocity*dt))
		if math.Abs(s.Velocity) < .008 && math.Abs(s.Offset) < .008 {
			delete(f.patches, k)
			continue
		}
		f.patches[k] = s
	}
}

// BendAt combines two wind harmonics with the local spring offset.
func (f *Field) BendAt(x, y int, t float64) float64 {
	wind := .75*math.Sin(t*1.45+float64(x)*.13+float64(y)*.16) + .3*math.Sin(t*.62+float64(y)*.037)
	return wind + f.patches[key{x, y}].Offset
}
