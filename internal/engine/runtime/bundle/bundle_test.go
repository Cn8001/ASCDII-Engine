package bundle

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
)

func TestUnit_WriteThenReadRoundTrips(t *testing.T) {
	// Arrange
	src := fstest.MapFS{
		"g.ascdiiproj":        {Data: []byte(`{"name":"g"}`)},
		"scenes/a.scene.json": {Data: []byte(`{}`)},
		"build/old":           {Data: []byte("stale")},
	}
	var out bytes.Buffer
	ctx := context.Background()

	// Act
	err := Write(ctx, &out, strings.NewReader("PLAYERBINARY"), src, "build")
	fsys, errRead := Read(ctx, bytes.NewReader(out.Bytes()), int64(out.Len()))

	// Assert
	if err != nil || errRead != nil {
		t.Fatal(err, errRead)
	}
	if !bytes.HasPrefix(out.Bytes(), []byte("PLAYERBINARY")) {
		t.Fatal("player bytes lost")
	}
	b, err := fs.ReadFile(fsys, "scenes/a.scene.json")
	if err != nil || string(b) != "{}" {
		t.Fatalf("read %v %q", err, b)
	}
	if _, err := fs.Stat(fsys, "build/old"); err == nil {
		t.Fatal("build dir must be skipped")
	}
}

func TestUnit_ReadPlainBinaryReportsNoBundle(t *testing.T) {
	// Arrange
	plain := bytes.Repeat([]byte("x"), 64)

	// Act
	_, err := Read(context.Background(), bytes.NewReader(plain), int64(len(plain)))

	// Assert
	if !errors.Is(err, ErrNoBundle) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_BuildFromBundledPlayerDoesNotNest(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	player := dir + "/player"
	if err := os.WriteFile(player, []byte("ELF-PLAYER"), 0o755); err != nil {
		t.Fatal(err)
	}
	first, second := dir+"/a", dir+"/b"
	src := fstest.MapFS{"g.ascdiiproj": {Data: []byte(`{}`)}}

	// Act
	err1 := Build(ctx, first, player, src)
	err2 := Build(ctx, second, first, src)

	// Assert
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if err1 != nil || err2 != nil || len(a) != len(b) { // salts differ, sizes match unless the bundle nested
		t.Fatalf("err %v %v nested %d vs %d", err1, err2, len(a), len(b))
	}
}

const meta = `{"format":1,"guid":"019a0000-0000-7000-8000-000000000001"}`

func TestUnit_FolderBuildPacksOnlyGameData(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	player := filepath.Join(dir, "player")
	if err := os.WriteFile(player, []byte("PLAYER"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := fstest.MapFS{
		"g.ascdiiproj":             {Data: []byte(`{}`)},
		"scenes/a.scene.json":      {Data: []byte(`{}`)},
		"scenes/a.scene.json.meta": {Data: []byte(meta)},
		"README.md":                {Data: []byte("notes")},
		".gitignore":               {Data: []byte("build/")},
		"build/old":                {Data: []byte("x")},
	}

	// Act
	exe, err := BuildFolder(context.Background(), filepath.Join(dir, "Game"), "game", player, src, "build")
	fsys, errData := OpenData(exe)
	_, none := OpenData(player)
	raw, _ := os.ReadFile(filepath.Join(dir, "Game", DataDir, PakFile))
	sums, _ := os.ReadFile(filepath.Join(dir, "Game", SumsFile))

	// Assert
	if err != nil || errData != nil || !errors.Is(none, ErrNoData) {
		t.Fatalf("errs %v %v %v", err, errData, none)
	}
	if _, err := fs.Stat(fsys, "scenes/a.scene.json"); err != nil {
		t.Fatal("the scene is missing from the pak")
	}
	for _, gone := range []string{"README.md", ".gitignore", "build/old", "scenes/a.scene.json.meta"} {
		if _, err := fs.Stat(fsys, gone); err == nil {
			t.Fatalf("%s should not be in a build", gone)
		}
	}
	if m, err := asset.ReadMeta(fsys, "scenes/a.scene.json"); err != nil || m.GUID == "" {
		t.Fatalf("the asset index lost the guid: %v", err)
	}
	if bytes.Contains(raw, []byte("scene.json")) {
		t.Fatal("the pak should not show file names in the clear")
	}
	if !strings.Contains(string(sums), "  game\n") || !strings.Contains(string(sums), "data/game.pak") {
		t.Fatalf("sums %s", sums)
	}
}
