package raster

import (
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

func frag(c color.Color, depth float64) Frag { return Frag{Albedo: c, Depth: depth} }

func TestUnit_PlotKeepsNearerDot(t *testing.T) {
	// Arrange
	b := New(2, 2)
	b.Clear(color.RGB(1, 2, 3))
	red, green := color.RGB(100, 0, 0), color.RGB(0, 100, 0)

	// Act
	b.Plot(1, 2, frag(red, 6))
	b.Plot(1, 2, frag(green, 5))
	b.Plot(2, 2, frag(green, 5))

	// Assert
	if got := b.At(1, 2).Color; got != red {
		t.Fatalf("occlusion failed: %+v", got)
	}
	if got := b.At(2, 2).Color; got != green {
		t.Fatalf("neighbour altered: %+v", got)
	}
	if b.At(-1, 0) != nil {
		t.Fatal("expected clipping")
	}
}

func TestUnit_EqualDepthLaterWins(t *testing.T) {
	// Arrange
	b := New(1, 1)
	b.Clear(color.Color{})

	// Act
	b.Plot(0, 0, frag(color.RGB(1, 0, 0), 2))
	b.Plot(0, 0, frag(color.RGB(0, 1, 0), 2))

	// Assert
	if b.At(0, 0).Color != color.RGB(0, 1, 0) {
		t.Fatal("later write on equal depth must win")
	}
}

func TestUnit_TriangleInterpolatesDepth(t *testing.T) {
	// Arrange
	b := New(8, 4)
	b.Clear(color.Color{})
	v := func(x, y, d float64) Vertex { return Vertex{X: x, Y: y, Depth: d, WX: x} }

	// Act
	b.Triangle(v(0, 0, 0), v(16, 0, 16), v(0, 16, 0), frag(color.RGB(9, 9, 9), 0))

	// Assert
	near, far := b.At(10, 1), b.At(1, 1)
	if near.Flags&FlagEmpty != 0 || far.Flags&FlagEmpty != 0 {
		t.Fatal("triangle did not cover samples")
	}
	if !(near.Depth > far.Depth) || near.X < 10 {
		t.Fatalf("depth not interpolated: near %v far %v", near.Depth, far.Depth)
	}
}

func TestUnit_PickReturnsNearestTag(t *testing.T) {
	// Arrange
	b := New(1, 1)
	b.Clear(color.Color{})
	b.SetTag(4)
	b.Plot(0, 0, frag(color.RGB(1, 1, 1), 1))
	b.SetTag(9)
	b.Plot(1, 3, frag(color.RGB(1, 1, 1), 3))

	// Act
	tag := b.Pick(0, 0)

	// Assert
	if tag != 9 {
		t.Fatalf("got %d", tag)
	}
}

func TestUnit_ShapesStayInsideBuffer(t *testing.T) {
	// Arrange
	b := New(20, 10)
	b.Clear(color.RGB(1, 1, 1))
	f := frag(color.RGB(200, 100, 80), 1)

	// Act
	b.Rect(1, 1, 50, 50, f)
	b.Line(-5, -5, 60, 70, 3, f)
	b.Ellipse(10, 15, 8, 6, f)
	b.Polygon([]Point{{6, 4}, {16, 8}, {5, 18}}, f)
	b.QuadBezier(Point{1, 20}, Point{15, 1}, Point{38, 39}, 12, 1, f)

	// Assert
	if len(b.Dots()) != 40*40 {
		t.Fatalf("dots %d", len(b.Dots()))
	}
}

func TestUnit_NormalRoundTrips(t *testing.T) {
	// Arrange
	n := geom.V3(0, -1, 1)

	// Act
	back := PackNormal(n).Vec()

	// Assert
	want := n.Norm()
	if d := back.Sub(want).Len(); d > .02 {
		t.Fatalf("error %f", d)
	}
}

func TestUnit_RowSlopeTiltsDepth(t *testing.T) {
	// Arrange
	b := New(1, 2)
	b.Clear(color.Color{})
	f := Frag{Albedo: color.RGB(1, 1, 1), Depth: 10, Row0: 4, RowDepth: -1, RowZ: -.5}

	// Act
	b.Rect(0, 0, 2, 8, f)

	// Assert
	top, bottom := b.At(0, 0), b.At(0, 7)
	if top.Depth != 14 || bottom.Depth != 7 || top.Z != 2 {
		t.Fatalf("top %v bottom %v z %v", top.Depth, bottom.Depth, top.Z)
	}
}
