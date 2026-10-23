package animator

import (
	"bytes"
	"testing"
)

func controller() *Controller {
	return &Controller{
		Parameters: []Param{{Name: "speed", Type: Float}, {Name: "jump", Type: Trigger}},
		States:     []State{{Name: "idle", Clip: "idle"}, {Name: "run", Clip: "run"}, {Name: "jump", Clip: "jump"}},
		Start:      "idle",
		Transitions: []Transition{
			{From: "idle", To: "run", When: []Cond{{Param: "speed", Op: ">", Value: .1}}},
			{From: "run", To: "idle", When: []Cond{{Param: "speed", Op: "<", Value: .1}}},
			{From: Any, To: "jump", When: []Cond{{Param: "jump"}}},
			{From: "jump", To: "idle", Exit: true},
		},
	}
}

func TestUnit_TransitionsFollowParameters(t *testing.T) {
	// Arrange
	c := controller()

	// Act
	toRun, _, moved := c.Next("idle", map[string]float64{"speed": 2}, false)
	stay, _, still := c.Next("idle", map[string]float64{"speed": 0}, false)
	toJump, used, _ := c.Next("run", map[string]float64{"speed": 2, "jump": 1}, false)
	wait, _, early := c.Next("jump", nil, false)
	back, _, done := c.Next("jump", nil, true)

	// Assert
	if !moved || toRun != "run" || still || stay != "idle" {
		t.Fatalf("idle: %s %v %s %v", toRun, moved, stay, still)
	}
	if toJump != "jump" || len(used) != 1 || used[0] != "jump" {
		t.Fatalf("trigger: %s %v", toJump, used)
	}
	if early || wait != "jump" || !done || back != "idle" {
		t.Fatalf("exit: %s %v %s %v", wait, early, back, done)
	}
}

func TestUnit_ControllerRoundTrips(t *testing.T) {
	// Arrange
	var b bytes.Buffer

	// Act
	err := Encode(&b, controller())
	c, errDec := Decode(&b)

	// Assert
	if err != nil || errDec != nil || len(c.Transitions) != 4 || c.Transitions[2].From != Any {
		t.Fatalf("errs %v %v controller %+v", err, errDec, c)
	}
}
