package prop

import (
	"fmt"
	"slices"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

type binding struct {
	prop Prop
	get  func() any
	set  func(any) error
}

// Binder lets a type expose its fields by pointer instead of writing a switch
// per property. Build it on demand; it holds no state of its own.
type Binder struct{ items []binding }

func Bind() *Binder { return &Binder{} }

func (b *Binder) add(p Prop, get func() any, set func(any) error) *Binder {
	b.items = append(b.items, binding{prop: p, get: get, set: set})
	return b
}

func (b *Binder) Float(name string, p *float64, lo, hi float64) *Binder {
	return b.add(Prop{Name: name, Type: Float, Min: lo, Max: hi}, func() any { return *p }, func(v any) error {
		f, err := ToFloat(v)
		if err != nil {
			return err
		}
		if lo != hi {
			f = geom.Clamp(f, lo, hi)
		}
		*p = f
		return nil
	})
}

func (b *Binder) Int(name string, p *int, lo, hi int) *Binder {
	return b.add(Prop{Name: name, Type: Int, Min: float64(lo), Max: float64(hi), Step: 1}, func() any { return *p }, func(v any) error {
		f, err := ToFloat(v)
		if err != nil {
			return err
		}
		n := int(f)
		if lo != hi {
			n = max(lo, min(hi, n))
		}
		*p = n
		return nil
	})
}

func (b *Binder) Bool(name string, p *bool) *Binder {
	return b.add(Prop{Name: name, Type: Bool}, func() any { return *p }, func(v any) error {
		x, err := ToBool(v)
		if err != nil {
			return err
		}
		*p = x
		return nil
	})
}

func (b *Binder) String(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: String}, p)
}

func (b *Binder) Asset(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: Asset}, p)
}

func (b *Binder) text(pr Prop, p *string) *Binder {
	return b.add(pr, func() any { return *p }, func(v any) error {
		s, err := ToString(v)
		if err != nil {
			return err
		}
		*p = s
		return nil
	})
}

func (b *Binder) Enum(name string, p *string, options ...string) *Binder {
	return b.add(Prop{Name: name, Type: Enum, Options: options}, func() any { return *p }, func(v any) error {
		s, err := ToString(v)
		if err != nil {
			return err
		}
		if !slices.Contains(options, s) {
			return fmt.Errorf("%w: %q not in %v", ErrInvalidValue, s, options)
		}
		*p = s
		return nil
	})
}

func (b *Binder) Color(name string, p *color.Color) *Binder {
	return b.add(Prop{Name: name, Type: Color}, func() any { return *p }, func(v any) error {
		c, err := ToColor(v)
		if err != nil {
			return err
		}
		*p = c
		return nil
	})
}

func (b *Binder) Vec3(name string, p *geom.Vec3) *Binder {
	return b.add(Prop{Name: name, Type: Vec3}, func() any { return *p }, func(v any) error {
		x, err := ToVec3(v)
		if err != nil {
			return err
		}
		*p = x
		return nil
	})
}

func (b *Binder) Props() []Prop {
	out := make([]Prop, len(b.items))
	for i, it := range b.items {
		out[i] = it.prop
		out[i].Value = it.get()
	}
	return out
}

// Set also accepts "pos.x" style names so animation can key single axes.
func (b *Binder) Set(name string, v any) error {
	for _, it := range b.items {
		if it.prop.Name == name {
			if err := it.set(v); err != nil {
				return fmt.Errorf("set %s: %w", name, err)
			}
			return nil
		}
	}
	base, axis := splitAxis(name)
	if axis != "" {
		for _, it := range b.items {
			if it.prop.Name != base || it.prop.Type != Vec3 {
				continue
			}
			f, err := ToFloat(v)
			if err != nil {
				return fmt.Errorf("set %s: %w", name, err)
			}
			cur := it.get().(geom.Vec3)
			switch axis {
			case "x":
				cur.X = f
			case "y":
				cur.Y = f
			default:
				cur.Z = f
			}
			return it.set(cur)
		}
	}
	return fmt.Errorf("%w: %s", ErrUnknownProp, name)
}

func (b *Binder) Has(name string) bool {
	base, _ := splitAxis(name)
	for _, it := range b.items {
		if it.prop.Name == name || it.prop.Name == base {
			return true
		}
	}
	return false
}

// Model is an asset reference restricted to imported models in the inspector.
func (b *Binder) Model(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: ModelRef}, p)
}

func (b *Binder) Texture(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: TexRef}, p)
}

func (b *Binder) Material(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: MatRef}, p)
}

func (b *Binder) Prefab(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: PrefabRef}, p)
}

// Object holds a scene reference; see scene.RefOf for the format.
func (b *Binder) Object(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: ObjectRef}, p)
}

// Component holds a reference to a component of kind, or of any kind for "".
func (b *Binder) Component(name, kind string, p *string) *Binder {
	pr := Prop{Name: name, Type: CompRef}
	if kind != "" {
		pr.Options = []string{kind}
	}
	return b.text(pr, p)
}

// Script is an asset reference restricted to Lua files in the inspector.
func (b *Binder) Script(name string, p *string) *Binder {
	return b.text(Prop{Name: name, Type: ScriptRef}, p)
}
