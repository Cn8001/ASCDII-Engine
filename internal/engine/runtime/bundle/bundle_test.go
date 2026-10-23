package bundle

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
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
	if err1 != nil || err2 != nil || !bytes.Equal(a, b) {
		t.Fatalf("err %v %v nested %d vs %d", err1, err2, len(a), len(b))
	}
}

func TestUnit_ReleaseBuildsEveryTarget(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	for _, n := range []string{"lin", "win.exe"} {
		if err := os.WriteFile(dir+"/"+n, []byte("PLAYER-"+n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	targets := []Target{{OS: "linux", Arch: "amd64", Player: dir + "/lin"}, {OS: "windows", Arch: "amd64", Player: dir + "/win.exe"}}
	src := fstest.MapFS{"g.ascdiiproj": {Data: []byte(`{}`)}}

	// Act
	rel, err := MakeRelease(context.Background(), dir+"/out", "My Game", "v1.2.0", targets, src)

	// Assert
	if err != nil || len(rel.Files) != 5 {
		t.Fatalf("err %v files %v", err, rel.Files)
	}
	sums, _ := os.ReadFile(rel.Dir + "/SHA256SUMS")
	if !strings.Contains(string(sums), "My-Game-1.2.0-windows-amd64.exe.zip") {
		t.Fatalf("sums %s", sums)
	}
}
