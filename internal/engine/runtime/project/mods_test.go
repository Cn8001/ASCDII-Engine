package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestUnit_ManifestKeepsModSettings(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	file, err := Create(ctx, dir, "demo", nil)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Open(ctx, os.DirFS(dir), nil)
	if err != nil {
		t.Fatal(err)
	}
	m := p.Manifest
	m.Mods = ModSettings{Enabled: true, API: "scripts/modapi.lua"}

	// Act
	err = SaveManifest(ctx, file, m)
	got, errRead := ReadManifest(os.DirFS(dir), filepath.Base(file))

	// Assert
	if err != nil || errRead != nil || got.Mods != m.Mods {
		t.Fatalf("err %v %v mods %+v", err, errRead, got.Mods)
	}
}

func TestUnit_LoadModsPutsAssetsUnderTheModFolder(t *testing.T) {
	// Arrange
	_, lib := Registry(nil)
	mods := fstest.MapFS{
		"coins/main.lua":     {Data: []byte("log('x')")},
		"coins/lib/util.lua": {Data: []byte("return 1")},
		"empty/readme.txt":   {Data: []byte("no main")},
		"stray.lua":          {Data: []byte("not in a folder")},
	}

	// Act
	sources, errs := LoadMods(context.Background(), lib, mods)

	// Assert
	if len(errs) != 0 || len(sources) != 1 || sources[0].Name != "coins" || sources[0].Main != "log('x')" {
		t.Fatalf("errs %v sources %+v", errs, sources)
	}
	if _, _, err := lib.Script("mods/coins/lib/util.lua"); err != nil {
		t.Fatal(err)
	}
}
