package pipeline

import (
	"errors"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time {
	c.t = c.t.Add(time.Millisecond)
	return c.t
}

func names(p *Pipeline) []string {
	var out []string
	for _, i := range p.Infos() {
		out = append(out, i.Name)
	}
	return out
}

func TestUnit_AddOrdersByStage(t *testing.T) {
	// Arrange
	var order []string
	mk := func(n string, s Stage) Pass { return Func(n, s, func(*Frame) { order = append(order, n) }) }
	clk := &fakeClock{}

	// Act
	p := New(clk.now, mk("hud", Overlay), mk("geo", Geometry), mk("post", Post), mk("geo2", Geometry))
	p.Render(NewFrame(2, 2))

	// Assert
	want := []string{"geo", "geo2", "post", "hud"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order %v", order)
		}
	}
	if p.Infos()[0].Micros != 1000 {
		t.Fatalf("timing %+v", p.Infos()[0])
	}
}

func TestUnit_DisabledPassIsSkipped(t *testing.T) {
	// Arrange
	ran := false
	clk := &fakeClock{}
	p := New(clk.now, Func("x", Post, func(*Frame) { ran = true }))

	// Act
	err := p.SetEnabled("x", false)
	p.Render(NewFrame(1, 1))

	// Assert
	if err != nil || ran {
		t.Fatalf("err %v ran %v", err, ran)
	}
	if !errors.Is(p.SetEnabled("missing", true), ErrPassNotFound) {
		t.Fatal("want ErrPassNotFound")
	}
}

func TestUnit_MoveStaysInsideStage(t *testing.T) {
	// Arrange
	clk := &fakeClock{}
	noop := func(*Frame) {}
	p := New(clk.now, Func("a", Geometry, noop), Func("b", Post, noop), Func("c", Post, noop), Func("d", Overlay, noop))

	// Act
	err := p.Move("c", 0)

	// Assert
	got := names(p)
	if err != nil || got[1] != "c" || got[2] != "b" || got[0] != "a" {
		t.Fatalf("got %v", got)
	}
}

func TestUnit_StandardChainLightsAndResolves(t *testing.T) {
	// Arrange
	clk := &fakeClock{}
	geo := Func("geo", Geometry, func(f *Frame) {
		f.Ctx.Box(geom.Footprint(geom.V3(0, 0, 0), geom.V3(6, 6, 2)), drawM(color.RGB(200, 180, 160)))
	})
	p := New(clk.now, append(Standard(), geo)...)
	f := NewFrame(30, 12)
	cam := view.Default()
	cam.Span = 20

	// Act
	f.Begin(cam, 0, light.DefaultEnv())
	p.Render(f)

	// Assert
	x, y, _ := f.View.Project(geom.V3(0, 0, 2))
	d := f.Buf.At(int(x), int(y))
	if d.Flags&raster.FlagEmpty != 0 || d.Color == d.Albedo {
		t.Fatalf("not lit: %+v", d)
	}
	c, _ := f.Cells.At(int(x)/2, int(y)/4)
	if c.BG == (color.Color{}) && c.FG == (color.Color{}) {
		t.Fatal("not resolved")
	}
}

func TestUnit_PixelsLeavesTransparentCellsUnderOverlay(t *testing.T) {
	// Arrange
	p := New(time.Now, Standard()...)
	p.Add(Func("hud", Overlay, func(f *Frame) { f.Cells.Text(0, 0, "a", color.RGB(255, 255, 255), color.RGB(1, 2, 3)) }))
	f := NewFrame(3, 2)
	f.Begin(view.Default(), 0, light.Env{})

	// Act
	err := UsePixels(p)
	p.Render(f)

	// Assert
	if err != nil || enabled(p, "braille") || !enabled(p, "pixels") {
		t.Fatalf("braille still on: %v %v", err, names(p))
	}
	for i, c := range f.Cells.Cells() {
		want := rune(0)
		if i == 0 {
			want = 'a'
		}
		if c.Rune != want {
			t.Fatalf("cell %d rune %q", i, c.Rune)
		}
	}
	if c, _ := f.Cells.At(1, 1); c.BG != f.Buf.CellAverage(1, 1) {
		t.Fatalf("cell colors %+v, average %+v", c, f.Buf.CellAverage(1, 1))
	}
}

func enabled(p *Pipeline, name string) bool {
	for _, i := range p.Infos() {
		if i.Name == name {
			return i.Enabled
		}
	}
	return false
}
