package dragonfall

import (
	"bytes"
	"context"
	"io/fs"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
)

func open(t testing.TB) *app.Engine {
	t.Helper()
	e, err := app.Open(context.Background(), Template(), app.Options{Cols: 160, Rows: 50, Packs: []project.Pack{Pack()}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestUnit_TemplateMatchesGenerator(t *testing.T) {
	// Arrange
	want, err := TemplateFiles()
	if err != nil {
		t.Fatal(err)
	}

	// Act
	for name, b := range want {
		got, err := fs.ReadFile(Template(), name)

		// Assert
		if err != nil || !bytes.Equal(got, b) {
			t.Fatalf("%s is stale, run: go run ./internal/dragonfall/gen", name)
		}
	}
}

func TestUnit_SceneRendersBraille(t *testing.T) {
	// Arrange
	e := open(t)

	// Act
	g := e.Render()

	// Assert
	braille := 0
	for _, c := range g.Cells() {
		if c.Rune >= 0x2800 && c.Rune <= 0x28ff {
			braille++
		}
	}
	if braille < 2000 {
		t.Fatalf("world too sparse: %d braille cells", braille)
	}
}

func TestUnit_CastleAndDragonBlockTheirFootprints(t *testing.T) {
	// Arrange
	e := open(t)
	s := e.Scene()
	hero := s.Find("hero")

	// Act
	wall := s.Blocked(geom.V2(104, 68), hero)
	tower := s.Blocked(geom.V2(121, 68), hero)
	dragon := s.Blocked(geom.V2(143, 73), hero)
	road := s.Blocked(geom.V2(114, 76), hero)

	// Assert
	if !wall || !tower || !dragon || road {
		t.Fatalf("wall %v tower %v dragon %v road %v", wall, tower, dragon, road)
	}
}

func TestUnit_HeroCannotWalkThroughTheWall(t *testing.T) {
	// Arrange
	e := open(t)
	hero := e.Scene().Find("hero")
	hero.Local.Pos = geom.V3(104, 74, 0)

	// Act
	for i := 0; i < 90; i++ {
		keys := []input.Key{}
		if i%4 == 0 {
			keys = append(keys, 'w')
		}
		if _, err := e.Update(1.0/30, keys); err != nil {
			t.Fatal(err)
		}
	}

	// Assert
	if y := hero.Local.Pos.Y; y < 70 || y > 73 {
		t.Fatalf("hero y %f", y)
	}
	if !hero.Node.(*Hero).Walking {
		t.Fatal("player behaviour did not drive the walk cycle")
	}
}

func TestUnit_KeysToggleDragonAndMap(t *testing.T) {
	// Arrange
	e := open(t)

	// Act
	_, err := e.Update(.1, []input.Key{'e', 'm'})

	// Assert
	if err != nil || !e.Scene().Find("dragon").Node.(*Dragon).Awake || !e.Scene().Find("hud").Node.(*HUD).ShowMap {
		t.Fatalf("toggles failed: %v", err)
	}
}

func BenchmarkScene160x50(b *testing.B) {
	e := open(b)
	e.Scene().Find("dragon").Node.(*Dragon).Awake = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Update(1.0/30, nil)
		e.Render()
	}
}

// BenchmarkScene220x60 is the editor at full screen; it logs each pass so a
// slow frame points at its cause.
func BenchmarkScene220x60(b *testing.B) {
	e := open(b)
	e.Resize(220, 60)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Update(1.0/30, nil)
		e.Render()
	}
	b.StopTimer()
	for _, in := range e.Pipeline().Infos() {
		b.Logf("%-10s %6d us", in.Name, in.Micros)
	}
}
