// Package scene is the world model: a tree of GameObjects, each a Transform
// plus any number of components, with prefab instances and animation clips.
// Drawing, logic and collision all come from components.
package scene

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/physics"
)

var (
	ErrNodeNotFound  = errors.New("object not found")
	ErrUnknownKind   = errors.New("unknown component")
	ErrInvalidParent = errors.New("invalid parent")
	ErrPrefabLocked  = errors.New("object belongs to a prefab instance")
)

type Transform struct {
	Pos   geom.Vec3
	Rot   geom.Vec3 // degrees around X, Y and Z (up)
	Scale geom.Vec3
	Flip  bool // mirrors sprites, the 2.5D way of facing left
}

func Identity() Transform { return Transform{Scale: geom.V3(1, 1, 1)} }

// Uniform is the mean scale, for content that only scales evenly.
func (t Transform) Uniform() float64 {
	if t.Scale == (geom.Vec3{}) {
		return 1
	}
	return (t.Scale.X + t.Scale.Y + t.Scale.Z) / 3
}

func (t Transform) scale() geom.Vec3 {
	if t.Scale == (geom.Vec3{}) {
		return geom.V3(1, 1, 1)
	}
	return t.Scale
}

// Size scales a local extent by this transform, for components sized in
// local units.
func (t Transform) Size(v geom.Vec3) geom.Vec3 { return v.Mul(t.scale()) }

// Inward maps an offset in the parent's space into this transform's local
// space, the inverse of Apply without the translation.
func (t Transform) Inward(d geom.Vec3) geom.Vec3 {
	d = d.UnrotateEuler(t.Rot)
	if t.Flip {
		d.X = -d.X
	}
	k := t.scale()
	return geom.V3(inv(d.X, k.X), inv(d.Y, k.Y), inv(d.Z, k.Z))
}

func inv(v, k float64) float64 {
	if k == 0 {
		return v // a flattened axis has no inverse; keep the drag usable
	}
	return v / k
}

// Apply maps a point from this transform's local space to its parent's.
func (t Transform) Apply(p geom.Vec3) geom.Vec3 {
	p = p.Mul(t.scale())
	if t.Flip {
		p.X = -p.X
	}
	return t.Pos.Add(p.RotateEuler(t.Rot))
}

// Then places child local transform c inside parent t.
func (t Transform) Then(c Transform) Transform {
	return Transform{Pos: t.Apply(c.Pos), Rot: t.Rot.Add(c.Rot), Scale: t.scale().Mul(c.scale()), Flip: t.Flip != c.Flip}
}

// Local is the child transform that Then places at world w, so an object
// can change parents without moving. Rotations add like Then's, which is
// exact for turns around one axis.
func (t Transform) Local(w Transform) Transform {
	k, ws := t.scale(), w.scale()
	return Transform{
		Pos:   t.Inward(w.Pos.Sub(t.Pos)),
		Rot:   w.Rot.Sub(t.Rot),
		Scale: geom.V3(inv(ws.X, k.X), inv(ws.Y, k.Y), inv(ws.Z, k.Z)),
		Flip:  w.Flip != t.Flip,
	}
}

// Dir turns a local direction into this transform's parent space.
func (t Transform) Dir(d geom.Vec3) geom.Vec3 {
	if t.Flip {
		d.X = -d.X
	}
	return d.RotateEuler(t.Rot)
}

// Component is anything attached to a GameObject. What it can do comes from
// the optional interfaces below, so one component may draw, collide and think.
type Component interface {
	prop.Editable
	Kind() string
}

type Renderer interface {
	Draw(c *draw.Context, t Transform)
}

type Bounder interface {
	Bounds(t Transform) geom.AABB
}

// Blocker gives a component a collision shape finer than its bounds.
type Blocker interface {
	Blocks(t Transform, p geom.Vec2) bool
}

// CellDrawer writes whole cells after Braille resolve, for text and glyph art.
type CellDrawer interface {
	DrawCells(f *pipeline.Frame, t Transform)
}

// Ticker runs in play and in the editor's Live preview, for simulation that
// is part of how something looks, such as grass springs or a flapping wing.
type Ticker interface {
	Tick(c *Ctx, o *GameObject, dt float64)
}

// Updater is game logic; it runs in play mode only.
type Updater interface {
	Update(c *Ctx, o *GameObject, dt float64)
}

