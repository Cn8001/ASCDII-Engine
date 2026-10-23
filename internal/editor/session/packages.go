package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

var ErrPackageLocked = errors.New("package assets are read only")

// locked reports paths inside an installed package.
func locked(rel string) bool {
	rel = filepath.ToSlash(rel)
	return rel == pack.Dir || strings.HasPrefix(rel, pack.Dir+"/")
}

func lockedErr(rel string) error {
	return fmt.Errorf("%s: %w; remove the package or copy what you need", rel, ErrPackageLocked)
}

type Package struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Modules     []string `json:"modules"`
	Missing     []string `json:"missing"`   // modules this editor was built without
	Source      string   `json:"source"`    // "builtin:<name>" or a file path, for Install
	Installed   string   `json:"installed"` // version in the project, "" when not installed
	Files       int      `json:"files"`
}

type Packages struct {
	Installed []Package `json:"installed"`
	Available []Package `json:"available"`
}

func (s *Session) describe(m pack.Manifest, source string, fsys fs.FS) Package {
	p := Package{Name: m.Name, Version: m.Version, Description: m.Description, Author: m.Author, Modules: m.Modules, Source: source, Missing: []string{}}
	for _, need := range m.Modules {
		if !slices.ContainsFunc(s.d.Modules, func(x project.Module) bool { return x.Name == need }) {
			p.Missing = append(p.Missing, need)
		}
	}
	if files, err := pack.Files(fsys); err == nil {
		p.Files = len(files)
	}
	return p
}

// sources are the packages that can be installed: built into the editor,
// and every .ascdiipack in the user's package folder.
func (s *Session) sources() []pack.Source {
	out := append([]pack.Source(nil), s.d.Packages...)
	if s.d.PackageDir == "" {
		return out
	}
	entries, err := os.ReadDir(s.d.PackageDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), pack.Ext) {
			continue
		}
		src, closer, err := pack.Open(filepath.Join(s.d.PackageDir, e.Name()))
		if err != nil {
			continue
		}
		_ = closer.Close()
		out = append(out, pack.Source{Manifest: src.Manifest, Origin: src.Origin}) // reopened when listed or installed
	}
	return out
}

// PackageList is what the package manager shows.
func (s *Session) PackageList(ctx context.Context) (Packages, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return Packages{}, err
	}
	out := Packages{Installed: []Package{}, Available: []Package{}}
	installed, err := pack.Installed(os.DirFS(s.dir))
	if err != nil {
		return out, err
	}
	have := map[string]string{}
	for _, m := range installed {
		sub, err := fs.Sub(os.DirFS(s.dir), pack.Dir+"/"+m.Name)
		if err != nil {
			continue
		}
		p := s.describe(m, "", sub)
		p.Installed = m.Version
		out.Installed = append(out.Installed, p)
		have[m.Name] = m.Version
	}
	for _, src := range s.sources() {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		origin := src.Origin
		if origin == "" {
			origin = "builtin:" + src.Manifest.Name
		}
		if src.FS == nil {
			src2, closer, err := pack.Open(origin)
			if err != nil {
				continue
			}
			src = src2
			defer closer.Close()
		}
		p := s.describe(src.Manifest, origin, src.FS)
		p.Installed = have[src.Manifest.Name]
		out.Available = append(out.Available, p)
	}
	return out, nil
}

// open finds a package by its source: "builtin:<name>" or a file path.
func (s *Session) openSource(source string) (pack.Source, func(), error) {
	if name, ok := strings.CutPrefix(source, "builtin:"); ok {
		for _, src := range s.d.Packages {
			if src.Manifest.Name == name {
				return src, func() {}, nil
			}
		}
		return pack.Source{}, nil, fmt.Errorf("%s: %w", name, pack.ErrNotPack)
	}
	src, closer, err := pack.Open(source)
	if err != nil {
		return pack.Source{}, nil, err
	}
	return src, func() { _ = closer.Close() }, nil
}

