package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

var (
	ErrNotEditable  = errors.New("not editable")
	ErrNotInstance  = errors.New("not part of a prefab instance")
	ErrNotReference = errors.New("not a matching reference")
)

type Node struct {
	ID         uint64   `json:"id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"` // the main component, for the tree icon
	Visible    bool     `json:"visible"`
	Prefab     string   `json:"prefab,omitempty"`
	Overrides  int      `json:"overrides,omitempty"`
	Inherited  bool     `json:"inherited,omitempty"`
	Components []string `json:"components,omitempty"`
	Children   []Node   `json:"children,omitempty"`
}

func nodeOf(e *scene.GameObject) Node {
	n := Node{ID: e.ID, Name: e.Name, Visible: e.Visible, Prefab: e.Prefab, Overrides: len(e.Overrides), Inherited: e.Inherited()}
	if m := e.Main(); m != nil {
		n.Kind = m.Kind()
	}
	for _, c := range e.Components {
		n.Components = append(n.Components, c.Kind())
	}
	for _, c := range e.Children {
		n.Children = append(n.Children, nodeOf(c))
	}
	return n
}

func (s *Session) Tree() []Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Node{}
	for _, c := range s.current().Root.Children {
		out = append(out, nodeOf(c))
	}
	return out
}

type Kind struct {
	Kind string `json:"kind"`
	scene.Meta
}

type Kinds struct {
	Components []Kind   `json:"components"`
	Prefabs    []string `json:"prefabs"`
	Models     []string `json:"models"`
}

func (s *Session) Kinds() Kinds {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Kinds{Components: []Kind{}, Prefabs: s.lib.PrefabNames(), Models: s.lib.ModelNames()}
	for _, k := range s.reg.Kinds() {
		out.Components = append(out.Components, Kind{Kind: k, Meta: s.reg.Meta(k)})
	}
	return out
}

func (s *Session) Props(id uint64) ([]prop.Prop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.current().ByID(id)
	if err != nil {
		return nil, err
	}
	return e.Props(), nil
}

// edit1 runs fn on an entity of the edited scene, or of the play copy while
// playing, recording undo only for the former.
func (s *Session) edit1(id uint64, label string, fn func(sc *scene.Scene, e *scene.GameObject) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	e, err := sc.ByID(id)
	if err != nil {
		return err
	}
	if s.play == nil {
		s.checkpoint(label)
	}
	if err := fn(sc, e); err != nil {
		return err
	}
	s.changed()
	return nil
}

func (s *Session) SetProp(id uint64, name string, v any) error {
	return s.edit1(id, fmt.Sprintf("prop:%d:%s", id, name), func(sc *scene.Scene, e *scene.GameObject) error {
		p, ok := prop.Get(e, name)
		if ok && p.Locked {
			return fmt.Errorf("set %s: %w", name, scene.ErrPrefabLocked)
		}
		if str, isStr := v.(string); ok && isStr && p.Type.Ref() {
			v = s.lib.Ref(str)
		}
		return sc.SetProp(e, name, v)
	})
}

// SetRef points a reference property of id at another object, or at one of
// its components: key names it ("light#2"), or "" picks the first one of
// the kind the property wants. target 0 clears the reference.
func (s *Session) SetRef(id uint64, name string, target uint64, key string) error {
	s.mu.Lock()
	sc := s.current()
	o, err := sc.ByID(id)
	var t *scene.GameObject
	if err == nil && target != 0 {
		t, err = sc.ByID(target)
	}
	if err != nil {
		s.mu.Unlock()
		return err
	}
	name = o.Resolve(name)
	p, ok := prop.Get(o, name)
	s.mu.Unlock()
	if !ok || (p.Type != prop.ObjectRef && p.Type != prop.CompRef) {
		return fmt.Errorf("set %s: %w", name, ErrNotReference)
	}
	ref := ""
	switch {
	case t == nil:
	case p.Type == prop.ObjectRef:
		ref = scene.RefOf(o, t)
	default:
		c := t.ComponentByKey(key)
		if c == nil && key == "" && len(p.Options) > 0 {
			c = t.Component(p.Options[0])
		} else if c == nil && key == "" {
			c = t.Main()
		}
		if c == nil || (len(p.Options) > 0 && c.Kind() != p.Options[0]) {
			return fmt.Errorf("set %s: %s has no matching component: %w", name, t.Name, ErrNotReference)
		}
		ref = scene.CompRefOf(o, t, c)
	}
	return s.SetProp(id, name, ref)
}

