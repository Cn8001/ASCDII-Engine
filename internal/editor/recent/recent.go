// Package recent remembers the projects opened in the editor.
package recent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

const limit = 12

// Store is the JSON file the list lives in; the editor keeps it in its app
// folder (appdir.Recent).
type Store struct{ Path string }

func (s Store) Load(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load recent: %w", err)
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("load recent: %w", err)
	}
	return out, nil
}

// Add moves path to the front and drops entries that no longer exist.
func (s Store) Add(ctx context.Context, path string) ([]string, error) {
	list, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	list = slices.DeleteFunc(list, func(p string) bool {
		_, err := os.Stat(p)
		return p == path || err != nil
	})
	list = append([]string{path}, list...)
	if len(list) > limit {
		list = list[:limit]
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("save recent: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return nil, fmt.Errorf("save recent: %w", err)
	}
	if err := os.WriteFile(s.Path, b, 0o644); err != nil {
		return nil, fmt.Errorf("save recent: %w", err)
	}
	return list, nil
}
