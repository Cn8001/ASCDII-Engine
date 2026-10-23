// Package service is the editor's API surface bound into the Wails frontend.
// It adapts session calls, owns the app context and maps errors to codes.
package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/session"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
)

// Context holds the Wails app context, which only exists after startup.
type Context struct {
	mu  sync.RWMutex
	ctx context.Context
}

func (c *Context) Set(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx = ctx
}

func (c *Context) Get() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

type Filter struct{ Name, Pattern string }

type Dialogs interface {
	OpenFile(ctx context.Context, title string, filters []Filter) (string, error)
	PickDir(ctx context.Context, title string) (string, error)
}

type Deps struct {
	Session *session.Session
	Ctx     *Context
	Dialogs Dialogs
	Recent  recent.Store
	Reveal  func(ctx context.Context, path string) error
	Edit    func(ctx context.Context, path string, line int) error
}

type Services struct {
	Project  *ProjectService
	Scene    *SceneService
	Viewport *ViewportService
	Pipeline *PipelineService
	Assets   *AssetService
	Anim     *AnimService
	Build    *BuildService
	Console  *ConsoleService
}

func New(d Deps) *Services {
	return &Services{
		Project:  &ProjectService{d: d},
		Scene:    &SceneService{d: d},
		Viewport: &ViewportService{d: d},
		Pipeline: &PipelineService{d: d},
		Assets:   &AssetService{d: d},
		Anim:     &AnimService{d: d},
		Build:    &BuildService{d: d},
		Console:  &ConsoleService{d: d},
	}
}

// All lists the services for Wails Bind.
func (s *Services) All() []any {
	return []any{s.Project, s.Scene, s.Viewport, s.Pipeline, s.Assets, s.Anim, s.Build, s.Console}
}

// Code is the stable error code the UI switches on.
func Code(err error) string {
	table := []struct {
		target error
		code   string
	}{
		{scene.ErrNodeNotFound, "not_found"},
		{scene.ErrUnknownKind, "unknown_kind"},
		{scene.ErrInvalidParent, "invalid_parent"},
		{scene.ErrPrefabLocked, "prefab_locked"},
		{prop.ErrUnknownProp, "unknown_prop"},
		{prop.ErrInvalidValue, "invalid_value"},
		{pipeline.ErrPassNotFound, "pass_not_found"},
		{session.ErrNotEditable, "not_editable"},
		{session.ErrNoProject, "no_project"},
		{session.ErrNotPlaying, "not_playing"},
		{session.ErrBadAxis, "bad_axis"},
		{session.ErrNotInstance, "not_instance"},
		{asset.ErrExists, "exists"},
		{session.ErrUnknownTemplate, "unknown_template"},
		{project.ErrNoManifest, "no_manifest"},
		{project.ErrMissingPack, "missing_pack"},
		{asset.ErrNotFound, "asset_not_found"},
		{asset.ErrInvalidArt, "invalid_art"},
		{players.ErrNoPlayer, "no_player"},
		{input.ErrUnknownKey, "unknown_key"},
		{fs.ErrExist, "exists"},
		{fs.ErrNotExist, "missing"},
		{errCancelled, "cancelled"},
		{context.Canceled, "cancelled"},
	}
	for _, t := range table {
		if errors.Is(err, t.target) {
			return t.code
		}
	}
	return "internal"
}

var errCancelled = errors.New("dialog cancelled")

// wrap prefixes the code so the frontend can read it from the rejected promise.
func wrap(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", Code(err), err)
}

func wrap1[T any](v T, err error) (T, error) { return v, wrap(err) }
