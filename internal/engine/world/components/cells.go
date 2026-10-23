package components

import (
	"math"
	"unicode/utf8"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func cellOf(f *pipeline.Frame, p geom.Vec3) (int, int, float64) {
	x, y, d := f.View.Project(p)
	return int(math.Floor(x / raster.SubW)), int(math.Floor(y / raster.SubH)), d
}

// Sprite stamps glyph art as whole cells, hidden where nearer geometry is drawn.
type Sprite struct {
	Asset string
	lib   *asset.Library
}

func NewSprite(lib *asset.Library, name string) *Sprite { return &Sprite{Asset: name, lib: lib} }

func (*Sprite) Kind() string                { return "sprite" }
func (s *Sprite) bind() *prop.Binder        { return prop.Bind().Asset("asset", &s.Asset) }
func (s *Sprite) Props() []prop.Prop        { return s.bind().Props() }
func (s *Sprite) Set(n string, v any) error { return s.bind().Set(n, v) }
func (s *Sprite) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos, geom.V3(2, 1, 2))
}

func (s *Sprite) DrawCells(f *pipeline.Frame, t scene.Transform) {
	if s.lib == nil || s.Asset == "" {
		return
	}
	art, err := s.lib.Art(s.Asset)
	if err != nil {
		return
	}
	col, row, depth := cellOf(f, t.Pos)
	for y := 0; y < art.Height; y++ {
		for x := 0; x < art.Width; x++ {
			p, _ := art.At(x, y)
			if p.Transparent {
				continue
			}
			cx, cy := col+x-art.PivotX, row+y-art.PivotY
			if f.Buf.CellDepth(cx, cy) > depth+2 {
				continue
			}
			f.Cells.Set(cx, cy, cell.Cell{Rune: p.Rune, FG: p.FG, BG: p.BG})
		}
	}
}

// Label is text in the world or pinned to a screen corner for HUDs.
type Label struct {
	Text   string
	Color  color.Color
	Back   color.Color
	Anchor string
	Solid  bool // fill the background behind the text
}

func NewLabel(text string) *Label {
	return &Label{Text: text, Color: color.RGB(232, 226, 210), Back: color.RGB(18, 18, 18), Anchor: "world"}
}

func Anchors() []string {
	return []string{"world", "top-left", "top", "top-right", "bottom-left", "bottom", "bottom-right", "center"}
}

func (*Label) Kind() string { return "label" }
func (l *Label) bind() *prop.Binder {
	return prop.Bind().String("text", &l.Text).Color("color", &l.Color).Color("back", &l.Back).
		Enum("anchor", &l.Anchor, Anchors()...).Bool("solid", &l.Solid)
}
func (l *Label) Props() []prop.Prop                  { return l.bind().Props() }
func (l *Label) Set(n string, v any) error           { return l.bind().Set(n, v) }
func (l *Label) Draw(*draw.Context, scene.Transform) {}
func (l *Label) Bounds(t scene.Transform) geom.AABB {
	if l.Anchor != "world" {
		return geom.AABB{}
	}
	return geom.Footprint(t.Pos, geom.V3(1, 1, 1))
}

func (l *Label) DrawCells(f *pipeline.Frame, t scene.Transform) {
	w, g := utf8.RuneCountInString(l.Text), f.Cells
	ox, oy := int(t.Pos.X), int(t.Pos.Y)
	var x, y int
	switch l.Anchor {
	case "world":
		col, row, depth := cellOf(f, t.Pos)
		if f.Buf.CellDepth(col, row) > depth+2 {
			return
		}
		x, y = col-w/2, row
	case "top-left":
		x, y = ox, oy
	case "top":
		x, y = (g.W-w)/2+ox, oy
	case "top-right":
		x, y = g.W-w-ox, oy
	case "bottom-left":
		x, y = ox, g.H-1-oy
	case "bottom":
		x, y = (g.W-w)/2+ox, g.H-1-oy
	case "bottom-right":
		x, y = g.W-w-ox, g.H-1-oy
	default:
		x, y = (g.W-w)/2+ox, g.H/2+oy
	}
	for i, r := range []rune(l.Text) {
		bg := l.Back
		if !l.Solid {
			if c, ok := g.At(x+i, y); ok {
				bg = c.BG
			}
		}
		g.Set(x+i, y, cell.Cell{Rune: r, FG: l.Color, BG: bg})
	}
}
