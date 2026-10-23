// Package material is the material asset, a .material.json file: a color,
// an optional texture and how the texture repeats.
package material

import (
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

const Ext = ".material.json"

type Material struct {
	Albedo   color.Color `json:"albedo"`
	Texture  string      `json:"texture,omitempty"` // texture asset reference
	Tiling   geom.Vec2   `json:"tiling"`
	Offset   geom.Vec2   `json:"offset"`
	Emissive float64     `json:"emissive,omitempty"`
	Unlit    bool        `json:"unlit,omitempty"`
}

func Default() *Material {
	return &Material{Albedo: color.RGB(255, 255, 255), Tiling: geom.V2(1, 1)}
}

func (m *Material) bind() *prop.Binder {
	return prop.Bind().Color("albedo", &m.Albedo).Texture("texture", &m.Texture).
		Float("tilingX", &m.Tiling.X, -100, 100).Float("tilingY", &m.Tiling.Y, -100, 100).
		Float("offsetX", &m.Offset.X, -10, 10).Float("offsetY", &m.Offset.Y, -10, 10).
		Float("emissive", &m.Emissive, 0, 1).Bool("unlit", &m.Unlit)
}

func (m *Material) Props() []prop.Prop        { return m.bind().Props() }
func (m *Material) Set(n string, v any) error { return m.bind().Set(n, v) }

func Decode(r io.Reader) (*Material, error) {
	m := Default()
	if err := json.NewDecoder(r).Decode(m); err != nil {
		return nil, fmt.Errorf("decode material: %w", err)
	}
	return m, nil
}

func Encode(w io.Writer, m *Material) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return fmt.Errorf("encode material: %w", err)
	}
	return nil
}
