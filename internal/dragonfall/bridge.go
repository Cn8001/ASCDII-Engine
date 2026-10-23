package dragonfall

import (
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Bridge is a plank deck with posts and rails, spanning along y.
type Bridge struct {
	Length, Width float64
	Wood          color.Color
}

func NewBridge() *Bridge { return &Bridge{Length: 9, Width: 6, Wood: color.RGB(132, 98, 64)} }

func (*Bridge) Kind() string { return "df.bridge" }
func (b *Bridge) bind() *prop.Binder {
	return prop.Bind().Float("length", &b.Length, 1, 60).Float("width", &b.Width, 1, 30).Color("wood", &b.Wood)
}
func (b *Bridge) Props() []prop.Prop        { return b.bind().Props() }
func (b *Bridge) Set(n string, v any) error { return b.bind().Set(n, v) }
func (b *Bridge) Bounds(t scene.Transform) geom.AABB {
	return geom.Footprint(t.Pos, geom.V3(b.Width, b.Length, 1.4))
}

func (b *Bridge) Draw(c *draw.Context, t scene.Transform) {
	wood := draw.M(b.Wood)
	hw, hl := b.Width/2, b.Length/2
	for y := -hl; y < hl; y += 1.05 {
		c.BoxFaces(geom.AABB{Min: t.Pos.Add(geom.V3(-hw, y, 0)), Max: t.Pos.Add(geom.V3(hw, y+.8, .3))}, wood.Tint(1.12), wood.Tint(.7))
	}
	for _, x := range []float64{-hw + .2, hw - .2} {
		for y := -hl; y <= hl+.01; y += 2.25 {
			c.Cylinder(t.Pos.Add(geom.V3(x, y, 0)), .15, 1.3, wood, wood.Tint(.8), 6)
		}
		c.Box(geom.AABB{Min: t.Pos.Add(geom.V3(x-.1, -hl, 1.05)), Max: t.Pos.Add(geom.V3(x+.1, hl, 1.25))}, wood.Tint(1.1))
	}
	c.Shadow(t.Pos, hw+1, hl, .5)
}
