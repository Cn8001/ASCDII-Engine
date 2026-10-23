package draw

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/noise"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

func packed(n geom.Vec3) raster.Normal { return raster.PackNormal(n) }

type Tile struct {
	Height float64
	Top    Material
	Side   Material
	Skip   bool
}

// Heightmap is sampled per integer tile; tile (x, y) covers [x, x+1) by [y, y+1).
type Heightmap interface {
	Tile(x, y int) Tile
}

type TerrainOptions struct {
	Grain  float64 // per quarter tile brightness jitter, 0..1
	Seed   uint32
	Max    float64 // tallest height, widens culling so cliffs never pop
	Smooth bool    // blend top colors across tile corners
}

// Terrain draws tile tops split in 2x2 jittered quads and cliff faces wherever a
// neighbour is lower, which is what makes the ground read as 2.5D.
// tileCache samples the visible tiles once per frame; corners and cliff
// sides ask for the same neighbours many times.
type tileCache struct {
	x0, y0, w, h int
	tiles        []Tile
	src          Heightmap
}

func newTileCache(src Heightmap, x0, y0, x1, y1 int) *tileCache {
	c := &tileCache{x0: x0, y0: y0, w: x1 - x0 + 1, h: y1 - y0 + 1, src: src}
	c.tiles = make([]Tile, c.w*c.h)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.tiles[y*c.w+x] = src.Tile(x0+x, y0+y)
		}
	}
	return c
}

func (c *tileCache) Tile(x, y int) Tile {
	tx, ty := x-c.x0, y-c.y0
	if tx < 0 || ty < 0 || tx >= c.w || ty >= c.h {
		return c.src.Tile(x, y)
	}
	return c.tiles[ty*c.w+tx]
}

func (c *Context) Terrain(src Heightmap, o TerrainOptions) {
	lo, hi := c.View.GroundRect(2 + o.Max*1.5)
	x0, y0 := int(math.Floor(lo.X)), int(math.Floor(lo.Y))
	x1, y1 := int(math.Ceil(hi.X)), int(math.Ceil(hi.Y))
	h := newTileCache(src, x0-1, y0-1, x1+1, y1+1)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			t := h.Tile(x, y)
			if t.Skip {
				continue
			}
			c.tileTop(h, x, y, t, o)
			c.tileSides(h, x, y, t)
		}
	}
}

// corner averages the top colors of the tiles meeting at a corner that share
// this tile's height, so smoothing never bleeds across a cliff or a shore.
func corner(h Heightmap, x, y int, own Tile) color.Color {
	var r, g, b, n float64
	for _, d := range [4][2]int{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}} {
		t := h.Tile(x+d[0], y+d[1])
		if t.Skip || math.Abs(t.Height-own.Height) > .01 || t.Top.Water != own.Top.Water {
			continue
		}
		cr, cg, cb := t.Top.Albedo.Float()
		r, g, b, n = r+cr, g+cg, b+cb, n+1
	}
	if n == 0 {
		return own.Top.Albedo
	}
	return color.FromFloat(r/n, g/n, b/n)
}

func (c *Context) tileTop(h Heightmap, x, y int, t Tile, o TerrainOptions) {
	fx, fy, z := float64(x), float64(y), t.Height
	if o.Smooth {
		cols := []color.Color{corner(h, x, y, t), corner(h, x+1, y, t), corner(h, x+1, y+1, t), corner(h, x, y+1, t)}
		c.FaceShaded([]geom.Vec3{geom.V3(fx, fy, z), geom.V3(fx+1, fy, z), geom.V3(fx+1, fy+1, z), geom.V3(fx, fy+1, z)}, cols, up(), t.Top)
		return
	}
	if o.Grain <= 0 {
		c.Face([]geom.Vec3{geom.V3(fx, fy, z), geom.V3(fx+1, fy, z), geom.V3(fx+1, fy+1, z), geom.V3(fx, fy+1, z)}, up(), t.Top)
		return
	}
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			jit := (noise.Unit(noise.Hash(x*2+i, y*2+j, o.Seed)) - .5) * 2 * o.Grain
			sx, sy := fx+float64(i)*.5, fy+float64(j)*.5
			c.Face([]geom.Vec3{geom.V3(sx, sy, z), geom.V3(sx+.5, sy, z), geom.V3(sx+.5, sy+.5, z), geom.V3(sx, sy+.5, z)}, up(), t.Top.Tint(1+jit))
		}
	}
}

func (c *Context) tileSides(h Heightmap, x, y int, t Tile) {
	fx, fy, z := float64(x), float64(y), t.Height
	type side struct {
		dx, dy int
		n      geom.Vec3
		a, b   geom.Vec2
	}
	for _, s := range [4]side{
		{0, 1, south(), geom.V2(fx, fy+1), geom.V2(fx+1, fy+1)},
		{0, -1, north(), geom.V2(fx, fy), geom.V2(fx+1, fy)},
		{-1, 0, west(), geom.V2(fx, fy), geom.V2(fx, fy+1)},
		{1, 0, east(), geom.V2(fx+1, fy), geom.V2(fx+1, fy+1)},
	} {
		n := h.Tile(x+s.dx, y+s.dy)
		if !n.Skip && n.Height >= z {
			continue
		}
		bottom := n.Height
		if n.Skip {
			bottom = math.Min(0, z) - 1
		}
		c.Face([]geom.Vec3{geom.V3(s.a.X, s.a.Y, bottom), geom.V3(s.b.X, s.b.Y, bottom), geom.V3(s.b.X, s.b.Y, z), geom.V3(s.a.X, s.a.Y, z)}, s.n, t.Side)
	}
}
