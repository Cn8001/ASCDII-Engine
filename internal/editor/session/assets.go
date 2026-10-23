package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type Preview struct {
	Cols  int    `json:"cols"`
	Rows  int    `json:"rows"`
	Cells string `json:"cells"`
}

// Assets rescans the project folder, so files added outside the editor show up.
func (s *Session) Assets(ctx context.Context) ([]asset.Entry, error) {
	s.mu.Lock()
	dir := s.dir
	err := s.need()
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return asset.List(ctx, os.DirFS(dir), project.BuildDir)
}

// Reload reads art and prefabs from disk again and rebuilds prefab instances.
func (s *Session) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	if err := s.lib.Load(ctx, os.DirFS(s.dir), project.BuildDir); err != nil {
		return err
	}
	s.touch()
	return s.edit.Instantiate()
}

func (s *Session) ArtPreview(name string) (Preview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	art, err := s.lib.Sprite(name)
	if err != nil {
		return Preview{}, err
	}
	g := cell.New(art.Width, art.Height)
	for y := 0; y < art.Height; y++ {
		for x := 0; x < art.Width; x++ {
			if p, _ := art.At(x, y); !p.Transparent {
				g.Set(x, y, cell.Cell{Rune: p.Rune, FG: p.FG, BG: p.BG})
			}
		}
	}
	return Preview{Cols: g.W, Rows: g.H, Cells: EncodeCells(g)}, nil
}

// AddSprite places glyph art at the editor camera target.
func (s *Session) AddSprite(name string) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.lib.Sprite(name); err != nil {
		return 0, err
	}
	if s.play == nil {
		s.checkpoint("")
	}
	sc := s.current()
	file := s.lib.Resolve(name)
	e := scene.NewGameObject(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), components.NewSprite(s.lib, s.lib.Ref(file)))
	e.Local.Pos = s.spawnPoint()
	got, err := sc.Graft(nil, e, -1)
	if err != nil {
		return 0, err
	}
	s.sel = got.ID
	s.changed()
	return got.ID, nil
}
