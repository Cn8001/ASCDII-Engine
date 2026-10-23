// Package kit is the built in content: primitive nodes and the behaviours
// that make a scene playable without writing Go.
package kit

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

// Register installs every built in node and behaviour. lib may be nil when no
// project is loaded; sprites then draw nothing.
func Register(r *scene.Registry, lib *asset.Library) {
	r.Node("box", func() scene.Node { return NewBox(geom.V3(2, 2, 2), color.RGB(168, 132, 92)) })
	r.Node("cylinder", func() scene.Node { return NewCylinder(1, 3, color.RGB(140, 146, 150)) })
	r.Node("cone", func() scene.Node { return NewCone(1.5, 3, color.RGB(64, 120, 84)) })
	r.Node("blob", func() scene.Node { return NewBlob(1.2, color.RGB(78, 138, 92)) })
	r.Node("ground", func() scene.Node { return NewGround(40, 40, color.RGB(64, 92, 60)) })
	r.Node("water", func() scene.Node { return NewWater(12, 6, color.RGB(36, 92, 112)) })
	r.Node("light", func() scene.Node { return NewLight(8, color.RGB(255, 190, 110)) })
	r.Node("emitter", func() scene.Node { return NewEmitter() })
	r.Node("label", func() scene.Node { return NewLabel("label") })
	r.Node("sprite", func() scene.Node { return NewSprite(lib, "") })
	r.Behaviour("player", func() scene.Behaviour { return NewPlayer() })
	r.Behaviour("toggle", func() scene.Behaviour { return NewToggle("e", "", "visible") })
	r.Behaviour("daynight", func() scene.Behaviour { return NewDayNight() })
	r.Behaviour("quit", func() scene.Behaviour { return &Quit{Press: "q"} })
}

func scaleOf(t scene.Transform) float64 {
	if t.Scale == 0 {
		return 1
	}
	return t.Scale
}

type Box struct {
	Size     geom.Vec3
	Top      color.Color
	Side     color.Color
	Emissive float64
}

func NewBox(size geom.Vec3, c color.Color) *Box {
	return &Box{Size: size, Top: color.Scale(c, 1.12), Side: c}
}

func (*Box) Kind() string { return "box" }
func (b *Box) bind() *prop.Binder {
	return prop.Bind().Vec3("size", &b.Size).Color("top", &b.Top).Color("side", &b.Side).Float("emissive", &b.Emissive, 0, 1)
}
func (b *Box) Props() []prop.Prop        { return b.bind().Props() }
func (b *Box) Set(n string, v any) error { return b.bind().Set(n, v) }
func (b *Box) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos, b.Size.Scale(scaleOf(t)))
}
func (b *Box) Draw(c *draw.Context, t scene.Transform) {
	c.BoxFaces(b.Bounds(t), draw.M(b.Top).Glow(b.Emissive), draw.M(b.Side).Glow(b.Emissive))
	sz := b.Size.Scale(scaleOf(t))
	c.Shadow(t.Pos, sz.X*.7, sz.Y*.7, .55)
}

type Cylinder struct {
	Radius, Height float64
	Color          color.Color
	Segments       int
}

func NewCylinder(r, h float64, c color.Color) *Cylinder {
	return &Cylinder{Radius: r, Height: h, Color: c, Segments: 14}
}

func (*Cylinder) Kind() string { return "cylinder" }
func (y *Cylinder) bind() *prop.Binder {
	return prop.Bind().Float("radius", &y.Radius, .05, 100).Float("height", &y.Height, .05, 200).
		Color("color", &y.Color).Int("segments", &y.Segments, 6, 48)
}
func (y *Cylinder) Props() []prop.Prop        { return y.bind().Props() }
func (y *Cylinder) Set(n string, v any) error { return y.bind().Set(n, v) }
func (y *Cylinder) Bounds(t scene.Transform) geom.AABB {
	k := scaleOf(t)
	return geom.Footprint(t.Pos, geom.V3(y.Radius*2*k, y.Radius*2*k, y.Height*k))
}
func (y *Cylinder) Draw(c *draw.Context, t scene.Transform) {
	k := scaleOf(t)
	m := draw.M(y.Color)
	c.Cylinder(t.Pos, y.Radius*k, y.Height*k, m.Tint(1.1), m, y.Segments)
	c.Shadow(t.Pos, y.Radius*k*1.2, y.Radius*k, .5)
}

type Cone struct {
	Radius, Height float64
	Color          color.Color
	Sway           float64
}

func NewCone(r, h float64, c color.Color) *Cone { return &Cone{Radius: r, Height: h, Color: c} }

func (*Cone) Kind() string { return "cone" }
func (o *Cone) bind() *prop.Binder {
	return prop.Bind().Float("radius", &o.Radius, .05, 100).Float("height", &o.Height, .05, 200).
		Color("color", &o.Color).Float("sway", &o.Sway, 0, 2)
}
func (o *Cone) Props() []prop.Prop        { return o.bind().Props() }
func (o *Cone) Set(n string, v any) error { return o.bind().Set(n, v) }
func (o *Cone) Bounds(t scene.Transform) geom.AABB {
	k := scaleOf(t)
	return geom.Footprint(t.Pos, geom.V3(o.Radius*2*k, o.Radius*2*k, o.Height*k))
}
func (o *Cone) Draw(c *draw.Context, t scene.Transform) {
	k := scaleOf(t)
	lean := geom.V2(o.Sway*math.Sin(c.Time*1.3+t.Pos.X*.3), 0)
	c.Cone(t.Pos, o.Radius*k, o.Height*k, draw.M(o.Color), 10, lean)
	c.Shadow(t.Pos, o.Radius*k, o.Radius*k*.8, .5)
}

