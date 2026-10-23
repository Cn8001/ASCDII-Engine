package project

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// EnsureMeta gives every asset of a project folder a .meta with a UUIDv7.
func EnsureMeta(ctx context.Context, dir string, src guid.Source) ([]string, error) {
	return asset.EnsureMeta(ctx, dir, src, BuildDir)
}

// Migrate rewrites path references in a scene to GUIDs, for projects saved
// before .meta files existed. It returns how many references changed.
func Migrate(s *scene.Scene, lib *asset.Library) int {
	n := 0
	s.Walk(func(e *scene.GameObject, _ bool) bool {
		if e.Inherited() {
			return true
		}
		if e.Prefab != "" && !guid.Valid(e.Prefab) {
			if ref := lib.Ref(e.Prefab); ref != e.Prefab {
				e.Prefab = ref
				n++
			}
		}
		for _, p := range e.Props() {
			if !p.Type.Ref() {
				continue
			}
			v, _ := p.Value.(string)
			if v == "" || guid.Valid(v) {
				continue
			}
			if ref := lib.Ref(v); ref != v && e.Set(p.Name, ref) == nil {
				n++
			}
		}
		return true
	})
	return n
}

// RenameSprites gives sprites saved as plain .json, from before sprites had
// their own extension, the .sprite.json name. Their .meta moves along, so
// references by GUID still hold. It returns the new paths.
func RenameSprites(ctx context.Context, dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && (d.Name() == BuildDir || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".json") {
			return nil
		}
		if _, ok := asset.Classify(p); ok {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("rename sprites: %w", err)
		}
		if _, err := asset.DecodeSprite(p, b); err != nil {
			return nil // some other json, not ours to touch
		}
		to := strings.TrimSuffix(p, filepath.Ext(p)) + asset.SpriteExt
		if _, err := os.Stat(to); err == nil {
			return nil
		}
		if err := os.Rename(p, to); err != nil {
			return fmt.Errorf("rename sprites: %w", err)
		}
		if err := os.Rename(p+asset.MetaExt, to+asset.MetaExt); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("rename sprites: %w", err)
		}
		rel, _ := filepath.Rel(dir, to)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}
