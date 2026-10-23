// Package components is the built in content: primitives, lights, effects
// and the logic that makes a scene playable without writing Go.
package components

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Register installs every built in node and behaviour. lib may be nil when no
// project is loaded; sprites then draw nothing.
func Register(r *scene.Registry, lib *asset.Library) {
	obj := func(cat, label string) scene.Meta { return scene.Meta{Category: cat, Label: label, Object: true} }
	r.Register("box", func() scene.Component {
		b := NewBox(geom.V3(2, 2, 2), color.RGB(168, 132, 92))
		b.lib = lib
		return b
	}, obj("3d", "Cube"))
	r.Register("blob", func() scene.Component { return NewBlob(1.2, color.RGB(78, 138, 92)) }, obj("3d", "Sphere"))
	r.Register("cylinder", func() scene.Component { return NewCylinder(1, 3, color.RGB(140, 146, 150)) }, obj("3d", "Cylinder"))
	r.Register("cone", func() scene.Component { return NewCone(1.5, 3, color.RGB(64, 120, 84)) }, obj("3d", "Cone"))
	r.Register("plane", func() scene.Component {
		p := NewPlane(6, 6, color.RGB(150, 146, 138))
		p.lib = lib
		return p
	}, obj("3d", "Plane"))
	r.Register("quad", func() scene.Component {
		q := NewQuad(2, 2, color.RGB(220, 216, 208))
		q.lib = lib
		return q
	}, obj("3d", "Quad"))
	r.Register("mesh", func() scene.Component { return NewMesh(lib, "") }, scene.Meta{Category: "3d", Label: "Mesh"})
	r.Register("ground", func() scene.Component { return NewGround(40, 40, color.RGB(64, 92, 60)) }, obj("3d", "Ground"))
	r.Register("water", func() scene.Component { return NewWater(12, 6, color.RGB(36, 92, 112)) }, obj("3d", "Water"))
	r.Register("light", func() scene.Component { return NewLight(8, color.RGB(255, 190, 110)) }, obj("light", "Point Light"))
	r.Register("emitter", func() scene.Component { return NewEmitter() }, obj("effects", "Particles"))
	r.Register("label", func() scene.Component { return NewLabel("label") }, obj("ui", "Text"))
	r.Register("sprite", func() scene.Component { return NewSprite(lib, "") }, obj("ui", "Sprite"))
	r.Register("audio_source", func() scene.Component { return NewAudioSource() }, scene.Meta{Category: "audio", Label: "Audio Source"})
	r.Register("box_collider", func() scene.Component { return NewBoxCollider() }, scene.Meta{Category: "physics", Label: "Box Collider"})
	r.Register("circle_collider", func() scene.Component { return NewCircleCollider() }, scene.Meta{Category: "physics", Label: "Circle Collider"})
	r.Register("mesh_collider", func() scene.Component { return &MeshCollider{} }, scene.Meta{Category: "physics", Label: "Mesh Collider"})
	r.Register("rigidbody", func() scene.Component { return NewRigidbody() }, scene.Meta{Category: "physics", Label: "Rigidbody"})
	r.Register("paint", func() scene.Component { return &Paint{} }, scene.Meta{Category: "3d", Label: "Paint"})
	r.Register("animator", func() scene.Component { return NewAnimator(lib) }, scene.Meta{Category: "logic", Label: "Animator"})
	r.Register("player", func() scene.Component { return NewPlayer() }, scene.Meta{Category: "logic", Label: "Player"})
	r.Register("toggle", func() scene.Component { return NewToggle("e", "", "visible") }, scene.Meta{Category: "logic", Label: "Toggle"})
	r.Register("daynight", func() scene.Component { return NewDayNight() }, scene.Meta{Category: "logic", Label: "Day Night"})
	r.Register("quit", func() scene.Component { return &Quit{Press: "q"} }, scene.Meta{Category: "logic", Label: "Quit Key"})
}

type Box struct {
	Size     geom.Vec3
	Top      color.Color
	Side     color.Color
	Emissive float64
	Material string // material asset; replaces the colors when set
	lib      *asset.Library
}

func NewBox(size geom.Vec3, c color.Color) *Box {
	return &Box{Size: size, Top: color.Scale(c, 1.12), Side: c}
}

func (*Box) Kind() string { return "box" }
func (b *Box) bind() *prop.Binder {
	return prop.Bind().Vec3("size", &b.Size).Color("top", &b.Top).Color("side", &b.Side).Float("emissive", &b.Emissive, 0, 1).
		Material("material", &b.Material)
}
func (b *Box) Props() []prop.Prop        { return b.bind().Props() }
func (b *Box) Set(n string, v any) error { return b.bind().Set(n, v) }
func (b *Box) local() geom.AABB {
	h := b.Size.Scale(.5)
	return geom.AABB{Min: geom.V3(-h.X, -h.Y, 0), Max: geom.V3(h.X, h.Y, b.Size.Z)}
}
func (b *Box) Bounds(t scene.Transform) geom.AABB { return boundsOf(t, b.local()) }

