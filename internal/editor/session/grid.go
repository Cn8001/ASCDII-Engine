package session

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// grid draws a ground grid on the dots nothing else covered, in the Scene
// view while editing, so an empty scene still shows where the ground is.
// Lines every unit thin out to every 10 or 100 as the view zooms out.
func (s *Session) grid(f *pipeline.Frame) {
	if s.viewMode != SceneView || s.play != nil {
		return
	}
	v, b := f.View, f.Buf
	o := v.Unproject(0, 0, 0)
	dx, dy := v.Unproject(1, 0, 0).Sub(o), v.Unproject(0, 1, 0).Sub(o)
	step := math.Max(dx.Len(), dy.Len()) // world units one dot covers
	if step <= 0 || math.IsNaN(step) {
		return
	}
	unit := 1.0
	for unit/step < 4 {
		unit *= 10
	}
	half := step * .55
	near := func(c, u float64) bool {
		r := math.Mod(c, u)
		if r < 0 {
			r += u
		}
		return r < half || u-r < half
	}
	minor, major := color.RGB(46, 46, 44), color.RGB(70, 68, 64)
	axisX, axisY := color.RGB(118, 62, 52), color.RGB(70, 104, 64)
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			d := b.At(x, y)
			if d.Flags&raster.FlagEmpty == 0 {
				continue
			}
			p := o.Add(dx.Scale(float64(x) + .5)).Add(dy.Scale(float64(y) + .5))
			var c color.Color
			switch {
			case math.Abs(p.Y) < half:
				c = axisX
			case math.Abs(p.X) < half:
				c = axisY
			case near(p.X, unit*10) || near(p.Y, unit*10):
				c = major
			case near(p.X, unit) || near(p.Y, unit):
				c = minor
			default:
				continue
			}
			d.Color, d.Albedo = c, c
		}
	}
}
