package components

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func keyIs(k input.Key, name string) bool {
	want, err := input.ParseName(name)
	return err == nil && want == k
}

// Player walks, jumps and slides along obstacles. Terminals send no key up
// events, so a press holds the direction briefly and key repeat extends it.
type Player struct {
	Speed      float64
	JumpSpeed  float64
	Gravity    float64
	Radius     float64
	Follow     bool
	FollowRate float64
	Up         string
	Down       string
	Left       string
	Right      string
	Jump       string
	held       [4]float64
	vz         float64
	moving     bool
}

func NewPlayer() *Player {
	return &Player{Speed: 7, JumpSpeed: 9, Gravity: 26, Radius: .6, Follow: true, FollowRate: 5,
		Up: "w", Down: "s", Left: "a", Right: "d", Jump: "space"}
}

func (*Player) Kind() string { return "player" }
func (p *Player) bind() *prop.Binder {
	return prop.Bind().Float("speed", &p.Speed, 0, 80).Float("jumpSpeed", &p.JumpSpeed, 0, 80).
		Float("gravity", &p.Gravity, 0, 200).Float("radius", &p.Radius, 0, 10).
		Bool("follow", &p.Follow).Float("followRate", &p.FollowRate, .1, 30).
		String("up", &p.Up).String("down", &p.Down).String("left", &p.Left).String("right", &p.Right).String("jump", &p.Jump)
}
func (p *Player) Props() []prop.Prop        { return p.bind().Props() }
func (p *Player) Set(n string, v any) error { return p.bind().Set(n, v) }

// Moving reports whether the player walked on the last update.
func (p *Player) Moving() bool { return p.moving }

func (p *Player) Key(c *scene.Ctx, e *scene.GameObject, k input.Key) bool {
	dir := -1
	switch {
	case keyIs(k, p.Up) || k == input.Up:
		dir = 0
	case keyIs(k, p.Down) || k == input.Down:
		dir = 1
	case keyIs(k, p.Left) || k == input.Left:
		dir = 2
	case keyIs(k, p.Right) || k == input.Right:
		dir = 3
	case keyIs(k, p.Jump):
		if e.Local.Pos.Z <= 0 && p.vz == 0 {
			p.vz = p.JumpSpeed
		}
		return true
	default:
		return false
	}
	now := c.Scene.Time
	hold := .55 // covers the terminal's initial repeat delay
	if p.held[dir] > now {
		hold = .13
	}
	p.held[dir] = now + hold
	if dir < 2 {
		p.held[1-dir] = 0
	} else {
		p.held[5-dir] = 0
	}
	return true
}

func (p *Player) Update(c *scene.Ctx, e *scene.GameObject, dt float64) {
	now := c.Scene.Time
	held := func(i int) bool { return p.held[i] > now }
	// With releases known every held key counts, so diagonals work; without
	// them only the timers of the latest presses do.
	if in := c.Input; in != nil && in.Exact() {
		keys := [4]struct {
			name  string
			arrow input.Key
		}{{p.Up, input.Up}, {p.Down, input.Down}, {p.Left, input.Left}, {p.Right, input.Right}}
		held = func(i int) bool {
			k, err := input.ParseName(keys[i].name)
			return in.Down(keys[i].arrow, now) || err == nil && in.Down(k, now)
		}
	}
	var d geom.Vec2
	if held(0) {
		d.Y--
	}
	if held(1) {
		d.Y++
	}
	if held(2) {
		d.X--
	}
	if held(3) {
		d.X++
	}
	p.moving = d != (geom.Vec2{})
	if p.moving {
		step := d.Norm().Scale(p.Speed * dt)
		pos := e.Local.Pos
		if nx := geom.V2(pos.X+step.X, pos.Y); !p.blocked(c.Scene, e, nx) {
			pos.X = nx.X
		}
		if ny := geom.V2(pos.X, pos.Y+step.Y); !p.blocked(c.Scene, e, ny) {
			pos.Y = ny.Y
		}
		e.Local.Pos = pos
		if d.X != 0 {
			e.Local.Flip = d.X < 0
		}
	}
	if e.Local.Pos.Z > 0 || p.vz != 0 {
		p.vz -= p.Gravity * dt
		e.Local.Pos.Z += p.vz * dt
		if e.Local.Pos.Z <= 0 && p.vz < 0 {
			e.Local.Pos.Z, p.vz = 0, 0
		}
	}
	if name := e.Resolve("walking"); name != "walking" {
		_ = e.Set(name, p.moving)
	}
	if p.Follow {
		c.Scene.FollowView(e, geom.V3(e.Local.Pos.X, e.Local.Pos.Y, 0), dt, p.FollowRate)
	}
}

