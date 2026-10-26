package session

import (
	"context"
	"os"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/winbrand"
)

func (s *Session) Targets() []string { return s.d.Players.Available() }

type BuildResult struct {
	Files []string `json:"files"`
	Dir   string   `json:"dir"`
}

// prepare saves pending edits so a build always matches what is on screen.
func (s *Session) prepare(ctx context.Context) (string, project.Manifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return "", project.Manifest{}, err
	}
	if s.unsaved {
		if err := s.save(ctx, s.scenePath); err != nil {
			return "", project.Manifest{}, err
		}
	}
	return s.dir, s.proj.Manifest, nil
}

// BuildOptions are the choices of the build dialog.
type BuildOptions struct {
	Single  bool   `json:"single"`  // one executable instead of a folder with data
	Version string `json:"version"` // "" keeps the manifest's
	Icon    string `json:"icon"`    // image for Windows executables, a guid or a path; "" for the player's
}

// CanBrand reports whether this editor writes icons into Windows builds;
// only the Windows editor links the resource writer.
func (s *Session) CanBrand() bool { return winbrand.Supported }

// Build writes build/<os>/<Name>/ with the player, its packed game data and
// SHA256SUMS, or one executable when single is set. The version and icon
// are stored in the manifest first, so the game reports them and the next
// build starts from them.
func (s *Session) Build(ctx context.Context, goos string, o BuildOptions) (BuildResult, error) {
	dir, m, err := s.prepare(ctx)
	if err != nil {
		return BuildResult{}, err
	}
	version, single := o.Version, o.Single
	if version == "" {
		version = m.Version
	}
	if version != m.Version || o.Icon != m.Icon {
		s.mu.Lock()
		s.proj.Manifest.Version, s.proj.Manifest.Icon = version, o.Icon
		m = s.proj.Manifest
		err = project.SaveManifest(ctx, filepath.Join(dir, s.proj.File), m)
		s.mu.Unlock()
		if err != nil {
			return BuildResult{}, err
		}
	}
	tmp, err := os.MkdirTemp("", "ascdii-player-*")
	if err != nil {
		return BuildResult{}, err
	}
	defer os.RemoveAll(tmp)
	player, err := s.d.Players.Extract(ctx, players.For(m.Display), goos, "amd64", tmp)
	if err != nil {
		return BuildResult{}, err
	}
	s.mu.Lock()
	brand, err := winbrand.For(m, os.DirFS(dir), s.lib)
	s.mu.Unlock()
	if err != nil {
		return BuildResult{}, err
	}
	if player, err = winbrand.Brand(ctx, player, filepath.Join(tmp, "branded"), brand); err != nil {
		return BuildResult{}, err
	}
	t := bundle.Target{OS: goos, Arch: "amd64"}
	name := project.FileName(m.Name)
	if !single {
		exe, err := bundle.BuildFolder(ctx, filepath.Join(dir, project.BuildDir, goos, name), name+t.Ext(), player, os.DirFS(dir), project.BuildDir)
		if err != nil {
			return BuildResult{}, err
		}
		if m.Mods.Enabled {
			if err := bundle.AddModsDir(filepath.Dir(exe)); err != nil {
				return BuildResult{}, err
			}
		}
		return BuildResult{Files: []string{exe}, Dir: filepath.Dir(exe)}, nil
	}
	out := filepath.Join(dir, project.BuildDir, name+t.Ext())
	if err := bundle.Build(ctx, out, player, os.DirFS(dir), project.BuildDir); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{Files: []string{out}, Dir: filepath.Dir(out)}, nil
}
