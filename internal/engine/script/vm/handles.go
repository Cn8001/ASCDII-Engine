package vm

import (
	"math"
	"slices"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// handle is a Lua table standing for a GameObject; nil for nil.
func (v *VM) handle(o *scene.GameObject) lua.LValue {
	if o == nil || o == v.ctx.Scene.Root {
		return lua.LNil
	}
	t := v.L.NewTable()
	t.RawSetString(idField, lua.LNumber(o.ID))
	v.L.SetMetatable(t, v.object)
	return t
}

func (v *VM) objectOf(t *lua.LTable) *scene.GameObject {
	id, ok := t.RawGetString(idField).(lua.LNumber)
	if !ok {
		return nil
	}
	o, err := v.ctx.Scene.ByID(uint64(id))
	if err != nil {
		return nil
	}
	return o
}

func (v *VM) checkObject(L *lua.LState, n int) *scene.GameObject {
	t, ok := L.Get(n).(*lua.LTable)
	if !ok {
		L.ArgError(n, "object expected")
	}
	o := v.objectOf(t)
	if o == nil {
		L.RaiseError("object is gone")
	}
	return o
}

// component is what scripts see for c: a script's own self table, so its
// functions can be called, or a handle whose fields are c's properties.
func (v *VM) component(o *scene.GameObject, c scene.Component) lua.LValue {
	if c == nil {
		return lua.LNil
	}
	if s, ok := c.(*Script); ok {
		if s.prepare(v.ctx, o) {
			return s.self
		}
		return lua.LNil
	}
	t := v.L.NewTable()
	t.RawSetString(idField, lua.LNumber(o.ID))
	t.RawSetString(keyField, lua.LString(o.Key(c)))
	v.L.SetMetatable(t, v.comp)
	return t
}

// value converts a property for Lua, resolving references to handles.
func (v *VM) value(o *scene.GameObject, p prop.Prop) lua.LValue {
	ref, _ := p.Value.(string)
	switch p.Type {
	case prop.ObjectRef:
		return v.handle(v.ctx.Scene.Lookup(o, ref))
	case prop.CompRef:
		kind := ""
		if len(p.Options) > 0 {
			kind = p.Options[0]
		}
		t, c := v.ctx.Scene.LookupComponent(o, ref, kind)
		if t == nil {
			return lua.LNil
		}
		return v.component(t, c)
	}
	return v.toLua(prop.Encode(p))
}

func (v *VM) get(L *lua.LState, o *scene.GameObject, name string) lua.LValue {
	p, ok := prop.Get(o, o.Resolve(name))
	if !ok {
		L.RaiseError("%s has no property %s", o.Name, name)
	}
	return v.value(o, p)
}

func (v *VM) set(L *lua.LState, o *scene.GameObject, name string, val lua.LValue) {
	if err := o.Set(name, v.fromLua(o, val)); err != nil {
		L.RaiseError("%v", err)
	}
}

// field reads the built in fields of a GameObject handle and its methods.
func (v *VM) field(o *scene.GameObject, k string) (lua.LValue, bool) {
	switch k {
	case "id":
		return lua.LNumber(o.ID), true
	case "name":
		return lua.LString(o.Name), true
	case "tag":
		return lua.LString(o.Tag), true
	case "active":
		return lua.LBool(o.Visible), true
	case "solid":
		return lua.LBool(o.Solid), true
	case "position", "world_position":
		return v.toLua(o.World().Pos), true
	case "local_position":
		return v.toLua(o.Local.Pos), true
	case "rotation":
		return v.toLua(o.World().Rot), true
	case "local_rotation":
		return v.toLua(o.Local.Rot), true
	case "scale", "local_scale":
		return v.toLua(o.Local.Scale), true
	case "world_scale":
		return v.toLua(o.World().Scale), true
	case "forward":
		return v.toLua(o.World().Dir(geom.V3(1, 0, 0))), true
	case "right":
		return v.toLua(o.World().Dir(geom.V3(0, 1, 0))), true
	case "up":
		return v.toLua(o.World().Dir(geom.V3(0, 0, 1))), true
	case "parent":
		return v.handle(o.Parent()), true
	case "root":
		r := o
		for r.Parent() != nil && r.Parent() != v.ctx.Scene.Root {
			r = r.Parent()
		}
		return v.handle(r), true
	case "prefab":
		if o.Prefab == "" {
			return lua.LNil, true
		}
		return lua.LString(v.lib.Resolve(o.Prefab)), true
	case "children":
		t := v.L.NewTable()
		for _, c := range o.Children {
			t.Append(v.handle(c))
		}
		return t, true
	}
	if m, ok := v.methods[k]; ok {
		return m, true
	}
	return lua.LNil, false
}

func (v *VM) setField(L *lua.LState, o *scene.GameObject, k string, val lua.LValue) bool {
	switch k {
	case "name":
		o.Name = lua.LVAsString(val)
	case "tag":
		o.Tag = lua.LVAsString(val)
	case "active":
		o.Visible = lua.LVAsBool(val)
	case "solid":
		o.Solid = lua.LVAsBool(val)
	case "position", "rotation", "local_position", "local_rotation", "scale", "local_scale":
		t, ok := val.(*lua.LTable)
		if !ok {
			L.RaiseError("%s needs a vec3", k)
		}
		p := toVec(t)
		switch k {
		case "position":
			w := o.World()
			w.Pos = p
			o.SetWorld(w)
		case "rotation":
			w := o.World()
			w.Rot = p
			o.SetWorld(w)
		case "local_position":
			o.Local.Pos = p
		case "local_rotation":
			o.Local.Rot = p
		default:
			o.Local.Scale = p
		}
	case "parent":
		v.setParent(L, o, val, true)
	default:
		return false
	}
	return true
}

// setParent moves o under the object behind val, or to the top for nil.
func (v *VM) setParent(L *lua.LState, o *scene.GameObject, val lua.LValue, keepWorld bool) {
	var parent uint64
	if t, ok := val.(*lua.LTable); ok {
		if p := v.objectOf(t); p != nil {
			parent = p.ID
		}
	}
	if err := v.ctx.Scene.Reparent(o.ID, parent, -1, keepWorld); err != nil {
		L.RaiseError("%v", err)
	}
}

func (v *VM) installObject() {
	L := v.L
	v.object = L.NewTable()
	v.object.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		o := v.checkObject(L, 1)
		val, _ := v.field(o, L.CheckString(2))
		L.Push(val)
		return 1
	}))
	v.object.RawSetString("__newindex", L.NewFunction(func(L *lua.LState) int {
		o := v.checkObject(L, 1)
		if k := L.CheckString(2); !v.setField(L, o, k, L.Get(3)) {
			L.RaiseError("cannot set %s on an object; use a script field instead", k)
		}
		return 0
	}))
	same := L.NewFunction(func(L *lua.LState) int {
		a, b := L.CheckTable(1), L.CheckTable(2)
		L.Push(lua.LBool(a.RawGetString(idField) == b.RawGetString(idField) && a.RawGetString(keyField) == b.RawGetString(keyField)))
		return 1
	})
	v.object.RawSetString("__eq", same)
	v.object.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		if o := v.objectOf(L.CheckTable(1)); o != nil {
			L.Push(lua.LString("object " + o.Name))
		} else {
			L.Push(lua.LString("object (gone)"))
		}
		return 1
	}))

	v.comp = L.NewTable()
	comp := func(L *lua.LState) (*scene.GameObject, scene.Component, string) {
		t := L.CheckTable(1)
		o := v.objectOf(t)
		key := lua.LVAsString(t.RawGetString(keyField))
		if o == nil {
			L.RaiseError("component is gone")
		}
		c := o.ComponentByKey(key)
		if c == nil {
			L.RaiseError("component %s is gone", key)
		}
		return o, c, key
	}
	v.comp.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		o, c, key := comp(L)
		switch k := L.CheckString(2); k {
		case "enabled":
			L.Push(lua.LBool(o.Enabled(c)))
		case "kind":
			L.Push(lua.LString(c.Kind()))
		case "object":
			L.Push(v.handle(o))
		default:
			if cl, ok := c.(scene.Callable); ok && slices.Contains(cl.Methods(), k) {
				L.Push(v.method(o, cl, k))
				return 1
			}
			p, ok := prop.Get(o, key+"."+k)
			if !ok {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(v.value(o, p))
		}
		return 1
	}))
	v.comp.RawSetString("__newindex", L.NewFunction(func(L *lua.LState) int {
		o, c, key := comp(L)
		if k := L.CheckString(2); k == "enabled" {
			o.SetEnabled(c, lua.LVAsBool(L.Get(3)))
		} else {
			v.set(L, o, key+"."+k, L.Get(3))
		}
		return 0
	}))
	v.comp.RawSetString("__eq", same)
	v.comp.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("component " + lua.LVAsString(L.CheckTable(1).RawGetString(keyField))))
		return 1
	}))
	v.installMethods()
}

