package components

import (
	"image"
	stdcolor "image/color"
	"math"
	"strings"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/texture"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func play(s *scene.Scene, keys []input.Key, seconds float64) *scene.Ctx {
	c := &scene.Ctx{Scene: s}
	_ = s.Update(c, 0, keys)
	for t := 0.0; t < seconds; t += 1.0 / 60 {
		_ = s.Update(c, 1.0/60, nil)
	}
	return c
}

func TestUnit_PlayerWalksAndStopsAtWall(t *testing.T) {
	// Arrange
	s := scene.New("t")
	hero := s.Add("hero", NewBlob(.5, color.RGB(200, 180, 120))).With(NewPlayer())
	s.Add("wall", NewBox(geom.V3(1, 10, 2), color.RGB(90, 90, 90))).At(3, 0, 0).Blocking()

	// Act
	play(s, []input.Key{'d'}, .5)

	// Assert
	x := hero.Local.Pos.X
	if x <= .5 || x > 2.5-.6+1e-9 {
		t.Fatalf("hero x %f", x)
	}
	if hero.Local.Flip {
		t.Fatal("facing right expected")
	}
}

func TestUnit_PlayerJumpReturnsToGround(t *testing.T) {
	// Arrange
	s := scene.New("t")
	hero := s.Add("hero", NewBlob(.5, color.RGB(200, 180, 120))).With(NewPlayer())

	// Act
	c := &scene.Ctx{Scene: s}
	_ = s.Update(c, 0, []input.Key{' '})
	_ = s.Update(c, .2, nil)
	mid := hero.Local.Pos.Z
	play(s, nil, 2)

	// Assert
	if mid <= 0 || hero.Local.Pos.Z != 0 {
		t.Fatalf("mid %f end %f", mid, hero.Local.Pos.Z)
	}
}

func TestUnit_ToggleFlipsTargetProperty(t *testing.T) {
	// Arrange
	s := scene.New("t")
	lamp := s.Add("lamp", NewEmitter())
	s.Add("switch", nil).With(NewToggle("e", "lamp", "active"))

	// Act
	play(s, []input.Key{'e'}, 0)

	// Assert
	if lamp.Component("emitter").(*Emitter).Active {
		t.Fatal("toggle did not switch")
	}
}

func TestUnit_DayNightRunsClockForward(t *testing.T) {
	// Arrange
	s := scene.New("t")
	s.Env.Hour = 16.5
	s.Add("clock", nil).With(NewDayNight())

	// Act
	play(s, []input.Key{'t'}, 3)

	// Assert
	if !IsNight(s.Env.Hour) {
		t.Fatalf("hour %f", s.Env.Hour)
	}
}

func TestUnit_QuitSetsFlag(t *testing.T) {
	// Arrange
	s := scene.New("t")
	s.Add("q", nil).With(&Quit{Press: "q"})

	// Act
	c := play(s, []input.Key{'q'}, 0)

	// Assert
	if !c.Quit {
		t.Fatal("quit not requested")
	}
}

func TestUnit_RegisteredKindsRender(t *testing.T) {
	// Arrange
	reg := scene.NewRegistry()
	Register(reg, nil)
	s := scene.New("t")
	for _, k := range reg.Kinds() {
		n, err := reg.New(k)
		if err != nil {
			t.Fatal(err)
		}
		s.Add(k, n)
	}
	get := func() *scene.Scene { return s }
	p := pipeline.New(time.Now, append(pipeline.Standard(), &scene.Geometry{Scene: get}, &scene.Cells{Scene: get})...)
	f := pipeline.NewFrame(60, 20)

	// Act
	scene.Render(p, f, s)

	// Assert
	if len(reg.Kinds()) != 17 || !reg.Meta("box").Object || reg.Meta("player").Object {
		t.Fatalf("kinds %v", reg.Kinds())
	}
}

func TestUnit_MeshDrawsRotatedModel(t *testing.T) {
	// Arrange
	lib := asset.NewLibrary(scene.NewRegistry())
	m, _, err := mesh.ParseOBJ(strings.NewReader("v 0 0 0\nv 4 0 0\nv 4 4 0\nv 0 4 0\nf 1 2 3 4\n"), mesh.Options{})
	if err != nil {
		t.Fatal(err)
	}
	lib.PutModel("models/wall.obj", m)
	s := scene.New("t")
	o := s.Add("wall", NewMesh(lib, "models/wall.obj"))
	o.Local.Rot = geom.V3(0, 0, 30)
	get := func() *scene.Scene { return s }
	p := pipeline.New(time.Now, append(pipeline.Standard(), &scene.Geometry{Scene: get})...)
	f := pipeline.NewFrame(60, 30)
	s.Camera.Target, s.Camera.Span = geom.V3(0, 0, 2), 12

	// Act
	scene.Render(p, f, s)
	b := o.Bounds()

	// Assert
	drawn := 0
	for _, d := range f.Buf.Dots() {
		if d.Tag == uint32(o.ID) {
			drawn++
		}
	}
	if drawn == 0 || math.Abs(b.Max.X-4*math.Cos(math.Pi/6)) > 1e-9 || math.Abs(b.Max.Y-2) > 1e-9 || math.Abs(b.Max.Z-4) > 1e-9 {
		t.Fatalf("drawn %d bounds %+v", drawn, b)
	}
}

func TestUnit_TexturedRotatedBoxSamplesItsMaterial(t *testing.T) {
	// Arrange
	lib := asset.NewLibrary(scene.NewRegistry())
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for i, c := range []stdcolor.NRGBA{{R: 255, A: 255}, {B: 255, A: 255}, {B: 255, A: 255}, {R: 255, A: 255}} {
		img.Set(i%2, i/2, c)
	}
	lib.PutTexture("textures/check.png", texture.FromImage(img, texture.Options{Filter: texture.Nearest}))
	mat := material.Default()
	mat.Texture = "textures/check.png"
	lib.PutMaterial("materials/check.material.json", mat)
	box := NewBox(geom.V3(4, 4, 4), color.RGB(0, 255, 0))
	box.lib, box.Material = lib, "materials/check.material.json"
	s := scene.New("t")
	o := s.Add("box", box)
	o.Local.Rot = geom.V3(0, 0, 30)
	get := func() *scene.Scene { return s }
	p := pipeline.New(time.Now, &scene.Geometry{Scene: get})
	f := pipeline.NewFrame(60, 30)
	s.Camera.Target, s.Camera.Span = geom.V3(0, 0, 2), 14

	// Act
	scene.Render(p, f, s)

	// Assert
	red, blue, green := 0, 0, 0
	for _, d := range f.Buf.Dots() {
		if d.Tag != uint32(o.ID) {
			continue
		}
		switch {
		case d.Albedo.R > 200 && d.Albedo.B < 50:
			red++
		case d.Albedo.B > 200 && d.Albedo.R < 50:
			blue++
		case d.Albedo.G > 200:
			green++
		}
	}
	if red == 0 || blue == 0 || green != 0 {
		t.Fatalf("red %d blue %d green %d", red, blue, green)
	}
}
