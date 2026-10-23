package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

var (
	ErrNotEditable = errors.New("not editable")
	ErrNotInstance = errors.New("not part of a prefab instance")
)

type Node struct {
	ID         uint64   `json:"id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Visible    bool     `json:"visible"`
	Prefab     string   `json:"prefab,omitempty"`
	Overrides  int      `json:"overrides,omitempty"`
	Inherited  bool     `json:"inherited,omitempty"`
	Behaviours []string `json:"behaviours,omitempty"`
	Children   []Node   `json:"children,omitempty"`
}

func nodeOf(e *scene.Entity) Node {
	n := Node{ID: e.ID, Name: e.Name, Visible: e.Visible, Prefab: e.Prefab, Overrides: len(e.Overrides), Inherited: e.Inherited()}
	if e.Node != nil {
		n.Kind = e.Node.Kind()
	}
	for _, b := range e.Behaviours {
		n.Behaviours = append(n.Behaviours, b.Kind())
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

type Kinds struct {
	Nodes      []string `json:"nodes"`
	Behaviours []string `json:"behaviours"`
	Prefabs    []string `json:"prefabs"`
}

func (s *Session) Kinds() Kinds {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Kinds{Nodes: s.reg.NodeKinds(), Behaviours: s.reg.BehaviourKinds(), Prefabs: s.lib.PrefabNames()}
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
func (s *Session) edit1(id uint64, label string, fn func(sc *scene.Scene, e *scene.Entity) error) error {
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
	return s.edit1(id, fmt.Sprintf("prop:%d:%s", id, name), func(sc *scene.Scene, e *scene.Entity) error {
		p, ok := prop.Get(e, name)
		if ok && p.Locked {
			return fmt.Errorf("set %s: %w", name, scene.ErrPrefabLocked)
		}
		if str, isStr := v.(string); ok && isStr && (p.Type == prop.Asset || p.Type == prop.ScriptRef) {
			v = s.lib.Ref(str)
		}
		return sc.SetProp(e, name, v)
	})
}

// Revert drops overrides on an entity inside a prefab instance: one property,
// or every override of the instance when called on its root with "".
func (s *Session) Revert(id uint64, name string) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.Entity) error { return sc.Revert(e, name) })
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

// Add creates "empty", a registered node kind or "prefab:<path>" under parent.
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
	if s.play == nil {
		s.checkpoint("")
	}
	var node scene.Node
	name, prefab := kind, ""
	switch {
	case kind == "empty":
	case strings.HasPrefix(kind, "prefab:"):
		prefab, name = s.lib.Ref(strings.TrimPrefix(kind, "prefab:")), ""
	default:
		n, err := s.reg.NewNode(kind)
		if err != nil {
			return 0, err
		}
		node = n
	}
	e := scene.NewEntity(name, node)
	if parent == 0 {
		e.Local.Pos = s.spawnPoint()
	}
	e.Prefab = prefab
	got, err := sc.Graft(p, e, -1)
	if err != nil {
		return 0, err
	}
	s.sel = got.ID
	s.changed()
	return got.ID, nil
}

func (s *Session) AddBehaviour(id uint64, kind string) error {
	return s.edit1(id, "", func(_ *scene.Scene, e *scene.Entity) error {
		if e.Prefab != "" || e.Inherited() {
			return scene.ErrPrefabLocked
		}
		b, err := s.reg.NewBehaviour(kind)
		if err != nil {
			return err
		}
		e.With(b)
		return nil
	})
}

func (s *Session) RemoveBehaviour(id uint64, kind string) error {
	return s.edit1(id, "", func(_ *scene.Scene, e *scene.Entity) error {
		for i, b := range e.Behaviours {
			if b.Kind() == kind {
				e.Behaviours = append(e.Behaviours[:i], e.Behaviours[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("%w: behaviour %s", scene.ErrUnknownKind, kind)
	})
}

func (s *Session) Remove(id uint64) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.Entity) error {
		if s.sel == id {
			s.sel = 0
		}
		return sc.Remove(e.ID)
	})
}

func (s *Session) Reparent(id, parent uint64, index int) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.Entity) error { return sc.Reparent(e.ID, parent, index) })
}

func (s *Session) Unpack(id uint64) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.Entity) error { return sc.Unpack(e.ID) })
}

func (s *Session) Duplicate(id uint64) (uint64, error) {
	var out uint64
	err := s.edit1(id, "", func(sc *scene.Scene, e *scene.Entity) error {
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
			c.Node, c.Behaviours, c.Children = nil, nil, nil
		}
		got, err := sc.Graft(e.Parent(), c, -1)
		if err != nil {
			return err
		}
		out, s.sel = got.ID, got.ID
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
	e, err := s.edit.ByID(id)
	if err != nil {
		return err
	}
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
	for _, p := range sc.Camera.Props() {
		p.Name, p.Group = "camera."+p.Name, "camera"
		out = append(out, p)
	}
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
	case strings.HasPrefix(name, "camera."):
		err = sc.Camera.Set(strings.TrimPrefix(name, "camera."), v)
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
