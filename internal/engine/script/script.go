// Package script runs Lua behaviours. Each scripted entity owns a sandboxed
// Lua state with hooks start(self), update(self, dt) and key(self, k).
// Scripts reload when their source changes, which is how the editor and an
// external text editor hot swap game logic.
package script

import (
	"fmt"
	"math"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

const Kind = "script"

func Register(r *scene.Registry, lib *asset.Library) {
	r.Behaviour(Kind, func() scene.Behaviour { return New(lib, "") })
}

type Script struct {
	File    string
	lib     *asset.Library
	L       *lua.LState
	self    *lua.LTable
	version int
	failed  bool
	ctx     *scene.Ctx
	owner   *scene.Entity
	says    []saying
}

func New(lib *asset.Library, file string) *Script { return &Script{File: file, lib: lib} }

func (*Script) Kind() string                   { return Kind }
func (s *Script) bind() *prop.Binder           { return prop.Bind().Script("file", &s.File) }
func (s *Script) Props() []prop.Prop           { return s.bind().Props() }
func (s *Script) Set(name string, v any) error { return s.bind().Set(name, v) }

// log names the file, not its GUID, so the console points at something readable.
func (s *Script) log(level, format string, args ...any) {
	src := s.File
	if s.lib != nil {
		src = s.lib.Resolve(s.File)
	}
	s.ctx.Logf(level, src, format, args...)
}

// load (re)builds the Lua state when the source version moved.
func (s *Script) load() bool {
	if s.File == "" || s.lib == nil {
		return false
	}
	text, ver, err := s.lib.Script(s.File)
	if err != nil {
		if !s.failed {
			s.log("error", "%v", err)
			s.failed = true
		}
		return false
	}
	if ver == s.version && s.L != nil {
		return !s.failed
	}
	if s.L != nil {
		s.L.Close()
	}
	s.version, s.failed = ver, false
	s.L = newState()
	s.install()
	s.self = s.handle(s.owner.ID)
	fn, err := s.L.Load(strings.NewReader(text), s.lib.Resolve(s.File))
	if err == nil {
		s.L.Push(fn)
		err = s.L.PCall(0, lua.MultRet, nil)
	}
	if err != nil {
		s.fail(err)
		return false
	}
	if s.version > 1 {
		s.log("info", "reloaded")
	}
	s.call("start", s.self)
	return !s.failed
}

// fail keeps only the first line; the Lua traceback adds nothing for game code.
func (s *Script) fail(err error) {
	s.failed = true
	msg, _, _ := strings.Cut(err.Error(), "\n")
	s.log("error", "%s", msg)
}

// call runs a global hook if the script defines it and returns its first result.
func (s *Script) call(name string, args ...lua.LValue) lua.LValue {
	fn, ok := s.L.GetGlobal(name).(*lua.LFunction)
	if !ok {
		return lua.LNil
	}
	if err := s.L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, args...); err != nil {
		s.fail(err)
		return lua.LNil
	}
	ret := s.L.Get(-1)
	s.L.Pop(1)
	return ret
}

func (s *Script) Update(c *scene.Ctx, e *scene.Entity, dt float64) {
	s.ctx, s.owner = c, e
	if !s.load() {
		return
	}
	s.call("update", s.self, lua.LNumber(dt))
	s.expireSays()
}

func (s *Script) Key(c *scene.Ctx, e *scene.Entity, k input.Key) bool {
	s.ctx, s.owner = c, e
	if !s.load() {
		return false
	}
	return lua.LVAsBool(s.call("key", s.self, lua.LString(k.String())))
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

func toLua(L *lua.LState, v any) lua.LValue {
	switch x := v.(type) {
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
		t := L.NewTable()
		for i, k := range []string{"x", "y", "z"} {
			if i < len(x) {
				t.RawSetString(k, lua.LNumber(x[i]))
			}
		}
		return t
	}
	return lua.LString(fmt.Sprint(v))
}

func fromLua(v lua.LValue) any {
	switch x := v.(type) {
	case lua.LBool:
		return bool(x)
	case lua.LNumber:
		return float64(x)
	case lua.LString:
		return string(x)
	case *lua.LTable:
		return map[string]any{"x": float64(lua.LVAsNumber(x.RawGetString("x"))), "y": float64(lua.LVAsNumber(x.RawGetString("y"))), "z": float64(lua.LVAsNumber(x.RawGetString("z")))}
	}
	return nil
}

func round(v float64) float64 { return math.Round(v*1e6) / 1e6 }