// Callable components offer methods to scripts, such as an animator's
// play or a rigidbody's add_force; properties alone cannot express those.
type Callable interface {
	Methods() []string
	Call(c *Ctx, o *GameObject, name string, args []any) (any, error)
}

// Collider is a footprint on the ground plane plus a height range, what
// rigidbodies and walkers collide with.
type Collider interface {
	Shape(o *GameObject, t Transform) physics.Shape
	Span(o *GameObject, t Transform) (z0, z1 float64)
	IsTrigger() bool
}

// CollisionHandler hears about contacts that begin, such as a script's
// on_collision and on_trigger hooks.
type CollisionHandler interface {
	Collide(c *Ctx, o, other *GameObject, trigger bool)
}

// DotPainter recolors the dots its own object drew, after geometry and
// before lighting; the paint component uses it.
type DotPainter interface {
	PaintDots(f *pipeline.Frame, o *GameObject, t Transform)
}

// AudioOut plays sound assets by reference; the terminal player mixes them
// to a speaker, the editor hands them to its web view.
type AudioOut interface {
	Play(ref string, volume float64, loop bool) (uint64, error)
	Stop(voice uint64)
	SetVolume(voice uint64, v float64)
	Playing(voice uint64) bool
}

// ClipDriver is a component that plays its object's clips itself, so the
// scene does not also play them.
type ClipDriver interface {
	DrivesClips() bool
}

type KeyHandler interface {
	Key(c *Ctx, o *GameObject, k input.Key) bool
}

type Ctx struct {
	Scene    *Scene
	Quit     bool
	Log      func(level, source, msg string) // nil drops messages
	Input    *input.State                    // nil outside play
	Pipeline *pipeline.Pipeline              // what scripts tune; nil in tests
	Audio    AudioOut                        // nil plays nothing
	Shared   map[any]any                     // per run state of modules such as the Lua VM, keyed like context values
	scale    float64
	scaled   bool
}

// TimeScale slows or speeds up the game; 0 pauses it.
func (c *Ctx) TimeScale() float64 {
	if !c.scaled {
		return 1
	}
	return c.scale
}

func (c *Ctx) SetTimeScale(k float64) { c.scale, c.scaled = max(0, k), true }

func (c *Ctx) Logf(level, source, format string, args ...any) {
	if c.Log != nil {
		c.Log(level, source, fmt.Sprintf(format, args...))
	}
}

type GameObject struct {
	ID         uint64 // runtime id, fresh on every load
	FID        uint64 // id within its scene or prefab file; references use it
	Name       string
	Tag        string
	Visible    bool
	Solid      bool
	Local      Transform
	Components []Component
	Children   []*GameObject
	Prefab     string // asset reference; components and children come from the prefab
	Overrides  []Override
	off        map[Component]bool
	slots      map[Component]int // the n of a component's key; kept when others are removed or moved
	high       map[string]int    // highest slot handed out per kind, so a removed one is not reused
	parent     *GameObject
	scene      *Scene
	inherited  bool
	srcFID     uint64 // an instance root's FID inside its prefab file
}

// NewGameObject makes a detached object for Graft.
func NewGameObject(name string, comps ...Component) *GameObject {
	o := &GameObject{Name: name, Visible: true, Local: Identity()}
	return o.With(comps...)
}

func (o *GameObject) Parent() *GameObject { return o.parent }

func (o *GameObject) Scene() *Scene { return o.scene }

// Inherited reports objects created by a prefab instance; they are not saved.
func (o *GameObject) Inherited() bool { return o.inherited }

func (o *GameObject) World() Transform {
	if o.parent == nil {
		return o.Local
	}
	return o.parent.World().Then(o.Local)
}

// SetWorld moves the object so its world transform becomes w.
func (o *GameObject) SetWorld(w Transform) {
	if o.parent == nil {
		o.Local = w
		return
	}
	o.Local = o.parent.World().Local(w)
}

// MoveWorld moves the object by d in world units, whatever its parent's
// position, rotation or scale; physics and walkers move in world space.
func (o *GameObject) MoveWorld(d geom.Vec3) {
	if o.parent == nil {
		o.Local.Pos = o.Local.Pos.Add(d)
		return
	}
	o.Local.Pos = o.Local.Pos.Add(o.parent.World().Inward(d))
}

