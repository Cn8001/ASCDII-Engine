package scene

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
)

// Override is one property of a prefab instance that differs from the prefab.
// Path names the entity inside the instance, "" for the instance root.
type Override struct {
	Path  string `json:"path,omitempty"`
	Prop  string `json:"prop"`
	Value any    `json:"value"`
}

// instanceLocal are the root properties every instance owns anyway; they are
// never overrides.
func instanceLocal(name string) bool {
	return slices.Contains([]string{"name", "visible", "solid", "pos", "pos.x", "pos.y", "pos.z", "scale", "flip"}, name)
}

// InstanceRoot is the nearest prefab instance containing e, or nil.
func (e *Entity) InstanceRoot() *Entity {
	for a := e; a != nil; a = a.parent {
		if a.Prefab != "" && !a.inherited {
			return a
		}
		if !a.inherited {
			return nil
		}
	}
	return nil
}

func PathFrom(root, e *Entity) string {
	var parts []string
	for a := e; a != nil && a != root; a = a.parent {
		parts = append(parts, a.Name)
	}
	slices.Reverse(parts)
	return strings.Join(parts, "/")
}

// FindPath walks names below root; the first match wins on duplicate names.
func FindPath(root *Entity, p string) *Entity {
	if p == "" {
		return root
	}
	cur := root
	for _, name := range strings.Split(p, "/") {
		var next *Entity
		for _, c := range cur.Children {
			if c.Name == name {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

// SetProp edits an entity; inside a prefab instance the change is recorded
// as an override so it survives the prefab being rebuilt.
func (s *Scene) SetProp(e *Entity, name string, v any) error {
	root := e.InstanceRoot()
	if root == nil || (e == root && instanceLocal(name)) {
		return e.Set(name, v)
	}
	if e.inherited && name == "name" {
		return fmt.Errorf("rename %s: %w", e.Name, ErrPrefabLocked)
	}
	if err := e.Set(name, v); err != nil {
		return err
	}
	p, ok := prop.Get(e, name)
	if !ok {
		return fmt.Errorf("%w: %s", prop.ErrUnknownProp, name)
	}
	ov := Override{Path: PathFrom(root, e), Prop: name, Value: prop.Encode(p)}
	for i := range root.Overrides {
		if root.Overrides[i].Path == ov.Path && root.Overrides[i].Prop == ov.Prop {
			root.Overrides[i] = ov
			return nil
		}
	}
	root.Overrides = append(root.Overrides, ov)
	return nil
}

// Revert drops overrides of e (one property, or all when name is "") and
// rebuilds the instance from its prefab.
func (s *Scene) Revert(e *Entity, name string) error {
	root := e.InstanceRoot()
	if root == nil {
		return nil
	}
	at := PathFrom(root, e)
	root.Overrides = slices.DeleteFunc(root.Overrides, func(o Override) bool {
		return (e == root && name == "") || (o.Path == at && (name == "" || o.Prop == name))
	})
	return s.Instantiate()
}

func applyOverrides(root *Entity) error {
	var errs []error
	for _, ov := range root.Overrides {
		t := FindPath(root, ov.Path)
		if t == nil {
			continue // the prefab no longer has that child; keep the override harmless
		}
		if err := t.Set(ov.Prop, ov.Value); err != nil {
			errs = append(errs, fmt.Errorf("override %s %s: %w", ov.Path, ov.Prop, err))
		}
	}
	return errors.Join(errs...)
}
