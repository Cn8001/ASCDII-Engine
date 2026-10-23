package session

import (
	"context"
	"os"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
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

// Build writes build/<os>/<Name>/ with the player, its packed game data and
// SHA256SUMS, or one executable when single is set. A version, when given,
// is stored in the manifest first, so the game reports it.
func (s *Session) Build(ctx context.Context, goos string, single bool, version string) (BuildResult, error) {
	dir, m, err := s.prepare(ctx)
	if err != nil {
		return BuildResult{}, err
	}
	if version != "" && version != m.Version {
		s.mu.Lock()
		s.proj.Manifest.Version = version
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
	player, err := s.d.Players.Extract(ctx, goos, "amd64", tmp)
	if err != nil {
		return BuildResult{}, err
	}
	t := bundle.Target{OS: goos, Arch: "amd64"}
	name := project.FileName(m.Name)
	if !single {
		exe, err := bundle.BuildFolder(ctx, filepath.Join(dir, project.BuildDir, goos, name), name+t.Ext(), player, os.DirFS(dir), project.BuildDir)
		if err != nil {
			return BuildResult{}, err
		}
		return BuildResult{Files: []string{exe}, Dir: filepath.Dir(exe)}, nil
	}
	out := filepath.Join(dir, project.BuildDir, name+t.Ext())
	if err := bundle.Build(ctx, out, player, os.DirFS(dir), project.BuildDir); err != nil {
		return BuildResult{}, err
	}
	return BuildResult{Files: []string{out}, Dir: filepath.Dir(out)}, nil
}
