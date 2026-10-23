package components

import (
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Animator plays its object's clips as the states of a controller asset and
// moves between them when the transition conditions hold.
type Animator struct {
	Controller string
	Speed      float64
	State      string // the active state; setting it jumps there
	lib        *asset.Library
	params     map[string]float64
	started    float64 // scene time the state began
	running    bool
	jump       string
}

func NewAnimator(lib *asset.Library) *Animator { return &Animator{Speed: 1, lib: lib} }

func (*Animator) Kind() string { return "animator" }
func (a *Animator) bind() *prop.Binder {
	return prop.Bind().Controller("controller", &a.Controller).Float("speed", &a.Speed, 0, 10).String("state", &a.State)
}
func (a *Animator) Props() []prop.Prop { return a.bind().Props() }
func (a *Animator) Set(n string, v any) error {
	before := a.State
	if err := a.bind().Set(n, v); err != nil {
		return err
	}
	if n == "state" && a.running && a.State != before {
		a.jump, a.State = a.State, before
	}
	return nil
}

// DrivesClips tells the scene to leave this object's clips to the animator
// while it runs, and to the timeline otherwise.
func (a *Animator) DrivesClips() bool { return a.running }

func (a *Animator) controller() *animator.Controller {
	if a.lib == nil || a.Controller == "" {
		return nil
	}
	c, err := a.lib.Controller(a.Controller)
	if err != nil {
		return nil
	}
	return c
}

func clipOf(s *scene.Scene, o *scene.GameObject, name string) *anim.Clip {
	for i := range s.Clips {
		c := &s.Clips[i]
		if c.Name == name && (c.Target == o.Name || c.Target == "") {
			return c
		}
	}
	return nil
}

func (a *Animator) enter(c *scene.Ctx, state string) {
	a.State, a.started = state, c.Scene.Time
}

func (a *Animator) Tick(c *scene.Ctx, o *scene.GameObject, _ float64) {
	ctrl := a.controller()
	if ctrl == nil {
		a.running = false
		return
	}
	if !a.running {
		a.params = map[string]float64{}
		for _, p := range ctrl.Parameters {
			a.params[p.Name] = p.Value
		}
		a.running = true
		a.enter(c, ctrl.Start)
	}
	if a.jump != "" {
		a.enter(c, a.jump)
		a.jump = ""
	}
	st, _ := ctrl.State(a.State)
	speed := a.Speed * st.Speed
	if st.Speed == 0 {
		speed = a.Speed
	}
	clip := clipOf(c.Scene, o, st.Clip)
	finished := clip == nil || (c.Scene.Time-a.started)*speed >= clip.Length()
	if next, used, ok := ctrl.Next(a.State, a.params, finished); ok {
		for _, u := range used {
			a.params[u] = 0
		}
		a.enter(c, next)
		st, _ = ctrl.State(next)
		clip = clipOf(c.Scene, o, st.Clip)
	}
	if clip == nil {
		return
	}
	cp := *clip
	cp.Offset, cp.Speed = a.started, speed
	_ = cp.Apply(c.Scene.Time, o) // a property the clip names but the object lacks is skipped
}

func (*Animator) Methods() []string { return []string{"play", "set", "get", "trigger"} }

func (a *Animator) Call(_ *scene.Ctx, _ *scene.GameObject, name string, args []any) (any, error) {
	str := func(i int) (string, error) {
		if i >= len(args) {
			return "", fmt.Errorf("argument %d missing", i+1)
		}
		return prop.ToString(args[i])
	}
	if a.params == nil {
		a.params = map[string]float64{}
	}
	key, err := str(0)
	if err != nil {
		return nil, err
	}
	switch name {
	case "play":
		if ctrl := a.controller(); ctrl != nil {
			if _, ok := ctrl.State(key); !ok {
				return nil, fmt.Errorf("no state %q", key)
			}
		}
		a.jump = key
	case "trigger":
		a.params[key] = 1
	case "set":
		if len(args) < 2 {
			return nil, fmt.Errorf("set %s needs a value", key)
		}
		if b, ok := args[1].(bool); ok {
			a.params[key] = 0
			if b {
				a.params[key] = 1
			}
			return nil, nil
		}
		v, err := prop.ToFloat(args[1])
		if err != nil {
			return nil, err
		}
		a.params[key] = v
	case "get":
		if ctrl := a.controller(); ctrl != nil {
			for _, p := range ctrl.Parameters {
				if p.Name == key && p.Type != animator.Float {
					return a.params[key] != 0, nil
				}
			}
		}
		return a.params[key], nil
	}
	return nil, nil
}
