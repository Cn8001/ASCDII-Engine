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
}

func (d *Dot) World() geom.Vec3 { return geom.V3(float64(d.X), float64(d.Y), float64(d.Z)) }

type Buffer struct {
	Cols, Rows int
	W, H       int
	dots       []Dot
	tag        uint32
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
	b.tag = 0
}

// SetTag marks every following write, which is how the editor picks entities.
func (b *Buffer) SetTag(tag uint32) { b.tag = tag }

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
		Depth: float32(depth), X: float32(f.World.X), Y: float32(f.World.Y), Z: float32(z), Tag: b.tag,
	}
}

// Pick returns the tag of the nearest tagged dot inside a cell, zero if none.
func (b *Buffer) Pick(col, row int) uint32 {
	var tag uint32
	best := float32(math.Inf(-1))
	for dy := 0; dy < SubH; dy++ {
		for dx := 0; dx < SubW; dx++ {
			d := b.At(col*SubW+dx, row*SubH+dy)
			if d != nil && d.Tag != 0 && d.Depth >= best {
				best, tag = d.Depth, d.Tag
			}
		}
	}
	return tag
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
