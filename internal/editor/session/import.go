package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
)

var ErrUnsupported = errors.New("unsupported file type")

func defaultFolder(k asset.Kind) string {
	switch k {
	case asset.Model:
		return "models"
	case asset.Script:
		return "scripts"
	case asset.Prefab:
		return "prefabs"
	case asset.Scene:
		return "scenes"
	}
	return "art"
}

// Import copies files from outside the project into folder, or a folder by
// kind when folder is "", and returns their project paths. A model brings
// its .mtl files and textures, keeping their paths relative to it.
func (s *Session) Import(ctx context.Context, files []string, folder string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return nil, err
	}
	var out []string
	var errs []error
	for _, src := range files {
		rel, err := s.import1(ctx, src, folder)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, rel)
	}
	if len(out) > 0 {
		s.touch()
	}
	return out, errors.Join(errs...)
}

func (s *Session) import1(ctx context.Context, src, folder string) (string, error) {
	k, ok := asset.Classify(src)
	if !ok {
		return "", fmt.Errorf("import %s: %w", filepath.Base(src), ErrUnsupported)
	}
	if rel, err := filepath.Rel(s.dir, src); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel), nil // already part of the project
	}
	if folder == "" {
		folder = defaultFolder(k)
	}
	rel := s.freeName(path.Join(filepath.ToSlash(folder), filepath.Base(src)))
	b, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("import %s: %w", filepath.Base(src), err)
	}
	if err := s.writeAsset(ctx, rel, b); err != nil {
		return "", err
	}
	switch k {
	case asset.Model:
		return rel, s.importModel(ctx, src, rel)
	case asset.Script:
		s.lib.PutScript(rel, string(b))
	case asset.Prefab:
		s.lib.PutPrefab(rel, b)
	case asset.Art:
		if a, err := asset.DecodeArt(rel, b); err == nil {
			s.lib.PutArt(rel, a)
		}
	}
	return rel, nil
}

func (s *Session) importModel(ctx context.Context, src, rel string) error {
	dir := filepath.Dir(src)
	deps, err := mesh.Files(os.DirFS(dir), filepath.Base(src))
	if err != nil {
		return err
	}
	for _, f := range deps {
		if !local(f) {
			continue // outside the model's folder; the model draws without it
		}
		to := filepath.Join(s.dir, filepath.FromSlash(path.Join(path.Dir(rel), f)))
		if _, err := os.Stat(to); err == nil {
			continue // shared with a model imported before
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f)))
		if err != nil {
			continue
		}
		if err := writeFile(ctx, to, b); err != nil {
			return err
		}
	}
	m, err := mesh.Load(ctx, os.DirFS(s.dir), rel, mesh.Options{})
	if err != nil {
		return err
	}
	s.lib.PutModel(rel, m)
	return nil
}

func local(p string) bool {
	return p != "" && !path.IsAbs(p) && !strings.HasPrefix(path.Clean(p), "..") && !strings.Contains(p, ":")
}

// freeName appends " 2", " 3" and so on until rel names no existing file.
func (s *Session) freeName(rel string) string {
	ext := path.Ext(rel)
	for _, multi := range []string{asset.PrefabExt, asset.SceneExt} {
		if strings.HasSuffix(rel, multi) {
			ext = multi
		}
	}
	stem := strings.TrimSuffix(rel, ext)
	out := rel
	for n := 2; ; n++ {
		if _, err := os.Stat(filepath.Join(s.dir, filepath.FromSlash(out))); err != nil {
			return out
		}
		out = fmt.Sprintf("%s %d%s", stem, n, ext)
	}
}