func (b *Box) Draw(c *draw.Context, t scene.Transform) {
	top, side := draw.M(b.Top).Glow(b.Emissive), draw.M(b.Side).Glow(b.Emissive)
	if m, ok := materialOf(b.lib, b.Material); ok {
		top, side = m, m
	}
	l := b.local()
	x0, y0, z0, x1, y1, z1 := l.Min.X, l.Min.Y, l.Min.Z, l.Max.X, l.Max.Y, l.Max.Z
	v := geom.V3
	q := func(m draw.Material, out geom.Vec3, p0, p1, p2, p3 geom.Vec3) {
		quad(c, t, m, out, [4]geom.Vec3{p0, p1, p2, p3}, unitUV())
	}
	q(top, v(0, 0, 1), v(x0, y1, z1), v(x1, y1, z1), v(x1, y0, z1), v(x0, y0, z1))
	q(side, v(0, 1, 0), v(x0, y1, z0), v(x1, y1, z0), v(x1, y1, z1), v(x0, y1, z1))
	q(side, v(0, -1, 0), v(x1, y0, z0), v(x0, y0, z0), v(x0, y0, z1), v(x1, y0, z1))
	q(side, v(1, 0, 0), v(x1, y1, z0), v(x1, y0, z0), v(x1, y0, z1), v(x1, y1, z1))
	q(side, v(-1, 0, 0), v(x0, y0, z0), v(x0, y1, z0), v(x0, y1, z1), v(x0, y0, z1))
	q(side, v(0, 0, -1), v(x0, y0, z0), v(x1, y0, z0), v(x1, y1, z0), v(x0, y1, z0))
	sz := t.Size(b.Size)
	c.Shadow(t.Pos, sz.X*.7, sz.Y*.7, .55)
}

// Plane is a flat textured rectangle lying on its object's ground.
type Plane struct {
	Width, Depth float64
	Color        color.Color
	Material     string
	lib          *asset.Library
}

func NewPlane(w, d float64, c color.Color) *Plane { return &Plane{Width: w, Depth: d, Color: c} }

func (*Plane) Kind() string { return "plane" }
func (p *Plane) bind() *prop.Binder {
	return prop.Bind().Float("width", &p.Width, .1, 2000).Float("depth", &p.Depth, .1, 2000).Color("color", &p.Color).
		Material("material", &p.Material)
}
func (p *Plane) Props() []prop.Prop        { return p.bind().Props() }
func (p *Plane) Set(n string, v any) error { return p.bind().Set(n, v) }
func (p *Plane) local() geom.AABB {
	return geom.AABB{Min: geom.V3(-p.Width/2, -p.Depth/2, 0), Max: geom.V3(p.Width/2, p.Depth/2, .01)}
}
func (p *Plane) Bounds(t scene.Transform) geom.AABB { return boundsOf(t, p.local()) }
func (p *Plane) Draw(c *draw.Context, t scene.Transform) {
	m := draw.M(p.Color)
	if mm, ok := materialOf(p.lib, p.Material); ok {
		m = mm
	}
	w, d := p.Width/2, p.Depth/2
	v := geom.V3
	quad(c, t, m, v(0, 0, 1), [4]geom.Vec3{v(-w, d, 0), v(w, d, 0), v(w, -d, 0), v(-w, -d, 0)}, unitUV())
}

// Quad stands upright facing the default camera, for signs, pictures and
// cut out sprites with alpha.
type Quad struct {
	Width, Height float64
	Color         color.Color
	Material      string
	TwoSided      bool
	lib           *asset.Library
}

func NewQuad(w, h float64, c color.Color) *Quad {
	return &Quad{Width: w, Height: h, Color: c, TwoSided: true}
}

func (*Quad) Kind() string { return "quad" }
func (q *Quad) bind() *prop.Binder {
	return prop.Bind().Float("width", &q.Width, .1, 500).Float("height", &q.Height, .1, 500).Color("color", &q.Color).
		Material("material", &q.Material).Bool("twoSided", &q.TwoSided)
}
func (q *Quad) Props() []prop.Prop        { return q.bind().Props() }
func (q *Quad) Set(n string, v any) error { return q.bind().Set(n, v) }
func (q *Quad) local() geom.AABB {
	return geom.AABB{Min: geom.V3(-q.Width/2, -.01, 0), Max: geom.V3(q.Width/2, .01, q.Height)}
}
func (q *Quad) Bounds(t scene.Transform) geom.AABB { return boundsOf(t, q.local()) }
func (q *Quad) Draw(c *draw.Context, t scene.Transform) {
	m := draw.M(q.Color)
	if mm, ok := materialOf(q.lib, q.Material); ok {
		m = mm
	}
	w, h := q.Width/2, q.Height
	v := geom.V3
	uv := unitUV()
	quad(c, t, m, v(0, 1, 0), [4]geom.Vec3{v(-w, 0, 0), v(w, 0, 0), v(w, 0, h), v(-w, 0, h)}, uv)
	if q.TwoSided {
		quad(c, t, m, v(0, -1, 0), [4]geom.Vec3{v(w, 0, 0), v(-w, 0, 0), v(-w, 0, h), v(w, 0, h)}, [4]geom.Vec2{uv[1], uv[0], uv[3], uv[2]})
	}
}

