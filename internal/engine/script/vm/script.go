// Package vm runs Lua scripts as components. All scripts of a run share one
// Lua state with an environment each, so they keep their own globals but can
// call each other. Hooks are start(self), update(self, dt), key(self, k),
// on_destroy(self) and draw(self, d). Scripts reload when their source
// changes, which is how an external editor hot swaps game logic.
package vm

import (
	"maps"
	"path"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const Kind = "script"

func Register(r *scene.Registry, lib *asset.Library) {
	r.Register(Kind, func() scene.Component { return &Script{lib: lib, reg: r} }, scene.Meta{Category: "logic", Label: "Script"})
}

type Script struct {
	File    string
	vars    map[string]any // inspector values by field name, kept as set until a field declares a type
	lib     *asset.Library
	reg     *scene.Registry
	vm      *VM
	env     *lua.LTable
	self    *lua.LTable
	version int
	failed  bool
	owner   *scene.GameObject
	fields  []Field
	known   int // script version fields were read for
	dirty   map[string]bool
	draw    struct {
		c *draw.Context
		t scene.Transform
		d *lua.LTable
	}
}

// New makes a script outside a registry; it cannot add components from Lua.
func New(lib *asset.Library, file string) *Script { return &Script{File: file, lib: lib} }

func (*Script) Kind() string { return Kind }

// Name is the script file's base name, which is how scripts find each other.
func (s *Script) Name() string {
	file := s.File
	if s.lib != nil {
		file = s.lib.Resolve(file)
	}
	return strings.TrimSuffix(path.Base(file), ".lua")
}

// declared returns the script's fields, read from the source when no run has
// loaded it, so the inspector shows them while editing.
func (s *Script) declared() []Field {
	if s.env != nil || s.lib == nil || s.File == "" {
		return s.fields
	}
	text, ver, err := s.lib.Script(s.File)
	if err != nil {
		return nil
	}
	if ver != s.known {
		s.fields, s.known = declare(text, s.lib.Resolve(s.File)), ver
	}
	return s.fields
}

// stored is the field's inspector value, or its declared default.
func (s *Script) stored(f Field) any {
	if raw, ok := s.vars[f.Name]; ok {
		if v, err := coerce(f, raw); err == nil {
			return v
		}
	}
	return f.Default
}

// live is what the inspector shows: during a run, plain values come from
// the game, since scripts change their own fields.
func (s *Script) live(f Field) any {
	if s.self != nil && !s.dirty[f.Name] && (f.Type == prop.Float || f.Type == prop.Int || f.Type == prop.Bool || f.Type == prop.String) {
		if v, err := coerce(f, s.vm.fromLua(s.owner, s.self.RawGetString(f.Name))); err == nil {
			return v
		}
	}
	return s.stored(f)
}

func (s *Script) Props() []prop.Prop {
	out := prop.Bind().Script("file", &s.File).Props()
	for _, f := range s.declared() {
		out = append(out, prop.Prop{Name: f.Name, Type: f.Type, Value: s.live(f), Options: f.Options, Min: f.Min, Max: f.Max})
	}
	return out
}

func (s *Script) Set(name string, v any) error {
	if name == "file" {
		s.known = -1
		return prop.Bind().Script("file", &s.File).Set(name, v)
	}
	for _, f := range s.declared() {
		if f.Name != name {
			continue
		}
		x, err := coerce(f, v)
		if err != nil {
			return err
		}
		v = x
	}
	if s.vars == nil {
		s.vars = map[string]any{}
	}
	s.vars[name] = v
	if s.dirty == nil {
		s.dirty = map[string]bool{}
	}
	s.dirty[name] = true
	return nil
}

// Vars are the inspector values, for copying a script with its settings.
func (s *Script) Vars() map[string]any { return maps.Clone(s.vars) }

func (s *Script) log(level, format string, args ...any) {
	src := s.File
	if s.lib != nil {
		src = s.lib.Resolve(s.File)
	}
	s.vm.ctx.Logf(level, src, format, args...)
}

// prepare binds the script to a run and (re)loads it when its source moved.
func (s *Script) prepare(c *scene.Ctx, o *scene.GameObject) bool {
	s.owner = o
	if s.File == "" || s.lib == nil {
		return false
	}
	v := vmFor(c, s.lib, s.reg)
	if s.vm != v {
		s.vm, s.version, s.env, s.self = v, 0, nil, nil
	}
	text, ver, err := s.lib.Script(s.File)
	if err != nil {
		if !s.failed {
			s.log("error", "%v", err)
			s.failed = true
		}
		return false
	}
	if ver == s.version && s.env != nil {
		return !s.failed
	}
	reload := s.env != nil
	s.version, s.failed = ver, false
	s.env = v.env()
	if s.self == nil {
		s.self = v.selfTable(s)
	}
	v.bind(s)
	fn, err := v.L.Load(strings.NewReader(text), s.lib.Resolve(s.File))
	if err != nil {
		s.fail(err)
		return false
	}
	fn.Env = s.env
	v.L.Push(fn)
	if err := v.L.PCall(0, lua.MultRet, nil); err != nil {
		s.fail(err)
		return false
	}
	s.fields = nil
	if t, ok := s.env.RawGetString("fields").(*lua.LTable); ok {
		s.fields = parseFields(t, text)
	}
	for _, f := range s.fields {
		s.apply(f)
	}
	s.dirty = nil
	if reload {
		s.log("info", "reloaded")
	}
	s.call("start", s.self)
	return !s.failed
}

// apply puts a field's value on self, resolving references to handles.
func (s *Script) apply(f Field) {
	v, val := s.vm, s.stored(f)
	var lv lua.LValue
	switch f.Type {
	case prop.ObjectRef:
		lv = v.handle(v.ctx.Scene.Lookup(s.owner, val.(string)))
	case prop.CompRef:
		kind := ""
		if len(f.Options) > 0 {
			kind = f.Options[0]
		}
		o, c := v.ctx.Scene.LookupComponent(s.owner, val.(string), kind)
		lv = lua.LNil
		if o != nil {
			lv = v.component(o, c)
		}
	case prop.Color:
		lv = lua.LString(val.(color.Color).Hex())
	case prop.Vec3:
		lv = v.toLua(val.(geom.Vec3))
	default:
		lv = v.toLua(val)
	}
	s.self.RawSetString(f.Name, lv)
}

// fail keeps only the first line; the Lua traceback adds nothing for game code.
func (s *Script) fail(err error) {
	s.failed = true
	msg, _, _ := strings.Cut(err.Error(), "\n")
	s.log("error", "%s", msg)
}

// call runs a hook if the script defines it and returns its first result.
func (s *Script) call(name string, args ...lua.LValue) lua.LValue {
	fn, ok := s.env.RawGetString(name).(*lua.LFunction)
	if !ok || s.failed {
		return lua.LNil
	}
	L := s.vm.L
	if err := L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, args...); err != nil {
		s.fail(err)
		return lua.LNil
	}
	ret := L.Get(-1)
	L.Pop(1)
	return ret
}

func (s *Script) Update(c *scene.Ctx, o *scene.GameObject, dt float64) {
	if !s.prepare(c, o) {
		return
	}
	s.vm.step(dt)
	for _, f := range s.fields {
		if s.dirty[f.Name] {
			s.apply(f)
		}
	}
	s.dirty = nil
	s.call("update", s.self, lua.LNumber(dt))
}

func (s *Script) Key(c *scene.Ctx, o *scene.GameObject, k input.Key) bool {
	if !s.prepare(c, o) {
		return false
	}
	return lua.LVAsBool(s.call("key", s.self, lua.LString(k.String())))
}

// Draw runs the draw hook while a game runs; scripts do not run in the editor.
func (s *Script) Draw(c *draw.Context, t scene.Transform) {
	if s.env == nil || s.failed {
		return
	}
	if _, ok := s.env.RawGetString("draw").(*lua.LFunction); !ok {
		return
	}
	s.draw.c, s.draw.t = c, t
	if s.draw.d == nil {
		s.draw.d = s.vm.drawer(&s.draw.c, &s.draw.t)
	}
	s.call("draw", s.self, s.draw.d)
}
