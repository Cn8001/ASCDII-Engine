package project

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
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
	if p.File != "My-Game.ascdiiproj" || file == "" || p.Manifest.Version != "0.0.1" || p.Manifest.Engine != "0.0.1" {
		t.Fatalf("file %s manifest %+v", p.File, p.Manifest)
	}
	if b, err := os.ReadFile(dir + "/.gitignore"); err != nil || string(b) != "build/\n" {
		t.Fatalf("gitignore %q %v", b, err)
	}
	s, err := p.StartScene(ctx)
	if err != nil || len(s.Root.Children) != 0 {
		t.Fatalf("start scene should be empty: %v", err)
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
	if !errors.Is(err, ErrMissingPack) {
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

func TestUnit_OnlyNamedPacksAreInstalled(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	installed := false
	sample := Pack{Name: "sample", Install: func(r *scene.Registry, _ *asset.Library) {
		installed = true
		r.Register("sample.thing", func() scene.Component { return nil }, scene.Meta{})
	}}
	if _, err := Create(ctx, dir, "Plain", nil); err != nil {
		t.Fatal(err)
	}

	// Act
	p, err := Open(ctx, os.DirFS(dir), []Pack{sample})

	// Assert
	if err != nil || installed || slices.Contains(p.Registry.Kinds(), "sample.thing") {
		t.Fatalf("err %v installed %v", err, installed)
	}
}
