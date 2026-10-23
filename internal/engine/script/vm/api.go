package vm

import (
	"math"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func (v *VM) install() {
	v.installVec()
	v.installObject()
	installFields(v.L)
	v.installGlobals()
	v.installEngine()
	v.installMods()
}

// selfTable is a script's self: the owner's handle, plus the script's own
// functions and free state, so self:open() and self.count both work.
func (v *VM) selfTable(s *Script) *lua.LTable {
	L := v.L
	t := L.NewTable()
	t.RawSetString(idField, lua.LNumber(s.owner.ID))
	mt := L.NewTable()
	ud := L.NewUserData()
	ud.Value = s
	mt.RawSetString("__script", ud)
	mt.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		k, ok := L.Get(2).(lua.LString)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		if o := v.objectOf(t); o != nil {
			if val, ok := v.field(o, string(k)); ok {
				L.Push(val)
				return 1
			}
		}
		L.Push(L.GetField(s.env, string(k)))
		return 1
	}))
	mt.RawSetString("__newindex", L.NewFunction(func(L *lua.LState) int {
		k, val := L.Get(2), L.Get(3)
		if name, ok := k.(lua.LString); ok {
			if o := v.objectOf(t); o != nil && v.setField(L, o, string(name), val) {
				return 0
			}
		}
		t.RawSet(k, val)
		return 0
	}))
	mt.RawSetString("__eq", v.object.RawGetString("__eq"))
	mt.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("script " + s.lib.Resolve(s.File)))
		return 1
	}))
	L.SetMetatable(t, mt)
	return t
}

// bind adds the globals that need to know which script calls them.
func (v *VM) bind(s *Script) {
	s.env.RawSetString("log", v.L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, L.GetTop())
		for i := range parts {
			parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
		}
		s.log("info", "%s", strings.Join(parts, " "))
		return 0
	}))
	s.env.RawSetString("say", v.L.NewFunction(func(L *lua.LState) int {
		if _, ok := L.Get(1).(lua.LString); ok {
			L.Insert(s.self, 1) // say("text") talks over the script's own object
		}
		return v.say(L, 1)
	}))
}

func (v *VM) walk(fn func(o *scene.GameObject)) {
	v.ctx.Scene.Walk(func(o *scene.GameObject, _ bool) bool {
		if o != v.ctx.Scene.Root {
			fn(o)
		}
		return true
	})
}

// list returns every object passing keep, as a Lua array.
func (v *VM) list(keep func(o *scene.GameObject) bool) *lua.LTable {
	t := v.L.NewTable()
	v.walk(func(o *scene.GameObject) {
		if keep(o) {
			t.Append(v.handle(o))
		}
	})
	return t
}

func (v *VM) key(L *lua.LState, n int) input.Key {
	k, err := input.ParseName(lower(L.CheckString(n)))
	if err != nil {
		L.ArgError(n, err.Error())
	}
	return k
}

