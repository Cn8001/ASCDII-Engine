package script

import (
	"math"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/kit"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

const idField = "__id"

type saying struct {
	id     uint64
	expire float64
}

// handle is a Lua table standing for an entity. Methods come from a shared
// metatable; any other field is free script state, as in self.open = true.
func (s *Script) handle(id uint64) *lua.LTable {
	t := s.L.NewTable()
	t.RawSetString(idField, lua.LNumber(id))
	s.L.SetMetatable(t, s.L.GetGlobal("__entity"))
	return t
}

func (s *Script) entity(L *lua.LState, n int) *scene.Entity {
	t := L.CheckTable(n)
	id := uint64(lua.LVAsNumber(t.RawGetString(idField)))
	e, err := s.ctx.Scene.ByID(id)
	if err != nil {
		L.RaiseError("entity is gone")
	}
	return e
}

func (s *Script) install() {
	L := s.L
	methods := map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			p, ok := prop.Get(s.entity(L, 1), L.CheckString(2))
			if !ok {
				L.RaiseError("unknown property %s", L.CheckString(2))
			}
			L.Push(toLua(L, prop.Encode(p)))
			return 1
		},
		"set": func(L *lua.LState) int {
			if err := s.entity(L, 1).Set(L.CheckString(2), fromLua(L.CheckAny(3))); err != nil {
				L.RaiseError("%v", err)
			}
			return 0
		},
		"pos": func(L *lua.LState) int {
			p := s.entity(L, 1).Local.Pos
			L.Push(lua.LNumber(round(p.X)))
			L.Push(lua.LNumber(round(p.Y)))
			L.Push(lua.LNumber(round(p.Z)))
			return 3
		},
		"moveTo": func(L *lua.LState) int {
			e := s.entity(L, 1)
			e.Local.Pos = geom.V3(float64(L.CheckNumber(2)), float64(L.CheckNumber(3)), float64(L.OptNumber(4, lua.LNumber(e.Local.Pos.Z))))
			return 0
		},
		"move": func(L *lua.LState) int {
			e := s.entity(L, 1)
			e.Local.Pos = e.Local.Pos.Add(geom.V3(float64(L.CheckNumber(2)), float64(L.CheckNumber(3)), float64(L.OptNumber(4, 0))))
			return 0
		},
		"name": func(L *lua.LState) int {
			L.Push(lua.LString(s.entity(L, 1).Name))
			return 1
		},
		"tween": func(L *lua.LState) int { return s.tween(L, 1) },
		"say":   func(L *lua.LState) int { return s.say(L, 1) },
		"destroy": func(L *lua.LState) int {
			_ = s.ctx.Scene.Remove(s.entity(L, 1).ID)
			return 0
		},
	}
	meta := L.NewTable()
	index := L.NewTable()
	for k, fn := range methods {
		index.RawSetString(k, L.NewFunction(fn))
	}
	meta.RawSetString("__index", index)
	L.SetGlobal("__entity", meta)
	globals := map[string]lua.LGFunction{
		"find": func(L *lua.LState) int {
			e := s.ctx.Scene.Find(L.CheckString(1))
			if e == nil {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(s.handle(e.ID))
			return 1
		},
		"dist": func(L *lua.LState) int {
			a, b := s.entity(L, 1).World().Pos, s.entity(L, 2).World().Pos
			L.Push(lua.LNumber(a.XY().Sub(b.XY()).Len()))
			return 1
		},
		"tween": func(L *lua.LState) int { return s.tween(L, 1) },
		"say":   func(L *lua.LState) int { return s.say(L, 1) },
		"spawn": func(L *lua.LState) int {
			e, err := s.ctx.Scene.Spawn(L.CheckString(1), geom.V3(float64(L.CheckNumber(2)), float64(L.CheckNumber(3)), float64(L.OptNumber(4, 0))))
			if err != nil {
				L.RaiseError("%v", err)
			}
			L.Push(s.handle(e.ID))
			return 1
		},
		"log": func(L *lua.LState) int {
			parts := make([]string, L.GetTop())
			for i := range parts {
				parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
			}
			s.log("info", "%s", strings.Join(parts, " "))
			return 0
		},
		"time": func(L *lua.LState) int {
			L.Push(lua.LNumber(s.ctx.Scene.Time))
			return 1
		},
		"hour": func(L *lua.LState) int {
			L.Push(lua.LNumber(s.ctx.Scene.Env.Hour))
			return 1
		},
		"setHour": func(L *lua.LState) int {
			s.ctx.Scene.Env.Hour = math.Mod(float64(L.CheckNumber(1))+24, 24)
			return 0
		},
		"quit": func(L *lua.LState) int {
			s.ctx.Quit = true
			return 0
		},
	}
	for k, fn := range globals {
		L.SetGlobal(k, L.NewFunction(fn))
	}
}

func (s *Script) tween(L *lua.LState, at int) int {
	e := s.entity(L, at)
	ease := anim.Ease(L.OptString(at+4, string(anim.InOut)))
	if err := s.ctx.Scene.TweenTo(e, L.CheckString(at+1), fromLua(L.CheckAny(at+2)), float64(L.OptNumber(at+3, 1)), ease); err != nil {
		L.RaiseError("%v", err)
	}
	return 0
}

// say floats a label over an entity for a few seconds; saying the same text
// again only extends it, so calling it every frame is cheap.
func (s *Script) say(L *lua.LState, at int) int {
	e := s.owner
	if L.Get(at).Type() == lua.LTTable {
		e = s.entity(L, at)
		at++
	}
	text := L.CheckString(at)
	until := s.ctx.Scene.Time + float64(L.OptNumber(at+1, 2))
	for i, sy := range s.says {
		if l, err := s.ctx.Scene.ByID(sy.id); err == nil && l.Parent() == e && l.Node.(*kit.Label).Text == text {
			s.says[i].expire = until
			return 0
		}
	}
	label := kit.NewLabel(text)
	label.Solid = true
	child := e.Add("say", label)
	h := e.Bounds().Size().Z
	if h == 0 {
		h = 2
	}
	child.Local.Pos = geom.V3(0, 0, h+.8)
	s.says = append(s.says, saying{id: child.ID, expire: until})
	return 0
}

func (s *Script) expireSays() {
	kept := s.says[:0]
	for _, sy := range s.says {
		if s.ctx.Scene.Time < sy.expire {
			kept = append(kept, sy)
			continue
		}
		_ = s.ctx.Scene.Remove(sy.id)
	}
	s.says = kept
}
