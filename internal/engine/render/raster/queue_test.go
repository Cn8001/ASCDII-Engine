package raster

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/color"
)

// scribble draws overlapping triangles, shaded triangles and plots, many on
// equal depths, under changing tags, so order and depth ties both matter.
func scribble(b *Buffer) {
	r := rand.New(rand.NewPCG(1, 2))
	v := func() Vertex {
		return Vertex{X: r.Float64()*float64(b.W+20) - 10, Y: r.Float64()*float64(b.H+20) - 10, Depth: float64(r.IntN(3)), R: r.Float64(), G: r.Float64(), B: r.Float64()}
	}
	for i := range 400 {
		b.SetTag(uint32(i % 7))
		b.SetPart(uint16(i % 3))
		f := frag(color.RGB(uint8(i), uint8(i*3), uint8(i*7)), float64(r.IntN(3)))
		switch i % 3 {
		case 0:
			b.Triangle(v(), v(), v(), f)
		case 1:
			b.TriangleShaded(v(), v(), v(), f)
		default:
			b.Rect(r.IntN(b.W), r.IntN(b.H), r.IntN(12), r.IntN(12), f)
		}
	}
}

func TestUnit_DeferredDrawMatchesImmediate(t *testing.T) {
	// Arrange
	want, got := New(60, 40), New(60, 40)
	want.Clear(color.Color{})
	got.Clear(color.Color{})
	scribble(want)

	// Act
	got.Defer()
	scribble(got)
	got.Flush()

	// Assert
	if !slices.Equal(want.Dots(), got.Dots()) {
		t.Fatal("deferred rasterization differs from immediate")
	}
}

func TestUnit_ReadWhileDeferredSeesQueuedWrites(t *testing.T) {
	// Arrange
	b := New(4, 4)
	b.Clear(color.Color{})
	b.Defer()
	b.Plot(3, 5, frag(color.RGB(9, 0, 0), 1))

	// Act
	d := b.At(3, 5)

	// Assert
	if d.Color != color.RGB(9, 0, 0) {
		t.Fatalf("queued write not visible to a read: %+v", d.Color)
	}
}