// Bounds unites what every component reports, for culling, picking and gizmos.
func (o *GameObject) Bounds() geom.AABB {
	var b geom.AABB
	t := o.World()
	for _, c := range o.Components {
		if bc, ok := c.(Bounder); ok && o.Enabled(c) {
			b = b.Union(bc.Bounds(t))
		}
	}
	return b
}

func (o *GameObject) At(x, y, z float64) *GameObject { o.Local.Pos = geom.V3(x, y, z); return o }
func (o *GameObject) Scaled(k float64) *GameObject   { o.Local.Scale = geom.V3(k, k, k); return o }
func (o *GameObject) Blocking() *GameObject          { o.Solid = true; return o }

func (o *GameObject) With(comps ...Component) *GameObject {
	for _, c := range comps {
		if c != nil {
			o.Components = append(o.Components, c)
		}
	}
	return o
}

// Add attaches a child; it panics only if o was never attached to a scene,
// which is a programming error in content code.
func (o *GameObject) Add(name string, comps ...Component) *GameObject {
	if o.scene == nil {
		panic("scene: Add on a detached object")
	}
	c := o.scene.newObject(name, comps...)
	o.attach(c, -1)
	return c
}

func (o *GameObject) attach(c *GameObject, index int) {
	c.parent = o
	if index < 0 || index > len(o.Children) {
		o.Children = append(o.Children, c)
		return
	}
	o.Children = append(o.Children[:index], append([]*GameObject{c}, o.Children[index:]...)...)
}

func (o *GameObject) detach() {
	p := o.parent
	if p == nil {
		return
	}
	for i, c := range p.Children {
		if c == o {
			p.Children = append(p.Children[:i], p.Children[i+1:]...)
			break
		}
	}
	o.parent = nil
}

// Enabled reports whether a component runs; disabled ones keep their settings.
func (o *GameObject) Enabled(c Component) bool { return !o.off[c] }

func (o *GameObject) SetEnabled(c Component, on bool) {
	if o.off == nil {
		o.off = map[Component]bool{}
	}
	if on {
		delete(o.off, c)
	} else {
		o.off[c] = true
	}
}

// Component returns the first component of a kind, or nil.
func (o *GameObject) Component(kind string) Component {
	for _, c := range o.Components {
		if c.Kind() == kind {
			return c
		}
	}
	return nil
}

func (o *GameObject) ComponentsOf(kind string) []Component {
	var out []Component
	for _, c := range o.Components {
		if c.Kind() == kind {
			out = append(out, c)
		}
	}
	return out
}

// Main is the component the editor shows as the object's kind: the first one
// that draws, else the first one.
func (o *GameObject) Main() Component {
	for _, c := range o.Components {
		if _, ok := c.(Renderer); ok {
			return c
		}
	}
	if len(o.Components) > 0 {
		return o.Components[0]
	}
	return nil
}

// RemoveComponent detaches the component at index.
func (o *GameObject) RemoveComponent(i int) error {
	if i < 0 || i >= len(o.Components) {
		return fmt.Errorf("%w: component %d", ErrUnknownKind, i)
	}
	o.number() // the others keep the keys they had with it present
	delete(o.off, o.Components[i])
	delete(o.slots, o.Components[i])
	o.Components = append(o.Components[:i], o.Components[i+1:]...)
	return nil
}

// ReplaceComponent puts c in place of the component at index; c takes over
// its key and enabled state, so references to it still hold.
func (o *GameObject) ReplaceComponent(i int, c Component) error {
	if i < 0 || i >= len(o.Components) {
		return fmt.Errorf("%w: component %d", ErrUnknownKind, i)
	}
	o.number()
	old := o.Components[i]
	on, n := o.Enabled(old), o.slots[old]
	delete(o.off, old)
	delete(o.slots, old)
	o.Components[i] = c
	o.slots[c] = n
	o.SetEnabled(c, on)
	return nil
}

// MoveComponent reorders components, which changes draw order; keys stay.
func (o *GameObject) MoveComponent(from, to int) error {
	if from < 0 || from >= len(o.Components) || to < 0 || to >= len(o.Components) {
		return fmt.Errorf("%w: component %d", ErrUnknownKind, from)
	}
	o.number()
	c := o.Components[from]
	o.Components = append(o.Components[:from], o.Components[from+1:]...)
	o.Components = append(o.Components[:to], append([]Component{c}, o.Components[to:]...)...)
	return nil
}

