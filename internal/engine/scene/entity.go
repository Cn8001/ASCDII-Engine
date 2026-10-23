// Package scene is the entity tree: transforms, one drawable node per entity,
// behaviours for game logic, prefab instances and animation clips.
package scene

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
)

var (
	ErrNodeNotFound  = errors.New("entity not found")
	ErrUnknownKind   = errors.New("unknown kind")
	ErrInvalidParent = errors.New("invalid parent")
	ErrPrefabLocked  = errors.New("entity belongs to a prefab instance")
)

type Transform struct {
	Pos   geom.Vec3
	Scale float64
	Flip  bool
}

func (t Transform) scale() float64 {
	if t.Scale == 0 {
		return 1
	}
	return t.Scale
}

// Then places child local transform c inside parent t.
func (t Transform) Then(c Transform) Transform {
	local := c.Pos.Scale(t.scale())
	if t.Flip {
		local.X = -local.X
	}
	return Transform{Pos: t.Pos.Add(local), Scale: t.scale() * c.scale(), Flip: t.Flip != c.Flip}
}

type Node interface {
	prop.Editable
	Kind() string
	Draw(c *draw.Context, t Transform)
	Bounds(t Transform) geom.AABB
}

// Blocker gives a node a collision shape finer than its bounds.
type Blocker interface {
	Blocks(t Transform, p geom.Vec2) bool
}

// CellDrawer writes whole cells after Braille resolve, for text and glyph art.
type CellDrawer interface {
	DrawCells(f *pipeline.Frame, t Transform)
}

// Ticker is for nodes with their own simulation, such as grass springs.
type Ticker interface {
	Tick(c *Ctx, e *Entity, dt float64)
}

type Behaviour interface {
	prop.Editable
	Kind() string
	Update(c *Ctx, e *Entity, dt float64)
}

type KeyHandler interface {
	Key(c *Ctx, e *Entity, k input.Key) bool
}

type Ctx struct {
	Scene *Scene
	Quit  bool
	Log   func(level, source, msg string) // nil drops messages
}

func (c *Ctx) Logf(level, source, format string, args ...any) {
	if c.Log != nil {
		c.Log(level, source, fmt.Sprintf(format, args...))
	}
}

type Entity struct {
	ID         uint64
	Name       string
	Visible    bool
	Solid      bool
	Local      Transform
	Node       Node
	Behaviours []Behaviour
	Children   []*Entity
	Prefab     string // asset reference; node, behaviours and children come from the prefab
	Overrides  []Override
	off        map[string]bool
	parent     *Entity
	scene      *Scene
	inherited  bool
}

func (e *Entity) Parent() *Entity { return e.parent }

// Inherited reports entities created by a prefab instance; they are not saved.
func (e *Entity) Inherited() bool { return e.inherited }

func (e *Entity) World() Transform {
	if e.parent == nil {
		return e.Local
	}
	return e.parent.World().Then(e.Local)
}

func (e *Entity) Bounds() geom.AABB {
	if e.Node == nil {
		return geom.AABB{}
	}
	return e.Node.Bounds(e.World())
}

func (e *Entity) At(x, y, z float64) *Entity { e.Local.Pos = geom.V3(x, y, z); return e }
func (e *Entity) Scaled(k float64) *Entity   { e.Local.Scale = k; return e }
func (e *Entity) Blocking() *Entity          { e.Solid = true; return e }

func (e *Entity) With(b ...Behaviour) *Entity {
	e.Behaviours = append(e.Behaviours, b...)
	return e
}

// Add attaches a child; it panics only if e was never attached to a scene,
// which is a programming error in content code.
func (e *Entity) Add(name string, n Node) *Entity {
	if e.scene == nil {
		panic("scene: Add on a detached entity")
	}
	c := e.scene.newEntity(name, n)
	e.attach(c, -1)
	return c
}

func (e *Entity) attach(c *Entity, index int) {
	c.parent = e
	if index < 0 || index > len(e.Children) {
		e.Children = append(e.Children, c)
		return
	}
	e.Children = append(e.Children[:index], append([]*Entity{c}, e.Children[index:]...)...)
}

func (e *Entity) detach() {
	p := e.parent
	if p == nil {
		return
	}
	for i, c := range p.Children {
		if c == e {
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			break
		}
	}
	e.parent = nil
}

// Enabled reports whether a behaviour runs; disabled ones keep their settings.
func (e *Entity) Enabled(kind string) bool { return !e.off[kind] }

func (e *Entity) SetEnabled(kind string, on bool) {
	if e.off == nil {
		e.off = map[string]bool{}
	}
	if on {
		delete(e.off, kind)
	} else {
		e.off[kind] = true
	}
}

func (e *Entity) Behaviour(kind string) Behaviour {
	for _, b := range e.Behaviours {
		if b.Kind() == kind {
			return b
		}
	}
	return nil
}

func (e *Entity) bind() *prop.Binder {
	return prop.Bind().
		String("name", &e.Name).
		Bool("visible", &e.Visible).
		Bool("solid", &e.Solid).
		Vec3("pos", &e.Local.Pos).
		Float("scale", &e.Local.Scale, .01, 100).
		Bool("flip", &e.Local.Flip)
}

// Props lists entity, node and behaviour properties. Behaviour properties are
// prefixed with their kind so names never collide. Inside a prefab instance,
// properties that differ from the prefab are marked overridden.
func (e *Entity) Props() []prop.Prop {
	out := group(e.bind().Props(), "entity")
	if e.Node != nil {
		out = append(out, group(e.Node.Props(), e.Node.Kind())...)
	}
	for _, b := range e.Behaviours {
		on := prop.Prop{Name: b.Kind() + ".enabled", Type: prop.Bool, Value: e.Enabled(b.Kind()), Group: b.Kind()}
		out = append(out, on)
		for _, p := range group(b.Props(), b.Kind()) {
			p.Name = b.Kind() + "." + p.Name
			out = append(out, p)
		}
	}
	root := e.InstanceRoot()
	if root == nil {
		return out
	}
	at := PathFrom(root, e)
	for i := range out {
		if e.inherited && out[i].Name == "name" {
			out[i].Locked = true // child paths are matched by name
		}
		for _, ov := range root.Overrides {
			if ov.Path == at && ov.Prop == out[i].Name {
				out[i].Overridden = true
			}
		}
	}
	return out
}

func group(ps []prop.Prop, g string) []prop.Prop {
	for i := range ps {
		ps[i].Group = g
	}
	return ps
}

func (e *Entity) Set(name string, v any) error {
	b := e.bind()
	if b.Has(name) {
		return b.Set(name, v)
	}
	if kind, rest, ok := strings.Cut(name, "."); ok {
		if beh := e.Behaviour(kind); beh != nil {
			if rest == "enabled" {
				on, err := prop.ToBool(v)
				if err != nil {
					return fmt.Errorf("set %s: %w", name, err)
				}
				e.SetEnabled(kind, on)
				return nil
			}
			return beh.Set(rest, v)
		}
	}
	if e.Node != nil {
		return e.Node.Set(name, v)
	}
	return fmt.Errorf("%w: %s", prop.ErrUnknownProp, name)
}

// NewEntity makes a detached entity for Graft.
func NewEntity(name string, n Node) *Entity {
	return &Entity{Name: name, Visible: true, Node: n, Local: Transform{Scale: 1}}
}
