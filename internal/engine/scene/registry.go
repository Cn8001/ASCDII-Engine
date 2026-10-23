package scene

import (
	"fmt"
	"slices"
)

// Registry maps kind names from scene files to constructors. It is built at
// startup and injected; there is no global registry.
type Registry struct {
	nodes      map[string]func() Node
	behaviours map[string]func() Behaviour
}

func NewRegistry() *Registry {
	return &Registry{nodes: map[string]func() Node{}, behaviours: map[string]func() Behaviour{}}
}

func (r *Registry) Node(kind string, f func() Node)           { r.nodes[kind] = f }
func (r *Registry) Behaviour(kind string, f func() Behaviour) { r.behaviours[kind] = f }

func (r *Registry) NewNode(kind string) (Node, error) {
	f, ok := r.nodes[kind]
	if !ok {
		return nil, fmt.Errorf("%w: node %q", ErrUnknownKind, kind)
	}
	return f(), nil
}

func (r *Registry) NewBehaviour(kind string) (Behaviour, error) {
	f, ok := r.behaviours[kind]
	if !ok {
		return nil, fmt.Errorf("%w: behaviour %q", ErrUnknownKind, kind)
	}
	return f(), nil
}

func (r *Registry) NodeKinds() []string      { return sorted(r.nodes) }
func (r *Registry) BehaviourKinds() []string { return sorted(r.behaviours) }

func sorted[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
