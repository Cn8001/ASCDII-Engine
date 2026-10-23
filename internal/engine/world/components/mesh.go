package components

import (
	"path"

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
	Material string // material asset for every part, instead of the model's own
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
	return prop.Bind().Model("model", &m.Model).Material("material", &m.Material).Color("tint", &m.Tint).Float("emissive", &m.Emissive, 0, 1).
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
	return boundsOf(t, msh.Bounds)
}

// partMaterial is what a part draws with: the override material, its own
// texture tinted by Kd, or its flat color.
func (m *Mesh) partMaterial(msh *mesh.Mesh, p mesh.Part) draw.Material {
	if dm, ok := materialOf(m.lib, m.Material); ok {
		dm.Albedo = color.Mul(dm.Albedo, m.Tint)
		return dm
	}
	if p.Texture != "" && m.lib != nil {
		dir := path.Dir(m.lib.Resolve(m.Model))
		if tex, err := m.lib.Texture(path.Join(dir, p.Texture)); err == nil {
			return draw.Material{Albedo: color.Mul(p.Kd, m.Tint), Texture: tex}.Glow(m.Emissive)
		}
	}
	return draw.M(color.Mul(p.Color, m.Tint)).Glow(m.Emissive)
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
		mats = append(mats, m.partMaterial(msh, p))
	}
	m.mats = mats
	mirror := mirrored(t)
	for _, tr := range msh.Tris {
		a, b, d := w[tr.V[0]], w[tr.V[1]], w[tr.V[2]]
		mat := mats[tr.Part]
		switch {
		case mat.Texture != nil && tr.HasUV():
			ua, ub, ud := msh.UV[tr.T[0]], msh.UV[tr.T[1]], msh.UV[tr.T[2]]
			if mirror {
				b, d, ub, ud = d, b, ud, ub
			}
			c.FacetUV(a, b, d, ua, ub, ud, mat)
			if m.TwoSided {
				c.FacetUV(a, d, b, ua, ud, ub, mat)
			}
		case m.TwoSided:
			c.Tri(a, b, d, mat)
		default:
			if mirror {
				b, d = d, b
			}
			c.Facet(a, b, d, mat)
		}
	}
	if m.Shadow {
		s := m.Bounds(t).Size()
		c.Shadow(t.Pos, s.X*.45, s.Y*.45, .5)
	}
}
