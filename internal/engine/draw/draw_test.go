package draw

import (
	"math"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/view"
)

func newContext(cols, rows int) *Context {
	buf := raster.New(cols, rows)
	buf.Clear(color.Color{})
	cam := view.Default()
	cam.Span = 20
	return &Context{Buf: buf, View: cam.View(buf.W, buf.H)}
}

func centerDot(c *Context, p geom.Vec3) *raster.Dot {
	x, y, _ := c.View.Project(p)
	return c.Buf.At(int(x), int(y))
}

func TestUnit_FrontBoxOccludesBackBox(t *testing.T) {
	// Arrange
	c := newContext(40, 20)
	red, green := M(color.RGB(200, 0, 0)), M(color.RGB(0, 200, 0))

	// Act
	c.Box(geom.Footprint(geom.V3(0, 1, 0), geom.V3(2, 1, 2)), red)
	c.Box(geom.Footprint(geom.V3(0, -1, 0), geom.V3(2, 1, 4)), green)

	// Assert
	if d := centerDot(c, geom.V3(0, 1.5, 1)); d.Albedo != red.Albedo {
		t.Fatalf("front face hidden: %+v", d.Albedo)
	}
}

func TestUnit_BoxTopHasUpNormal(t *testing.T) {
	// Arrange
	c := newContext(40, 20)

	// Act
	c.Box(geom.Footprint(geom.V3(0, 0, 0), geom.V3(4, 4, 1)), M(color.RGB(9, 9, 9)))

	// Assert
	d := centerDot(c, geom.V3(0, 0, 1))
	if n := d.Normal.Vec(); n.Z < .99 {
		t.Fatalf("normal %+v", n)
	}
	if math.Abs(float64(d.Z)-1) > .01 {
		t.Fatalf("world z %v", d.Z)
	}
}

func TestUnit_SpriteInFrontOfWallWins(t *testing.T) {
	// Arrange
	c := newContext(40, 20)
	wall, hero := M(color.RGB(90, 90, 90)), M(color.RGB(250, 200, 100))
	c.Box(geom.AABB{Min: geom.V3(-5, -1, 0), Max: geom.V3(5, 0, 6)}, wall)

	// Act
	c.Sprite(geom.V3(0, .4, 0), .1, false).Color(hero).Rect(-5, -40, 10, 40)

	// Assert
	x, y, _ := c.View.Project(geom.V3(0, .4, 0))
	top := c.Buf.At(int(x), int(y)-int(3*c.View.Zoom))
	if top.Albedo != hero.Albedo {
		t.Fatalf("wall poked through sprite: %+v", top.Albedo)
	}
}

func TestUnit_SpriteBehindWallIsHidden(t *testing.T) {
	// Arrange
	c := newContext(40, 20)
	wall, hero := M(color.RGB(90, 90, 90)), M(color.RGB(250, 200, 100))
	c.Box(geom.AABB{Min: geom.V3(-5, -1, 0), Max: geom.V3(5, 0, 6)}, wall)

	// Act
	c.Sprite(geom.V3(0, -2, 0), .1, false).Color(hero).Rect(-5, -20, 10, 20)

	// Assert
	x, y, _ := c.View.Project(geom.V3(0, -2, 1))
	if d := c.Buf.At(int(x), int(y)); d.Albedo != wall.Albedo {
		t.Fatalf("sprite drawn through wall: %+v", d.Albedo)
	}
}

func TestUnit_LaterSpriteCallsStackInFront(t *testing.T) {
	// Arrange
	c := newContext(20, 10)
	s := c.Sprite(geom.V3(0, 0, 0), .1, false)

	// Act
	s.Color(M(color.RGB(1, 0, 0))).Rect(-10, -20, 20, 20)
	s.Color(M(color.RGB(0, 2, 0))).Rect(-5, -15, 10, 10)

	// Assert
	x, y, _ := c.View.Project(geom.V3(0, 0, 0))
	if d := c.Buf.At(int(x), int(y)-int(1*c.View.Zoom)); d.Albedo != color.RGB(0, 2, 0) {
		t.Fatalf("got %+v", d.Albedo)
	}
}

type flat struct{}

func (flat) Tile(x, y int) Tile {
	if x == 0 && y == 0 {
		return Tile{Height: 2, Top: M(color.RGB(9, 9, 9)), Side: M(color.RGB(5, 5, 5))}
	}
	return Tile{Top: M(color.RGB(3, 3, 3)), Side: M(color.RGB(3, 3, 3))}
}

func TestUnit_TerrainDrawsCliffFaceTowardCamera(t *testing.T) {
	// Arrange
	c := newContext(40, 20)

	// Act
	c.Terrain(flat{}, TerrainOptions{Max: 2})

	// Assert
	if d := centerDot(c, geom.V3(.5, 1, 1)); d.Albedo != color.RGB(5, 5, 5) || d.Normal.Vec().Y < .99 {
		t.Fatalf("cliff missing: %+v", d)
	}
}

func TestUnit_ShadowAndLightAccumulate(t *testing.T) {
	// Arrange
	c := newContext(10, 5)

	// Act
	c.Shadow(geom.V3(1, 2, 0), 1, .5, .6)
	c.Light(light.Point{Radius: 3})

	// Assert
	if len(c.Shadows) != 1 || c.Shadows[0].Strength != .6 || len(c.Lights) != 1 {
		t.Fatalf("shadows %+v", c.Shadows)
	}
}

func TestUnit_SpriteWorldProjectsBackToArtPixel(t *testing.T) {
	// Arrange
	c := newContext(40, 20)
	s := c.Sprite(geom.V3(2, 1, 0), .1, true)

	// Act
	w := s.World(10, -30)
	x, y, _ := c.View.Project(w)

	// Assert
	p := s.pt(10, -30)
	if math.Abs(x-float64(p.X)) > 1 || math.Abs(y-float64(p.Y)) > 1 {
		t.Fatalf("world %+v projects to %f,%f want %+v", w, x, y, p)
	}
}

type twoTone struct{}

func (twoTone) Tile(x, y int) Tile {
	c := color.RGB(0, 0, 0)
	if x > 0 {
		c = color.RGB(200, 200, 200)
	}
	return Tile{Top: M(c), Side: M(c)}
}

func TestUnit_SmoothTerrainBlendsAcrossTiles(t *testing.T) {
	// Arrange
	c := newContext(40, 20)

	// Act
	c.Terrain(twoTone{}, TerrainOptions{Smooth: true})

	// Assert
	mid := centerDot(c, geom.V3(1, .5, 0)).Albedo
	if mid.R < 60 || mid.R > 160 {
		t.Fatalf("edge not blended: %+v", mid)
	}
}
