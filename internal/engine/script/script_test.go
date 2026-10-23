package script

import (
	"math"
	"strings"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/kit"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

type logs struct{ lines []string }

func (l *logs) add(level, source, msg string) { l.lines = append(l.lines, level+" "+source+" "+msg) }

func setup(t *testing.T, src string) (*scene.Scene, *scene.Entity, *asset.Library, *logs, *scene.Ctx) {
	t.Helper()
	reg := scene.NewRegistry()
	lib := asset.NewLibrary(reg)
	lib.PutScript("scripts/door.lua", src)
	s := scene.New("t")
	door := s.Add("door", kit.NewBox(geom.V3(2, 1, 3), color.RGB(120, 90, 60))).With(New(lib, "scripts/door.lua"))
	l := &logs{}
	return s, door, lib, l, &scene.Ctx{Scene: s, Log: l.add}
}

const door = `
function start(self)
  self.open = false
  self.ticks = 0
end

function key(self, k)
  if k == "e" then
    self.open = not self.open
    tween(self, "pos.z", self.open and 4 or 0, 1, "linear")
    return true
  end
end

function update(self, dt)
  self.ticks = self.ticks + 1
  self:set("visible", true)
end
`

func TestUnit_KeyHookTweensProperty(t *testing.T) {
	// Arrange
	s, d, _, l, c := setup(t, door)

	// Act
	_ = s.Update(c, 0, []input.Key{'e'})
	for i := 0; i < 30; i++ {
		_ = s.Update(c, 1.0/60, nil)
	}

	// Assert
	if z := d.Local.Pos.Z; math.Abs(z-2) > .1 {
		t.Fatalf("z %f logs %v", z, l.lines)
	}
}

func TestUnit_ErrorsAreLoggedWithLineAndStopScript(t *testing.T) {
	// Arrange
	s, _, _, l, c := setup(t, "function update(self, dt)\n  local x = nil + 1\nend\n")

	// Act
	_ = s.Update(c, .1, nil)
	_ = s.Update(c, .1, nil)

	// Assert
	if len(l.lines) != 1 || !strings.Contains(l.lines[0], "door.lua:2") || !strings.HasPrefix(l.lines[0], "error") {
		t.Fatalf("logs %v", l.lines)
	}
}

func TestUnit_SourceChangeReloads(t *testing.T) {
	// Arrange
	s, d, lib, l, c := setup(t, "function update(self) self:moveTo(1, 1) end")
	_ = s.Update(c, .1, nil)

	// Act
	lib.PutScript("scripts/door.lua", "function update(self) self:moveTo(5, 5) end")
	_ = s.Update(c, .1, nil)

	// Assert
	if d.Local.Pos.X != 5 || len(l.lines) != 1 || !strings.Contains(l.lines[0], "reloaded") {
		t.Fatalf("pos %+v logs %v", d.Local.Pos, l.lines)
	}
}

func TestUnit_SandboxHasNoFileAccess(t *testing.T) {
	// Arrange
	s, _, _, l, c := setup(t, "function start(self) dofile('/etc/passwd') end")

	// Act
	_ = s.Update(c, .1, nil)

	// Assert
	if len(l.lines) != 1 || !strings.Contains(l.lines[0], "non-function") {
		t.Fatalf("logs %v", l.lines)
	}
}

func TestUnit_FindDistAndSay(t *testing.T) {
	// Arrange
	s, d, _, l, c := setup(t, `
function update(self)
  local hero = find("hero")
  if hero and dist(self, hero) < 3 then say("press E", 1) end
end`)
	s.Add("hero", kit.NewBlob(.5, color.RGB(1, 1, 1))).At(1, 0, 0)

	// Act
	_ = s.Update(c, .1, nil)
	_ = s.Update(c, .1, nil)
	shown := len(d.Children)
	s.Find("hero").Local.Pos.X = 50
	for i := 0; i < 20; i++ {
		_ = s.Update(c, .1, nil)
	}

	// Assert
	if shown != 1 || len(d.Children) != 0 || len(l.lines) != 0 {
		t.Fatalf("shown %d after %d logs %v", shown, len(d.Children), l.lines)
	}
}
