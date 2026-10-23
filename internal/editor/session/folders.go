package session

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

var ErrBadPath = errors.New("path leaves the project")

// clean checks a project relative path; "" is the project folder itself.
func clean(rel string) (string, error) {
	rel = path.Clean("/" + filepath.ToSlash(rel))[1:]
	hidden := rel != "" && strings.HasPrefix(path.Base(rel), ".")
	if rel == project.BuildDir || strings.HasPrefix(rel, project.BuildDir+"/") || hidden {
		return "", fmt.Errorf("%s: %w", rel, ErrBadPath)
	}
	return rel, nil
}

// Folders lists the project's folders, the tree of the Project window.
func (s *Session) Folders(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	dir, err := s.dir, s.need()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	out := []string{}
	err = fs.WalkDir(os.DirFS(dir), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !d.IsDir() || p == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || p == project.BuildDir {
			return fs.SkipDir
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Session) CreateFolder(ctx context.Context, rel string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	rel, err := clean(rel)
	if err != nil || rel == "" {
		return "", fmt.Errorf("create folder %q: %w", rel, ErrBadPath)
	}
	if locked(rel) {
		return "", lockedErr(rel)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	rel = s.freeName(rel)
	if err := os.MkdirAll(filepath.Join(s.dir, filepath.FromSlash(rel)), 0o755); err != nil {
		return "", fmt.Errorf("create folder: %w", err)
	}
	return rel, nil
}

// Move puts an asset or a folder into folder ("" for the project folder).
// Assets keep their GUIDs, so nothing that references them breaks.
func (s *Session) Move(ctx context.Context, from, folder string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	from = s.lib.Resolve(from)
	folder, err := clean(folder)
	if err != nil {
		return "", err
	}
	to := path.Join(folder, path.Base(from))
	if to == from {
		return to, nil
	}
	info, err := os.Stat(filepath.Join(s.dir, filepath.FromSlash(from)))
	if err != nil {
		return "", fmt.Errorf("move %s: %w", from, err)
	}
	if !info.IsDir() {
		return s.move(ctx, from, to)
	}
	return s.moveFolder(ctx, from, to)
}

// RenameFolder gives a folder a new name in place.
func (s *Session) RenameFolder(ctx context.Context, from, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	if strings.ContainsAny(name, `/\`) || name == "" {
		return "", fmt.Errorf("rename %s: %w", from, ErrBadPath)
	}
	return s.moveFolder(ctx, from, path.Join(path.Dir(from), name))
}

func (s *Session) moveFolder(ctx context.Context, from, to string) (string, error) {
	from, err := clean(from)
	if err != nil || from == "" {
		return "", fmt.Errorf("move folder: %w", ErrBadPath)
	}
	if to, err = clean(to); err != nil {
		return "", err
	}
	if locked(from) || locked(to) {
		return "", lockedErr(from)
	}
	if strings.HasPrefix(to+"/", from+"/") {
		return "", fmt.Errorf("move %s into itself: %w", from, ErrBadPath)
	}
	if _, err := os.Stat(filepath.Join(s.dir, filepath.FromSlash(to))); err == nil {
		return "", fmt.Errorf("move %s: %w", to, asset.ErrExists)
	}
	entries, err := asset.List(ctx, os.DirFS(s.dir), project.BuildDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(s.dir, filepath.FromSlash(to))), 0o755); err != nil {
		return "", fmt.Errorf("move folder: %w", err)
	}
	if err := os.Rename(filepath.Join(s.dir, filepath.FromSlash(from)), filepath.Join(s.dir, filepath.FromSlash(to))); err != nil {
		return "", fmt.Errorf("move folder: %w", err)
	}
	for _, e := range entries {
		if rest, ok := strings.CutPrefix(e.Path, from+"/"); ok {
			moved := to + "/" + rest
			s.lib.Rename(e.Path, moved)
			if s.scenePath == e.Path {
				s.scenePath = moved
			}
			if s.proj.Manifest.Start == e.Path {
				s.proj.Manifest.Start = moved
				if err := project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), s.proj.Manifest); err != nil {
					return "", err
				}
			}
		}
	}
	s.touch()
	return to, nil
}

// Delete removes an asset with its .meta, or a folder with everything in
// it; references to what was deleted show as missing.
func (s *Session) Delete(ctx context.Context, rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	rel, err := clean(s.lib.Resolve(rel))
	if err != nil || rel == "" {
		return fmt.Errorf("delete: %w", ErrBadPath)
	}
	if locked(rel) {
		return fmt.Errorf("delete %s: remove the package in the package manager: %w", rel, ErrPackageLocked)
	}
	if rel == s.scenePath {
		return fmt.Errorf("delete %s: %w", rel, ErrNotEditable) // the open scene
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	abs := filepath.Join(s.dir, filepath.FromSlash(rel))
	entries, err := asset.List(ctx, os.DirFS(s.dir), project.BuildDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Path == rel || strings.HasPrefix(e.Path, rel+"/") {
			s.lib.Remove(e.Path)
		}
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("delete %s: %w", rel, err)
	}
	if err := os.Remove(abs + asset.MetaExt); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete %s: %w", rel, err)
	}
	s.touch()
	return nil
}
