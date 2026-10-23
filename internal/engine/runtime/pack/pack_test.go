package pack

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

const guid = `{"format":1,"guid":"019a0000-0000-7000-8000-000000000001"}`

func source() Source {
	return Source{Manifest: Manifest{Name: "props", Version: "1.0.0", Modules: []string{"extra"}}, FS: fstest.MapFS{
		"prefabs/crate.prefab.json":      {Data: []byte(`{}`)},
		"prefabs/crate.prefab.json.meta": {Data: []byte(guid)},
		"Game.ascdiiproj":                {Data: []byte(`{}`)},
		".git/config":                    {Data: []byte(``)},
		"build/game":                     {Data: []byte(``)},
	}}
}

func nobody(string) (string, bool) { return "", false }

func TestUnit_InstallCopiesAssetsAndManifest(t *testing.T) {
	// Arrange
	dir := t.TempDir()

	// Act
	err := Install(context.Background(), dir, source(), nobody)
	list, errList := Installed(os.DirFS(dir))

	// Assert
	if err != nil || errList != nil || len(list) != 1 || list[0].Modules[0] != "extra" {
		t.Fatalf("err %v %v list %+v", err, errList, list)
	}
	for _, f := range []string{"prefabs/crate.prefab.json", "prefabs/crate.prefab.json.meta", "pack.json"} {
		if _, err := os.Stat(filepath.Join(dir, "packages", "props", f)); err != nil {
			t.Fatalf("missing %s", f)
		}
	}
	for _, f := range []string{"Game.ascdiiproj", ".git", "build"} {
		if _, err := os.Stat(filepath.Join(dir, "packages", "props", f)); err == nil {
			t.Fatalf("%s should not be installed", f)
		}
	}
}

func TestUnit_InstallRefusesGUIDClashButAllowsUpgrade(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	elsewhere := func(string) (string, bool) { return "prefabs/mine.prefab.json", true }
	itself := func(string) (string, bool) { return "packages/props/prefabs/crate.prefab.json", true }

	// Act
	clash := Install(ctx, dir, source(), elsewhere)
	upgrade := Install(ctx, dir, source(), itself)

	// Assert
	if !errors.Is(clash, ErrConflict) || upgrade != nil {
		t.Fatalf("clash %v upgrade %v", clash, upgrade)
	}
}

func TestUnit_ExportOpensBackAndRemoveDeletes(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	file := filepath.Join(dir, "props"+Ext)
	var b bytes.Buffer

	// Act
	err := Export(ctx, &b, source().FS, source().Manifest)
	_ = os.WriteFile(file, b.Bytes(), 0o644)
	src, closer, errOpen := Open(file)
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	defer closer.Close()
	files, _ := Files(src.FS)
	errInstall := Install(ctx, dir, src, nobody)
	errRemove := Remove(ctx, dir, "props")
	bad := Export(ctx, &b, source().FS, Manifest{Name: "Bad Name"})

	// Assert
	if err != nil || errInstall != nil || errRemove != nil || src.Manifest.Version != "1.0.0" || len(files) != 2 {
		t.Fatalf("errs %v %v %v manifest %+v files %v", err, errInstall, errRemove, src.Manifest, files)
	}
	if _, err := os.Stat(filepath.Join(dir, "packages", "props")); err == nil || !errors.Is(bad, ErrBadName) {
		t.Fatalf("package still installed or bad name accepted: %v", bad)
	}
}