func (p *Player) blocked(s *scene.Scene, self *scene.GameObject, at geom.Vec2) bool {
	r := p.Radius
	for _, o := range [5]geom.Vec2{{}, {X: r}, {X: -r}, {Y: r * .5}, {Y: -r * .5}} {
		if s.Blocked(at.Add(o), self) {
			return true
		}
	}
	return false
}

// Toggle flips a bool property on a key press, the generic switch for doors,
// lamps and creatures.
type Toggle struct {
	Press  string
	Target string // object reference, empty for the owner
	Prop   string
}

func NewToggle(key, target, p string) *Toggle { return &Toggle{Press: key, Target: target, Prop: p} }

func (*Toggle) Kind() string { return "toggle" }
func (t *Toggle) bind() *prop.Binder {
	return prop.Bind().String("key", &t.Press).Object("target", &t.Target).String("prop", &t.Prop)
}
func (t *Toggle) Props() []prop.Prop        { return t.bind().Props() }
func (t *Toggle) Set(n string, v any) error { return t.bind().Set(n, v) }

func (t *Toggle) Key(c *scene.Ctx, e *scene.GameObject, k input.Key) bool {
	if !keyIs(k, t.Press) {
		return false
	}
	target := e
	if t.Target != "" {
		if target = c.Scene.Lookup(e, t.Target); target == nil {
			return false
		}
	}
	p, ok := prop.Get(target, t.Prop)
	if !ok {
		return false
	}
	on, err := prop.ToBool(p.Value)
	if err != nil {
		return false
	}
	return target.Set(t.Prop, !on) == nil
}

// DayNight runs the clock forward to the other phase instead of jumping, so
// switching to night plays a sunset.
type DayNight struct {
	Press  string
	Day    float64
	Night  float64
	Speed  float64 // hours per second during a transition
	target float64
	moving bool
}

func NewDayNight() *DayNight { return &DayNight{Press: "t", Day: 16.5, Night: 0.5, Speed: 7} }

func (*DayNight) Kind() string { return "daynight" }
func (d *DayNight) bind() *prop.Binder {
	return prop.Bind().String("key", &d.Press).Float("day", &d.Day, 0, 24).Float("night", &d.Night, 0, 24).Float("speed", &d.Speed, .1, 48)
}
func (d *DayNight) Props() []prop.Prop        { return d.bind().Props() }
func (d *DayNight) Set(n string, v any) error { return d.bind().Set(n, v) }

func (d *DayNight) Key(c *scene.Ctx, _ *scene.GameObject, k input.Key) bool {
	if !keyIs(k, d.Press) {
		return false
	}
	h := c.Scene.Env.Hour
	next := d.Night
	if near(h, d.Night) || (d.moving && near(d.target, d.Night)) {
		next = d.Day
	}
	d.target = h + math.Mod(next-h+48, 24)
	d.moving = true
	c.Scene.Env.Auto = true
	return true
}

func near(a, b float64) bool { return math.Abs(math.Mod(a-b+36, 24)-12) > 11.5 }

func (d *DayNight) Update(c *scene.Ctx, _ *scene.GameObject, dt float64) {
	if !d.moving {
		return
	}
	h := c.Scene.Env.Hour
	if h >= d.target {
		c.Scene.Env.Hour = math.Mod(d.target, 24)
		d.moving = false
		return
	}
	c.Scene.Env.Hour = math.Min(d.target, h+d.Speed*dt)
}

// IsNight is for HUD text and content that reacts to the clock.
func IsNight(hour float64) bool {
	h := math.Mod(hour+24, 24)
	return h < 5.5 || h > 19.5
}

type Quit struct{ Press string }

func (*Quit) Kind() string                { return "quit" }
func (q *Quit) Props() []prop.Prop        { return prop.Bind().String("key", &q.Press).Props() }
func (q *Quit) Set(n string, v any) error { return prop.Bind().String("key", &q.Press).Set(n, v) }
func (q *Quit) Key(c *scene.Ctx, _ *scene.GameObject, k input.Key) bool {
	if keyIs(k, q.Press) {
		c.Quit = true
		return true
	}
	return false
}
