package vm

import (
	"fmt"
	"math"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type vmKey struct{}

const (
	idField  = "__id"
	keyField = "__key"
)

// VM is the Lua state of one run. Every script gets its own environment in
// it, so scripts keep separate globals but can call into each other, which
// separate states would not allow.
type VM struct {
	L       *lua.LState
	ctx     *scene.Ctx
	lib     *asset.Library
	reg     *scene.Registry // nil when scripts were made without one
	object  *lua.LTable     // metatable of GameObject handles
	comp    *lua.LTable     // metatable of component handles
	vec     *lua.LTable     // metatable of vec3 values
	methods map[string]*lua.LFunction
	frame   float64 // scene time of the last once per frame step
	follow  uint64
	rate    float64
	says    []saying
}

type saying struct {
	id     uint64
	expire float64
}

// vmFor returns the run's VM, creating it on first use.
func vmFor(c *scene.Ctx, lib *asset.Library, reg *scene.Registry) *VM {
	if c.Shared == nil {
		c.Shared = map[any]any{}
	}
	if v, ok := c.Shared[vmKey{}].(*VM); ok {
		v.ctx = c
		if v.reg == nil {
			v.reg = reg
		}
		return v
	}
	v := &VM{L: newState(), ctx: c, lib: lib, reg: reg, frame: -1}
	v.install()
	c.Shared[vmKey{}] = v
	return v
}

// newState opens only side effect free libraries; scripts cannot touch files.
func newState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true, CallStackSize: 256, RegistrySize: 1024 * 16})
	for _, lib := range []struct {
		name string
		fn   lua.LGFunction
	}{{lua.BaseLibName, lua.OpenBase}, {lua.TabLibName, lua.OpenTable}, {lua.StringLibName, lua.OpenString}, {lua.MathLibName, lua.OpenMath}} {
		L.Push(L.NewFunction(lib.fn))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	for _, unsafe := range []string{"dofile", "loadfile", "load", "loadstring", "require", "module"} {
		L.SetGlobal(unsafe, lua.LNil)
	}
	return L
}

// env is a fresh global table for one script that reads through to _G.
func (v *VM) env() *lua.LTable {
	e := v.L.NewTable()
	mt := v.L.NewTable()
	mt.RawSetString("__index", v.L.Get(lua.GlobalsIndex))
	v.L.SetMetatable(e, mt)
	return e
}

// step runs once per frame, from whichever script updates first.
func (v *VM) step(dt float64) {
	s := v.ctx.Scene
	if v.frame == s.Time {
		return
	}
	v.frame = s.Time
	if v.follow != 0 {
		if o, err := s.ByID(v.follow); err == nil {
			p := o.World().Pos
			s.Camera.Follow(geom.V3(p.X, p.Y, 0), dt, v.rate)
		}
	}
	kept := v.says[:0]
	for _, sy := range v.says {
		if s.Time < sy.expire {
			kept = append(kept, sy)
			continue
		}
		_ = s.Remove(sy.id)
	}
	v.says = kept
}

func (v *VM) vec3(x, y, z float64) *lua.LTable {
	t := v.L.NewTable()
	t.RawSetString("x", lua.LNumber(x))
	t.RawSetString("y", lua.LNumber(y))
	t.RawSetString("z", lua.LNumber(z))
	v.L.SetMetatable(t, v.vec)
	return t
}

func toVec(t *lua.LTable) geom.Vec3 {
	return geom.V3(float64(lua.LVAsNumber(t.RawGetString("x"))), float64(lua.LVAsNumber(t.RawGetString("y"))), float64(lua.LVAsNumber(t.RawGetString("z"))))
}

// checkVec reads a vec3 at n, or three numbers starting there.
func (v *VM) checkVec(L *lua.LState, n int) geom.Vec3 {
	if t, ok := L.Get(n).(*lua.LTable); ok {
		return toVec(t)
	}
	return geom.V3(float64(L.CheckNumber(n)), float64(L.CheckNumber(n+1)), float64(L.OptNumber(n+2, 0)))
}

