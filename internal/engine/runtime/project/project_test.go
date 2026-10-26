package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/script/vm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func TestUnit_CreateThenOpenStarter(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()

	// Act
	file, err := Create(ctx, dir, "My Game", nil)
	p, errOpen := Open(ctx, os.DirFS(dir), nil)

	// Assert
	if err != nil || errOpen != nil {
		t.Fatal(err, errOpen)
	}
	if p.File != "My-Game.ascdiiproj" || file == "" || p.Manifest.Version != "0.0.1" || p.Manifest.Engine != version.Engine {
		t.Fatalf("file %s manifest %+v", p.File, p.Manifest)
	}
	if b, err := os.ReadFile(dir + "/.gitignore"); err != nil || string(b) != "build/\n" {
		t.Fatalf("gitignore %q %v", b, err)
	}
	s, err := p.StartScene(ctx)
	if err != nil || len(s.Root.Children) != 1 || s.Root.Children[0].Component("camera") == nil {
		t.Fatalf("start scene should hold only a camera: %v", err)
	}
}

func TestUnit_CreateRefusesExisting(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	_, _ = Create(ctx, dir, "g", nil)

	// Act
	_, err := Create(ctx, dir, "g", nil)

	// Assert
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUnit_CreateFromTemplateRenames(t *testing.T) {
	// Arrange
	tpl := fstest.MapFS{
		"Template.ascdiiproj": {Data: []byte(`{"format":1,"name":"Template","version":"1.2.0","start":"scenes/a.scene.json"}`)},
		"scenes/a.scene.json": {Data: []byte(`{"version":1,"name":"a","entities":[]}`)},
	}
	dir := t.TempDir()
	ctx := context.Background()

	// Act
	_, err := Create(ctx, dir, "Copy", tpl)
	p, errOpen := Open(ctx, os.DirFS(dir), nil)

	// Assert
	if err != nil || errOpen != nil || p.Manifest.Name != "Copy" || p.Manifest.Version != "1.2.0" || p.Manifest.Start != "scenes/a.scene.json" {
		t.Fatalf("err %v %v %+v", err, errOpen, p)
	}
}

func TestUnit_OpenRequiresPacks(t *testing.T) {
	// Arrange
	fsys := fstest.MapFS{"g.ascdiiproj": {Data: []byte(`{"name":"g","start":"x","packs":["dragonfall"]}`)}}

	// Act
	_, err := Open(context.Background(), fsys, nil)

	// Assert
	if !errors.Is(err, ErrMissingModule) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_OpenWithoutManifestFails(t *testing.T) {
	// Arrange
	fsys := fstest.MapFS{"a.txt": {Data: []byte("x")}}

	// Act
	_, err := Open(context.Background(), fsys, nil)

	// Assert
	if !errors.Is(err, ErrNoManifest) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_CreateWritesMetaAndMigrateUsesGUIDs(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	if _, err := Create(ctx, dir, "g", nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/scripts.lua", []byte("x=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	made, err := EnsureMeta(ctx, dir, guid.Default())
	p, errOpen := Open(ctx, os.DirFS(dir), nil)
	s, _ := p.StartScene(ctx)
	e := s.Add("door", nil).With(vm.New(p.Library, "scripts.lua"))

	// Act
	n := Migrate(s, p.Library)

	// Assert
	if err != nil || errOpen != nil || len(made) != 1 || n != 1 {
		t.Fatalf("err %v %v made %v n %d", err, errOpen, made, n)
	}
	if v := e.Component("script").(*vm.Script).File; !guid.Valid(v) {
		t.Fatalf("file %q", v)
	}
	if _, err := os.Stat(dir + "/scenes/main.scene.json.meta"); err != nil {
		t.Fatal("scene has no meta")
	}
}

func TestUnit_OnlyNamedModulesAreInstalled(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	installed := false
	sample := Module{Name: "sample", Install: func(r *scene.Registry, _ *asset.Library) {
		installed = true
		r.Register("sample.thing", func() scene.Component { return nil }, scene.Meta{})
	}}
	if _, err := Create(ctx, dir, "Plain", nil); err != nil {
		t.Fatal(err)
	}

	// Act
	p, err := Open(ctx, os.DirFS(dir), []Module{sample})

	// Assert
	if err != nil || installed || slices.Contains(p.Registry.Kinds(), "sample.thing") {
		t.Fatalf("err %v installed %v", err, installed)
	}
}

func TestUnit_RenameSpritesKeepsMetaAndSkipsOtherJSON(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	sprite := `{"name":"t","palette":{"x":{"glyph":"X","fg":"#00ff22","bg":"#000000"}},"rows":["x"]}`
	_ = os.MkdirAll(filepath.Join(dir, "art"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "art", "tree.json"), []byte(sprite), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "art", "tree.json.meta"), []byte(`{"guid":"x"}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"volume":3}`), 0o644)

	// Act
	got, err := RenameSprites(context.Background(), dir)

	// Assert
	_, errMeta := os.Stat(filepath.Join(dir, "art", "tree.sprite.json.meta"))
	_, errOther := os.Stat(filepath.Join(dir, "settings.json"))
	if err != nil || len(got) != 1 || got[0] != "art/tree.sprite.json" || errMeta != nil || errOther != nil {
		t.Fatalf("err %v got %v meta %v other %v", err, got, errMeta, errOther)
	}
}