// Key names a component in property paths: "light" for the first light,
// "light#2" for the second. Keys are saved in references, overrides and clips,
// so a component keeps its key when another one is removed or reordered.
func (o *GameObject) Key(c Component) string {
	o.number()
	if n := o.slots[c]; n > 1 {
		return c.Kind() + "#" + strconv.Itoa(n)
	}
	return c.Kind()
}

// number gives components without a slot the next one of their kind, in
// list order, so objects built in code number like they always did.
func (o *GameObject) number() {
	if o.slots == nil {
		o.slots, o.high = map[Component]int{}, map[string]int{}
	}
	for _, c := range o.Components {
		if _, ok := o.slots[c]; !ok {
			o.high[c.Kind()]++
			o.slots[c] = o.high[c.Kind()]
		}
	}
}

// setSlot gives c a slot read from a file.
func (o *GameObject) setSlot(c Component, n int) {
	if o.slots == nil {
		o.slots, o.high = map[Component]int{}, map[string]int{}
	}
	o.slots[c] = n
	o.high[c.Kind()] = max(o.high[c.Kind()], n)
}

// ComponentByKey finds a component by the key its properties use, such as
// "light" or "light#2".
func (o *GameObject) ComponentByKey(key string) Component {
	if c, rest := o.split(key + "."); rest == "" {
		return c
	}
	return nil
}

// split finds the component a property path starts with. Kinds may contain
// dots ("df.hero"), so the longest matching key wins.
func (o *GameObject) split(name string) (Component, string) {
	var best Component
	rest := ""
	for _, c := range o.Components {
		key := o.Key(c)
		if strings.HasPrefix(name, key+".") && (best == nil || len(key) > len(name)-len(rest)-1) {
			best, rest = c, name[len(key)+1:]
		}
	}
	return best, rest
}

func (o *GameObject) bind() *prop.Binder {
	return prop.Bind().
		String("name", &o.Name).
		String("tag", &o.Tag).
		Bool("visible", &o.Visible).
		Bool("solid", &o.Solid).
		Vec3("pos", &o.Local.Pos).
		Vec3("rot", &o.Local.Rot).
		Vec3("scale", &o.Local.Scale).
		Bool("flip", &o.Local.Flip)
}

// Props lists the object's own fields, then every component's properties
// prefixed with its key. Inside a prefab instance, properties that differ
// from the prefab are marked overridden.
func (o *GameObject) Props() []prop.Prop {
	out := group(o.bind().Props(), "transform")
	for _, c := range o.Components {
		key := o.Key(c)
		out = append(out, prop.Prop{Name: key + ".enabled", Type: prop.Bool, Value: o.Enabled(c), Group: key})
		for _, p := range group(c.Props(), key) {
			p.Name = key + "." + p.Name
			out = append(out, p)
		}
	}
	for i := range out {
		if out[i].Type == prop.ObjectRef || out[i].Type == prop.CompRef {
			out[i].Display = o.display(out[i])
		}
	}
	root := o.InstanceRoot()
	if root == nil {
		return out
	}
	at := PathFrom(root, o)
	for i := range out {
		if o.inherited && out[i].Name == "name" {
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

// Resolve maps a short property name such as "radius", used by older files,
// scripts and clips, to its full name such as "light.radius".
func (o *GameObject) Resolve(name string) string {
	if o.bind().Has(name) {
		return name
	}
	if c, _ := o.split(name); c != nil {
		return name
	}
	base, axis := name, ""
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		if s := name[i+1:]; s == "x" || s == "y" || s == "z" {
			base, axis = name[:i], name[i:]
		}
	}
	for _, c := range o.Components {
		for _, p := range c.Props() {
			if p.Name == base {
				return o.Key(c) + "." + base + axis
			}
		}
	}
	return name
}

func (o *GameObject) Set(name string, v any) error {
	name = o.Resolve(name)
	b := o.bind()
	if b.Has(name) {
		return b.Set(name, v)
	}
	if c, rest := o.split(name); c != nil {
		if rest != "enabled" {
			return c.Set(rest, v)
		}
		on, err := prop.ToBool(v)
		if err != nil {
			return fmt.Errorf("set %s: %w", name, err)
		}
		o.SetEnabled(c, on)
		return nil
	}
	return fmt.Errorf("%w: %s", prop.ErrUnknownProp, name)
}
