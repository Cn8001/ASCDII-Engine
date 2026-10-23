package scene

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

// Prefabs resolves a prefab name to a fresh entity tree.
type Prefabs interface {
	Prefab(name string) (*Entity, error)
}

type Scene struct {
	Name    string
	Root    *Entity
	Camera  view.Camera
	Env     light.Env
	Clips   []anim.Clip
	Time    float64
	next    atomic.Uint64
	prefabs Prefabs
}

func New(name string) *Scene {
	s := &Scene{Name: name, Camera: view.Default(), Env: light.DefaultEnv()}
	s.Root = &Entity{Name: "root", Visible: true, scene: s}
	return s
}

func (s *Scene) UsePrefabs(p Prefabs) { s.prefabs = p }

func (s *Scene) newEntity(name string, n Node) *Entity {
	return &Entity{ID: s.next.Add(1), Name: name, Visible: true, Node: n, scene: s, Local: Transform{Scale: 1}}
}

func (s *Scene) Add(name string, n Node) *Entity { return s.Root.Add(name, n) }

// Walk visits entities depth first; returning false skips the children.
func (s *Scene) Walk(fn func(e *Entity, visible bool) bool) {
	var rec func(e *Entity, visible bool)
	rec = func(e *Entity, visible bool) {
		for _, c := range e.Children {
			v := visible && c.Visible
			if fn(c, v) {
				rec(c, v)
			}
		}
	}
	rec(s.Root, true)
}

func (s *Scene) Find(name string) *Entity {
	var out *Entity
	s.Walk(func(e *Entity, _ bool) bool {
		if out == nil && e.Name == name {
			out = e
		}
		return out == nil
	})
	return out
}

func (s *Scene) ByID(id uint64) (*Entity, error) {
	var out *Entity
	s.Walk(func(e *Entity, _ bool) bool {
		if e.ID == id {
			out = e
		}
		return out == nil
	})
	if out == nil {
		return nil, fmt.Errorf("%w: %d", ErrNodeNotFound, id)
	}
	return out, nil
}

func (s *Scene) Remove(id uint64) error {
	e, err := s.ByID(id)
	if err != nil {
		return err
	}
	if e.inherited {
		return fmt.Errorf("remove %s: %w", e.Name, ErrPrefabLocked)
	}
	e.detach()
	return nil
}

// Reparent moves an entity under parent (0 for root) at index, refusing cycles.
func (s *Scene) Reparent(id, parent uint64, index int) error {
	e, err := s.ByID(id)
	if err != nil {
		return err
	}
	p := s.Root
	if parent != 0 {
		if p, err = s.ByID(parent); err != nil {
			return err
		}
	}
	if e.inherited || p.inherited || p.Prefab != "" {
		return fmt.Errorf("reparent %s: %w", e.Name, ErrPrefabLocked)
	}
	for a := p; a != nil; a = a.parent {
		if a == e {
			return fmt.Errorf("reparent %s under %s: %w", e.Name, p.Name, ErrInvalidParent)
		}
	}
	e.detach()
	p.attach(e, index)
	return nil
}

// Animate binds a clip to an entity by name and returns the scene for chaining.
func (s *Scene) Animate(target string, c anim.Clip) *Scene {
	c.Target = target
	s.Clips = append(s.Clips, c)
	return s
}

// Blocked is the single collision query; it reads the same bounds used for drawing.
func (s *Scene) Blocked(p geom.Vec2, ignore *Entity) bool {
	hit := false
	s.Walk(func(e *Entity, visible bool) bool {
		if hit || !visible {
			return false
		}
		if e == ignore {
			return false
		}
		if e.Solid && e.Node != nil {
			if b, ok := e.Node.(Blocker); ok {
				hit = b.Blocks(e.World(), p)
			} else {
				hit = e.Bounds().Contains2D(p)
			}
		}
		return !hit
	})
	return hit
}

// Update advances time, routes keys, runs behaviours and node tickers, then
// samples clips so animation always wins over logic for the same property.
func (s *Scene) Update(c *Ctx, dt float64, keys []input.Key) error {
	s.Time += dt
	var all []*Entity
	s.Walk(func(e *Entity, _ bool) bool { all = append(all, e); return true })
	for _, k := range keys {
		for _, e := range all {
			for _, b := range e.Behaviours {
				if !e.Enabled(b.Kind()) {
					continue
				}
				if h, ok := b.(KeyHandler); ok && h.Key(c, e, k) {
					break
				}
			}
		}
	}
	for _, e := range all {
		for _, b := range e.Behaviours {
			if e.Enabled(b.Kind()) {
				b.Update(c, e, dt)
			}
		}
		if t, ok := e.Node.(Ticker); ok {
			t.Tick(c, e, dt)
		}
	}
	return s.ApplyClips()
}

