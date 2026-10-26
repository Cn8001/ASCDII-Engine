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

// PartProp prefixes the material slot of one model part, as in
// "parts/eyes" for the part an OBJ names with "usemtl eyes".
const PartProp = "parts/"

// Mesh draws an imported model with the object's full transform.
type Mesh struct {
	Model    string
	Material string // material asset for every part, instead of the model's own
	Tint     color.Color
	Emissive float64
	TwoSided bool // for open models whose faces are wound inconsistently
	Shadow   bool
	slots    map[string]*string // material per part name; listed and saved only for parts the model has
	lib      *asset.Library
	world    []geom.Vec3 // scratch reused every frame, so drawing does not allocate
	mats     []draw.Material
}

func NewMesh(lib *asset.Library, model string) *Mesh {
	return &Mesh{Model: model, Tint: color.RGB(255, 255, 255), Shadow: true, lib: lib}
}

func (*Mesh) Kind() string { return "mesh" }

// bind lists a slot per part of the current model after the fixed
// properties; scenes load them after model, as script fields load after file.
func (m *Mesh) bind() *prop.Binder {
	b := prop.Bind().Model("model", &m.Model).Material("material", &m.Material).Color("tint", &m.Tint).Float("emissive", &m.Emissive, 0, 1).
		Bool("twoSided", &m.TwoSided).Bool("shadow", &m.Shadow)
	if msh := m.mesh(); msh != nil {
		for _, p := range msh.Parts {
			b.Material(PartProp+p.Material, m.slot(p.Material))
		}
	}
	return b
}

func (m *Mesh) slot(part string) *string {
	if m.slots == nil {
		m.slots = map[string]*string{}
	}
	s, ok := m.slots[part]
	if !ok {
		s = new(string)
		m.slots[part] = s
	}
	return s
}

// PartName is the name of the model's part i, the slot a drop on it fills.
func (m *Mesh) PartName(i int) (string, bool) {
	msh := m.mesh()
	if msh == nil || i < 0 || i >= len(msh.Parts) {
		return "", false
	}
	return msh.Parts[i].Material, true
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

// partMaterial is what a part draws with: its own slot, the material for
// every part, its texture tinted by Kd, or its flat color.
func (m *Mesh) partMaterial(msh *mesh.Mesh, p mesh.Part) draw.Material {
	for _, ref := range []string{m.slotRef(p.Material), m.Material} {
		if dm, ok := materialOf(m.lib, ref); ok {
			dm.Albedo = color.Mul(dm.Albedo, m.Tint)
			return dm
		}
	}
	if p.Texture != "" && m.lib != nil {
		dir := path.Dir(m.lib.Resolve(m.Model))
		if tex, err := m.lib.Texture(path.Join(dir, p.Texture)); err == nil {
			return draw.Material{Albedo: color.Mul(p.Kd, m.Tint), Texture: tex}.Glow(m.Emissive)
		}
	}
	return draw.M(color.Mul(p.Color, m.Tint)).Glow(m.Emissive)
}

func (m *Mesh) slotRef(part string) string {
	if s, ok := m.slots[part]; ok {
		return *s
	}
	return ""
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
	part := -1
	for _, tr := range msh.Tris {
		if tr.Part != part {
			part = tr.Part
			c.Buf.SetPart(uint16(part + 1))
		}
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
	c.Buf.SetPart(0)
	if m.Shadow {
		s := m.Bounds(t).Size()
		c.Shadow(t.Pos, s.X*.45, s.Y*.45, .5)
	}
}
