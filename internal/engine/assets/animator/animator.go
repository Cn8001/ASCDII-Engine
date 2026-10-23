// Package animator is the animator controller asset, a .animator.json file:
// parameters, states that each play a clip, and transitions between them
// with conditions on the parameters.
package animator

import (
	"encoding/json"
	"fmt"
	"io"
)

const Ext = ".animator.json"

// Any as a transition's From lets it leave every state.
const Any = "any"

type ParamType string

const (
	Float   ParamType = "float"
	Bool    ParamType = "bool"
	Trigger ParamType = "trigger" // true for one transition, then reset
)

type Param struct {
	Name  string    `json:"name"`
	Type  ParamType `json:"type"`
	Value float64   `json:"value,omitempty"` // bools and triggers use 0 and 1
}

type State struct {
	Name  string  `json:"name"`
	Clip  string  `json:"clip"` // a clip of the scene, by name, on the animated object
	Speed float64 `json:"speed,omitempty"`
	X     float64 `json:"x,omitempty"` // position in the animator window
	Y     float64 `json:"y,omitempty"`
}

type Cond struct {
	Param string  `json:"param"`
	Op    string  `json:"op,omitempty"` // >, <, ==, !=; empty means the bool or trigger is set
	Value float64 `json:"value,omitempty"`
}

type Transition struct {
	From string `json:"from"`
	To   string `json:"to"`
	When []Cond `json:"when,omitempty"`
	Exit bool   `json:"exit,omitempty"` // also wait for the state's clip to finish
}

type Controller struct {
	Parameters  []Param      `json:"parameters"`
	States      []State      `json:"states"`
	Start       string       `json:"start"`
	Transitions []Transition `json:"transitions"`
}

func Default() *Controller {
	return &Controller{Parameters: []Param{}, States: []State{{Name: "idle", Clip: "idle", Speed: 1}}, Start: "idle", Transitions: []Transition{}}
}

func Decode(r io.Reader) (*Controller, error) {
	c := Default()
	if err := json.NewDecoder(r).Decode(c); err != nil {
		return nil, fmt.Errorf("decode animator: %w", err)
	}
	return c, nil
}

func Encode(w io.Writer, c *Controller) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("encode animator: %w", err)
	}
	return nil
}

func (c *Controller) State(name string) (State, bool) {
	for _, s := range c.States {
		if s.Name == name {
			return s, true
		}
	}
	return State{}, false
}

// Next picks the first transition out of state whose conditions hold for
// params; finished says whether the state's clip has played to its end.
// It returns the target state and the triggers the transition consumed.
func (c *Controller) Next(state string, params map[string]float64, finished bool) (string, []string, bool) {
	for _, t := range c.Transitions {
		if (t.From != state && t.From != Any) || t.To == state {
			continue
		}
		if t.Exit && !finished {
			continue
		}
		if !t.Exit && len(t.When) == 0 {
			continue // a transition with no condition and no exit would fire every frame
		}
		ok, used := true, []string(nil)
		for _, w := range t.When {
			if !w.holds(params[w.Param]) {
				ok = false
				break
			}
			if c.typeOf(w.Param) == Trigger {
				used = append(used, w.Param)
			}
		}
		if ok {
			return t.To, used, true
		}
	}
	return state, nil, false
}

func (c *Controller) typeOf(name string) ParamType {
	for _, p := range c.Parameters {
		if p.Name == name {
			return p.Type
		}
	}
	return Float
}

func (w Cond) holds(v float64) bool {
	switch w.Op {
	case ">":
		return v > w.Value
	case "<":
		return v < w.Value
	case "==":
		return v == w.Value
	case "!=":
		return v != w.Value
	}
	return v != 0
}
