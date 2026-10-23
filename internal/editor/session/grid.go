package session

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// grid draws a ground grid on the dots nothing else covered, in the Scene
// view while editing, so an empty scene still shows where the ground is.
// Lines thin out as the view zooms out, like a CAD grid.
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
	// The finest of 1, 2, 5, 10, 20, 50... units that keeps lines 8 dots apart.
	unit := 1.0
	for i := 0; unit/step < 8; i++ {
		unit *= [3]float64{2, 2.5, 2}[i%3]
	}
	major := unit * 5
	half := step * .55
	near := func(c, u float64) bool {
		r := math.Mod(c, u)
		if r < 0 {
			r += u
		}
		return r < half || u-r < half
	}
	ink := color.RGB(20, 20, 20)
	axisX, axisY := color.RGB(170, 70, 56), color.RGB(80, 140, 72)
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			d := b.At(x, y)
			if d.Flags&raster.FlagEmpty == 0 {
				continue
			}
			p := o.Add(dx.Scale(float64(x) + .5)).Add(dy.Scale(float64(y) + .5))
			// Lines are tinted from the background, so they stay quiet on any sky.
			var c color.Color
			switch {
			case math.Abs(p.Y) < half:
				c = color.Mix(d.Color, axisX, .55)
			case math.Abs(p.X) < half:
				c = color.Mix(d.Color, axisY, .55)
			case near(p.X, major) || near(p.Y, major):
				c = color.Mix(d.Color, ink, .22)
			case near(p.X, unit) || near(p.Y, unit):
				c = color.Mix(d.Color, ink, .1)
			default:
				continue
			}
			d.Color, d.Albedo = c, c
		}
	}
}
