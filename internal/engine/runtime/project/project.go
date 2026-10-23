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

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/script/vm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const (
	Ext      = ".ascdiiproj"
	BuildDir = "build"
)

var (
	ErrNoManifest    = errors.New("no " + Ext + " manifest")
	ErrMissingModule = errors.New("code module not available")
)

type Manifest struct {
	Format  int         `json:"format"`
	Name    string      `json:"name"`
	Version string      `json:"version"` // semver of the game, used for release names
	Engine  string      `json:"engine"`  // engine version the project was last saved with
	Start   string      `json:"start"`
	Cols    int         `json:"cols,omitempty"`
	Rows    int         `json:"rows,omitempty"`
	FPS     int         `json:"fps,omitempty"`
	Modules []string    `json:"modules,omitempty"` // compiled Go components the project uses
	Legacy  []string    `json:"packs,omitempty"`   // the name modules had before packages existed
	Icon    string      `json:"icon,omitempty"`    // image of the Windows executable, a guid or a path
	Mods    ModSettings `json:"mods,omitzero"`
}

// ModSettings decide whether players' mods load, and the script that defines
// everything they can do.
type ModSettings struct {
	Enabled bool   `json:"enabled"`
	API     string `json:"api,omitempty"`
}

// Module is compiled content, component kinds written in Go, that a project
// can require; the editor and player list the modules they were built with.
type Module struct {
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
	m.Modules, m.Legacy = append(m.Modules, m.Legacy...), nil
	return m, nil
}

// Registry builds the component set a project can use.
func Registry(modules []Module) (*scene.Registry, *asset.Library) {
	reg := scene.NewRegistry()
	lib := asset.NewLibrary(reg)
	components.Register(reg, lib)
	vm.Register(reg, lib)
	for _, p := range modules {
		p.Install(reg, lib)
	}
	return reg, lib
}

func Open(ctx context.Context, fsys fs.FS, modules []Module) (*Project, error) {
	file, err := Find(fsys)
	if err != nil {
		return nil, err
	}
	m, err := ReadManifest(fsys, file)
	if err != nil {
		return nil, err
	}
	// Only the modules the manifest names are installed, so a new project
	// shows the engine's own components and not those of a sample it never uses.
	var use []Module
	for _, need := range m.Modules {
		i := slices.IndexFunc(modules, func(p Module) bool { return p.Name == need })
		if i < 0 {
			return nil, fmt.Errorf("%w: %s", ErrMissingModule, need)
		}
		use = append(use, modules[i])
	}
	reg, lib := Registry(use)
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
