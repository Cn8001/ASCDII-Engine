package vm

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type world struct {
	s   *scene.Scene
	reg *scene.Registry
	lib *asset.Library
	c   *scene.Ctx
	log *logs
}

func newWorld(t *testing.T, scripts map[string]string) *world {
	t.Helper()
	reg := scene.NewRegistry()
	lib := asset.NewLibrary(reg)
	components.Register(reg, lib)
	Register(reg, lib)
	for name, src := range scripts {
		lib.PutScript(name, src)
	}
	s := scene.New("t")
	l := &logs{}
	get := func() *scene.Scene { return s }
	p := pipeline.New(time.Now, append(pipeline.Standard(), &scene.Geometry{Scene: get})...)
	return &world{s: s, reg: reg, lib: lib, c: &scene.Ctx{Scene: s, Log: l.add, Pipeline: p}, log: l}
}

func (w *world) script(t *testing.T, file string) *Script {
	t.Helper()
	c, err := w.reg.New(Kind)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Set("file", file)
	return c.(*Script)
}

func (w *world) run(frames int) {
	for i := 0; i < frames; i++ {
		_ = w.s.Update(w.c, 1.0/60, nil)
	}
}

const mover = `
fields = {
  speed = range(4, 0, 10),
  target = object(),
  lamp = component("light"),
  mode = enum("idle", "chase"),
  tint = color("#ff8800"),
  offset = vec3(0, 0, 1),
  label = "gate",
}
`

func TestUnit_FieldsAreDeclaredInSourceOrder(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mover.lua": mover})
	sc := w.script(t, "scripts/mover.lua")

	// Act
	props := sc.Props()

	// Assert
	var names []string
	for _, p := range props {
		names = append(names, p.Name+":"+string(p.Type))
	}
	want := "file:script speed:float target:object lamp:component mode:enum tint:color offset:vec3 label:string"
	if strings.Join(names, " ") != want {
		t.Fatalf("props %v", names)
	}
	if props[1].Max != 10 || props[3].Options[0] != "light" || props[6].Value != geom.V3(0, 0, 1) {
		t.Fatalf("details %+v", props)
	}
}

func TestUnit_FieldValuesSurviveSaveAndLoad(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/mover.lua": mover})
	door := w.s.Add("door", components.NewBox(geom.V3(1, 1, 1), color.RGB(1, 2, 3)))
	o := w.s.Add("mover", w.script(t, "scripts/mover.lua"))
	_ = o.Set("speed", 7)
	_ = o.Set("target", scene.RefOf(o, door))
	var b bytes.Buffer
	if err := scene.Encode(&b, w.s); err != nil {
		t.Fatal(err)
	}

	// Act
	back, err := scene.Decode(&b, w.reg, nil)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	m := back.Find("mover")
	speed, _ := prop.Get(m, "speed")
	target, _ := prop.Get(m, "target")
	if speed.Value != 7.0 || target.Display != "door" {
		t.Fatalf("speed %v target %+v", speed.Value, target)
	}
}

func TestUnit_ObjectFieldIsAHandleAndScriptsCallEachOther(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{
		"scripts/door.lua": `
function start(self) self.opened = 0 end
function open(self) self.opened = self.opened + 1; self.position = self.position + vec3(0, 0, 3) end`,
		"scripts/lever.lua": `
fields = { door = object() }
function update(self)
  if not self.pulled then
    self.door:component("door"):open()
    self.pulled = true
  end
end`,
	})
	door := w.s.Add("gate", w.script(t, "scripts/door.lua"))
	lever := w.s.Add("lever", w.script(t, "scripts/lever.lua"))
	_ = lever.Set("door", scene.RefOf(lever, door))

	// Act
	w.run(3)

	// Assert
	if door.Local.Pos.Z != 3 || len(w.log.lines) != 0 {
		t.Fatalf("z %v logs %v", door.Local.Pos.Z, w.log.lines)
	}
}

