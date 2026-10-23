package pipeline

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/braille"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/par"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
)

func isEditable(p Pass) bool {
	_, ok := p.(prop.Editable)
	return ok
}

// Standard is the default chain; scene passes are added by the caller.
func Standard() []Pass {
	return []Pass{&Clear{}, &Shadows{Strength: 1}, &Lit{}, NewWater(), NewFog(), NewOutline(), NewGrade(), NewBraille()}
}

type Clear struct{}

func (*Clear) Name() string { return "clear" }
func (*Clear) Stage() Stage { return Geometry }
func (*Clear) Run(f *Frame) {
	f.Buf.Clear(f.Env.Sky)
	f.Cells.Clear(cell.Cell{Rune: ' ', BG: f.Env.Sky})
}

type Shadows struct{ Strength float64 }

func (*Shadows) Name() string { return "shadows" }
func (*Shadows) Stage() Stage { return Lighting }

func (s *Shadows) Props() []prop.Prop {
	return prop.Bind().Float("strength", &s.Strength, 0, 1).Props()
}
func (s *Shadows) Set(n string, v any) error {
	return prop.Bind().Float("strength", &s.Strength, 0, 1).Set(n, v)
}

// Run stretches contact shadows away from the sun and darkens only dots lying
// near the caster's ground plane, so the caster never shadows itself.
func (s *Shadows) Run(f *Frame) {
	sun := f.Env.Sun.Dir.Norm()
	lean := geom.V2(-sun.X, -sun.Y).Scale(1 / math.Max(sun.Z, .35))
	for _, c := range f.Ctx.Shadows {
		if c.RX <= 0 || c.RY <= 0 {
			continue
		}
		ctr := c.At.XY().Add(lean.Scale(c.RX * .45))
		x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, d := range [4]geom.Vec2{{X: -c.RX, Y: -c.RY}, {X: c.RX, Y: -c.RY}, {X: -c.RX, Y: c.RY}, {X: c.RX, Y: c.RY}} {
			x, y, _ := f.View.Project(geom.V3(ctr.X+d.X, ctr.Y+d.Y, c.At.Z))
			x0, y0, x1, y1 = math.Min(x0, x), math.Min(y0, y), math.Max(x1, x), math.Max(y1, y)
		}
		k := c.Strength * s.Strength
		for y := max(0, int(y0)); y <= min(f.Buf.H-1, int(y1)); y++ {
			for x := max(0, int(x0)); x <= min(f.Buf.W-1, int(x1)); x++ {
				d := f.Buf.At(x, y)
				if d.Flags&raster.FlagEmpty != 0 || float64(d.Z) > c.At.Z+.35 || float64(d.Z) < c.At.Z-1.5 {
					continue
				}
				dx, dy := (float64(d.X)-ctr.X)/c.RX, (float64(d.Y)-ctr.Y)/c.RY
				r2 := dx*dx + dy*dy
				if r2 >= 1 {
					continue
				}
				v := uint8(geom.Clamp(k*math.Sqrt(1-r2), 0, 1) * 255)
				d.Shadow = max(d.Shadow, v)
			}
		}
	}
}

type Lit struct{}

func (*Lit) Name() string { return "lighting" }
func (*Lit) Stage() Stage { return Lighting }
func (*Lit) Run(f *Frame) {
	sh := light.NewShader(f.Env, f.Ctx.Lights, f.Time)
	dots, w := f.Buf.Dots(), f.Buf.W
	par.Rows(f.Buf.H, func(lo, hi int) {
		for i := lo * w; i < hi*w; i++ {
			d := &dots[i]
			if d.Flags&raster.FlagUnlit != 0 {
				continue
			}
			r, g, b := sh.Irradiance(d.Normal.Vec(), d.World(), float64(d.Shadow)/255)
			e := float64(d.Emissive) / 255
			ar, ag, ab := d.Albedo.Float()
			d.Color = color.FromFloat(ar*(r*(1-e)+e*1.15), ag*(g*(1-e)+e*1.15), ab*(b*(1-e)+e*1.15))
		}
	})
}

type Water struct{ Strength, Speed float64 }

func NewWater() *Water { return &Water{Strength: .8, Speed: 1} }

func (*Water) Name() string { return "water" }
func (*Water) Stage() Stage { return Post }
func (w *Water) bind() *prop.Binder {
	return prop.Bind().Float("strength", &w.Strength, 0, 2).Float("speed", &w.Speed, 0, 4)
}
func (w *Water) Props() []prop.Prop        { return w.bind().Props() }
func (w *Water) Set(n string, v any) error { return w.bind().Set(n, v) }

// Run draws drifting glints keyed to world position so ripples stay put on screen.
func (w *Water) Run(f *Frame) {
	t := f.Time * w.Speed
	glint := color.Mix(f.Env.Sun.Color, color.RGB(255, 255, 255), .4)
	dots, width := f.Buf.Dots(), f.Buf.W
	par.Rows(f.Buf.H, func(lo, hi int) {
		for i := lo * width; i < hi*width; i++ {
			d := &dots[i]
			if d.Flags&raster.FlagWater == 0 {
				continue
			}
			x, y := float64(d.X), float64(d.Y)
			wave := math.Sin(x*1.7+t*1.9+math.Sin(y*.9+t*.7)*1.6) * math.Sin(y*2.3-t*1.3)
			switch {
			case wave > .78:
				d.Color = color.Mix(d.Color, glint, (wave-.78)/.22*.7*w.Strength)
			case wave < -.6:
				d.Color = color.Scale(d.Color, 1-(-.6-wave)*.5*w.Strength)
			}
		}
	})
}

type Fog struct{ Density, Start, End, Height float64 }

func NewFog() *Fog { return &Fog{Density: .35, Start: 6, End: 40, Height: 1.2} }

