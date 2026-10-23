package vm

import (
	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

// maxDepth stops tables that contain themselves.
const maxDepth = 8

// plain copies a Lua value into Go data that refers to nothing in the state
// it came from: numbers, strings, booleans, vectors and tables of them. This
// is all that crosses between a mod and the game; object and component
// handles, functions and userdata become nil, so a mod can never hold
// anything of the engine.
func plain(x lua.LValue, depth int) any {
	switch x := x.(type) {
	case lua.LBool:
		return bool(x)
	case lua.LNumber:
		return float64(x)
	case lua.LString:
		return string(x)
	case *lua.LTable:
		if x.RawGetString(idField) != lua.LNil || depth >= maxDepth {
			return nil
		}
		if isVec(x) {
			return toVec(x)
		}
		if n := x.Len(); n > 0 {
			out := make([]any, 0, n)
			for i := 1; i <= n; i++ {
				out = append(out, plain(x.RawGetInt(i), depth+1))
			}
			return out
		}
		out := map[string]any{}
		x.ForEach(func(k, val lua.LValue) {
			if name, ok := k.(lua.LString); ok {
				if p := plain(val, depth+1); p != nil {
					out[string(name)] = p
				}
			}
		})
		return out
	}
	return nil
}

func isVec(t *lua.LTable) bool {
	for _, k := range [3]string{"x", "y", "z"} {
		if _, ok := t.RawGetString(k).(lua.LNumber); !ok {
			return false
		}
	}
	return true
}

// fromPlain builds plain data in v's state, the inverse of plain.
func (v *VM) fromPlain(x any) lua.LValue {
	switch x := x.(type) {
	case []any:
		t := v.L.NewTable()
		for _, e := range x {
			t.Append(v.fromPlain(e))
		}
		return t
	case map[string]any:
		t := v.L.NewTable()
		for k, e := range x {
			t.RawSetString(k, v.fromPlain(e))
		}
		return t
	case geom.Vec3:
		return v.vec3(x.X, x.Y, x.Z)
	}
	return v.toLua(x)
}
