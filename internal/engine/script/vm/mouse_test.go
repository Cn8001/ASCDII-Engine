package vm

import (
	"strings"
	"testing"

	"github.com/Cn8001/ASCDII-Engine/internal/engine/core/input"
	"github.com/Cn8001/ASCDII-Engine/internal/engine/world/components"
)

func TestUnit_MouseReportsCellButtonsAndGroundPoint(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/m.lua": `
function update(self, dt)
  local m = mouse()
  if m then log(m.x, m.y, m.left, m.clicked_left, m.wheel, m.world ~= nil) end
end`})
	w.s.Add("Main Camera", components.NewCamera())
	w.s.Add("m", w.script(t, "scripts/m.lua"))
	w.c.Input = input.NewState()
	w.c.Input.SetScreen(80, 24)
	w.c.Input.Apply(input.Event{Mouse: true, X: 40, Y: 12, Button: input.ButtonLeft, Down: true}, 0)

	// Act
	w.run(1)

	// Assert
	if len(w.log.lines) == 0 || !strings.HasSuffix(w.log.lines[len(w.log.lines)-1], " 40 12 true true 0 true") {
		t.Fatalf("logs %v", w.log.lines)
	}
}

func TestUnit_MouseIsNilBeforeAnyReport(t *testing.T) {
	// Arrange
	w := newWorld(t, map[string]string{"scripts/m.lua": `
function update(self, dt) log(mouse() == nil) end`})
	w.s.Add("m", w.script(t, "scripts/m.lua"))
	w.c.Input = input.NewState()

	// Act
	w.run(1)

	// Assert
	if len(w.log.lines) == 0 || !strings.HasSuffix(w.log.lines[0], " true") {
		t.Fatalf("logs %v", w.log.lines)
	}
}