func (*Fog) Name() string { return "fog" }
func (*Fog) Stage() Stage { return Post }
func (g *Fog) bind() *prop.Binder {
	return prop.Bind().Float("density", &g.Density, 0, 1).Float("start", &g.Start, 0, 200).
		Float("end", &g.End, 1, 400).Float("height", &g.Height, 0, 10)
}
func (g *Fog) Props() []prop.Prop        { return g.bind().Props() }
func (g *Fog) Set(n string, v any) error { return g.bind().Set(n, v) }

// Run fades with distance behind the camera target plus a low ground mist.
func (g *Fog) Run(f *Frame) {
	if g.Density <= 0 {
		return
	}
	dots, w := f.Buf.Dots(), f.Buf.W
	par.Rows(f.Buf.H, func(lo, hi int) {
		for i := lo * w; i < hi*w; i++ {
			d := &dots[i]
			if d.Flags&raster.FlagEmpty != 0 {
				continue
			}
			far := geom.Smoothstep(g.Start, g.End, -float64(d.Depth))
			low := 0.0
			if g.Height > 0 {
				low = (1 - geom.Smoothstep(-.5, g.Height, float64(d.Z))) * .35
			}
			if k := math.Min(1, (far+low)*g.Density); k > .004 {
				d.Color = color.Mix(d.Color, f.Env.Fog, k)
			}
		}
	})
}

type Outline struct{ Threshold, Strength float64 }

func NewOutline() *Outline { return &Outline{Threshold: 1.2, Strength: .3} }

func (*Outline) Name() string { return "outline" }
func (*Outline) Stage() Stage { return Post }
func (o *Outline) bind() *prop.Binder {
	return prop.Bind().Float("threshold", &o.Threshold, .05, 20).Float("strength", &o.Strength, 0, 1)
}
func (o *Outline) Props() []prop.Prop        { return o.bind().Props() }
func (o *Outline) Set(n string, v any) error { return o.bind().Set(n, v) }

// Run darkens the near side of depth breaks; at two colors per cell a crisp
// silhouette does more for legibility than any shading.
func (o *Outline) Run(f *Frame) {
	b := f.Buf
	// Reads neighbour depth only and writes own color, so bands never race.
	par.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			for x := 0; x < b.W; x++ {
				d := b.At(x, y)
				if d.Flags&raster.FlagEmpty != 0 {
					continue
				}
				for _, n := range [4]*raster.Dot{b.At(x+1, y), b.At(x-1, y), b.At(x, y+1), b.At(x, y-1)} {
					if n != nil && d.Depth-n.Depth > float32(o.Threshold) {
						d.Color = color.Scale(d.Color, 1-o.Strength)
						break
					}
				}
			}
		}
	})
}

type Grade struct{ Exposure, Contrast, Saturation, Warmth, Vignette float64 }

func NewGrade() *Grade {
	return &Grade{Exposure: 1, Contrast: 1.08, Saturation: 1.1, Warmth: .04, Vignette: .25}
}

func (*Grade) Name() string { return "grade" }
func (*Grade) Stage() Stage { return Post }
func (g *Grade) bind() *prop.Binder {
	return prop.Bind().Float("exposure", &g.Exposure, 0, 3).Float("contrast", &g.Contrast, .5, 2).
		Float("saturation", &g.Saturation, 0, 2).Float("warmth", &g.Warmth, -.5, .5).Float("vignette", &g.Vignette, 0, 1)
}
func (g *Grade) Props() []prop.Prop        { return g.bind().Props() }
func (g *Grade) Set(n string, v any) error { return g.bind().Set(n, v) }

func (g *Grade) Run(f *Frame) {
	b := f.Buf
	cx, cy := float64(b.W)/2, float64(b.H)/2
	par.Rows(b.H, func(lo, hi int) { g.rows(b, cx, cy, lo, hi) })
}

func (g *Grade) rows(b *raster.Buffer, cx, cy float64, lo, hi int) {
	for y := lo; y < hi; y++ {
		for x := 0; x < b.W; x++ {
			d := b.At(x, y)
			r, gr, bl := d.Color.Float()
			r, gr, bl = r*g.Exposure, gr*g.Exposure, bl*g.Exposure
			l := .299*r + .587*gr + .114*bl
			r, gr, bl = l+(r-l)*g.Saturation, l+(gr-l)*g.Saturation, l+(bl-l)*g.Saturation
			r, gr, bl = (r-.5)*g.Contrast+.5, (gr-.5)*g.Contrast+.5, (bl-.5)*g.Contrast+.5
			r, bl = r+g.Warmth*.1, bl-g.Warmth*.1
			if g.Vignette > 0 {
				dx, dy := (float64(x)-cx)/cx, (float64(y)-cy)/cy
				k := 1 - g.Vignette*geom.Smoothstep(.45, 1.5, dx*dx+dy*dy)
				r, gr, bl = r*k, gr*k, bl*k
			}
			d.Color = color.FromFloat(r, gr, bl)
		}
	}
}

// Braille encodes dots into glyphs; Calm is how different two colors in one
// cell must be before they are drawn as dots instead of a flat block.
type Braille struct{ Calm float64 }

func NewBraille() *Braille { return &Braille{Calm: 520} }

func (*Braille) Name() string                { return "braille" }
func (*Braille) Stage() Stage                { return Resolve }
func (b *Braille) bind() *prop.Binder        { return prop.Bind().Float("calm", &b.Calm, 0, 6000) }
func (b *Braille) Props() []prop.Prop        { return b.bind().Props() }
func (b *Braille) Set(n string, v any) error { return b.bind().Set(n, v) }
func (b *Braille) Run(f *Frame)              { braille.Resolve(f.Buf, f.Cells, int(b.Calm)) }