// InstallPackage copies a package into the project and loads it. Modules
// it needs are added to the manifest and become available at once.
func (s *Session) InstallPackage(ctx context.Context, source string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", err
	}
	src, done, err := s.openSource(source)
	if err != nil {
		return "", err
	}
	defer done()
	var add []project.Module
	for _, need := range src.Manifest.Modules {
		if slices.Contains(s.proj.Manifest.Modules, need) {
			continue
		}
		i := slices.IndexFunc(s.d.Modules, func(m project.Module) bool { return m.Name == need })
		if i < 0 {
			return "", fmt.Errorf("package %s: %w: %s", src.Manifest.Name, project.ErrMissingModule, need)
		}
		add = append(add, s.d.Modules[i])
	}
	owner := func(guid string) (string, bool) {
		if !s.lib.Known(guid) {
			return "", false
		}
		return s.lib.Resolve(guid), true
	}
	if err := pack.Install(ctx, s.dir, src, owner); err != nil {
		return "", err
	}
	for _, m := range add {
		m.Install(s.reg, s.lib)
		s.proj.Manifest.Modules = append(s.proj.Manifest.Modules, m.Name)
	}
	if len(add) > 0 {
		if err := project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), s.proj.Manifest); err != nil {
			return "", err
		}
	}
	if err := s.lib.Load(ctx, os.DirFS(s.dir), project.BuildDir); err != nil {
		return "", err
	}
	s.touch()
	return pack.Dir + "/" + src.Manifest.Name, s.current().Instantiate()
}

// RemovePackage deletes an installed package. Its modules leave the
// manifest too, unless another package or the project's own scenes and
// prefabs still use their components.
func (s *Session) RemovePackage(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	home := pack.Dir + "/" + name
	if strings.HasPrefix(s.scenePath, home+"/") {
		return fmt.Errorf("remove %s: the open scene belongs to it; open another scene first: %w", name, ErrNotEditable)
	}
	installed, err := pack.Installed(os.DirFS(s.dir))
	if err != nil {
		return err
	}
	var gone pack.Manifest
	for _, m := range installed {
		if m.Name == name {
			gone = m
		}
	}
	entries, err := asset.List(ctx, os.DirFS(s.dir), project.BuildDir)
	if err != nil {
		return err
	}
	if err := pack.Remove(ctx, s.dir, name); err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Path, home+"/") {
			s.lib.Remove(e.Path)
		}
	}
	var open bytes.Buffer
	if err := scene.Encode(&open, s.edit); err != nil {
		return err
	}
	drop, err := project.Unneeded(ctx, os.DirFS(s.dir), s.d.Modules, gone, open.Bytes())
	if err != nil {
		return err
	}
	s.touch()
	if len(drop) == 0 {
		return s.edit.Instantiate()
	}
	s.proj.Manifest.Modules = slices.DeleteFunc(s.proj.Manifest.Modules, func(m string) bool { return slices.Contains(drop, m) })
	if err := project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), s.proj.Manifest); err != nil {
		return err
	}
	return s.reload(ctx)
}

// reload rebuilds the registry and library from disk after the module set
// changed, keeping the open scene with its unsaved edits.
func (s *Session) reload(ctx context.Context) error {
	var b bytes.Buffer
	if err := scene.Encode(&b, s.edit); err != nil {
		return err
	}
	p, err := project.Open(ctx, os.DirFS(s.dir), s.d.Modules)
	if err != nil {
		return err
	}
	sc, err := scene.Decode(&b, p.Registry, p.Library)
	if err != nil {
		return err
	}
	unsaved, path := s.unsaved, s.scenePath
	s.proj, s.reg, s.lib = p, p.Registry, p.Library
	s.load(sc, path)
	s.unsaved = unsaved
	return nil
}

// ExportPackage writes folder, metas included, as a .ascdiipack at out.
func (s *Session) ExportPackage(ctx context.Context, folder string, m pack.Manifest, out string) error {
	s.mu.Lock()
	dir, err := s.dir, s.need()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	folder, err = clean(folder)
	if err != nil {
		return err
	}
	root := filepath.Join(dir, filepath.FromSlash(folder))
	if m.Engine == "" {
		m.Engine = version.Engine
	}
	var b bytes.Buffer
	if err := pack.Export(ctx, &b, os.DirFS(root), m); err != nil {
		return err
	}
	if !strings.HasSuffix(out, pack.Ext) {
		out += pack.Ext
	}
	return writeFile(ctx, out, b.Bytes())
}
