package dragonfall

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall/realm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/noise"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

// Terrain turns the realm into a height field: raised crags, a sunken river
// with dirt banks and stone ruins.
type Terrain struct {
	Seed  int
	Water float64 // river surface height, negative sinks the channel
	cache *realm.Cache
}

func (*Terrain) Kind() string { return "df.terrain" }
func (t *Terrain) bind() *prop.Binder {
	return prop.Bind().Int("seed", &t.Seed, 0, 99999).Float("water", &t.Water, -3, 0)
}
func (t *Terrain) Props() []prop.Prop               { return t.bind().Props() }
func (t *Terrain) Set(n string, v any) error        { return t.bind().Set(n, v) }
func (t *Terrain) Bounds(scene.Transform) geom.AABB { return geom.AABB{} }
func (t *Terrain) realm() *realm.Realm              { return t.cache.Get(uint32(t.Seed)) }

type heights struct {
	r     *realm.Realm
	water float64
}

func shade(c color.Color, v uint32, amount float64) color.Color {
	return color.Scale(c, 1+(noise.Unit(v)-.5)*amount)
}

func (h heights) Tile(x, y int) draw.Tile {
	t := h.r.At(x, y)
	dirt := draw.M(shade(color.RGB(104, 82, 60), t.Variant, .2))
	switch t.Kind {
	case realm.Wood:
		return draw.Tile{Top: draw.M(shade(color.RGB(50, 80, 54), t.Variant, .16)), Side: dirt}
	case realm.Path:
		return draw.Tile{Height: -.04, Top: draw.M(shade(color.RGB(138, 116, 84), t.Variant, .18)), Side: dirt}
	case realm.Grove:
		return draw.Tile{Top: draw.M(shade(color.RGB(64, 82, 96), t.Variant, .2)), Side: dirt}
	case realm.Ruin:
		return draw.Tile{Height: .5, Top: draw.M(shade(color.RGB(124, 122, 120), t.Variant, .22)), Side: draw.M(color.RGB(92, 90, 92))}
	case realm.Crag:
		return draw.Tile{Height: 1 + float64(t.Elevation)*1.3, Top: draw.M(shade(color.RGB(118, 120, 124), t.Variant, .2)), Side: draw.M(shade(color.RGB(86, 86, 92), t.Variant, .2))}
	case realm.River, realm.Bridge:
		return draw.Tile{Height: h.water, Top: draw.Material{Albedo: shade(color.RGB(42, 98, 118), t.Variant, .1), Water: true}, Side: dirt}
	}
	return draw.Tile{Top: draw.M(shade(color.RGB(74, 108, 62), t.Variant, .18)), Side: dirt}
}

func (t *Terrain) Draw(c *draw.Context, _ scene.Transform) {
	c.Terrain(heights{t.realm(), t.Water}, draw.TerrainOptions{Smooth: true, Seed: uint32(t.Seed), Max: 5})
}

// Blocks keeps the player out of water and off cliffs; the bridge tiles stay open.
func (t *Terrain) Blocks(_ scene.Transform, p geom.Vec2) bool {
	tile := t.realm().At(int(math.Floor(p.X)), int(math.Floor(p.Y)))
	return tile.Kind == realm.River || tile.Kind == realm.Crag
}
