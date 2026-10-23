// Package prop is the property model shared by the inspector, scene files and
// animation tracks. Anything exposing Editable can be saved, edited and keyed.
package prop

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

var (
	ErrUnknownProp  = errors.New("unknown property")
	ErrInvalidValue = errors.New("invalid property value")
)

type Type string

const (
	Float     Type = "float"
	Int       Type = "int"
	Bool      Type = "bool"
	String    Type = "string"
	Color     Type = "color"
	Enum      Type = "enum"
	Vec3      Type = "vec3"
	Asset     Type = "asset"
	ScriptRef Type = "script"
	ModelRef  Type = "model"
	PrefabRef Type = "prefab"
	TexRef    Type = "texture"
	MatRef    Type = "material"
	CtrlRef   Type = "controller" // an animator controller
	AudioRef  Type = "audio"
	Data      Type = "data"      // structured data a component saves, not edited field by field
	ObjectRef Type = "object"    // a GameObject in the same scene or prefab
	CompRef   Type = "component" // Options[0], when set, limits the kind
)

// Ref reports whether values of t name assets, which are stored as GUIDs.
func (t Type) Ref() bool {
	switch t {
	case Asset, ScriptRef, ModelRef, PrefabRef, TexRef, MatRef, CtrlRef, AudioRef:
		return true
	}
	return false
}

type Prop struct {
	Name       string   `json:"name"`
	Type       Type     `json:"type"`
	Value      any      `json:"value"`
	Options    []string `json:"options,omitempty"`
	Min        float64  `json:"min,omitempty"`
	Max        float64  `json:"max,omitempty"`
	Step       float64  `json:"step,omitempty"`
	Group      string   `json:"group,omitempty"` // inspector section
	Locked     bool     `json:"locked,omitempty"`
	Overridden bool     `json:"overridden,omitempty"` // differs from its prefab
	Display    string   `json:"display,omitempty"`    // what a reference points at, for the inspector
}

type Editable interface {
	Props() []Prop
	Set(name string, v any) error
}

// Resolver lets an Editable accept short names, such as a component property
// without its component prefix.
type Resolver interface {
	Resolve(name string) string
}

func Get(e Editable, name string) (Prop, bool) {
	if r, ok := e.(Resolver); ok {
		name = r.Resolve(name)
	}
	base, axis := splitAxis(name)
	for _, p := range e.Props() {
		if p.Name == name {
			return p, true
		}
		if axis != "" && p.Name == base && p.Type == Vec3 {
			v := p.Value.(geom.Vec3)
			return Prop{Name: name, Type: Float, Value: component(v, axis)}, true
		}
	}
	return Prop{}, false
}

// Numeric conversions accept what JSON decoding and the editor bridge produce.
func ToFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %q", ErrInvalidValue, n)
		}
		return f, nil
	}
	return 0, fmt.Errorf("%w: %T is not a number", ErrInvalidValue, v)
}

func ToBool(v any) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		return strconv.ParseBool(b)
	}
	f, err := ToFloat(v)
	if err != nil {
		return false, fmt.Errorf("%w: %T is not a bool", ErrInvalidValue, v)
	}
	return f != 0, nil
}

func ToString(v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %T is not a string", ErrInvalidValue, v)
	}
	return s, nil
}

func ToColor(v any) (color.Color, error) {
	switch c := v.(type) {
	case color.Color:
		return c, nil
	case string:
		out, err := color.ParseHex(c)
		if err != nil {
			return color.Color{}, fmt.Errorf("%w: %w", ErrInvalidValue, err)
		}
		return out, nil
	}
	return color.Color{}, fmt.Errorf("%w: %T is not a color", ErrInvalidValue, v)
}

func ToVec3(v any) (geom.Vec3, error) {
	switch x := v.(type) {
	case geom.Vec3:
		return x, nil
	case []float64:
		if len(x) == 3 {
			return geom.V3(x[0], x[1], x[2]), nil
		}
	case []any:
		if len(x) == 3 {
			var out [3]float64
			for i, c := range x {
				f, err := ToFloat(c)
				if err != nil {
					return geom.Vec3{}, err
				}
				out[i] = f
			}
			return geom.V3(out[0], out[1], out[2]), nil
		}
	case map[string]any:
		var out [3]float64
		for i, k := range []string{"x", "y", "z"} {
			f, err := ToFloat(x[k])
			if err != nil {
				return geom.Vec3{}, err
			}
			out[i] = f
		}
		return geom.V3(out[0], out[1], out[2]), nil
	}
	return geom.Vec3{}, fmt.Errorf("%w: %T is not a vec3", ErrInvalidValue, v)
}

// Encode turns a value into its JSON friendly form for scene files and the bridge.
func Encode(p Prop) any {
	switch v := p.Value.(type) {
	case color.Color:
		return v.Hex()
	case geom.Vec3:
		return []float64{v.X, v.Y, v.Z}
	}
	return p.Value
}

func splitAxis(name string) (string, string) {
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return name, ""
	}
	switch name[i+1:] {
	case "x", "y", "z":
		return name[:i], name[i+1:]
	}
	return name, ""
}

func component(v geom.Vec3, axis string) float64 {
	switch axis {
	case "x":
		return v.X
	case "y":
		return v.Y
	}
	return v.Z
}
