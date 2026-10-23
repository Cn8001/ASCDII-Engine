// Package mesh reads Wavefront OBJ models and their MTL materials into
// triangles in engine space (Z up, world units).
package mesh

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

type Tri struct {
	V    [3]int // indices into Mesh.Pos, counter clockwise seen from outside
	T    [3]int // indices into Mesh.UV, -1 where the file gave none
	Part int
}

// HasUV reports whether every corner of t has texture coordinates.
func (t Tri) HasUV() bool { return t.T[0] >= 0 && t.T[1] >= 0 && t.T[2] >= 0 }

// Part is a run of triangles sharing one material, an OBJ "usemtl".
type Part struct {
	Material string      `json:"material"`
	Color    color.Color `json:"color"`             // Kd, times the texture's average when textured
	Kd       color.Color `json:"kd"`                // the diffuse color alone, the tint of a sampled texture
	Texture  string      `json:"texture,omitempty"` // map_Kd, relative to the model
}

type Mesh struct {
	Pos    []geom.Vec3
	UV     []geom.Vec2
	Tris   []Tri
	Parts  []Part
	Bounds geom.AABB
}

type Options struct {
	UpZ   bool    `json:"upZ"`   // the file is already Z up, as some CAD exports are
	Scale float64 `json:"scale"` // world units per file unit, 0 means 1
}

// engine turns a file position into engine space, where Z is up and the
// default camera looks from +Y, so a model's front view faces the camera.
// Both mappings mirror one axis, which is why the parser swaps winding.
func (o Options) engine(p geom.Vec3) geom.Vec3 {
	k := o.Scale
	if k == 0 {
		k = 1
	}
	if o.UpZ {
		return geom.V3(p.X, -p.Y, p.Z).Scale(k)
	}
	return geom.V3(p.X, p.Z, p.Y).Scale(k)
}

func (m *Mesh) fit() {
	if len(m.Pos) == 0 {
		return
	}
	b := geom.AABB{Min: m.Pos[0], Max: m.Pos[0]}
	for _, p := range m.Pos[1:] {
		b.Min = geom.V3(min(b.Min.X, p.X), min(b.Min.Y, p.Y), min(b.Min.Z, p.Z))
		b.Max = geom.V3(max(b.Max.X, p.X), max(b.Max.Y, p.Y), max(b.Max.Z, p.Z))
	}
	m.Bounds = b
}

// Normal is the outward face normal of t.
func (m *Mesh) Normal(t Tri) geom.Vec3 {
	a, b, c := m.Pos[t.V[0]], m.Pos[t.V[1]], m.Pos[t.V[2]]
	return b.Sub(a).Cross(c.Sub(a)).Norm()
}