// installMethods are the functions every object handle and script self has.
func (v *VM) installMethods() {
	m := map[string]lua.LGFunction{
		"component": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			k := L.CheckString(2)
			L.Push(v.component(o, v.find(o, k)))
			return 1
		},
		"components": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			t := L.NewTable()
			kind := L.OptString(2, "")
			for _, c := range o.Components {
				if kind == "" || c.Kind() == kind {
					t.Append(v.component(o, c))
				}
			}
			L.Push(t)
			return 1
		},
		"add_component": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			if v.reg == nil {
				L.RaiseError("add_component needs scripts made by the registry")
			}
			c, err := v.reg.New(L.CheckString(2))
			if err != nil {
				L.RaiseError("%v", err)
			}
			o.With(c)
			L.Push(v.component(o, c))
			return 1
		},
		"remove_component": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			c := v.componentOf(L.CheckTable(2))
			for i, x := range o.Components {
				if x == c {
					_ = o.RemoveComponent(i)
					break
				}
			}
			return 0
		},
		"child": func(L *lua.LState) int {
			L.Push(v.handle(scene.FindPath(v.checkObject(L, 1), L.CheckString(2))))
			return 1
		},
		"destroy": func(L *lua.LState) int {
			v.destroy(v.checkObject(L, 1))
			return 0
		},
		"get": func(L *lua.LState) int {
			L.Push(v.get(L, v.checkObject(L, 1), L.CheckString(2)))
			return 1
		},
		"set": func(L *lua.LState) int {
			v.set(L, v.checkObject(L, 1), L.CheckString(2), L.CheckAny(3))
			return 0
		},
		"move": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			w := o.World()
			w.Pos = w.Pos.Add(v.checkVec(L, 2))
			o.SetWorld(w)
			return 0
		},
		"move_to": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			w := o.World()
			w.Pos = v.checkVec(L, 2)
			o.SetWorld(w)
			return 0
		},
		"set_parent": func(L *lua.LState) int {
			v.setParent(L, v.checkObject(L, 1), L.Get(2), L.OptBool(3, true))
			return 0
		},
		"translate": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			d := v.checkVec(L, 2)
			w := o.World()
			if L.OptString(3, "self") != "world" {
				d = w.Dir(d)
			}
			w.Pos = w.Pos.Add(d)
			o.SetWorld(w)
			return 0
		},
		"rotate": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			o.Local.Rot = o.Local.Rot.Add(v.checkVec(L, 2))
			return 0
		},
		"look_at": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			var at geom.Vec3
			if t, ok := L.Get(2).(*lua.LTable); ok && v.objectOf(t) != nil {
				at = v.objectOf(t).World().Pos
			} else {
				at = v.checkVec(L, 2)
			}
			w := o.World()
			d := at.Sub(w.Pos)
			if d.X != 0 || d.Y != 0 {
				w.Rot.Z = math.Atan2(d.Y, d.X) * 180 / math.Pi
				o.SetWorld(w)
			}
			return 0
		},
		"transform_point": func(L *lua.LState) int {
			L.Push(v.toLua(v.checkObject(L, 1).World().Apply(v.checkVec(L, 2))))
			return 1
		},
		"inverse_transform_point": func(L *lua.LState) int {
			w := v.checkObject(L, 1).World()
			L.Push(v.toLua(w.Inward(v.checkVec(L, 2).Sub(w.Pos))))
			return 1
		},
		"distance": func(L *lua.LState) int {
			a, b := v.checkObject(L, 1).World().Pos, v.checkObject(L, 2).World().Pos
			L.Push(lua.LNumber(a.Sub(b).Len()))
			return 1
		},
		"tween": func(L *lua.LState) int { return v.tween(L, 1) },
		"say":   func(L *lua.LState) int { return v.say(L, 1) },
		"play": func(L *lua.LState) int {
			L.Push(lua.LBool(v.play(v.checkObject(L, 1), L.CheckString(2), true)))
			return 1
		},
		"stop": func(L *lua.LState) int {
			L.Push(lua.LBool(v.play(v.checkObject(L, 1), L.CheckString(2), false)))
			return 1
		},
		// Names from the first API, kept so older scripts run unchanged.
		"pos": func(L *lua.LState) int {
			p := v.checkObject(L, 1).Local.Pos
			L.Push(lua.LNumber(round(p.X)))
			L.Push(lua.LNumber(round(p.Y)))
			L.Push(lua.LNumber(round(p.Z)))
			return 3
		},
		"moveTo": func(L *lua.LState) int {
			o := v.checkObject(L, 1)
			o.Local.Pos = geom.V3(float64(L.CheckNumber(2)), float64(L.CheckNumber(3)), float64(L.OptNumber(4, lua.LNumber(o.Local.Pos.Z))))
			return 0
		},
	}
	v.methods = map[string]*lua.LFunction{}
	for k, fn := range m {
		v.methods[k] = v.L.NewFunction(fn)
	}
}

