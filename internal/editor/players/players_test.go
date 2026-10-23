package players

import (
	"context"
	"errors"
	"os"
	"testing"
	"testing/fstest"
)

func TestUnit_ExtractWritesExecutable(t *testing.T) {
	// Arrange
	s := Source{FS: fstest.MapFS{"ascdii-linux-amd64": {Data: []byte("bin")}}}
	dir := t.TempDir()

	// Act
	path, err := s.Extract(context.Background(), "linux", "amd64", dir)

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
	s := Source{FS: fstest.MapFS{}}

	// Act
	_, err := s.Extract(context.Background(), "windows", "amd64", t.TempDir())

	// Assert
	if !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("got %v", err)
	}
}
