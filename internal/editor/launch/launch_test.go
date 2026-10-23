package launch

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnit_FindPrefersProgramNextToSelf(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	name := Editor
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	_ = os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o755)
	t.Setenv("PATH", "")

	// Act
	got, err := Find(filepath.Join(dir, CLI), Editor)
	_, missing := Find(filepath.Join(dir, CLI), "nothing-here")

	// Assert
	if err != nil || got != filepath.Join(dir, name) || !errors.Is(missing, ErrNotFound) {
		t.Fatalf("got %q err %v missing %v", got, err, missing)
	}
}

func TestUnit_FromLauncherReadsTheMark(t *testing.T) {
	// Arrange
	t.Setenv(Env, "")
	before := FromLauncher()

	// Act
	t.Setenv(Env, "1")

	// Assert
	if before || !FromLauncher() {
		t.Fatalf("before %v after %v", before, FromLauncher())
	}
}
