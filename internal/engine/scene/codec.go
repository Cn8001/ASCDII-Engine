package scene

import (
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

const Version = 1

type compDoc struct {
	Kind  string         `json:"kind"`
	Off   bool           `json:"off,omitempty"`
	Props map[string]any `json:"props,omitempty"`
}

type entityDoc struct {
	Name       string      `json:"name"`
	Hidden     bool        `json:"hidden,omitempty"`
	Solid      bool        `json:"solid,omitempty"`
	Pos        geom.Vec3   `json:"pos"`
	Scale      float64     `json:"scale,omitempty"`
	Flip       bool        `json:"flip,omitempty"`
	Prefab     string      `json:"prefab,omitempty"`
	Overrides  []Override  `json:"overrides,omitempty"`
	Node       *compDoc    `json:"node,omitempty"`
	Behaviours []compDoc   `json:"behaviours,omitempty"`
	Children   []entityDoc `json:"children,omitempty"`
}

type sceneDoc struct {
	Version  int         `json:"version"`
	Name     string      `json:"name"`
	Camera   view.Camera `json:"camera"`
	Env      light.Env   `json:"env"`
	Clips    []anim.Clip `json:"clips,omitempty"`
	Entities []entityDoc `json:"entities"`
}

type prefabDoc struct {
	Version int       `json:"version"`
	Root    entityDoc `json:"root"`
}

func props(e prop.Editable) map[string]any {
	out := map[string]any{}
	for _, p := range e.Props() {
		out[p.Name] = prop.Encode(p)
	}
	return out
}

// docOf with expand set writes prefab content inline, used when saving a prefab.
func docOf(e *Entity, expand bool) entityDoc {
	d := entityDoc{Name: e.Name, Hidden: !e.Visible, Solid: e.Solid, Pos: e.Local.Pos, Flip: e.Local.Flip, Prefab: e.Prefab}
	if e.Local.Scale != 1 {
		d.Scale = e.Local.Scale
	}
	if e.Prefab != "" && !expand {
		d.Overrides = e.Overrides
		return d
	}
	d.Prefab = ""
	if e.Node != nil {
		d.Node = &compDoc{Kind: e.Node.Kind(), Props: props(e.Node)}
	}
	for _, b := range e.Behaviours {
		d.Behaviours = append(d.Behaviours, compDoc{Kind: b.Kind(), Off: !e.Enabled(b.Kind()), Props: props(b)})
	}
	for _, c := range e.Children {
		if expand || !c.inherited {
			d.Children = append(d.Children, docOf(c, expand))
		}
	}
	return d
}

func Encode(w io.Writer, s *Scene) error {
	doc := sceneDoc{Version: Version, Name: s.Name, Camera: s.Camera, Env: s.Env}
	for _, c := range s.Clips {
		if !c.Transient {
			doc.Clips = append(doc.Clips, c)
		}
	}
	for _, c := range s.Root.Children {
		if !c.inherited {
			doc.Entities = append(doc.Entities, docOf(c, false))
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encode scene: %w", err)
	}
	return nil
}

func Decode(r io.Reader, reg *Registry, prefabs Prefabs) (*Scene, error) {
	var doc sceneDoc
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode scene: %w", err)
	}
	s := New(doc.Name)
	s.Camera, s.Env, s.Clips = doc.Camera, doc.Env, doc.Clips
	s.UsePrefabs(prefabs)
	for _, d := range doc.Entities {
		e, err := s.build(d, reg)
		if err != nil {
			return nil, err
		}
		s.Root.attach(e, -1)
	}
	if err := s.Instantiate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scene) build(d entityDoc, reg *Registry) (*Entity, error) {
	e := s.newEntity(d.Name, nil)
	e.Visible, e.Solid, e.Prefab, e.Overrides = !d.Hidden, d.Solid, d.Prefab, d.Overrides
	e.Local = Transform{Pos: d.Pos, Scale: d.Scale, Flip: d.Flip}
	if e.Local.Scale == 0 {
		e.Local.Scale = 1
	}
	if d.Node != nil {
		n, err := reg.NewNode(d.Node.Kind)
		if err != nil {
			return nil, fmt.Errorf("entity %s: %w", d.Name, err)
		}
		if err := apply(n, d.Node.Props); err != nil {
			return nil, fmt.Errorf("entity %s: %w", d.Name, err)
		}
		e.Node = n
	}
	for _, bd := range d.Behaviours {
		b, err := reg.NewBehaviour(bd.Kind)
		if err != nil {
			return nil, fmt.Errorf("entity %s: %w", d.Name, err)
		}
		if err := apply(b, bd.Props); err != nil {
			return nil, fmt.Errorf("entity %s: %w", d.Name, err)
		}
		e.Behaviours = append(e.Behaviours, b)
		if bd.Off {
			e.SetEnabled(bd.Kind, false)
		}
	}
	for _, cd := range d.Children {
		c, err := s.build(cd, reg)
		if err != nil {
			return nil, err
		}
		e.attach(c, -1)
	}
	return e, nil
}

// apply ignores unknown properties so old files keep loading after a node
// drops a field.
func apply(e prop.Editable, m map[string]any) error {
	known := map[string]bool{}
	for _, p := range e.Props() {
		known[p.Name] = true
	}
	for k, v := range m {
		if !known[k] {
			continue
		}
		if err := e.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func EncodePrefab(w io.Writer, e *Entity) error {
	d := docOf(e, true)
	d.Pos = geom.Vec3{}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(prefabDoc{Version: Version, Root: d}); err != nil {
		return fmt.Errorf("encode prefab: %w", err)
	}
	return nil
}

// DecodePrefab builds a detached tree; Instantiate adopts it into a scene.
func DecodePrefab(r io.Reader, reg *Registry) (*Entity, error) {
	var doc prefabDoc
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode prefab: %w", err)
	}
	tmp := New("prefab")
	return tmp.build(doc.Root, reg)
}

// Clone deep copies a scene through its own file format, used for play mode.
func Clone(s *Scene, reg *Registry) (*Scene, error) {
	pr, pw := io.Pipe()
	go func() { pw.CloseWithError(Encode(pw, s)) }()
	out, err := Decode(pr, reg, s.prefabs)
	if err != nil {
		return nil, fmt.Errorf("clone scene: %w", err)
	}
	out.Time = s.Time
	return out, nil
}
