package components

import (
	"bytes"
	"fmt"
	"image"
	stdcolor "image/color"
	"math"
	"strings"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/draw"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/light"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/texture"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
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
	if len(reg.Kinds()) != 25 || !reg.Meta("box").Object || reg.Meta("player").Object {
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
	aim(s, geom.V3(0, 0, 2), 12)

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
	aim(s, geom.V3(0, 0, 2), 14)

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

func TestUnit_AnimatorFollowsControllerAndTimelineStillWorksWhenIdle(t *testing.T) {
	// Arrange
	lib := asset.NewLibrary(scene.NewRegistry())
	lib.PutController("anims/door.animator.json", &animator.Controller{
		Parameters:  []animator.Param{{Name: "open", Type: animator.Trigger}},
		States:      []animator.State{{Name: "shut", Clip: "shut"}, {Name: "rise", Clip: "rise"}},
		Start:       "shut",
		Transitions: []animator.Transition{{From: "shut", To: "rise", When: []animator.Cond{{Param: "open"}}}, {From: "rise", To: "shut", Exit: true}},
	})
	s := scene.New("t")
	door := s.Add("door", NewBox(geom.V3(1, 1, 1), color.RGB(1, 1, 1)))
	s.Clips = []anim.Clip{
		{Name: "shut", Target: "door", Mode: anim.Loop, Tracks: []anim.Track{{Prop: "pos.z", Keys: []anim.Key{{T: 0, V: 0.0}, {T: 1, V: 0.0}}}}},
		{Name: "rise", Target: "door", Mode: anim.Once, Tracks: []anim.Track{{Prop: "pos.z", Keys: []anim.Key{{T: 0, V: 0.0}, {T: 1, V: 4.0}}}}},
	}
	a := NewAnimator(lib)
	a.Controller = "anims/door.animator.json"
	c := &scene.Ctx{Scene: s}
	_ = s.ApplyClips()
	timeline := door.Local.Pos.Z

	// Act
	door.With(a)
	_ = s.Update(c, .1, nil)
	before := a.State
	_, err := a.Call(c, door, "trigger", []any{"open"})
	for i := 0; i < 6; i++ {
		_ = s.Update(c, .1, nil)
	}
	mid, midState := door.Local.Pos.Z, a.State
	for i := 0; i < 10; i++ {
		_ = s.Update(c, .1, nil)
	}

	// Assert
	if err != nil || before != "shut" || midState != "rise" || mid < 1 || a.State != "shut" || timeline != 0 {
		t.Fatalf("err %v before %s mid %s %.2f after %s", err, before, midState, mid, a.State)
	}
}

type contacts struct{ hits []string }

func (*contacts) Kind() string          { return "probe" }
func (*contacts) Props() []prop.Prop    { return nil }
func (*contacts) Set(string, any) error { return nil }
func (p *contacts) Collide(_ *scene.Ctx, _, other *scene.GameObject, trigger bool) {
	p.hits = append(p.hits, fmt.Sprintf("%s:%v", other.Name, trigger))
}

func step(s *scene.Scene, c *scene.Ctx, seconds float64) {
	for t := 0.0; t < seconds; t += 1.0 / 60 {
		_ = s.Update(c, 1.0/60, nil)
	}
}

func TestUnit_RigidbodyFallsLandsAndStopsAtWalls(t *testing.T) {
	// Arrange
	s := scene.New("t")
	c := &scene.Ctx{Scene: s}
	wall := NewBoxCollider()
	wall.Size = geom.V3(2, 10, 3)
	s.Add("wall", wall).At(5, 0, 0)
	probe := &contacts{}
	ball := s.Add("ball", NewCircleCollider(), NewRigidbody(), probe).At(0, 0, 4)
	rb := ball.Component("rigidbody").(*Rigidbody)
	rb.Velocity = geom.V3(3, 0, 0)
	rb.Drag = 0

	// Act
	step(s, c, 3)

	// Assert
	p := ball.Local.Pos
	if p.Z != 0 || p.X > 3.01 || p.X < 1 || len(probe.hits) != 1 || probe.hits[0] != "wall:false" {
		t.Fatalf("pos %+v hits %v", p, probe.hits)
	}
}

func TestUnit_TriggersReportButDoNotPush(t *testing.T) {
	// Arrange
	s := scene.New("t")
	c := &scene.Ctx{Scene: s}
	zone := NewCircleCollider()
	zone.Trigger, zone.Radius = true, 3
	zp := &contacts{}
	s.Add("zone", zone, zp).At(4, 0, 0)
	ball := s.Add("ball", NewCircleCollider(), NewRigidbody()).At(0, 0, 0)
	rb := ball.Component("rigidbody").(*Rigidbody)
	rb.Velocity, rb.Drag = geom.V3(6, 0, 0), 0

	// Act
	step(s, c, 1)

	// Assert
	if ball.Local.Pos.X < 5 || len(zp.hits) != 1 || zp.hits[0] != "ball:true" {
		t.Fatalf("x %.2f hits %v", ball.Local.Pos.X, zp.hits)
	}
}

func TestUnit_PlayerCannotWalkThroughCircleCollider(t *testing.T) {
	// Arrange
	s := scene.New("t")
	post := NewCircleCollider()
	post.Radius = 1.5
	s.Add("post", post).At(3, 0, 0)
	hero := s.Add("hero", NewBlob(.5, color.RGB(1, 1, 1))).With(NewPlayer())

	// Act
	play(s, []input.Key{'d'}, 2)

	// Assert
	if x := hero.Local.Pos.X; x > 1.5 || x < .5 {
		t.Fatalf("x %.2f", x)
	}
}

func TestUnit_DirectionalLightReplacesTheSun(t *testing.T) {
	// Arrange
	render := func(withLight bool) float64 {
		s := scene.New("t")
		s.Env.Auto, s.Env.Sun = false, light.Sun{Dir: geom.V3(0, 0, 1), Color: color.RGB(255, 255, 255), Intensity: 1}
		s.Env.Ambient = light.Ambient{}
		s.Add("floor", NewPlane(30, 30, color.RGB(200, 200, 200)))
		if withLight {
			sun := NewLight(1, color.RGB(255, 255, 255))
			sun.Type, sun.Intensity = "directional", .2
			s.Add("sun", sun)
		}
		get := func() *scene.Scene { return s }
		p := pipeline.New(time.Now, append(pipeline.Standard(), &scene.Geometry{Scene: get})...)
		f := pipeline.NewFrame(30, 12)
		aim(s, geom.V3(0, 0, 0), 10)
		scene.Render(p, f, s)
		d := f.Buf.At(f.Buf.W/2, f.Buf.H/2)
		return float64(d.Color.R)
	}

	// Act
	bright, dim := render(false), render(true)

	// Assert
	if dim >= bright*.5 {
		t.Fatalf("bright %.0f dim %.0f", bright, dim)
	}
}

func TestUnit_PaintRecolorsAndErasesOwnDots(t *testing.T) {
	// Arrange
	s := scene.New("t")
	box := s.Add("box", NewBox(geom.V3(4, 4, 4), color.RGB(0, 0, 255)))
	paint := &Paint{}
	paint.Add(Stroke{P: geom.V3(-1, 2, 2), R: 1.2, C: color.RGB(255, 0, 0)})
	paint.Add(Stroke{P: geom.V3(1.2, 2, 2), R: 1, Erase: true})
	box.With(paint)
	get := func() *scene.Scene { return s }
	p := pipeline.New(time.Now, &scene.Geometry{Scene: get}, &scene.Painted{Scene: get})
	f := pipeline.NewFrame(60, 30)
	aim(s, geom.V3(0, 0, 2), 12)
	var back bytes.Buffer
	_ = scene.Encode(&back, s)

	// Act
	scene.Render(p, f, s)
	reg := scene.NewRegistry()
	Register(reg, nil)
	loaded, err := scene.Decode(&back, reg, nil)

	// Assert
	red, blue := 0, 0
	for _, d := range f.Buf.Dots() {
		if d.Tag == uint32(box.ID) && d.Albedo.R > 200 {
			red++
		}
		if d.Tag == uint32(box.ID) && d.Albedo.B > 200 {
			blue++
		}
	}
	if red == 0 || blue == 0 {
		t.Fatalf("red %d blue %d", red, blue)
	}
	if err != nil || len(loaded.Find("box").Component("paint").(*Paint).Strokes) != 2 {
		t.Fatalf("strokes lost on save: %v", err)
	}
}

func TestUnit_PrewarmedEmitterDrawsWithoutTicking(t *testing.T) {
	// Arrange
	e := NewEmitter()
	b := raster.New(40, 20)
	c := &draw.Context{Buf: b, View: view.Default().View(b.W, b.H)}

	// Act
	e.Draw(c, scene.Identity())

	// Assert
	if e.sim == nil || e.sim.Count() == 0 {
		t.Fatal("a prewarmed emitter drew nothing before its first tick")
	}
}

// aim adds a camera object looking at target with span world units across.
func aim(s *scene.Scene, target geom.Vec3, span float64) {
	c := NewCamera()
	c.Span = span
	s.Add("camera", c).Local.Pos = target
}

func TestUnit_RigidbodyUnderTurnedParentMovesInWorldSpace(t *testing.T) {
	// Arrange
	s := scene.New("t")
	c := &scene.Ctx{Scene: s}
	wall := NewBoxCollider()
	wall.Size = geom.V3(2, 10, 3)
	s.Add("wall", wall).At(5, 0, 0)
	rig := s.Add("rig").At(2, -3, 1)
	rig.Local.Rot = geom.V3(0, 0, 90)
	ball := rig.Add("ball", NewCircleCollider(), NewRigidbody())
	ball.SetWorld(scene.Transform{Pos: geom.V3(0, 0, 4), Scale: geom.V3(1, 1, 1)})
	rb := ball.Component("rigidbody").(*Rigidbody)
	rb.Velocity = geom.V3(3, 0, 0)
	rb.Drag = 0

	// Act
	step(s, c, 3)

	// Assert
	p := ball.World().Pos
	if math.Abs(p.Z) > 1e-9 || p.X > 3.01 || p.X < 1 || math.Abs(p.Y) > 1e-9 {
		t.Fatalf("world pos %+v", p)
	}
}

func TestUnit_PlayerUnderTurnedParentWalksInWorldSpace(t *testing.T) {
	// Arrange
	s := scene.New("t")
	rig := s.Add("rig").At(-1, 2, 0)
	rig.Local.Rot = geom.V3(0, 0, 90)
	hero := rig.Add("hero", NewBlob(.5, color.RGB(200, 180, 120))).With(NewPlayer())
	hero.SetWorld(scene.Transform{Scale: geom.V3(1, 1, 1)})
	s.Add("wall", NewBox(geom.V3(1, 10, 2), color.RGB(90, 90, 90))).At(3, 0, 0).Blocking()

	// Act
	play(s, []input.Key{'d'}, .5)

	// Assert
	w := hero.World()
	if w.Pos.X <= .5 || w.Pos.X > 2.5-.6+1e-9 || math.Abs(w.Pos.Y) > 1e-9 || w.Flip {
		t.Fatalf("world %+v", w)
	}
}