type Blob struct {
	Radius float64
	Color  color.Color
	Glow   float64
}

func NewBlob(r float64, c color.Color) *Blob { return &Blob{Radius: r, Color: c} }

func (*Blob) Kind() string { return "blob" }
func (b *Blob) bind() *prop.Binder {
	return prop.Bind().Float("radius", &b.Radius, .05, 50).Color("color", &b.Color).Float("glow", &b.Glow, 0, 1)
}
func (b *Blob) Props() []prop.Prop        { return b.bind().Props() }
func (b *Blob) Set(n string, v any) error { return b.bind().Set(n, v) }
func (b *Blob) Bounds(t scene.Transform) geom.AABB {
	r := b.Radius * scaleOf(t)
	return geom.Footprint(t.Pos, geom.V3(r*2, r*2, r*2))
}
func (b *Blob) Draw(c *draw.Context, t scene.Transform) {
	r := b.Radius * scaleOf(t)
	c.Blob(t.Pos.Add(geom.V3(0, 0, r)), r, draw.M(b.Color).Glow(b.Glow))
	c.Shadow(t.Pos, r, r*.8, .55)
}

type Ground struct {
	Width, Depth float64
	Color        color.Color
	Grain        float64
	Seed         int
}

func NewGround(w, d float64, c color.Color) *Ground {
	return &Ground{Width: w, Depth: d, Color: c, Grain: .08, Seed: 1}
}

func (*Ground) Kind() string { return "ground" }
func (g *Ground) bind() *prop.Binder {
	return prop.Bind().Float("width", &g.Width, 1, 2000).Float("depth", &g.Depth, 1, 2000).
		Color("color", &g.Color).Float("grain", &g.Grain, 0, 1).Int("seed", &g.Seed, 0, 9999)
}
func (g *Ground) Props() []prop.Prop        { return g.bind().Props() }
func (g *Ground) Set(n string, v any) error { return g.bind().Set(n, v) }
func (g *Ground) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos, geom.V3(g.Width, g.Depth, .01))
}

type rect struct {
	b    geom.AABB
	tile draw.Tile
}

func (r rect) Tile(x, y int) draw.Tile {
	if !r.b.Contains2D(geom.V2(float64(x)+.5, float64(y)+.5)) {
		return draw.Tile{Skip: true}
	}
	return r.tile
}

func (g *Ground) Draw(c *draw.Context, t scene.Transform) {
	m := draw.M(g.Color)
	c.Terrain(rect{g.Bounds(t), draw.Tile{Height: t.Pos.Z, Top: m, Side: m.Tint(.7)}}, draw.TerrainOptions{Grain: g.Grain, Seed: uint32(g.Seed)})
}

type Water struct {
	Width, Depth float64
	Color        color.Color
}

func NewWater(w, d float64, c color.Color) *Water { return &Water{Width: w, Depth: d, Color: c} }

func (*Water) Kind() string { return "water" }
func (w *Water) bind() *prop.Binder {
	return prop.Bind().Float("width", &w.Width, 1, 2000).Float("depth", &w.Depth, 1, 2000).Color("color", &w.Color)
}
func (w *Water) Props() []prop.Prop        { return w.bind().Props() }
func (w *Water) Set(n string, v any) error { return w.bind().Set(n, v) }
func (w *Water) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos, geom.V3(w.Width, w.Depth, .01))
}
func (w *Water) Draw(c *draw.Context, t scene.Transform) {
	m := draw.Material{Albedo: w.Color, Water: true}
	c.Terrain(rect{w.Bounds(t), draw.Tile{Height: t.Pos.Z, Top: m, Side: m.Tint(.6)}}, draw.TerrainOptions{})
}

type Light struct {
	Radius    float64
	Color     color.Color
	Intensity float64
	Flicker   float64
}

func NewLight(r float64, c color.Color) *Light {
	return &Light{Radius: r, Color: c, Intensity: 1.2, Flicker: .15}
}

func (*Light) Kind() string { return "light" }
func (l *Light) bind() *prop.Binder {
	return prop.Bind().Float("radius", &l.Radius, .5, 200).Color("color", &l.Color).
		Float("intensity", &l.Intensity, 0, 5).Float("flicker", &l.Flicker, 0, 1)
}
func (l *Light) Props() []prop.Prop        { return l.bind().Props() }
func (l *Light) Set(n string, v any) error { return l.bind().Set(n, v) }
func (l *Light) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos.Sub(geom.V3(0, 0, l.Radius)), geom.V3(l.Radius*2, l.Radius*2, l.Radius*2))
}
func (l *Light) Draw(c *draw.Context, t scene.Transform) {
	c.Light(lightAt(t.Pos, l.Radius, l.Color, l.Intensity, l.Flicker))
}
