// Package project reads and creates .ascdiiproj game folders. A project is
// plain files: a manifest, scenes, prefabs and art, from any fs.FS so the same
// code serves a folder on disk, a template and a packed build.
package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/kit"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/script"
)

const (
	Ext      = ".ascdiiproj"
	BuildDir = "build"
)

var (
	ErrNoManifest  = errors.New("no " + Ext + " manifest")
	ErrMissingPack = errors.New("content pack not available")
)

type Manifest struct {
	Format  int      `json:"format"`
	Name    string   `json:"name"`
	Version string   `json:"version"` // semver of the game, used for release names
	Engine  string   `json:"engine"`  // engine version the project was last saved with
	Start   string   `json:"start"`
	Cols    int      `json:"cols,omitempty"`
	Rows    int      `json:"rows,omitempty"`
	FPS     int      `json:"fps,omitempty"`
	Packs   []string `json:"packs,omitempty"`
}

// Pack is compiled content, node kinds written in Go, that a project can require.
type Pack struct {
	Name    string
	Install func(r *scene.Registry, lib *asset.Library)
}

type Project struct {
	Manifest Manifest
	File     string
	FS       fs.FS
	Registry *scene.Registry
	Library  *asset.Library
}

func Find(fsys fs.FS) (string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return "", fmt.Errorf("find manifest: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), Ext) {
			return e.Name(), nil
		}
	}
	return "", ErrNoManifest
}

func ReadManifest(fsys fs.FS, name string) (Manifest, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest %s: %w", name, err)
	}
	return m, nil
}

// Registry builds the node and behaviour set a project can use.
func Registry(packs []Pack) (*scene.Registry, *asset.Library) {
	reg := scene.NewRegistry()
	lib := asset.NewLibrary(reg)
	kit.Register(reg, lib)
	script.Register(reg, lib)
	for _, p := range packs {
		p.Install(reg, lib)
	}
	return reg, lib
}

func Open(ctx context.Context, fsys fs.FS, packs []Pack) (*Project, error) {
	file, err := Find(fsys)
	if err != nil {
		return nil, err
	}
	m, err := ReadManifest(fsys, file)
	if err != nil {
		return nil, err
	}
	for _, need := range m.Packs {
		if !slices.ContainsFunc(packs, func(p Pack) bool { return p.Name == need }) {
			return nil, fmt.Errorf("%w: %s", ErrMissingPack, need)
		}
	}
	reg, lib := Registry(packs)
	if err := lib.Load(ctx, fsys, BuildDir); err != nil {
		return nil, fmt.Errorf("open %s: %w", m.Name, err)
	}
	return &Project{Manifest: m, File: file, FS: fsys, Registry: reg, Library: lib}, nil
}

func (p *Project) Scene(ctx context.Context, name string) (*scene.Scene, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := p.FS.Open(path.Clean(name))
	if err != nil {
		return nil, fmt.Errorf("open scene: %w", err)
	}
	defer f.Close()
	s, err := scene.Decode(f, p.Registry, p.Library)
	if err != nil {
		return nil, fmt.Errorf("scene %s: %w", name, err)
	}
	return s, nil
}

func (p *Project) StartScene(ctx context.Context) (*scene.Scene, error) {
	return p.Scene(ctx, p.Manifest.Start)
}

func (p *Project) Assets(ctx context.Context) ([]asset.Entry, error) {
	return asset.List(ctx, p.FS, BuildDir)
}