func TestUnit_ComponentHandlesEditProperties(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/dim.lua": `
function start(self)
  local l = find("lamp"):component("light")
  l.radius = l.radius * 2
  find("lamp"):component("emitter").enabled = false
end`})
	lamp := w.s.Add("lamp", components.NewLight(5, color.RGB(255, 255, 255)), components.NewEmitter())
	w.s.Add("rules", w.script(t, "scripts/dim.lua"))

	// Act
	w.run(1)

	// Assert
	if lamp.Component("light").(*components.Light).Radius != 10 || lamp.Enabled(lamp.Component("emitter")) {
		t.Fatalf("logs %v", w.log.lines)
	}
}

func TestUnit_FindHelpersAndCreateDestroy(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/q.lua": `
function start(self)
  result = #find_by_tag("enemy") .. "," .. #find_by_component("light") .. "," .. #find_all("orc")
  local made = create("spark")
  made:add_component("blob")
  destroy(find("trash"))
  self.result = result
end`})
	w.s.Add("orc").Tag = "enemy"
	w.s.Add("orc").Tag = "enemy"
	w.s.Add("lamp", components.NewLight(5, color.RGB(1, 1, 1)))
	w.s.Add("trash")
	q := w.s.Add("q", w.script(t, "scripts/q.lua"))

	// Act
	w.run(1)

	// Assert
	sc := q.Component(Kind).(*Script)
	if got := sc.self.RawGetString("result").String(); got != "2,1,2" {
		t.Fatalf("result %s logs %v", got, w.log.lines)
	}
	if w.s.Find("trash") != nil || w.s.Find("spark") == nil || w.s.Find("spark").Component("blob") == nil {
		t.Fatal("create or destroy failed")
	}
}

func TestUnit_EngineSettingsFromLua(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/fx.lua": `
function start(self)
  pipeline.pass("fog").density = 0.9
  pipeline.enable("outline", false)
  camera.span = 12
  env.hour = 22
  time_scale(0.5)
end`})
	w.s.Add("fx", w.script(t, "scripts/fx.lua"))

	// Act
	w.run(1)
	before := w.s.Time
	w.run(1)

	// Assert
	fog, _ := w.c.Pipeline.Pass("fog")
	if fog.(*pipeline.Fog).Density != .9 || w.c.Pipeline.Enabled("outline") || w.s.Camera.Span != 12 || w.s.Env.Hour != 22 {
		t.Fatalf("settings not applied, logs %v", w.log.lines)
	}
	if math.Abs(w.s.Time-before-1.0/120) > 1e-9 {
		t.Fatalf("time step %v", w.s.Time-before)
	}
}

func TestUnit_InspectorEditDuringRunReachesScript(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/m.lua": `
fields = { speed = 1 }
function update(self, dt) self.position = vec3(self.speed, 0, 0) end`})
	o := w.s.Add("m", w.script(t, "scripts/m.lua"))
	w.run(1)

	// Act
	_ = o.Set("speed", 9)
	w.run(1)

	// Assert
	if o.Local.Pos.X != 9 {
		t.Fatalf("x %v logs %v", o.Local.Pos.X, w.log.lines)
	}
}

func TestUnit_DrawHookAndOnDestroy(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/d.lua": `
function draw(self, d) d:box(vec3(0, 0, 0), vec3(2, 2, 2), "#ff0000") end
function on_destroy(self) gone = true; log("bye") end`})
	o := w.s.Add("d", w.script(t, "scripts/d.lua"))
	w.run(1)
	f := pipeline.NewFrame(40, 20)
	w.s.Camera.Target, w.s.Camera.Span = geom.V3(0, 0, 1), 10

	// Act
	scene.Render(w.c.Pipeline, f, w.s)
	drawn := 0
	for _, d := range f.Buf.Dots() {
		if d.Tag == uint32(o.ID) {
			drawn++
		}
	}
	w.s.Add("killer", w.script(t, "scripts/k.lua"))
	w.lib.PutScript("scripts/k.lua", `function start() destroy(find("d")) end`)
	w.run(1)

	// Assert
	if drawn == 0 || w.s.Find("d") != nil || len(w.log.lines) != 1 || !strings.Contains(w.log.lines[0], "bye") {
		t.Fatalf("drawn %d logs %v", drawn, w.log.lines)
	}
}

func TestUnit_Vec3Math(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/v.lua": `
function start(self)
  local a = vec3(1, 2, 2)
  local b = a * 2 + vec3(1, 0, 0)
  self.position = b
  self.len = a:len()
end`})
	o := w.s.Add("v", w.script(t, "scripts/v.lua"))

	// Act
	w.run(1)

	// Assert
	if o.Local.Pos != geom.V3(3, 4, 4) || o.Component(Kind).(*Script).self.RawGetString("len").String() != "3" {
		t.Fatalf("pos %+v logs %v", o.Local.Pos, w.log.lines)
	}
}

type fakeAudio struct{ played []string }

func (f *fakeAudio) Play(ref string, vol float64, _ bool) (uint64, error) {
	f.played = append(f.played, fmt.Sprintf("%s@%.1f", ref, vol))
	return uint64(len(f.played)), nil
}
func (f *fakeAudio) Stop(uint64)               {}
func (f *fakeAudio) SetVolume(uint64, float64) {}
func (f *fakeAudio) Playing(uint64) bool       { return true }

func TestUnit_ScriptsDriveAnimatorAudioAndBodies(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/boss.lua": `
function start(self)
  local a = self:component("animator")
  a:set("speed", 2)
  a:trigger("hit")
  self.speed = a:get("speed")
  local src = self:component("audio_source")
  src.volume = 0.5
  src:play()
  play_sound("sfx/boom.wav", 0.3)
  local rb = self:component("rigidbody")
  rb:add_force(vec3(4, 0, 0))
  rb.gravity = false
end
function on_collision(self, other) hit = other.name end
function update(self) self.hit = hit end`})
	ctrl := animator.Default()
	ctrl.Parameters = []animator.Param{{Name: "speed", Type: animator.Float}, {Name: "hit", Type: animator.Trigger}}
	w.lib.PutController("anims/boss.animator.json", ctrl)
	audio := &fakeAudio{}
	w.c.Audio = audio
	boss := w.s.Add("boss", components.NewCircleCollider(), components.NewRigidbody(), w.script(t, "scripts/boss.lua"))
	_ = boss.Set("rigidbody.drag", 0)
	an, _ := w.reg.New("animator")
	_ = an.Set("controller", "anims/boss.animator.json")
	src := components.NewAudioSource()
	src.Clip, src.PlayOnStart = "sfx/theme.wav", false
	boss.With(an, src)
	w.s.Add("pillar", components.NewCircleCollider()).At(4, 0, 0)

	// Act
	w.run(40)

	// Assert
	sc := boss.Component(Kind).(*Script)
	if got := sc.self.RawGetString("speed").String(); got != "2" {
		t.Fatalf("animator param %s logs %v", got, w.log.lines)
	}
	if len(audio.played) != 2 || audio.played[0] != "sfx/theme.wav@0.5" {
		t.Fatalf("audio %v logs %v", audio.played, w.log.lines)
	}
	if boss.Local.Pos.X < 1 || sc.self.RawGetString("hit").String() != "pillar" {
		t.Fatalf("x %.2f hit %s logs %v", boss.Local.Pos.X, sc.self.RawGetString("hit"), w.log.lines)
	}
}
