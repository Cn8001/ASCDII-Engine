package vm

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"

	lua "github.com/yuin/gopher-lua"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/geom"
)

// storeGlobals are save(slot, value) and load(slot): a game keeps plain
// data between runs, copied with the same rules as data crossing to mods.
// Both return nil and a message when there is no store or it fails, so a
// game without saves still runs.
func (v *VM) storeGlobals() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"save": func(L *lua.LState) int {
			slot := L.CheckString(1)
			if v.ctx.Store == nil {
				return fail(L, "no save store")
			}
			data, err := json.Marshal(jsonable(plain(L.Get(2), 0)))
			if err != nil {
				return fail(L, err.Error())
			}
			if err := v.ctx.Store.Save(luaContext(L), slot, data); err != nil {
				return fail(L, err.Error())
			}
			L.Push(lua.LTrue)
			return 1
		},
		"load": func(L *lua.LState) int {
			slot := L.CheckString(1)
			if v.ctx.Store == nil {
				return fail(L, "no save store")
			}
			data, err := v.ctx.Store.Load(luaContext(L), slot)
			if errors.Is(err, fs.ErrNotExist) {
				L.Push(lua.LNil)
				return 1
			}
			if err != nil {
				return fail(L, err.Error())
			}
			var x any
			if err := json.Unmarshal(data, &x); err != nil {
				return fail(L, err.Error())
			}
			L.Push(v.fromPlain(x))
			return 1
		},
	}
}

func fail(L *lua.LState, msg string) int {
	L.Push(lua.LNil)
	L.Push(lua.LString(msg))
	return 2
}

func luaContext(L *lua.LState) context.Context {
	if c := L.Context(); c != nil {
		return c
	}
	return context.Background()
}

// jsonable turns vectors into {x, y, z} objects; they load back as tables
// with those fields.
func jsonable(x any) any {
	switch x := x.(type) {
	case geom.Vec3:
		return map[string]float64{"x": x.X, "y": x.Y, "z": x.Z}
	case []any:
		for i, e := range x {
			x[i] = jsonable(e)
		}
	case map[string]any:
		for k, e := range x {
			x[k] = jsonable(e)
		}
	}
	return x
}
