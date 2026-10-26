// Package raster is the dot level G-buffer. Each terminal cell owns 2x4 dots and
// every dot keeps material, normal, depth and world position so that lighting
// and post passes run after all geometry, independent of draw order.
package raster

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

const (
	SubW = 2
	SubH = 4
)

type Flag uint8

const (
	FlagEmpty Flag = 1 << iota // background, never written by geometry
	FlagUnlit                  // lighting copies albedo untouched
	FlagWater
)

// Normal is a unit vector quantized to int8 to keep the G-buffer small.
type Normal [3]int8

func PackNormal(v geom.Vec3) Normal {
	v = v.Norm()
	return Normal{int8(math.Round(v.X * 127)), int8(math.Round(v.Y * 127)), int8(math.Round(v.Z * 127))}
}

func (n Normal) Vec() geom.Vec3 {
	return geom.V3(float64(n[0])/127, float64(n[1])/127, float64(n[2])/127)
}

// Frag is what a primitive writes; Depth grows toward the camera.
type Frag struct {
	Albedo   color.Color
	Emissive uint8
	Flags    Flag
	Normal   Normal
	Depth    float64
	World    geom.Vec3
	Row0     int     // reference row for the per row slopes below
	RowDepth float64 // lets flat 2D shapes stand as vertical planes
	RowZ     float64
	Tex      Sampler // multiplies Albedo per dot; triangles only
}

type Dot struct {
	Color    color.Color // final color, rewritten by lighting and post passes
	Albedo   color.Color
	Emissive uint8
	Flags    Flag
	Shadow   uint8
	Normal   Normal
	Depth    float32
	X, Y, Z  float32
	Tag      uint32
	Part     uint16
}

func (d *Dot) World() geom.Vec3 { return geom.V3(float64(d.X), float64(d.Y), float64(d.Z)) }

type Buffer struct {
	Cols, Rows int
	W, H       int
	dots       []Dot
	tag        uint32
	part       uint16
}

func New(cols, rows int) *Buffer {
	b := &Buffer{}
	b.Resize(cols, rows)
	return b
}

func (b *Buffer) Resize(cols, rows int) bool {
	cols, rows = max(1, cols), max(1, rows)
	if cols == b.Cols && rows == b.Rows && b.dots != nil {
		return false
	}
	b.Cols, b.Rows = cols, rows
	b.W, b.H = cols*SubW, rows*SubH
	b.dots = make([]Dot, b.W*b.H)
	return true
}

func (b *Buffer) Clear(bg color.Color) {
	empty := Dot{Color: bg, Albedo: bg, Flags: FlagEmpty | FlagUnlit, Depth: float32(math.Inf(-1))}
	if len(b.dots) > 0 {
		// Doubling copies fill the slice far faster than assigning one dot at a time.
		b.dots[0] = empty
		for n := 1; n < len(b.dots); n *= 2 {
			copy(b.dots[n:], b.dots[:n])
		}
	}
	b.tag, b.part = 0, 0
}

// Grow makes the buffer at least w by h dots, keeping it when it is larger,
// for scratch buffers reused at changing sizes.
func (b *Buffer) Grow(w, h int) {
	cols, rows := (w+SubW-1)/SubW, (h+SubH-1)/SubH
	if cols > b.Cols || rows > b.Rows {
		b.Resize(max(cols, b.Cols), max(rows, b.Rows))
	}
}

// ClearRect empties the dots in [x0,x1) by [y0,y1).
func (b *Buffer) ClearRect(x0, y0, x1, y1 int) {
	empty := Dot{Flags: FlagEmpty | FlagUnlit, Depth: float32(math.Inf(-1))}
	for y := max(0, y0); y < min(b.H, y1); y++ {
		row := b.dots[y*b.W : (y+1)*b.W]
		for x := max(0, x0); x < min(b.W, x1); x++ {
			row[x] = empty
		}
	}
}

// SetTag marks every following write, which is how the editor picks entities.
func (b *Buffer) SetTag(tag uint32) { b.tag, b.part = tag, 0 }

// SetPart marks following writes as part p-1 of the tagged object, so a drop
// can tell a model's parts apart; 0 clears it.
func (b *Buffer) SetPart(p uint16) { b.part = p }

func (b *Buffer) Inside(x, y int) bool { return x >= 0 && y >= 0 && x < b.W && y < b.H }

func (b *Buffer) At(x, y int) *Dot {
	if !b.Inside(x, y) {
		return nil
	}
	return &b.dots[y*b.W+x]
}

func (b *Buffer) Dots() []Dot { return b.dots }

// Plot keeps the later write on equal depth so layered sprite parts stack in call order.
func (b *Buffer) Plot(x, y int, f Frag) {
	if !b.Inside(x, y) {
		return
	}
	d := &b.dots[y*b.W+x]
	depth, z := f.Depth, f.World.Z
	if f.RowDepth != 0 || f.RowZ != 0 {
		rows := float64(y - f.Row0)
		depth += rows * f.RowDepth
		z += rows * f.RowZ
	}
	if float32(depth) < d.Depth {
		return
	}
	*d = Dot{
		Color: f.Albedo, Albedo: f.Albedo, Emissive: f.Emissive, Flags: f.Flags, Normal: f.Normal,
		Depth: float32(depth), X: float32(f.World.X), Y: float32(f.World.Y), Z: float32(z), Tag: b.tag, Part: b.part,
	}
}

// Pick returns the tag of the nearest tagged dot inside a cell, zero if none.
func (b *Buffer) Pick(col, row int) uint32 {
	tag, _ := b.PickPart(col, row)

	return tag
}

// PickPart is Pick with the part of the object that drew the dot, -1 when
// it marked none.
func (b *Buffer) PickPart(col int, row int) (uint32, int) {
	var tag uint32
	var part = -1
	best := float32(math.Inf(-1))
	for dy := 0; dy < SubH; dy++ {
		for dx := 0; dx < SubW; dx++ {
			d := b.At(col*SubW+dx, row*SubH+dy)
			if d != nil && d.Tag != 0 && d.Depth >= best {
				best, tag, part = d.Depth, d.Tag, int(d.Part)-1
			}
		}
	}
	return tag, part
}

// CellDepth is the nearest depth inside a cell, used to occlude cell sprites.
func (b *Buffer) CellDepth(col, row int) float64 {
	best := math.Inf(-1)
	for dy := 0; dy < SubH; dy++ {
		for dx := 0; dx < SubW; dx++ {
			if d := b.At(col*SubW+dx, row*SubH+dy); d != nil && float64(d.Depth) > best {
				best = float64(d.Depth)
			}
		}
	}
	return best
}

func (b *Buffer) CellAverage(col int, row int) color.Color {
	var r, g, bl, n int

	for dy := 0; dy < SubH; dy++ {
		for dx := 0; dx < SubW; dx++ {
			if d := b.At(col*SubW+dx, row*SubH+dy); d != nil {
				r, g, bl, n = r+int(d.Color.R), g+int(d.Color.G), bl+int(d.Color.B), n+1
			}
		}
	}
	if n == 0 {
		return color.Color{}
	}
	return color.RGB(uint8(r/n), uint8(g/n), uint8(bl/n))
}