// Revert drops overrides on an entity inside a prefab instance: one property,
// or every override of the instance when called on its root with "".
func (s *Session) Revert(id uint64, name string) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.GameObject) error { return sc.Revert(e, name) })
}

// Apply writes overrides back into the prefab file, so every instance gets
// them; name "" applies every override of the entity's instance.
func (s *Session) Apply(ctx context.Context, id uint64, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	e, err := s.edit.ByID(id)
	if err != nil {
		return err
	}
	root := e.InstanceRoot()
	if root == nil {
		return fmt.Errorf("apply %s: %w", e.Name, ErrNotInstance)
	}
	rel, raw, err := s.lib.PrefabSource(root.Prefab)
	if err != nil {
		return err
	}
	if locked(rel) {
		return lockedErr(rel)
	}
	src, err := scene.DecodePrefab(bytes.NewReader(raw), s.reg)
	if err != nil {
		return err
	}
	at := scene.PathFrom(root, e)
	var keep []scene.Override
	for _, ov := range root.Overrides {
		if name == "" && e == root || ov.Path == at && (name == "" || ov.Prop == name) {
			if t := scene.FindPath(src, ov.Path); t != nil {
				if err := t.Set(ov.Prop, ov.Value); err != nil {
					return err
				}
			}
			continue
		}
		keep = append(keep, ov)
	}
	var out bytes.Buffer
	if err := scene.EncodePrefab(&out, src); err != nil {
		return err
	}
	s.checkpoint("")
	if err := writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(rel)), out.Bytes()); err != nil {
		return err
	}
	s.lib.PutPrefab(rel, out.Bytes())
	root.Overrides = keep
	s.changed()
	return s.edit.Instantiate()
}

func (s *Session) spawnPoint() geom.Vec3 {
	t := s.camera().Target
	return geom.V3(t.X, t.Y, 0)
}

// object builds what the GameObject menu and drops create: "empty",
// "prefab:<ref>", "sprite:<ref>", "model:<ref>" or a registered component kind.
func (s *Session) object(kind string) (*scene.GameObject, error) {
	base := func(ref string) string {
		p := s.lib.Resolve(ref)
		return strings.TrimSuffix(path.Base(p), path.Ext(p))
	}
	switch {
	case kind == "empty":
		return scene.NewGameObject("GameObject"), nil
	case strings.HasPrefix(kind, "prefab:"):
		return s.current().Instance(s.lib.Ref(strings.TrimPrefix(kind, "prefab:")))
	case strings.HasPrefix(kind, "sprite:"):
		ref := strings.TrimPrefix(kind, "sprite:")
		if _, err := s.lib.Sprite(ref); err != nil {
			return nil, err
		}
		return scene.NewGameObject(base(ref), components.NewSprite(s.lib, s.lib.Ref(ref))), nil
	case strings.HasPrefix(kind, "model:"):
		ref := strings.TrimPrefix(kind, "model:")
		if _, err := s.lib.Model(ref); err != nil {
			return nil, err
		}
		return scene.NewGameObject(base(ref), components.NewMesh(s.lib, s.lib.Ref(ref))), nil
	}
	c, err := s.reg.New(kind)
	if err != nil {
		return nil, err
	}
	return scene.NewGameObject(s.reg.Meta(kind).Label, c), nil
}

// Add creates an object (see object for kinds) under parent.
func (s *Session) Add(parent uint64, kind string) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	p := sc.Root
	if parent != 0 {
		var err error
		if p, err = sc.ByID(parent); err != nil {
			return 0, err
		}
	}
	e, err := s.object(kind)
	if err != nil {
		return 0, err
	}
	if s.play == nil {
		s.checkpoint("")
	}
	if parent == 0 {
		e.Local.Pos = s.spawnPoint()
	}
	got, err := sc.Graft(p, e, -1)
	if err != nil {
		return 0, err
	}
	s.selectOnly(got.ID)
	s.changed()
	return got.ID, nil
}

// components edits the component list of an object the scene owns outright;
// prefab instances only carry property overrides.
func (s *Session) components(id uint64, fn func(e *scene.GameObject) error) error {
	return s.edit1(id, "", func(_ *scene.Scene, e *scene.GameObject) error {
		if e.Prefab != "" || e.Inherited() {
			return fmt.Errorf("edit components of %s: %w", e.Name, scene.ErrPrefabLocked)
		}
		return fn(e)
	})
}

