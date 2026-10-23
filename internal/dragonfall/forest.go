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

// Forest scatters pines on a jittered grid, so thousands of trees cost
// nothing to store and only visible ones are drawn.
type Forest struct {
	Seed    int
	Density float64
	Sway    float64
	cache   *realm.Cache
}

type tree struct {
	base geom.Vec3
	size float64
	hash uint32
}

func (*Forest) Kind() string { return "df.forest" }
func (f *Forest) bind() *prop.Binder {
	return prop.Bind().Int("seed", &f.Seed, 0, 99999).Float("density", &f.Density, 0, 1).Float("sway", &f.Sway, 0, 2)
}
func (f *Forest) Props() []prop.Prop               { return f.bind().Props() }
func (f *Forest) Set(n string, v any) error        { return f.bind().Set(n, v) }
func (f *Forest) Bounds(scene.Transform) geom.AABB { return geom.AABB{} }

func clearing(x, y float64) bool {
	return (x > 80 && x < 130 && y > 48 && y < 80) || (x > 128 && x < 160 && y > 60 && y < 84)
}

func (f *Forest) each(lo, hi geom.Vec2, fn func(tree)) {
	r := f.cache.Get(uint32(f.Seed))
	for wy := int(math.Floor(lo.Y/9)) * 9; wy <= int(hi.Y); wy += 9 {
		for wx := int(math.Floor(lo.X/10)) * 10; wx <= int(hi.X); wx += 10 {
			h := noise.Hash(wx, wy, 983)
			t := r.At(wx, wy)
			if (t.Kind != realm.Wood && h%8 != 0) || h%3 == 0 || noise.Unit(h>>12) > f.Density {
				continue
			}
			x, y := float64(wx)+float64(h%7)*.4, float64(wy)+float64((h>>4)%5)*.5
			k := r.At(int(x), int(y)).Kind
			if clearing(x, y) || k == realm.River || k == realm.Bridge || k == realm.Path || k == realm.Crag || realm.RiverAt(x, y) {
				continue
			}
			fn(tree{base: geom.V3(x, y, 0), size: .8 + noise.Unit(h>>5)*.55, hash: h})
		}
	}
}

func (f *Forest) Draw(c *draw.Context, _ scene.Transform) {
	lo, hi := c.View.GroundRect(14)
	f.each(lo, hi, func(t tree) { drawPine(c, t, f.Sway) })
}

func drawPine(c *draw.Context, t tree, sway float64) {
	k := t.size
	bark := draw.M(color.RGB(98, 74, 54))
	c.Cylinder(t.base, .45*k, 2.6*k, bark, bark, 8)
	wind := math.Sin(c.Time*1.3+t.base.X*.3+t.base.Y*.2) * sway * k
	greens := [3]color.Color{color.RGB(34, 84, 58), color.RGB(46, 104, 68), color.RGB(62, 126, 82)}
	tint := 1 + (noise.Unit(t.hash>>9)-.5)*.2
	for i, l := range [3]struct{ z, r, h float64 }{{2, 3.2, 3.4}, {4.2, 2.5, 3.1}, {6.3, 1.7, 3}} {
		lean := geom.V2(wind*float64(i+1)*.35, 0)
		base := t.base.Add(geom.V3(lean.X*.5, 0, l.z*k))
		c.Cone(base, l.r*k, l.h*k, draw.M(greens[i]).Tint(tint), 9, lean)
	}
	c.Shadow(t.base, 2.8*k, 1.9*k, .6)
}

func (f *Forest) Blocks(_ scene.Transform, p geom.Vec2) bool {
	hit := false
	f.each(p.Sub(geom.V2(10, 9)), p.Add(geom.V2(10, 9)), func(t tree) {
		if t.base.XY().Sub(p).Len() < .7*t.size {
			hit = true
		}
	})
	return hit
}
