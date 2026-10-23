package vm

import (
	"strings"
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
)

const coinsAPI = `
local score = 0
api.func("add", function(n) score = score + n return score end)
api.value("score", function() return score end)
api.func("hero", function() return {obj = find("hero"), all = find_all("hero"), n = 1} end)
api.event("key")
`

func (w *world) has(want string) bool {
	for _, l := range w.log.lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}

func TestUnit_PlainCopiesDataAndVectors(t *testing.T) {
	// Arrange
	a, b := &VM{L: newState()}, &VM{L: newState()}
	a.installVec()
	b.installVec()
	if err := a.L.DoString(`x = {1, "two", {pos = vec3(1, 2, 3), on = true}}`); err != nil {
		t.Fatal(err)
	}

	// Act
	b.L.SetGlobal("x", b.fromPlain(plain(a.L.GetGlobal("x"), 0)))
	err := b.L.DoString(`ok = x[1] == 1 and x[2] == "two" and x[3].on and (x[3].pos + vec3(1, 1, 1)).z == 4`)

	// Assert
	if err != nil || b.L.GetGlobal("ok") != lua.LTrue {
		t.Fatalf("err %v ok %v", err, b.L.GetGlobal("ok"))
	}
}

func TestUnit_ModCallsAPIFunctionsAndReadsValues(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mods.lua": coinsAPI})
	mod := ModSource{Name: "coins", Main: `function on_load() game.add(3) game.add(2) log("score", game.score()) end`}

	// Act
	errs := StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", []ModSource{mod})

	// Assert
	if len(errs) != 0 || !w.has("info mod:coins score 5") {
		t.Fatalf("errs %v logs %v", errs, w.log.lines)
	}
}

func TestUnit_ModSeesNoEngineGlobalsOrHandles(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mods.lua": coinsAPI})
	w.s.Add("hero", components.NewBox(geom.V3(1, 1, 1), color.RGB(1, 2, 3)))
	mod := ModSource{Name: "peek", Main: `
local h = game.hero()
log(type(find), type(spawn), type(camera), type(io), type(os), type(require), h.obj, #h.all, h.n)
`}

	// Act
	errs := StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", []ModSource{mod})

	// Assert
	if len(errs) != 0 || !w.has("mod:peek nil nil nil nil nil nil nil 0 1") {
		t.Fatalf("errs %v logs %v", errs, w.log.lines)
	}
}

func TestUnit_EmitReachesModHandlers(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{
		"scripts/mods.lua": coinsAPI,
		"scripts/keys.lua": `function update(self) mods.emit("key", "e", {n = 2}) end`,
	})
	w.s.Add("keys", nil).With(w.script(t, "scripts/keys.lua"))
	errs := StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", []ModSource{{Name: "k", Main: `function on_key(k, t) log("key", k, t.n) end`}})

	// Act
	w.run(1)

	// Assert
	if len(errs) != 0 || !w.has("mod:k key e 2") {
		t.Fatalf("errs %v logs %v", errs, w.log.lines)
	}
}

func TestUnit_UndeclaredEmitIsAnError(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{
		"scripts/mods.lua": coinsAPI,
		"scripts/keys.lua": `function update(self) mods.emit("secret") end`,
	})
	w.s.Add("keys", nil).With(w.script(t, "scripts/keys.lua"))
	_ = StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", nil)

	// Act
	w.run(1)

	// Assert
	if !w.has(ErrModEvent.Error()) {
		t.Fatalf("logs %v", w.log.lines)
	}
}

func TestUnit_EmitWithoutModsDoesNothing(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/keys.lua": `function update(self) mods.emit("anything") end`})
	w.s.Add("keys", nil).With(w.script(t, "scripts/keys.lua"))

	// Act
	w.run(1)

	// Assert
	if len(w.log.lines) != 0 {
		t.Fatalf("logs %v", w.log.lines)
	}
}

func TestUnit_RunawayModTimesOutAndOthersRun(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mods.lua": coinsAPI})
	mods := []ModSource{
		{Name: "spin", Main: `function on_load() while true do end end`},
		{Name: "calm", Main: `function on_load() log("fine") end`},
	}
	start := time.Now()

	// Act
	errs := StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", mods)

	// Assert
	if len(errs) != 0 || time.Since(start) > 2*time.Second || !w.has("error mod:spin on_load") || !w.has("mod:calm fine") {
		t.Fatalf("errs %v logs %v", errs, w.log.lines)
	}
}

func TestUnit_BrokenModIsLeftOut(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mods.lua": coinsAPI})
	mods := []ModSource{{Name: "bad", Main: `this is not lua`}, {Name: "good", Main: `function on_load() log("up") end`}}

	// Act
	errs := StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", mods)

	// Assert
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "mod bad") || !w.has("mod:good up") {
		t.Fatalf("errs %v logs %v", errs, w.log.lines)
	}
}

func TestUnit_AssetPathsStayInsideTheMod(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mods.lua": coinsAPI})
	mod := ModSource{Name: "coins", Main: `log(asset("art/coin.sprite.json"), pcall(asset, "../../data/game.pak"))`}

	// Act
	_ = StartMods(w.c, w.lib, w.reg, "scripts/mods.lua", []ModSource{mod})

	// Assert
	if !w.has("mod:coins mods/coins/art/coin.sprite.json false") {
		t.Fatalf("logs %v", w.log.lines)
	}
}
