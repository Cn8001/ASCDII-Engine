// Package texture turns images into mip mapped textures the rasterizer
// samples per Braille dot.
package texture

import (
	"image"
	"math"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/color"
)

type Filter string

const (
	Nearest Filter = "nearest" // crisp texels, for pixel art
	Linear  Filter = "linear"  // blended texels, for photos and painted art
)

type Options struct {
	MaxSize int     `json:"maxSize"` // longest side kept, 0 for 256
	Filter  Filter  `json:"filter"`
	Cutoff  float64 `json:"alphaCutoff"` // texels below this alpha are holes, 0 for .5
}

func (o Options) withDefaults() Options {
	if o.MaxSize <= 0 {
		o.MaxSize = 256
	}
	if o.Filter == "" {
		o.Filter = Linear
	}
	if o.Cutoff <= 0 {
		o.Cutoff = .5
	}
	return o
}

type level struct {
	w, h int
	pix  []uint8 // RGBA rows
}

type Texture struct {
	levels []level
	filter Filter
	cutoff uint8
}

// FromImage copies img, shrinking it to MaxSize, and builds its mip chain.
func FromImage(img image.Image, o Options) *Texture {
	o = o.withDefaults()
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		w, h = 1, 1
	}
	k := math.Min(1, float64(o.MaxSize)/float64(max(w, h)))
	tw, th := max(1, int(float64(w)*k)), max(1, int(float64(h)*k))
	base := level{w: tw, h: th, pix: make([]uint8, tw*th*4)}
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			var r, g, bl, a uint32
			n := uint32(0)
			// Average the source texels under this one, so shrinking does not alias.
			sx0, sx1 := b.Min.X+x*w/tw, b.Min.X+max((x+1)*w/tw, x*w/tw+1)
			sy0, sy1 := b.Min.Y+y*h/th, b.Min.Y+max((y+1)*h/th, y*h/th+1)
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, ca := img.At(sx, sy).RGBA()
					r, g, bl, a, n = r+cr, g+cg, bl+cb, a+ca, n+1
				}
			}
			i := (y*tw + x) * 4
			base.pix[i], base.pix[i+1], base.pix[i+2], base.pix[i+3] = unmul(r, a, n), unmul(g, a, n), unmul(bl, a, n), uint8(a/n>>8)
		}
	}
	t := &Texture{levels: []level{base}, filter: o.Filter, cutoff: uint8(o.Cutoff * 255)}
	for l := base; l.w > 1 || l.h > 1; {
		l = half(l)
		t.levels = append(t.levels, l)
	}
	return t
}

// unmul undoes the premultiplied alpha image.Color.RGBA returns.
func unmul(c, a, n uint32) uint8 {
	if a == 0 {
		return 0
	}
	return uint8(min(255, uint64(c)*255/uint64(a)))
}

func half(l level) level {
	w, h := max(1, l.w/2), max(1, l.h/2)
	out := level{w: w, h: h, pix: make([]uint8, w*h*4)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum [4]int
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					sx, sy := min(l.w-1, x*2+dx), min(l.h-1, y*2+dy)
					i := (sy*l.w + sx) * 4
					for c := range sum {
						sum[c] += int(l.pix[i+c])
					}
				}
			}
			i := (y*w + x) * 4
			for c := range sum {
				out.pix[i+c] = uint8(sum[c] / 4)
			}
		}
	}
	return out
}

func (t *Texture) Size() (int, int) { return t.levels[0].w, t.levels[0].h }

func (t *Texture) Levels() int { return len(t.levels) }

// Sample reads the texture at u, v (repeating outside 0..1) from the mip
// level lod; false means a hole under the alpha cutoff.
func (t *Texture) Sample(lod, u, v float64) (color.Color, bool) {
	l := t.levels[max(0, min(len(t.levels)-1, int(lod+.5)))] // rounds as math.Round for lod >= 0
	u, v = u-math.Floor(u), v-math.Floor(v)
	x, y := u*float64(l.w), (1-v)*float64(l.h) // v grows upward, as in OBJ files
	if t.filter == Nearest || (l.w == 1 && l.h == 1) {
		i := l.index(wrap(int(x), l.w), wrap(int(y), l.h))
		return color.Color{R: l.pix[i], G: l.pix[i+1], B: l.pix[i+2]}, l.pix[i+3] >= t.cutoff
	}
	x, y = x-.5, y-.5
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	// Wrapping once per sample, not per texel, is most of what sampling costs.
	cx0, cy0 := wrap(int(x0), l.w), wrap(int(y0), l.h)
	cx1, cy1 := cx0+1, cy0+1
	if cx1 == l.w {
		cx1 = 0
	}
	if cy1 == l.h {
		cy1 = 0
	}
	a, b, c, d := l.index(cx0, cy0), l.index(cx1, cy0), l.index(cx0, cy1), l.index(cx1, cy1)
	ka, kb, kc, kd := (1-fx)*(1-fy), fx*(1-fy), (1-fx)*fy, fx*fy
	p := l.pix
	mix := func(o int) float64 {
		return float64(p[a+o])*ka + float64(p[b+o])*kb + float64(p[c+o])*kc + float64(p[d+o])*kd
	}
	return color.Color{R: uint8(mix(0) + .5), G: uint8(mix(1) + .5), B: uint8(mix(2) + .5)}, mix(3) >= float64(t.cutoff)
}

func wrap(i, n int) int {
	i %= n
	if i < 0 {
		i += n
	}
	return i
}

func (l level) index(x, y int) int { return (y*l.w + x) * 4 }

// Average is the mean opaque color of the smallest mip level, for places
// that need one color, such as a model without UVs.
func (t *Texture) Average() color.Color {
	l := t.levels[len(t.levels)-1]
	return color.Color{R: l.pix[0], G: l.pix[1], B: l.pix[2]}
}
