package session

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

var ErrBadSettings = errors.New("invalid project settings")

// ProjectSettings are the manifest fields the developer edits. Cols, Rows and
// FPS of 0 keep the engine defaults.
type ProjectSettings struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Start   string `json:"start"`
	Cols    int    `json:"cols"`
	Rows    int    `json:"rows"`
	FPS     int    `json:"fps"`
	Mods    bool   `json:"mods"`
	ModAPI  string `json:"modApi"` // script reference, a guid or a path
	Icon    string `json:"icon"`   // chosen in the build dialog, kept as it is here
}

func (s *Session) ProjectSettings() (ProjectSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return ProjectSettings{}, err
	}
	m := s.proj.Manifest
	return ProjectSettings{Name: m.Name, Version: m.Version, Start: m.Start, Cols: m.Cols, Rows: m.Rows, FPS: m.FPS, Mods: m.Mods.Enabled, ModAPI: m.Mods.API, Icon: m.Icon}, nil
}

func (s *Session) SetProjectSettings(ctx context.Context, p ProjectSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	if err := s.checkSettings(p); err != nil {
		return err
	}
	m := s.proj.Manifest
	m.Name, m.Version, m.Start = strings.TrimSpace(p.Name), strings.TrimSpace(p.Version), p.Start
	m.Cols, m.Rows, m.FPS = p.Cols, p.Rows, p.FPS
	m.Mods = project.ModSettings{Enabled: p.Mods, API: p.ModAPI}
	m.Icon = p.Icon
	if err := project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), m); err != nil {
		return err
	}
	s.proj.Manifest = m
	return nil
}

func (s *Session) checkSettings(p ProjectSettings) error {
	switch {
	case strings.TrimSpace(p.Name) == "":
		return fmt.Errorf("%w: the name is empty", ErrBadSettings)
	case p.Cols < 0 || p.Rows < 0 || p.FPS < 0 || p.FPS > 240:
		return fmt.Errorf("%w: size and fps", ErrBadSettings)
	case p.Mods && p.ModAPI == "":
		return fmt.Errorf("%w: mods need an api script", ErrBadSettings)
	}
	if _, err := fs.Stat(os.DirFS(s.dir), p.Start); err != nil || !strings.HasSuffix(p.Start, asset.SceneExt) {
		return fmt.Errorf("%w: no start scene %s", ErrBadSettings, p.Start)
	}
	if p.ModAPI != "" {
		if _, _, err := s.lib.Script(p.ModAPI); err != nil {
			return fmt.Errorf("%w: api script: %w", ErrBadSettings, err)
		}
	}
	return nil
}