// find looks a component up by kind, by key ("light#2"), or for scripts by
// file name, so obj:component("door") finds door.lua.
func (v *VM) find(o *scene.GameObject, name string) scene.Component {
	if c := o.Component(name); c != nil {
		return c
	}
	if c := o.ComponentByKey(name); c != nil {
		return c
	}
	for _, c := range o.Components {
		if s, ok := c.(*Script); ok && s.Name() == name {
			return c
		}
	}
	return nil
}

// method binds a component method for a colon call: c:play("run").
func (v *VM) method(o *scene.GameObject, c scene.Callable, name string) *lua.LFunction {
	return v.L.NewFunction(func(L *lua.LState) int {
		args := make([]any, 0, L.GetTop())
		for i := 2; i <= L.GetTop(); i++ {
			args = append(args, v.fromLua(o, L.Get(i)))
		}
		out, err := c.Call(v.ctx, o, name, args)
		if err != nil {
			L.RaiseError("%s: %v", name, err)
		}
		L.Push(v.toLua(out))
		return 1
	})
}

// componentOf finds the component behind a handle or a script's self.
func (v *VM) componentOf(t *lua.LTable) scene.Component {
	if s, ok := v.L.GetMetatable(t).(*lua.LTable); ok {
		if sc, ok := s.RawGetString("__script").(*lua.LUserData); ok {
			c, _ := sc.Value.(*Script)
			return c
		}
	}
	o := v.objectOf(t)
	if o == nil {
		return nil
	}
	return o.ComponentByKey(lua.LVAsString(t.RawGetString(keyField)))
}

