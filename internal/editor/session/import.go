package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
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
	case asset.Texture:
		return "textures"
	case asset.Material:
		return "materials"
	case asset.Audio:
		return "audio"
	case asset.Animator:
		return "animators"
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
	if locked(folder) {
		return nil, lockedErr(folder)
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
	if rel, err := filepath.Rel(s.dir, src); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel), nil // already part of the project
	}
	info, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("import %s: %w", filepath.Base(src), err)
	}
	if info.IsDir() {
		return s.importDir(ctx, src, folder)
	}
	k, ok := asset.Classify(src)
	if !ok {
		return "", fmt.Errorf("import %s: %w", filepath.Base(src), ErrUnsupported)
	}
	if folder == "" {
		folder = defaultFolder(k)
	}
	rel := s.freeName(path.Join(filepath.ToSlash(folder), filepath.Base(src)))
	if err := s.copyAsset(ctx, src, rel); err != nil {
		return "", err
	}
	if k == asset.Model {
		return rel, s.importModel(ctx, src, rel)
	}
	return rel, nil
}

// importDir copies a folder as it is, metas included, which is how a
// prefab arrives together with its scripts, models and art.
func (s *Session) importDir(ctx context.Context, src, folder string) (string, error) {
	root := s.freeName(path.Join(filepath.ToSlash(folder), filepath.Base(src)))
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != src {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || strings.HasSuffix(p, asset.MetaExt) {
			return nil // metas travel with their asset
		}
		in, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		rel := path.Join(root, filepath.ToSlash(in))
		if _, ok := asset.Classify(p); !ok {
			b, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("import %s: %w", in, err)
			}
			return writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(rel)), b)
		}
		return s.copyAsset(ctx, p, rel)
	})
	if err != nil {
		return "", err
	}
	if err := s.lib.Load(ctx, os.DirFS(s.dir), project.BuildDir); err != nil {
		return "", err
	}
	return root, s.current().Instantiate()
}

// copyAsset copies one asset. A .meta next to the source comes along, so
// the asset keeps its GUID and whatever referenced it by that GUID still
// finds it; a GUID the project already uses gets replaced by a new one.
func (s *Session) copyAsset(ctx context.Context, src, rel string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("import %s: %w", filepath.Base(src), err)
	}
	if m, err := asset.ReadMeta(os.DirFS(filepath.Dir(src)), filepath.Base(src)); err == nil && !s.lib.Known(m.GUID) {
		raw, err := os.ReadFile(src + asset.MetaExt)
		if err != nil {
			return fmt.Errorf("import %s: %w", filepath.Base(src), err)
		}
		if err := writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(rel))+asset.MetaExt, raw); err != nil {
			return err
		}
		s.lib.SetGUID(rel, m.GUID)
	}
	if err := s.writeAsset(ctx, rel, b); err != nil {
		return err
	}
	k, _ := asset.Classify(rel)
	switch k {
	case asset.Script:
		s.lib.PutScript(rel, string(b))
	case asset.Prefab:
		s.lib.PutPrefab(rel, b)
	case asset.Art:
		if a, err := asset.DecodeArt(rel, b); err == nil {
			s.lib.PutArt(rel, a)
		}
	case asset.Texture:
		if t, err := asset.LoadTexture(os.DirFS(s.dir), rel); err == nil {
			s.lib.PutTexture(rel, t)
		}
	case asset.Material:
		if m, err := material.Decode(bytes.NewReader(b)); err == nil {
			s.lib.PutMaterial(rel, m)
		}
	}
	return nil
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
		dep := path.Join(path.Dir(rel), f)
		if k, _ := asset.Classify(dep); k != asset.Texture {
			if err := writeFile(ctx, to, b); err != nil {
				return err
			}
			continue
		}
		if err := s.copyAsset(ctx, filepath.Join(dir, filepath.FromSlash(f)), dep); err != nil {
			return err
		}
	}
	m, err := mesh.Load(ctx, os.DirFS(s.dir), rel, asset.ModelOptions(os.DirFS(s.dir), rel))
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
