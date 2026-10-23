package components

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Mesh draws an imported model with the object's full transform.
type Mesh struct {
	Model    string
	Tint     color.Color
	Emissive float64
	TwoSided bool // for open models whose faces are wound inconsistently
	Shadow   bool
	lib      *asset.Library
	world    []geom.Vec3 // scratch reused every frame, so drawing does not allocate
	mats     []draw.Material
}

func NewMesh(lib *asset.Library, model string) *Mesh {
	return &Mesh{Model: model, Tint: color.RGB(255, 255, 255), Shadow: true, lib: lib}
}

func (*Mesh) Kind() string { return "mesh" }
func (m *Mesh) bind() *prop.Binder {
	return prop.Bind().Model("model", &m.Model).Color("tint", &m.Tint).Float("emissive", &m.Emissive, 0, 1).
		Bool("twoSided", &m.TwoSided).Bool("shadow", &m.Shadow)
}
func (m *Mesh) Props() []prop.Prop        { return m.bind().Props() }
func (m *Mesh) Set(n string, v any) error { return m.bind().Set(n, v) }

func (m *Mesh) mesh() *mesh.Mesh {
	if m.lib == nil || m.Model == "" {
		return nil
	}
	msh, err := m.lib.Model(m.Model)
	if err != nil {
		return nil
	}
	return msh
}

func (m *Mesh) Bounds(t scene.Transform) geom.AABB {
	msh := m.mesh()
	if msh == nil {
		return geom.AABB{}
	}
	var b geom.AABB
	for i, p := range msh.Bounds.Corners() {
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

func (m *Mesh) Draw(c *draw.Context, t scene.Transform) {
	msh := m.mesh()
	if msh == nil {
		return
	}
	if cap(m.world) < len(msh.Pos) {
		m.world = make([]geom.Vec3, len(msh.Pos))
	}
	w := m.world[:len(msh.Pos)]
	for i, p := range msh.Pos {
		w[i] = t.Apply(p)
	}
	mats := m.mats[:0]
	for _, p := range msh.Parts {
		mats = append(mats, draw.M(color.Mul(p.Color, m.Tint)).Glow(m.Emissive))
	}
	m.mats = mats
	k := t.Size(geom.V3(1, 1, 1))
	mirror := t.Flip != (k.X*k.Y*k.Z < 0) // a mirrored transform turns every face inside out
	for _, tr := range msh.Tris {
		a, b, d := w[tr.V[0]], w[tr.V[1]], w[tr.V[2]]
		if mirror {
			b, d = d, b
		}
		if m.TwoSided {
			c.Tri(a, b, d, mats[tr.Part])
		} else {
			c.Facet(a, b, d, mats[tr.Part])
		}
	}
	if m.Shadow {
		s := m.Bounds(t).Size()
		c.Shadow(t.Pos, s.X*.45, s.Y*.45, .5)
	}
}