// destroy runs on_destroy hooks below o before removing it.
func (v *VM) destroy(o *scene.GameObject) {
	var walk func(x *scene.GameObject)
	walk = func(x *scene.GameObject) {
		for _, c := range x.Components {
			if s, ok := c.(*Script); ok && s.env != nil {
				s.call("on_destroy", s.self)
			}
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(o)
	_ = v.ctx.Scene.Remove(o.ID)
}

// play starts or stops the scene clips that target o by name.
func (v *VM) play(o *scene.GameObject, clip string, on bool) bool {
	found := false
	for i := range v.ctx.Scene.Clips {
		c := &v.ctx.Scene.Clips[i]
		if c.Name != clip || (c.Target != "" && c.Target != o.Name) {
			continue
		}
		found = true
		if on {
			*c = c.Delay(v.ctx.Scene.Time)
		} else {
			c.Mode = anim.Once
			*c = c.Delay(-1e9)
		}
	}
	return found
}

func (v *VM) tween(L *lua.LState, at int) int {
	o := v.checkObject(L, at)
	ease := anim.Ease(L.OptString(at+4, string(anim.InOut)))
	name := o.Resolve(L.CheckString(at + 1))
	if err := v.ctx.Scene.TweenTo(o, name, v.fromLua(o, L.CheckAny(at+2)), float64(L.OptNumber(at+3, 1)), ease); err != nil {
		L.RaiseError("%v", err)
	}
	return 0
}

// say floats a label over an object for a few seconds; saying the same text
// again only extends it, so calling it every frame is cheap.
func (v *VM) say(L *lua.LState, at int) int {
	o := v.checkObject(L, at)
	text := L.CheckString(at + 1)
	until := v.ctx.Scene.Time + float64(L.OptNumber(at+2, 2))
	for i, sy := range v.says {
		l, err := v.ctx.Scene.ByID(sy.id)
		if err != nil || l.Parent() != o {
			continue
		}
		if lb, ok := l.Component("label").(*components.Label); ok && lb.Text == text {
			v.says[i].expire = until
			return 0
		}
	}
	label := components.NewLabel(text)
	label.Solid = true
	child := o.Add("say", label)
	h := o.Bounds().Size().Z
	if h == 0 {
		h = 2
	}
	child.Local.Pos = geom.V3(0, 0, h+.8)
	v.says = append(v.says, saying{id: child.ID, expire: until})
	return 0
}
