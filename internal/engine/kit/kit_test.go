package kit

import (
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
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
	if lamp.Node.(*Emitter).Active {
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
	for _, k := range reg.NodeKinds() {
		n, err := reg.NewNode(k)
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
	if len(reg.NodeKinds()) < 10 || len(reg.BehaviourKinds()) != 4 {
		t.Fatalf("kinds %v %v", reg.NodeKinds(), reg.BehaviourKinds())
	}
}