// AddComponent returns the new component's key, the prefix of its properties.
func (s *Session) AddComponent(id uint64, kind string) (string, error) {
	key := ""
	err := s.components(id, func(e *scene.GameObject) error {
		c, err := s.reg.New(kind)
		if err != nil {
			return err
		}
		e.With(c)
		key = e.Key(c)
		return nil
	})
	return key, err
}

func (s *Session) RemoveComponent(id uint64, index int) error {
	return s.components(id, func(e *scene.GameObject) error { return e.RemoveComponent(index) })
}

func (s *Session) MoveComponent(id uint64, from, to int) error {
	return s.components(id, func(e *scene.GameObject) error { return e.MoveComponent(from, to) })
}

// ResetComponent puts a component back to the defaults of its kind.
func (s *Session) ResetComponent(id uint64, index int) error {
	return s.components(id, func(e *scene.GameObject) error {
		if index < 0 || index >= len(e.Components) {
			return fmt.Errorf("reset component %d: %w", index, scene.ErrUnknownKind)
		}
		c, err := s.reg.New(e.Components[index].Kind())
		if err != nil {
			return err
		}
		return e.ReplaceComponent(index, c)
	})
}

func (s *Session) Remove(id uint64) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.GameObject) error {
		if s.sel == id {
			s.selectOnly(0)
		}
		return sc.Remove(e.ID)
	})
}

func (s *Session) Reparent(id, parent uint64, index int) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.GameObject) error { return sc.Reparent(e.ID, parent, index, true) })
}

func (s *Session) Unpack(id uint64) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.GameObject) error { return sc.Unpack(e.ID) })
}

func (s *Session) Duplicate(id uint64) (uint64, error) { return s.duplicate(id, "") }

// duplicate copies id; copies made under the same label merge into one undo
// step.
func (s *Session) duplicate(id uint64, label string) (uint64, error) {
	var out uint64
	err := s.edit1(id, label, func(sc *scene.Scene, e *scene.GameObject) error {
		if e.Inherited() {
			return scene.ErrPrefabLocked
		}
		var b bytes.Buffer
		if err := scene.EncodePrefab(&b, e); err != nil {
			return err
		}
		c, err := scene.DecodePrefab(&b, s.reg)
		if err != nil {
			return err
		}
		c.Name, c.Local, c.Prefab = e.Name+" copy", e.Local, e.Prefab
		c.Overrides = append([]scene.Override(nil), e.Overrides...)
		c.Local.Pos = c.Local.Pos.Add(geom.V3(2, 1, 0))
		if c.Prefab != "" {
			c.Components, c.Children = nil, nil
		}
		old := scene.FIDs(c)
		got, err := sc.Graft(e.Parent(), c, -1)
		if err != nil {
			return err
		}
		scene.Remap(got, old)
		out = got.ID
		s.selectOnly(got.ID)
		return nil
	})
	return out, err
}

// SavePrefab writes the entity as a prefab asset and turns it into an instance.
func (s *Session) SavePrefab(ctx context.Context, id uint64, rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	if !strings.HasSuffix(rel, ".prefab.json") {
		rel += ".prefab.json"
	}
	if locked(rel) {
		return lockedErr(rel)
	}
	e, err := s.edit.ByID(id)
	if err != nil {
		return err
	}
	scene.Localize(e)
	var b bytes.Buffer
	if err := scene.EncodePrefab(&b, e); err != nil {
		return err
	}
	if err := s.writeAsset(ctx, rel, b.Bytes()); err != nil {
		return err
	}
	s.checkpoint("")
	s.lib.PutPrefab(rel, b.Bytes())
	e.Prefab, e.Overrides = s.lib.Ref(rel), nil
	s.changed()
	return s.edit.Instantiate()
}

// SceneProps exposes camera and environment as one inspector target.
func (s *Session) SceneProps() []prop.Prop {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	var out []prop.Prop
	for _, p := range sc.Env.Props() {
		p.Name, p.Group = "env."+p.Name, "environment"
		out = append(out, p)
	}
	return out
}

func (s *Session) SetSceneProp(name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	if s.play == nil {
		s.checkpoint("scene:" + name)
	}
	var err error
	switch {
	case strings.HasPrefix(name, "env."):
		err = sc.Env.Set(strings.TrimPrefix(name, "env."), v)
	default:
		err = fmt.Errorf("%w: %s", prop.ErrUnknownProp, name)
	}
	if err != nil {
		return err
	}
	s.changed()
	return nil
}
