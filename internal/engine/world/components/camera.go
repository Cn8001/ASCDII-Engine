package components

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Camera makes its object the game camera: the object's position is the
// point the camera looks at and its Z rotation turns the view, so parenting
// the object to another one carries the camera along.
type Camera struct {
	Span    float64 // world units across the view; smaller zooms in
	Pitch   float64 // degrees above the ground
	AnchorX float64 // where the target sits on screen, 0 to 1
	AnchorY float64
}

func NewCamera() *Camera {
	d := view.Default()
	return &Camera{Span: d.Span, Pitch: d.Pitch, AnchorX: d.AnchorX, AnchorY: d.AnchorY}
}

func (*Camera) Kind() string { return "camera" }
func (c *Camera) bind() *prop.Binder {
	return prop.Bind().Float("span", &c.Span, 4, 400).Float("pitch", &c.Pitch, 15, 89).
		Float("anchorX", &c.AnchorX, 0, 1).Float("anchorY", &c.AnchorY, 0, 1)
}
func (c *Camera) Props() []prop.Prop        { return c.bind().Props() }
func (c *Camera) Set(n string, v any) error { return c.bind().Set(n, v) }

func (c *Camera) View(t scene.Transform) view.Camera {
	return view.Camera{Target: t.Pos, Span: c.Span, Pitch: c.Pitch, Yaw: t.Rot.Z, AnchorX: c.AnchorX, AnchorY: c.AnchorY}
}

func (c *Camera) Place(o *scene.GameObject, v view.Camera) {
	w := o.World()
	w.Pos, w.Rot.Z = v.Target, v.Yaw
	o.SetWorld(w)
	c.Span, c.Pitch, c.AnchorX, c.AnchorY = geom.Clamp(v.Span, 4, 400), geom.Clamp(v.Pitch, 15, 89), v.AnchorX, v.AnchorY
}