func (v *VM) installGlobals() {
	L := v.L
	globals := map[string]lua.LGFunction{
		"find": func(L *lua.LState) int {
			L.Push(v.handle(v.ctx.Scene.Find(L.CheckString(1))))
			return 1
		},
		"find_all": func(L *lua.LState) int {
			name := L.CheckString(1)
			L.Push(v.list(func(o *scene.GameObject) bool { return o.Name == name }))
			return 1
		},
		"find_by_tag": func(L *lua.LState) int {
			tag := L.CheckString(1)
			L.Push(v.list(func(o *scene.GameObject) bool { return o.Tag == tag }))
			return 1
		},
		"find_by_component": func(L *lua.LState) int {
			kind := L.CheckString(1)
			L.Push(v.list(func(o *scene.GameObject) bool { return o.Component(kind) != nil }))
			return 1
		},
		"find_by_prefab": func(L *lua.LState) int {
			want := v.lib.Resolve(L.CheckString(1))
			L.Push(v.list(func(o *scene.GameObject) bool {
				return o.Prefab != "" && !o.Inherited() && v.lib.Resolve(o.Prefab) == want
			}))
			return 1
		},
		"create": func(L *lua.LState) int {
			parent := v.ctx.Scene.Root
			if L.GetTop() >= 2 {
				parent = v.checkObject(L, 2)
			}
			L.Push(v.handle(parent.Add(L.CheckString(1))))
			return 1
		},
		"spawn": func(L *lua.LState) int {
			ref := L.CheckString(1)
			if r := v.lib.Ref(ref); r != "" {
				ref = r
			}
			o, err := v.ctx.Scene.Spawn(ref, v.checkVec(L, 2))
			if err != nil {
				L.RaiseError("%v", err)
			}
			if last, ok := L.Get(L.GetTop()).(*lua.LTable); ok && L.GetTop() >= 3 && v.objectOf(last) != nil {
				v.setParent(L, o, last, true)
			}
			L.Push(v.handle(o))
			return 1
		},
		"destroy": func(L *lua.LState) int {
			v.destroy(v.checkObject(L, 1))
			return 0
		},
		"key_down": func(L *lua.LState) int {
			in := v.ctx.Input
			L.Push(lua.LBool(in != nil && in.Down(v.key(L, 1), v.ctx.Scene.Time)))
			return 1
		},
		"key_pressed": func(L *lua.LState) int {
			in := v.ctx.Input
			L.Push(lua.LBool(in != nil && in.Pressed(v.key(L, 1))))
			return 1
		},
		"time": func(L *lua.LState) int {
			L.Push(lua.LNumber(v.ctx.Scene.Time))
			return 1
		},
		"time_scale": func(L *lua.LState) int {
			if L.GetTop() > 0 {
				v.ctx.SetTimeScale(float64(L.CheckNumber(1)))
			}
			L.Push(lua.LNumber(v.ctx.TimeScale()))
			return 1
		},
		"play_sound": func(L *lua.LState) int {
			if v.ctx.Audio == nil {
				return 0
			}
			ref := L.CheckString(1)
			if r := v.lib.Ref(ref); r != "" {
				ref = r
			}
			if _, err := v.ctx.Audio.Play(ref, float64(L.OptNumber(2, 1)), false); err != nil {
				L.RaiseError("%v", err)
			}
			return 0
		},
		"tween": func(L *lua.LState) int { return v.tween(L, 1) },
		"say":   func(L *lua.LState) int { return v.say(L, 1) },
		"quit": func(L *lua.LState) int {
			v.ctx.Quit = true
			return 0
		},
		// Names from the first API, kept so older scripts run unchanged.
		"dist": func(L *lua.LState) int {
			a, b := v.checkObject(L, 1).World().Pos, v.checkObject(L, 2).World().Pos
			L.Push(lua.LNumber(a.XY().Sub(b.XY()).Len()))
			return 1
		},
		"hour": func(L *lua.LState) int {
			L.Push(lua.LNumber(v.ctx.Scene.Env.Hour))
			return 1
		},
		"setHour": func(L *lua.LState) int {
			v.ctx.Scene.Env.Hour = math.Mod(float64(L.CheckNumber(1))+24, 24)
			return 0
		},
	}
	for k, fn := range globals {
		L.SetGlobal(k, L.NewFunction(fn))
	}
}

// proxy is a Lua table whose fields read and write an Editable, such as the
// camera or a render pass; extra holds functions offered next to them.
func (v *VM) proxy(get func() prop.Editable, extra map[string]lua.LGFunction) *lua.LTable {
	L := v.L
	t := L.NewTable()
	mt := L.NewTable()
	fns := map[string]*lua.LFunction{}
	for k, fn := range extra {
		fns[k] = L.NewFunction(fn)
	}
	mt.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		k := L.CheckString(2)
		if fn, ok := fns[k]; ok {
			L.Push(fn)
			return 1
		}
		if k == "object" && t == v.camera {
			o, _ := v.ctx.Scene.MainCamera()
			L.Push(v.handle(o))
			return 1
		}
		e := get()
		if e == nil {
			L.Push(lua.LNil)
			return 1
		}
		p, ok := prop.Get(e, k)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(v.toLua(prop.Encode(p)))
		return 1
	}))
	mt.RawSetString("__newindex", L.NewFunction(func(L *lua.LState) int {
		e := get()
		if e == nil {
			L.RaiseError("nothing to set %s on", L.CheckString(2))
		}
		if err := e.Set(L.CheckString(2), v.fromLua(nil, L.Get(3))); err != nil {
			L.RaiseError("%v", err)
		}
		return 0
	}))
	L.SetMetatable(t, mt)
	return t
}

// arg skips the table itself when a proxy function is called with a colon.
func arg(L *lua.LState, self *lua.LTable) int {
	if t, ok := L.Get(1).(*lua.LTable); ok && t == self {
		return 2
	}
	return 1
}

