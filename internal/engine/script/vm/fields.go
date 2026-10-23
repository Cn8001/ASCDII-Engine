package vm

import (
	"context"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

const markField = "__field"

// Field is a script variable the inspector edits, declared in the script:
//
//	fields = { speed = 5, door = object(), lamp = component("light") }
type Field struct {
	Name    string
	Type    prop.Type
	Default any
	Options []string
	Min     float64
	Max     float64
}

// installFields defines the constructors a fields table uses.
func installFields(L *lua.LState) {
	mark := func(kind string, set func(L *lua.LState, t *lua.LTable)) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			t := L.NewTable()
			t.RawSetString(markField, lua.LString(kind))
			set(L, t)
			L.Push(t)
			return 1
		})
	}
	none := func(*lua.LState, *lua.LTable) {}
	L.SetGlobal("object", mark("object", none))
	L.SetGlobal("prefab", mark("prefab", none))
	L.SetGlobal("component", mark("component", func(L *lua.LState, t *lua.LTable) {
		t.RawSetString("kind", lua.LString(L.OptString(1, "")))
	}))
	L.SetGlobal("color", mark("color", func(L *lua.LState, t *lua.LTable) {
		t.RawSetString("value", lua.LString(L.OptString(1, "#ffffff")))
	}))
	L.SetGlobal("enum", mark("enum", func(L *lua.LState, t *lua.LTable) {
		opts := L.NewTable()
		for i := 1; i <= L.GetTop(); i++ {
			opts.Append(lua.LString(L.CheckString(i)))
		}
		t.RawSetString("options", opts)
	}))
	L.SetGlobal("range", mark("range", func(L *lua.LState, t *lua.LTable) {
		t.RawSetString("value", L.CheckNumber(1))
		t.RawSetString("min", L.CheckNumber(2))
		t.RawSetString("max", L.CheckNumber(3))
	}))
	L.SetGlobal("int", mark("int", func(L *lua.LState, t *lua.LTable) {
		t.RawSetString("value", L.OptNumber(1, 0))
	}))
}

// parseFields reads a fields table, ordered as the names appear in text.
func parseFields(t *lua.LTable, text string) []Field {
	var out []Field
	t.ForEach(func(k, v lua.LValue) {
		name, ok := k.(lua.LString)
		if !ok {
			return
		}
		if f, ok := fieldOf(string(name), v); ok {
			out = append(out, f)
		}
	})
	at := strings.Index(text, "fields")
	pos := func(name string) int {
		re := regexp.MustCompile(`(^|[^\w.])` + regexp.QuoteMeta(name) + `\s*=`)
		if loc := re.FindStringIndex(text[max(at, 0):]); loc != nil {
			return loc[0]
		}
		return math.MaxInt
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := pos(out[i].Name), pos(out[j].Name)
		if pi != pj {
			return pi < pj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func fieldOf(name string, v lua.LValue) (Field, bool) {
	f := Field{Name: name}
	switch x := v.(type) {
	case lua.LNumber:
		f.Type, f.Default = prop.Float, float64(x)
	case lua.LBool:
		f.Type, f.Default = prop.Bool, bool(x)
	case lua.LString:
		f.Type, f.Default = prop.String, string(x)
	case *lua.LTable:
		kind, marked := x.RawGetString(markField).(lua.LString)
		if !marked {
			if _, ok := x.RawGetString("x").(lua.LNumber); ok {
				f.Type, f.Default = prop.Vec3, toVec(x)
				return f, true
			}
			return f, false
		}
		switch kind {
		case "object":
			f.Type, f.Default = prop.ObjectRef, ""
		case "prefab":
			f.Type, f.Default = prop.PrefabRef, ""
		case "component":
			f.Type, f.Default = prop.CompRef, ""
			if k := lua.LVAsString(x.RawGetString("kind")); k != "" {
				f.Options = []string{k}
			}
		case "color":
			f.Type, f.Default = prop.Color, color.Hex(lua.LVAsString(x.RawGetString("value")))
		case "enum":
			f.Type = prop.Enum
			if opts, ok := x.RawGetString("options").(*lua.LTable); ok {
				opts.ForEach(func(_, o lua.LValue) { f.Options = append(f.Options, lua.LVAsString(o)) })
			}
			if len(f.Options) == 0 {
				return f, false
			}
			f.Default = f.Options[0]
		case "range":
			f.Type, f.Default = prop.Float, float64(lua.LVAsNumber(x.RawGetString("value")))
			f.Min, f.Max = float64(lua.LVAsNumber(x.RawGetString("min"))), float64(lua.LVAsNumber(x.RawGetString("max")))
		case "int":
			f.Type, f.Default = prop.Int, int(lua.LVAsNumber(x.RawGetString("value")))
		default:
			return f, false
		}
	default:
		return f, false
	}
	return f, true
}

// declare reads the fields of a script without running a game: the chunk
// runs in a throwaway state with a deadline, and calls into the game API
// simply fail, which leaves an already built fields table intact.
func declare(text, name string) []Field {
	L := newState()
	defer L.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	L.SetContext(ctx)
	installFields(L)
	vec := &VM{L: L}
	vec.installVec()
	fn, err := L.Load(strings.NewReader(text), name)
	if err != nil {
		return nil
	}
	L.Push(fn)
	_ = L.PCall(0, lua.MultRet, nil)
	t, ok := L.GetGlobal("fields").(*lua.LTable)
	if !ok {
		return nil
	}
	return parseFields(t, text)
}

// coerce turns a stored or inspector value into the field's Go type.
func coerce(f Field, v any) (any, error) {
	switch f.Type {
	case prop.Float:
		x, err := prop.ToFloat(v)
		if err != nil {
			return nil, err
		}
		if f.Max > f.Min {
			x = geom.Clamp(x, f.Min, f.Max)
		}
		return x, nil
	case prop.Int:
		x, err := prop.ToFloat(v)
		return int(math.Round(x)), err
	case prop.Bool:
		return prop.ToBool(v)
	case prop.Color:
		return prop.ToColor(v)
	case prop.Vec3:
		return prop.ToVec3(v)
	case prop.Enum:
		s, err := prop.ToString(v)
		if err != nil {
			return nil, err
		}
		for _, o := range f.Options {
			if o == s {
				return s, nil
			}
		}
		return nil, prop.ErrInvalidValue
	}
	return prop.ToString(v)
}
