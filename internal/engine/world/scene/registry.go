package scene

import (
	"fmt"
	"slices"
)

// Meta tells the editor where a component kind belongs in its menus.
type Meta struct {
	Category string `json:"category"` // "3d", "light", "effects", "ui", "logic", or a pack name
	Label    string `json:"label"`
	Object   bool   `json:"object"` // offered in the GameObject menu as a ready object
}

// Registry maps kind names from scene files to constructors. It is built at
// startup and injected; there is no global registry.
type Registry struct {
	kinds map[string]func() Component
	meta  map[string]Meta
	order []string
}

func NewRegistry() *Registry {
	return &Registry{kinds: map[string]func() Component{}, meta: map[string]Meta{}}
}

func (r *Registry) Register(kind string, f func() Component, m Meta) {
	if _, ok := r.kinds[kind]; !ok {
		r.order = append(r.order, kind)
	}
	r.kinds[kind] = f
	if m.Label == "" {
		m.Label = kind
	}
	r.meta[kind] = m
}

func (r *Registry) New(kind string) (Component, error) {
	f, ok := r.kinds[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}
	return f(), nil
}

// Kinds lists kinds in registration order, which is the order menus show.
func (r *Registry) Kinds() []string { return slices.Clone(r.order) }

func (r *Registry) Meta(kind string) Meta { return r.meta[kind] }