func (v *VM) installEngine() {
	L := v.L
	var cam *lua.LTable
	cam = v.proxy(func() prop.Editable { return v.ctx.Scene.CameraProps() }, map[string]lua.LGFunction{
		"follow": func(L *lua.LState) int {
			n := arg(L, cam)
			v.follow, v.rate = 0, float64(L.OptNumber(n+1, 5))
			if t, ok := L.Get(n).(*lua.LTable); ok {
				if o := v.objectOf(t); o != nil {
					v.follow = o.ID
				}
			}
			return 0
		},
	})
	v.camera = cam
	L.SetGlobal("camera", cam)
	L.SetGlobal("env", v.proxy(func() prop.Editable { return &v.ctx.Scene.Env }, nil))
	pipe := L.NewTable()
	pipe.RawSetString("pass", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(arg(L, pipe))
		if v.ctx.Pipeline == nil {
			L.Push(lua.LNil)
			return 1
		}
		p, err := v.ctx.Pipeline.Pass(name)
		e, ok := p.(prop.Editable)
		if err != nil || !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(v.proxy(func() prop.Editable { return e }, nil))
		return 1
	}))
	pipe.RawSetString("enable", L.NewFunction(func(L *lua.LState) int {
		n := arg(L, pipe)
		if v.ctx.Pipeline == nil {
			return 0
		}
		on := L.Get(n+1) == lua.LNil || lua.LVAsBool(L.Get(n+1))
		if err := v.ctx.Pipeline.SetEnabled(L.CheckString(n), on); err != nil {
			L.RaiseError("%v", err)
		}
		return 0
	}))
	pipe.RawSetString("enabled", L.NewFunction(func(L *lua.LState) int {
		n := arg(L, pipe)
		L.Push(lua.LBool(v.ctx.Pipeline != nil && v.ctx.Pipeline.Enabled(L.CheckString(n))))
		return 1
	}))
	L.SetGlobal("pipeline", pipe)
}

// drawer is the d argument of draw(self, d). Positions and sizes are in the
// object's local space, like a component's geometry.
func (v *VM) drawer(c **draw.Context, t *scene.Transform) *lua.LTable {
	L := v.L
	d := L.NewTable()
	col := func(L *lua.LState, n int) color.Color { return color.Hex(L.OptString(n, "#ffffff")) }
	at := func(L *lua.LState, n int) geom.Vec3 { return t.Apply(toVec(L.CheckTable(n))) }
	k := func() float64 { return t.Uniform() }
	fns := map[string]lua.LGFunction{
		"box": func(L *lua.LState) int {
			size := t.Size(toVec(L.CheckTable(3)))
			m := draw.M(col(L, 4)).Glow(float64(L.OptNumber(5, 0)))
			(*c).BoxFaces(geom.Footprint(at(L, 2), size), m.Tint(1.1), m)
			return 0
		},
		"blob": func(L *lua.LState) int {
			r := float64(L.CheckNumber(3)) * k()
			(*c).Blob(at(L, 2).Add(geom.V3(0, 0, r)), r, draw.M(col(L, 4)).Glow(float64(L.OptNumber(5, 0))))
			return 0
		},
		"cylinder": func(L *lua.LState) int {
			m := draw.M(col(L, 5))
			(*c).Cylinder(at(L, 2), float64(L.CheckNumber(3))*k(), float64(L.CheckNumber(4))*k(), m.Tint(1.1), m, 14)
			return 0
		},
		"cone": func(L *lua.LState) int {
			(*c).Cone(at(L, 2), float64(L.CheckNumber(3))*k(), float64(L.CheckNumber(4))*k(), draw.M(col(L, 5)), 12, geom.Vec2{})
			return 0
		},
		"point": func(L *lua.LState) int {
			(*c).Point(at(L, 2), float64(L.CheckNumber(3)), draw.M(col(L, 4)).Glow(float64(L.OptNumber(5, 1))))
			return 0
		},
		"light": func(L *lua.LState) int {
			(*c).Light(light.Point{Pos: at(L, 2), Radius: float64(L.CheckNumber(3)), Color: col(L, 4), Intensity: float64(L.OptNumber(5, 1))})
			return 0
		},
		"shadow": func(L *lua.LState) int {
			(*c).Shadow(at(L, 2), float64(L.CheckNumber(3))*k(), float64(L.OptNumber(4, L.CheckNumber(3)))*k(), .5)
			return 0
		},
	}
	for name, fn := range fns {
		d.RawSetString(name, L.NewFunction(fn))
	}
	return d
}
