package session

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/animator"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/material"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/mesh"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

const AssetsEvent = "assets"

// watch reloads changed project files so scripts edited in an external
// editor, or art and prefabs saved by other tools, apply without a restart.
func (s *Session) watch(ctx context.Context, dir string) {
	_ = asset.Watch(ctx, os.DirFS(dir), 500*time.Millisecond, func(changed []string) {
		s.reloadFiles(ctx, dir, changed)
	}, project.BuildDir)
}

func (s *Session) reloadFiles(ctx context.Context, dir string, changed []string) {
	s.mu.Lock()
	if s.dir != dir {
		s.mu.Unlock()
		return
	}
	prefabs := false
	for _, p := range changed {
		k, _ := asset.Classify(p)
		b, err := os.ReadFile(dir + "/" + p)
		if err != nil {
			continue // deleted; running content keeps its last version
		}
		if m, err := asset.ReadMeta(os.DirFS(dir), p); err == nil {
			s.lib.SetGUID(p, m.GUID) // a moved file brings its id along
		}
		switch k {
		case asset.Script:
			s.lib.PutScript(p, string(b))
		case asset.Prefab:
			s.lib.PutPrefab(p, b)
			prefabs = true
		case asset.Sprite:
			if a, err := asset.DecodeSprite(p, b); err == nil {
				s.lib.PutSprite(p, a)
			}
		case asset.Texture:
			if t, err := asset.LoadTexture(os.DirFS(dir), p); err == nil {
				s.lib.PutTexture(p, t)
			}
		case asset.Animator:
			if c, err := animator.Decode(bytes.NewReader(b)); err == nil {
				s.lib.PutController(p, c)
			}
		case asset.Material:
			if m, err := material.Decode(bytes.NewReader(b)); err == nil {
				s.lib.PutMaterial(p, m)
			}
		case asset.Model:
			if m, err := mesh.Load(ctx, os.DirFS(dir), p, asset.ModelOptions(os.DirFS(dir), p)); err == nil {
				s.lib.PutModel(p, m)
			}
		}
	}
	var err error
	if prefabs {
		err = s.edit.Instantiate()
	}
	s.touch()
	s.mu.Unlock()
	if err != nil {
		s.Log(ctx, "error", "prefab", err.Error())
	}
	s.Log(ctx, "info", "assets", strings.Join(changed, ", "))
	if s.d.Emit != nil {
		s.d.Emit.Emit(ctx, AssetsEvent, changed)
	}
}

func (s *Session) startWatch(ctx context.Context) {
	if s.stopWatch != nil {
		s.stopWatch()
	}
	wctx, cancel := context.WithCancel(ctx)
	s.stopWatch = cancel
	go s.watch(wctx, s.dir)
}

// Close stops background work; the editor calls it on shutdown and tests on cleanup.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopWatch != nil {
		s.stopWatch()
		s.stopWatch = nil
	}
}