func (s *Scene) ApplyClips() error {
	var errs []error
	for _, c := range s.Clips {
		e := s.Find(c.Target)
		if e == nil {
			continue
		}
		if err := c.Apply(s.Time, e); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Instantiate fills prefab instances from their source, replacing any
// previously inherited content so edits to a prefab propagate.
func (s *Scene) Instantiate() error {
	if s.prefabs == nil {
		return nil
	}
	var errs []error
	var instances []*Entity
	s.Walk(func(e *Entity, _ bool) bool {
		if e.Prefab != "" && !e.inherited {
			instances = append(instances, e)
			return false
		}
		return true
	})
	for _, e := range instances {
		src, err := s.prefabs.Prefab(e.Prefab)
		if err != nil {
			errs = append(errs, fmt.Errorf("instance %s: %w", e.Name, err))
			continue
		}
		e.Node, e.Behaviours, e.Children, e.off = src.Node, src.Behaviours, nil, src.off
		if e.Name == "" {
			e.Name = src.Name
		}
		for _, c := range src.Children {
			s.adopt(c, true)
			e.attach(c, -1)
		}
		if err := applyOverrides(e); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// adopt gives a foreign subtree fresh ids from this scene.
func (s *Scene) adopt(e *Entity, inherited bool) {
	e.ID, e.scene, e.inherited = s.next.Add(1), s, inherited
	for _, c := range e.Children {
		c.parent = e
		s.adopt(c, inherited)
	}
}

// Unpack turns a prefab instance into regular, editable entities.
func (s *Scene) Unpack(id uint64) error {
	e, err := s.ByID(id)
	if err != nil {
		return err
	}
	if e.Prefab == "" {
		return nil
	}
	e.Prefab, e.Overrides = "", nil
	var rec func(*Entity)
	rec = func(x *Entity) {
		x.inherited = false
		for _, c := range x.Children {
			rec(c)
		}
	}
	rec(e)
	return nil
}

// Graft attaches a detached tree, such as a decoded prefab copy, under parent
// with fresh ids and resolves any prefab instances inside it.
func (s *Scene) Graft(parent *Entity, e *Entity, index int) (*Entity, error) {
	if parent == nil {
		parent = s.Root
	}
	if parent.inherited || parent.Prefab != "" {
		return nil, fmt.Errorf("graft %s: %w", e.Name, ErrPrefabLocked)
	}
	s.adopt(e, false)
	parent.attach(e, index)
	return e, s.Instantiate()
}

// Preview advances time for editor playback: clips and node tickers run, but
// behaviours and input do not, so nothing in the edited scene is moved by logic.
func (s *Scene) Preview(c *Ctx, dt float64) error {
	s.Time += dt
	s.Walk(func(e *Entity, _ bool) bool {
		if t, ok := e.Node.(Ticker); ok {
			t.Tick(c, e, dt)
		}
		return true
	})
	return s.ApplyClips()
}

// TweenTo animates prop of target from its current value to "to", starting
// now. A newer tween on the same property replaces the old one.
func (s *Scene) TweenTo(target *Entity, name string, to any, seconds float64, e anim.Ease) error {
	p, ok := prop.Get(target, name)
	if !ok {
		return fmt.Errorf("tween %s: %w: %s", target.Name, prop.ErrUnknownProp, name)
	}
	id := fmt.Sprintf("tween:%d:%s", target.ID, name)
	c := anim.Tween(name, prop.Encode(p), to, math.Max(seconds, 1e-3), e).Delay(s.Time)
	c.Name, c.Target, c.Transient = id, target.Name, true
	for i := range s.Clips {
		if s.Clips[i].Name == id {
			s.Clips[i] = c
			return nil
		}
	}
	s.Clips = append(s.Clips, c)
	return nil
}

// Spawn instantiates a prefab at a world position under the root.
func (s *Scene) Spawn(prefab string, pos geom.Vec3) (*Entity, error) {
	e := NewEntity("", nil)
	e.Prefab, e.Local.Pos = prefab, pos
	return s.Graft(nil, e, -1)
}
