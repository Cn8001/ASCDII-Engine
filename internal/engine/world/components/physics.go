package components

import (
	"fmt"
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/physics"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Gravity pulls rigidbodies down in world units per second squared; it
// matches the player's jump so bodies and walkers fall alike.
const Gravity = 26

func span(t scene.Transform, offset geom.Vec3, height float64) (float64, float64) {
	z0 := t.Apply(offset).Z
	return z0, z0 + height*t.Size(geom.V3(1, 1, 1)).Z
}

// BoxCollider is a rectangle on the ground that turns with the object.
type BoxCollider struct {
	Size    geom.Vec3 // x and y on the ground, z the height
	Offset  geom.Vec3
	Trigger bool
}

func NewBoxCollider() *BoxCollider { return &BoxCollider{Size: geom.V3(2, 2, 2)} }

func (*BoxCollider) Kind() string { return "box_collider" }
func (b *BoxCollider) bind() *prop.Binder {
	return prop.Bind().Vec3("size", &b.Size).Vec3("offset", &b.Offset).Bool("trigger", &b.Trigger)
}
func (b *BoxCollider) Props() []prop.Prop        { return b.bind().Props() }
func (b *BoxCollider) Set(n string, v any) error { return b.bind().Set(n, v) }
func (b *BoxCollider) IsTrigger() bool           { return b.Trigger }
func (b *BoxCollider) Shape(_ *scene.GameObject, t scene.Transform) physics.Shape {
	c := t.Apply(b.Offset)
	k := t.Size(geom.V3(1, 1, 1))
	return physics.Box(geom.V2(c.X, c.Y), geom.V2(b.Size.X*math.Abs(k.X)/2, b.Size.Y*math.Abs(k.Y)/2), t.Rot.Z*math.Pi/180)
}
func (b *BoxCollider) Span(_ *scene.GameObject, t scene.Transform) (float64, float64) {
	return span(t, b.Offset, b.Size.Z)
}

type CircleCollider struct {
	Radius  float64
	Height  float64
	Offset  geom.Vec3
	Trigger bool
}

func NewCircleCollider() *CircleCollider { return &CircleCollider{Radius: 1, Height: 2} }

func (*CircleCollider) Kind() string { return "circle_collider" }
func (c *CircleCollider) bind() *prop.Binder {
	return prop.Bind().Float("radius", &c.Radius, .05, 500).Float("height", &c.Height, 0, 500).Vec3("offset", &c.Offset).Bool("trigger", &c.Trigger)
}
func (c *CircleCollider) Props() []prop.Prop        { return c.bind().Props() }
func (c *CircleCollider) Set(n string, v any) error { return c.bind().Set(n, v) }
func (c *CircleCollider) IsTrigger() bool           { return c.Trigger }
func (c *CircleCollider) Shape(_ *scene.GameObject, t scene.Transform) physics.Shape {
	p := t.Apply(c.Offset)
	k := t.Size(geom.V3(1, 1, 1))
	return physics.Circle{C: geom.V2(p.X, p.Y), R: c.Radius * math.Max(math.Abs(k.X), math.Abs(k.Y))}
}
func (c *CircleCollider) Span(_ *scene.GameObject, t scene.Transform) (float64, float64) {
	return span(t, c.Offset, c.Height)
}

// MeshCollider uses the footprint of its object's model: the convex hull of
// the model's corners seen from above.
type MeshCollider struct {
	Trigger bool
	cache   struct {
		t    scene.Transform
		m    *Mesh
		hull physics.Poly
		z    [2]float64
	}
}

func (*MeshCollider) Kind() string { return "mesh_collider" }
func (m *MeshCollider) bind() *prop.Binder {
	return prop.Bind().Bool("trigger", &m.Trigger)
}
func (m *MeshCollider) Props() []prop.Prop        { return m.bind().Props() }
func (m *MeshCollider) Set(n string, v any) error { return m.bind().Set(n, v) }
func (m *MeshCollider) IsTrigger() bool           { return m.Trigger }

// footprint recomputes the hull only when the object moved or turned.
func (m *MeshCollider) footprint(o *scene.GameObject, t scene.Transform) (physics.Poly, [2]float64) {
	msh, _ := o.Component("mesh").(*Mesh)
	if msh == m.cache.m && t == m.cache.t && m.cache.hull.Pts != nil {
		return m.cache.hull, m.cache.z
	}
	var pts []geom.Vec2
	z := [2]float64{t.Pos.Z, t.Pos.Z}
	if msh == nil {
		m.cache.t, m.cache.m, m.cache.hull, m.cache.z = t, nil, physics.Poly{}, z
		return m.cache.hull, z
	}
	if model := msh.mesh(); model != nil {
		z = [2]float64{math.Inf(1), math.Inf(-1)}
		for _, p := range model.Pos {
			w := t.Apply(p)
			pts = append(pts, geom.V2(w.X, w.Y))
			z = [2]float64{math.Min(z[0], w.Z), math.Max(z[1], w.Z)}
		}
	}
	m.cache.t, m.cache.m, m.cache.hull, m.cache.z = t, msh, physics.Hull(pts), z
	return m.cache.hull, z
}

func (m *MeshCollider) Shape(o *scene.GameObject, t scene.Transform) physics.Shape {
	h, _ := m.footprint(o, t)
	return h
}
func (m *MeshCollider) Span(o *scene.GameObject, t scene.Transform) (float64, float64) {
	_, z := m.footprint(o, t)
	return z[0], z[1]
}

// Rigidbody moves its object with velocity and gravity and pushes it out of
// other colliders. It needs a collider on the same object to collide.
type Rigidbody struct {
	Mass      float64
	Gravity   bool
	Drag      float64
	Bounce    float64
	Kinematic bool // moved by scripts only; others still bump into it
	Velocity  geom.Vec3
	touching  map[uint64]bool
}

func NewRigidbody() *Rigidbody { return &Rigidbody{Mass: 1, Gravity: true, Drag: .5, Bounce: .2} }

func (*Rigidbody) Kind() string { return "rigidbody" }
func (r *Rigidbody) bind() *prop.Binder {
	return prop.Bind().Float("mass", &r.Mass, .01, 1000).Bool("gravity", &r.Gravity).Float("drag", &r.Drag, 0, 20).
		Float("bounce", &r.Bounce, 0, 1).Bool("kinematic", &r.Kinematic).Vec3("velocity", &r.Velocity)
}
func (r *Rigidbody) Props() []prop.Prop        { return r.bind().Props() }
func (r *Rigidbody) Set(n string, v any) error { return r.bind().Set(n, v) }

func (*Rigidbody) Methods() []string { return []string{"add_force"} }

// Call adds an impulse: velocity changes by force divided by mass.
func (r *Rigidbody) Call(_ *scene.Ctx, _ *scene.GameObject, name string, args []any) (any, error) {
	if name != "add_force" || len(args) == 0 {
		return nil, fmt.Errorf("add_force needs a vector")
	}
	f, err := prop.ToVec3(args[0])
	if err != nil {
		return nil, err
	}
	r.Velocity = r.Velocity.Add(f.Scale(1 / math.Max(r.Mass, .01)))
	return nil, nil
}

type shape struct {
	s      physics.Shape
	z0, z1 float64
	trig   bool
}

func shapes(o *scene.GameObject) []shape {
	var out []shape
	t := o.World()
	for _, c := range o.Components {
		if col, ok := c.(scene.Collider); ok && o.Enabled(c) {
			z0, z1 := col.Span(o, t)
			out = append(out, shape{s: col.Shape(o, t), z0: z0, z1: z1, trig: col.IsTrigger()})
		}
	}
	return out
}

func bodyOf(o *scene.GameObject) *Rigidbody {
	for _, c := range o.Components {
		if r, ok := c.(*Rigidbody); ok && o.Enabled(c) && !r.Kinematic {
			return r
		}
	}
	return nil
}

func notify(c *scene.Ctx, o, other *scene.GameObject, trigger bool) {
	for _, h := range o.Components {
		if ch, ok := h.(scene.CollisionHandler); ok && o.Enabled(h) {
			ch.Collide(c, o, other, trigger)
		}
	}
}

func (r *Rigidbody) Update(c *scene.Ctx, o *scene.GameObject, dt float64) {
	if r.Kinematic || dt <= 0 {
		return
	}
	v := r.Velocity
	if r.Gravity {
		v.Z -= Gravity * dt
	}
	keep := math.Max(0, 1-r.Drag*dt)
	v.X, v.Y = v.X*keep, v.Y*keep
	// Velocity, pushes and the floor are in world space; MoveWorld turns them
	// into the parent's space, so bodies under moved or turned parents work.
	before := o.World().Pos
	o.MoveWorld(v.Scale(dt))
	mine := shapes(o)
	floor := 0.0
	touching := map[uint64]bool{}
	c.Scene.Walk(func(other *scene.GameObject, visible bool) bool {
		if other == o || !visible || other == c.Scene.Root {
			return other != o
		}
		theirs := shapes(other)
		for _, a := range mine {
			for _, b := range theirs {
				push, hit := physics.Overlap(a.s, b.s)
				if !hit {
					continue
				}
				// Standing on top: their top is below our feet.
				if !a.trig && !b.trig && b.z1 <= before.Z+.05 {
					floor = math.Max(floor, b.z1)
					continue
				}
				if a.z0 >= b.z1 || b.z0 >= a.z1 {
					continue
				}
				touching[other.ID] = true
				if a.trig || b.trig {
					continue
				}
				share := 1.0
				if ob := bodyOf(other); ob != nil {
					share = ob.Mass / (ob.Mass + r.Mass)
					other.MoveWorld(geom.V3(push.X, push.Y, 0).Scale(share - 1))
				}
				o.MoveWorld(geom.V3(push.X, push.Y, 0).Scale(share))
				n := push.Norm()
				if vn := v.X*n.X + v.Y*n.Y; vn < 0 {
					v.X -= n.X * vn * (1 + r.Bounce)
					v.Y -= n.Y * vn * (1 + r.Bounce)
				}
			}
		}
		return true
	})
	if z := o.World().Pos.Z; z < floor {
		o.MoveWorld(geom.V3(0, 0, floor-z))
		if v.Z < -2 {
			v.Z = -v.Z * r.Bounce
		} else {
			v.Z = 0
		}
	}
	r.Velocity = v
	for id := range touching {
		if r.touching[id] {
			continue
		}
		other, err := c.Scene.ByID(id)
		if err != nil {
			continue
		}
		trig := false
		for _, s := range append(shapes(other), mine...) {
			trig = trig || s.trig
		}
		notify(c, o, other, trig)
		if bodyOf(other) == nil {
			notify(c, other, o, trig) // a body would report the contact itself
		}
	}
	r.touching = touching
}
