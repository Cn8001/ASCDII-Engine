package recent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUnit_AddMovesToFrontAndDropsMissing(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	s := Store{Path: filepath.Join(dir, "cfg", "recent.json")}
	a, b := filepath.Join(dir, "a.ascdiiproj"), filepath.Join(dir, "b.ascdiiproj")
	for _, p := range []string{a, b} {
		_ = os.WriteFile(p, []byte("{}"), 0o644)
	}
	ctx := context.Background()

	// Act
	_, _ = s.Add(ctx, a)
	_, _ = s.Add(ctx, b)
	_ = os.Remove(b)
	list, err := s.Add(ctx, a)

	// Assert
	if err != nil || len(list) != 1 || list[0] != a {
		t.Fatalf("err %v list %v", err, list)
	}
}