func (v *VM) installVec() {
	L := v.L
	v.vec = L.NewTable()
	arith := func(op func(a, b geom.Vec3) geom.Vec3) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			a, b := v.operand(L, 1), v.operand(L, 2)
			r := op(a, b)
			L.Push(v.vec3(r.X, r.Y, r.Z))
			return 1
		})
	}
	v.vec.RawSetString("__add", arith(func(a, b geom.Vec3) geom.Vec3 { return a.Add(b) }))
	v.vec.RawSetString("__sub", arith(func(a, b geom.Vec3) geom.Vec3 { return a.Sub(b) }))
	v.vec.RawSetString("__mul", arith(func(a, b geom.Vec3) geom.Vec3 { return a.Mul(b) }))
	v.vec.RawSetString("__div", arith(func(a, b geom.Vec3) geom.Vec3 { return geom.V3(a.X/b.X, a.Y/b.Y, a.Z/b.Z) }))
	v.vec.RawSetString("__unm", L.NewFunction(func(L *lua.LState) int {
		a := toVec(L.CheckTable(1))
		L.Push(v.vec3(-a.X, -a.Y, -a.Z))
		return 1
	}))
	v.vec.RawSetString("__eq", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(toVec(L.CheckTable(1)) == toVec(L.CheckTable(2))))
		return 1
	}))
	v.vec.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		a := toVec(L.CheckTable(1))
		L.Push(lua.LString(fmt.Sprintf("(%g, %g, %g)", round(a.X), round(a.Y), round(a.Z))))
		return 1
	}))
	methods := L.NewTable()
	one := func(fn func(a geom.Vec3) lua.LValue) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			L.Push(fn(toVec(L.CheckTable(1))))
			return 1
		})
	}
	two := func(fn func(a, b geom.Vec3) lua.LValue) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			L.Push(fn(toVec(L.CheckTable(1)), toVec(L.CheckTable(2))))
			return 1
		})
	}
	methods.RawSetString("len", one(func(a geom.Vec3) lua.LValue { return lua.LNumber(a.Len()) }))
	methods.RawSetString("norm", one(func(a geom.Vec3) lua.LValue { n := a.Norm(); return v.vec3(n.X, n.Y, n.Z) }))
	methods.RawSetString("dot", two(func(a, b geom.Vec3) lua.LValue { return lua.LNumber(a.Dot(b)) }))
	methods.RawSetString("dist", two(func(a, b geom.Vec3) lua.LValue { return lua.LNumber(a.Sub(b).Len()) }))
	methods.RawSetString("cross", two(func(a, b geom.Vec3) lua.LValue { c := a.Cross(b); return v.vec3(c.X, c.Y, c.Z) }))
	methods.RawSetString("lerp", L.NewFunction(func(L *lua.LState) int {
		a, b, t := toVec(L.CheckTable(1)), toVec(L.CheckTable(2)), float64(L.CheckNumber(3))
		r := a.Add(b.Sub(a).Scale(t))
		L.Push(v.vec3(r.X, r.Y, r.Z))
		return 1
	}))
	v.vec.RawSetString("__index", methods)
	L.SetGlobal("vec3", L.NewFunction(func(L *lua.LState) int {
		L.Push(v.vec3(float64(L.OptNumber(1, 0)), float64(L.OptNumber(2, 0)), float64(L.OptNumber(3, 0))))
		return 1
	}))
}

// operand lets numbers mix with vectors: 2 * v scales every axis.
func (v *VM) operand(L *lua.LState, n int) geom.Vec3 {
	if t, ok := L.Get(n).(*lua.LTable); ok {
		return toVec(t)
	}
	k := float64(L.CheckNumber(n))
	return geom.V3(k, k, k)
}

// toLua converts a property value as prop.Encode writes it.
func (v *VM) toLua(x any) lua.LValue {
	switch x := x.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(x)
	case float64:
		return lua.LNumber(x)
	case int:
		return lua.LNumber(x)
	case string:
		return lua.LString(x)
	case []float64:
		if len(x) == 3 {
			return v.vec3(x[0], x[1], x[2])
		}
	case geom.Vec3:
		return v.vec3(x.X, x.Y, x.Z)
	}
	return lua.LString(fmt.Sprint(x))
}

// fromLua converts a Lua value for prop.Set; handles become references
// written relative to owner.
func (v *VM) fromLua(owner *scene.GameObject, x lua.LValue) any {
	switch x := x.(type) {
	case lua.LBool:
		return bool(x)
	case lua.LNumber:
		return float64(x)
	case lua.LString:
		return string(x)
	case *lua.LTable:
		if o := v.objectOf(x); o != nil {
			ref := scene.RefOf(owner, o)
			if key, ok := x.RawGetString(keyField).(lua.LString); ok {
				ref += "#" + string(key)
			}
			return ref
		}
		a := toVec(x)
		return []float64{a.X, a.Y, a.Z}
	}
	return nil
}

func round(x float64) float64 { return math.Round(x*1e6) / 1e6 }

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
