package session

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

const scriptTemplate = `-- %s
-- start(self) runs once, update(self, dt) every frame,
-- key(self, k) on a key press; return true to consume it.

function start(self)
end

function update(self, dt)
end

function key(self, k)
end
`

// NewScript writes a starter Lua file and returns its project path.
func (s *Session) NewScript(ctx context.Context, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	rel := strings.TrimSuffix(filepath.ToSlash(name), ".lua") + ".lua"
	if !strings.Contains(rel, "/") {
		rel = "scripts/" + rel
	}
	if locked(rel) {
		return "", lockedErr(rel)
	}
	abs := filepath.Join(s.dir, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err == nil {
		return "", fmt.Errorf("new script %s: %w", rel, fs.ErrExist)
	}
	text := fmt.Sprintf(scriptTemplate, filepath.Base(rel))
	if err := s.writeAsset(ctx, rel, []byte(text)); err != nil {
		return "", err
	}
	s.lib.PutScript(rel, text)
	return rel, nil
}

// RenameAsset moves a file and its .meta. References hold the GUID, so the
// scene, prefabs and scripts keep working without being rewritten.
func (s *Session) RenameAsset(ctx context.Context, from, to string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	from = s.lib.Resolve(from)
	to = filepath.ToSlash(strings.TrimPrefix(to, "/"))
	if !strings.Contains(to, "/") {
		to = filepath.ToSlash(filepath.Join(filepath.Dir(from), to))
	}
	return s.move(ctx, from, to)
}

// move renames one asset with its .meta and every place that names its path.
func (s *Session) move(ctx context.Context, from, to string) (string, error) {
	if locked(from) || locked(to) {
		return "", lockedErr(from)
	}
	if k, _ := asset.Classify(from); k != "" {
		if want := assetExt(from); !strings.HasSuffix(to, want) {
			to += want
		}
	}
	if err := asset.Move(ctx, s.dir, from, to); err != nil {
		return "", err
	}
	s.lib.Rename(from, to)
	if s.scenePath == from {
		s.scenePath = to
	}
	if s.proj.Manifest.Start == from {
		s.proj.Manifest.Start = to
		if err := project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), s.proj.Manifest); err != nil {
			return "", err
		}
	}
	s.touch()
	return to, nil
}

func assetExt(p string) string {
	for _, ext := range []string{asset.PrefabExt, asset.SceneExt, material.Ext, asset.ScriptExt, ".xp", ".json"} {
		if strings.HasSuffix(strings.ToLower(p), ext) {
			return ext
		}
	}
	return filepath.Ext(p)
}

// AbsPath resolves a project relative path for opening in other programs.
func (s *Session) AbsPath(rel string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, filepath.FromSlash(s.lib.Resolve(rel))), nil
}
