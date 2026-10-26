package appdir

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUnit_OpenCreatesTheOneFolder(t *testing.T) {
	// Arrange
	base := t.TempDir()

	// Act
	dir, err := Open(context.Background(), base)

	// Assert
	info, statErr := os.Stat(dir)
	if err != nil || statErr != nil || !info.IsDir() || dir != filepath.Join(base, "ASCDIIEngine") {
		t.Fatalf("dir %q err %v %v", dir, err, statErr)
	}
}
