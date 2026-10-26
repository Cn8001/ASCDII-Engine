package players

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"runtime"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/launch"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

func TestUnit_ExtractWritesExecutable(t *testing.T) {
	// Arrange
	s := Source{Dirs: []fs.FS{fstest.MapFS{}, fstest.MapFS{"ascdii-player-linux-amd64": {Data: []byte("bin")}}}}
	dir := t.TempDir()

	// Act
	path, err := s.Extract(context.Background(), launch.Player, "linux", "amd64", dir)

	// Assert
	info, statErr := os.Stat(path)
	if err != nil || statErr != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("err %v %v", err, statErr)
	}
	if got := s.Available(); len(got) != 1 || got[0] != "linux" {
		t.Fatalf("available %v", got)
	}
}

func TestUnit_ExtractMissingReportsNoPlayer(t *testing.T) {
	// Arrange
	s := Source{Dirs: []fs.FS{fstest.MapFS{}}}

	// Act
	_, err := s.Extract(context.Background(), launch.Player, "windows", "amd64", t.TempDir())

	// Assert
	if !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_SourceTreeLayoutServesBothPlatforms(t *testing.T) {
	// Arrange
	s := Source{Dirs: []fs.FS{fstest.MapFS{
		"linux/ascdii-player":       {Data: []byte("elf")},
		"windows/ascdii-player.exe": {Data: []byte("pe")},
	}}}

	// Act
	got := s.Available()

	// Assert
	if len(got) != 2 {
		t.Fatalf("available %v", got)
	}
}

func TestUnit_CopiedPlayerOfOtherPlatformNeedsNoRename(t *testing.T) {
	// Arrange
	other, file := "windows", "ascdii-player.exe"
	if runtime.GOOS == "windows" {
		other, file = "linux", "ascdii-player"
	}
	s := Source{Dirs: []fs.FS{fstest.MapFS{file: {Data: []byte("other")}}}}

	// Act
	path, err := s.Extract(context.Background(), launch.Player, other, "amd64", t.TempDir())
	b, errRead := os.ReadFile(path)

	// Assert
	if err != nil || errRead != nil || string(b) != "other" {
		t.Fatalf("got %q %v %v", b, err, errRead)
	}
	if got := s.Available(); len(got) != 1 || got[0] != other {
		t.Fatalf("available %v", got)
	}
}

func TestUnit_WindowGamesStartFromWindowPlayer(t *testing.T) {
	// Arrange
	s := Source{Dirs: []fs.FS{fstest.MapFS{
		"linux/ascdii-player":        {Data: []byte("terminal")},
		"linux/ascdii-player-window": {Data: []byte("window")},
	}}}
	dir := t.TempDir()

	// Act
	path, err := s.Extract(context.Background(), For(project.Display{Mode: project.DisplayWindow}), "linux", "amd64", dir)
	b, errRead := os.ReadFile(path)

	// Assert
	if err != nil || errRead != nil || string(b) != "window" || For(project.Display{}) != launch.Player {
		t.Fatalf("got %q %v %v", b, err, errRead)
	}
}
