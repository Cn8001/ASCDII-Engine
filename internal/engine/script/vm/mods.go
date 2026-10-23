package vm

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

var (
	ErrModAPI   = errors.New("mod api")
	ErrModEvent = errors.New("event is not declared in the mod api")
)

const (
	modBudget   = 50 * time.Millisecond // a mod call that runs longer is stopped
	modMaxDepth = 4                     // events emitted from inside mod calls
	ModsDir     = "mods"
)

var apiName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ModSource is one mod: its folder name and the text of its main.lua.
type ModSource struct {
	Name, Main string
}

// modHost is what the developer's API script exposes, and the mods using it.
type modHost struct {
	funcs  map[string]*lua.LFunction
	values map[string]*lua.LFunction
	events map[string]bool
	mods   []*mod
	depth  int
}

// mod runs in a Lua state of its own with only the standard math, string and
// table libraries, so nothing of the game is reachable but the api.
type mod struct {
	name string
	vm   *VM // a bare VM: only vec3 is installed
}

// StartMods runs the developer's API script in the game's VM, then every
// mod, then each mod's on_load. A mod that fails to load is reported and
// left out; the others still run.
func StartMods(c *scene.Ctx, lib *asset.Library, reg *scene.Registry, api string, sources []ModSource) []error {
	v := vmFor(c, lib, reg)
	v.mods = &modHost{funcs: map[string]*lua.LFunction{}, values: map[string]*lua.LFunction{}, events: map[string]bool{"load": true}}
	if err := v.runAPI(api); err != nil {
		v.mods = nil
		return []error{err}
	}
	var errs []error
	for _, src := range sources {
		m, err := v.loadMod(src)
		if err != nil {
			errs = append(errs, fmt.Errorf("mod %s: %w", src.Name, err))
			continue
		}
		v.mods.mods = append(v.mods.mods, m)
	}
	v.emit("load", nil)
	return errs
}

// runAPI runs the API script with the api global; its top level code
// declares what mods get.
func (v *VM) runAPI(ref string) error {
	text, _, err := v.lib.Script(ref)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrModAPI, err)
	}
	L := v.L
	env := v.env()
	api := L.NewTable()
	register := func(into map[string]*lua.LFunction) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			name := L.CheckString(1)
			if !apiName.MatchString(name) {
				L.ArgError(1, "not a valid name")
			}
			into[name] = L.CheckFunction(2)
			return 0
		})
	}
	api.RawSetString("func", register(v.mods.funcs))
	api.RawSetString("value", register(v.mods.values))
	api.RawSetString("event", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		if !apiName.MatchString(name) {
			L.ArgError(1, "not a valid name")
		}
		v.mods.events[name] = true
		return 0
	}))
	env.RawSetString("api", api)
	fn, err := L.Load(strings.NewReader(text), v.lib.Resolve(ref))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrModAPI, err)
	}
	fn.Env = env
	L.Push(fn)
	if err := L.PCall(0, 0, nil); err != nil {
		return fmt.Errorf("%w: %w", ErrModAPI, err)
	}
	return nil
}

func (v *VM) loadMod(src ModSource) (*mod, error) {
	m := &mod{name: src.Name, vm: &VM{L: newState()}}
	m.vm.installVec()
	L := m.vm.L
	game := L.NewTable()
	for name, fn := range v.mods.funcs {
		game.RawSetString(name, L.NewFunction(v.bridge(m, fn)))
	}
	for name, fn := range v.mods.values {
		game.RawSetString(name, L.NewFunction(v.bridge(m, fn)))
	}
	L.SetGlobal("game", game)
	logf := L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, 0, L.GetTop())
		for i := 1; i <= L.GetTop(); i++ {
			parts = append(parts, L.ToStringMeta(L.Get(i)).String())
		}
		v.ctx.Logf("info", "mod:"+m.name, "%s", strings.Join(parts, " "))
		return 0
	})
	L.SetGlobal("log", logf)
	L.SetGlobal("print", logf) // the standard print would write over the game's frame
	L.SetGlobal("asset", L.NewFunction(func(L *lua.LState) int {
		p := path.Clean(L.CheckString(1))
		if p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p) {
			L.ArgError(1, "outside the mod")
		}
		L.Push(lua.LString(path.Join(ModsDir, m.name, p)))
		return 1
	}))
	fn, err := L.Load(strings.NewReader(src.Main), path.Join(ModsDir, src.Name, "main.lua"))
	if err != nil {
		return nil, err
	}
	if err := m.run(func() error {
		L.Push(fn)
		return L.PCall(0, 0, nil)
	}); err != nil {
		return nil, err
	}
	return m, nil
}

// bridge is an api function as a mod sees it: arguments and the result are
// copied as plain values between the two states.
func (v *VM) bridge(m *mod, fn *lua.LFunction) lua.LGFunction {
	return func(L *lua.LState) int {
		args := make([]lua.LValue, 0, L.GetTop())
		for i := 1; i <= L.GetTop(); i++ {
			args = append(args, v.fromPlain(plain(L.Get(i), 0)))
		}
		if err := v.L.CallByParam(lua.P{Fn: fn, NRet: 1, Protect: true}, args...); err != nil {
			msg, _, _ := strings.Cut(err.Error(), "\n")
			L.RaiseError("%s", msg)
		}
		out := plain(v.L.Get(-1), 0)
		v.L.Pop(1)
		L.Push(m.vm.fromPlain(out))
		return 1
	}
}

// emit calls on_<event> in every mod that defines it.
func (v *VM) emit(event string, args []any) {
	h := v.mods
	if h == nil || h.depth >= modMaxDepth {
		return
	}
	h.depth++
	defer func() { h.depth-- }()
	for _, m := range h.mods {
		fn, ok := m.vm.L.GetGlobal("on_" + event).(*lua.LFunction)
		if !ok {
			continue
		}
		vals := make([]lua.LValue, 0, len(args))
		for _, a := range args {
			vals = append(vals, m.vm.fromPlain(a))
		}
		err := m.run(func() error {
			return m.vm.L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, vals...)
		})
		if err != nil {
			msg, _, _ := strings.Cut(err.Error(), "\n")
			v.ctx.Logf("error", "mod:"+m.name, "on_%s: %s", event, msg)
		}
	}
}

// run gives a call into the mod a deadline, so a runaway loop errors instead
// of freezing the game.
func (m *mod) run(fn func() error) error {
	L := m.vm.L
	if L.Context() != nil {
		return fn() // a nested call already runs under the outer deadline
	}
	ctx, cancel := context.WithTimeout(context.Background(), modBudget)
	defer cancel()
	L.SetContext(ctx)
	defer L.RemoveContext()
	return fn()
}

// installMods is the mods global of developer scripts: mods.emit sends an
// event declared by the api script to every mod.
func (v *VM) installMods() {
	L := v.L
	t := L.NewTable()
	t.RawSetString("emit", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(arg(L, t))
		if v.mods == nil {
			return 0
		}
		if !v.mods.events[name] {
			L.RaiseError("%v: %s", ErrModEvent, name)
		}
		var args []any
		for i := arg(L, t) + 1; i <= L.GetTop(); i++ {
			args = append(args, plain(L.Get(i), 0))
		}
		v.emit(name, args)
		return 0
	}))
	L.SetGlobal("mods", t)
}