func unitUV() [4]geom.Vec2 {
	return [4]geom.Vec2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}
}

// quad draws a local space rectangle facing out through the object's
// transform; corners may come in either order, the winding is fixed here.
func quad(c *draw.Context, t scene.Transform, m draw.Material, out geom.Vec3, p [4]geom.Vec3, uv [4]geom.Vec2) {
	if p[1].Sub(p[0]).Cross(p[2].Sub(p[0])).Dot(out) < 0 {
		p[1], p[3], uv[1], uv[3] = p[3], p[1], uv[3], uv[1]
	}
	var w [4]geom.Vec3
	for i := range p {
		w[i] = t.Apply(p[i])
	}
	if mirrored(t) {
		w[1], w[3], uv[1], uv[3] = w[3], w[1], uv[3], uv[1]
	}
	c.FacetUV(w[0], w[1], w[2], uv[0], uv[1], uv[2], m)
	c.FacetUV(w[0], w[2], w[3], uv[0], uv[2], uv[3], m)
}

// mirrored reports a transform that turns faces inside out.
func mirrored(t scene.Transform) bool {
	k := t.Size(geom.V3(1, 1, 1))
	return t.Flip != (k.X*k.Y*k.Z < 0)
}

// boundsOf is the world box around a local box after rotation and scale.
func boundsOf(t scene.Transform, local geom.AABB) geom.AABB {
	var b geom.AABB
	for i, p := range local.Corners() {
		w := t.Apply(p)
		if i == 0 {
			b = geom.AABB{Min: w, Max: w}
			continue
		}
		b.Min = geom.V3(math.Min(b.Min.X, w.X), math.Min(b.Min.Y, w.Y), math.Min(b.Min.Z, w.Z))
		b.Max = geom.V3(math.Max(b.Max.X, w.X), math.Max(b.Max.Y, w.Y), math.Max(b.Max.Z, w.Z))
	}
	return b
}

func materialOf(lib *asset.Library, ref string) (draw.Material, bool) {
	if lib == nil || ref == "" {
		return draw.Material{}, false
	}
	return lib.DrawMaterial(ref)
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
	k := t.Uniform()
	return geom.Footprint(t.Pos, geom.V3(y.Radius*2*k, y.Radius*2*k, y.Height*k))
}
func (y *Cylinder) Draw(c *draw.Context, t scene.Transform) {
	k := t.Uniform()
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
	k := t.Uniform()
	return geom.Footprint(t.Pos, geom.V3(o.Radius*2*k, o.Radius*2*k, o.Height*k))
}
func (o *Cone) Draw(c *draw.Context, t scene.Transform) {
	k := t.Uniform()
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
	r := b.Radius * t.Uniform()
	return geom.Footprint(t.Pos, geom.V3(r*2, r*2, r*2))
}
func (b *Blob) Draw(c *draw.Context, t scene.Transform) {
	r := b.Radius * t.Uniform()
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
	Type      string // point, spot or directional
	Radius    float64
	Color     color.Color
	Intensity float64
	Flicker   float64
	Angle     float64 // a spot's full cone in degrees
}

func NewLight(r float64, c color.Color) *Light {
	return &Light{Type: "point", Radius: r, Color: c, Intensity: 1.2, Flicker: .15, Angle: 50}
}

func (*Light) Kind() string { return "light" }
func (l *Light) bind() *prop.Binder {
	return prop.Bind().Enum("type", &l.Type, "point", "spot", "directional").Float("radius", &l.Radius, .5, 200).Color("color", &l.Color).
		Float("intensity", &l.Intensity, 0, 5).Float("flicker", &l.Flicker, 0, 1).Float("angle", &l.Angle, 1, 170)
}
func (l *Light) Props() []prop.Prop        { return l.bind().Props() }
func (l *Light) Set(n string, v any) error { return l.bind().Set(n, v) }

// Bounds is empty for a directional light, which lights everything and so
// must never be culled.
func (l *Light) Bounds(t scene.Transform) geom.AABB {
	if l.Type == "directional" {
		return geom.AABB{}
	}
	return geom.Footprint(t.Pos.Sub(geom.V3(0, 0, l.Radius)), geom.V3(l.Radius*2, l.Radius*2, l.Radius*2))
}

// Draw aims spot and directional lights down, turned by the object's rotation.
func (l *Light) Draw(c *draw.Context, t scene.Transform) {
	aim := geom.V3(0, 0, -1).RotateEuler(t.Rot)
	switch l.Type {
	case "directional":
		c.Sun = &light.Sun{Dir: aim.Scale(-1), Color: l.Color, Intensity: l.Intensity}
	case "spot":
		p := lightAt(t.Pos, l.Radius, l.Color, l.Intensity, l.Flicker)
		p.Dir, p.Cone = aim, math.Cos(l.Angle/2*math.Pi/180)
		c.Light(p)
	default:
		c.Light(lightAt(t.Pos, l.Radius, l.Color, l.Intensity, l.Flicker))
	}
}
