package scene

import (
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
)

// Version 2 stores a list of components per object; version 1 files (one
// node plus behaviours, a single scale number) are converted on load.
const Version = 2

type compDoc struct {
	Kind  string         `json:"kind"`
	Off   bool           `json:"off,omitempty"`
	Props map[string]any `json:"props,omitempty"`
}

type entityDoc struct {
	FID        uint64          `json:"fid,omitempty"`
	Name       string          `json:"name"`
	Tag        string          `json:"tag,omitempty"`
	Hidden     bool            `json:"hidden,omitempty"`
	Solid      bool            `json:"solid,omitempty"`
	Pos        geom.Vec3       `json:"pos"`
	Rot        *geom.Vec3      `json:"rot,omitempty"`
	Scale      json.RawMessage `json:"scale,omitempty"` // [x,y,z], or one number in version 1
	Flip       bool            `json:"flip,omitempty"`
	Prefab     string          `json:"prefab,omitempty"`
	Overrides  []Override      `json:"overrides,omitempty"`
	Components []compDoc       `json:"components,omitempty"`
	Node       *compDoc        `json:"node,omitempty"`       // version 1
	Behaviours []compDoc       `json:"behaviours,omitempty"` // version 1
	Children   []entityDoc     `json:"children,omitempty"`
}

type sceneDoc struct {
	Version  int         `json:"version"`
	Name     string      `json:"name"`
	Camera   view.Camera `json:"camera"`
	Env      light.Env   `json:"env"`
	Clips    []anim.Clip `json:"clips,omitempty"`
	Entities []entityDoc `json:"objects"`
	Legacy   []entityDoc `json:"entities,omitempty"` // version 1
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

func encodeScale(v geom.Vec3) json.RawMessage {
	if v == (geom.Vec3{}) || v == geom.V3(1, 1, 1) {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func decodeScale(raw json.RawMessage) (geom.Vec3, error) {
	if len(raw) == 0 {
		return geom.V3(1, 1, 1), nil
	}
	var k float64
	if err := json.Unmarshal(raw, &k); err == nil {
		if k == 0 {
			k = 1
		}
		return geom.V3(k, k, k), nil
	}
	var v geom.Vec3
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, fmt.Errorf("decode scale: %w", err)
	}
	return v, nil
}

// docOf with expand set writes prefab content inline, used when saving a prefab.
func docOf(e *GameObject, expand bool) entityDoc {
	d := entityDoc{FID: e.FID, Name: e.Name, Tag: e.Tag, Hidden: !e.Visible, Solid: e.Solid, Pos: e.Local.Pos, Flip: e.Local.Flip, Prefab: e.Prefab, Scale: encodeScale(e.Local.Scale)}
	if e.Local.Rot != (geom.Vec3{}) {
		r := e.Local.Rot
		d.Rot = &r
	}
	if e.Prefab != "" && !expand {
		d.Overrides = e.Overrides
		return d
	}
	d.Prefab = ""
	for _, c := range e.Components {
		d.Components = append(d.Components, compDoc{Kind: c.Kind(), Off: !e.Enabled(c), Props: props(c)})
	}
	for _, c := range e.Children {
		if expand || !c.inherited {
			d.Children = append(d.Children, docOf(c, expand))
		}
	}
	return d
}

func Encode(w io.Writer, s *Scene) error {
	doc := sceneDoc{Version: Version, Name: s.Name, Camera: s.Camera, Env: s.Env, Entities: []entityDoc{}}
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
	for _, d := range append(doc.Entities, doc.Legacy...) {
		e, err := s.build(d, reg)
		if err != nil {
			return nil, err
		}
		s.Root.attach(e, -1)
	}
	s.number(s.Root, map[uint64]bool{})
	if err := s.Instantiate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scene) build(d entityDoc, reg *Registry) (*GameObject, error) {
	e := s.newObject(d.Name)
	e.FID = d.FID
	e.Tag, e.Visible, e.Solid, e.Prefab, e.Overrides = d.Tag, !d.Hidden, d.Solid, d.Prefab, d.Overrides
	scale, err := decodeScale(d.Scale)
	if err != nil {
		return nil, fmt.Errorf("object %s: %w", d.Name, err)
	}
	e.Local = Transform{Pos: d.Pos, Scale: scale, Flip: d.Flip}
	if d.Rot != nil {
		e.Local.Rot = *d.Rot
	}
	comps := d.Components
	if d.Node != nil {
		comps = append([]compDoc{*d.Node}, comps...)
	}
	comps = append(comps, d.Behaviours...)
	for _, cd := range comps {
		c, err := reg.New(cd.Kind)
		if err != nil {
			return nil, fmt.Errorf("object %s: %w", d.Name, err)
		}
		if err := apply(c, cd.Props); err != nil {
			return nil, fmt.Errorf("object %s: %w", d.Name, err)
		}
		e.Components = append(e.Components, c)
		if cd.Off {
			e.SetEnabled(c, false)
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

// apply ignores unknown properties so old files keep loading after a
// component drops a field.
// apply sets the stored properties a component still has. Setting some can
// declare others, as a script's file declares its fields, so it repeats
// until nothing new appears.
func apply(e prop.Editable, m map[string]any) error {
	done := map[string]bool{}
	for {
		progress := false
		for _, p := range e.Props() {
			v, ok := m[p.Name]
			if !ok || done[p.Name] {
				continue
			}
			done[p.Name], progress = true, true
			if err := e.Set(p.Name, v); err != nil {
				return err
			}
		}
		if !progress {
			return nil
		}
	}
}

func EncodePrefab(w io.Writer, e *GameObject) error {
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
func DecodePrefab(r io.Reader, reg *Registry) (*GameObject, error) {
	var doc prefabDoc
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode prefab: %w", err)
	}
	tmp := New("prefab")
	tmp.UseFIDs(CountingFIDs()) // files from before FIDs get the same ones on every load
	root, err := tmp.build(doc.Root, reg)
	if err != nil {
		return nil, err
	}
	tmp.number(root, map[uint64]bool{})
	return root, nil
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
