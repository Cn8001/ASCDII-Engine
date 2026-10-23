// Package session is the editor's model: the open project, the scene being
// edited, play mode, selection, undo and the render loop that streams frames.
// Every exported method is safe to call from any goroutine.
package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/players"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/pack"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

var (
	ErrNoProject       = errors.New("no project open")
	ErrUnknownTemplate = errors.New("unknown template")
)

// Emitter pushes events to the UI; the Wails adapter implements it.
type Emitter interface {
	Emit(ctx context.Context, event string, data any)
}

type Template struct {
	Name string
	FS   fs.FS // nil creates the starter scene
}

type Deps struct {
	Modules    []project.Module
	Packages   []pack.Source // built into the editor, such as the Dragonfall sample
	PackageDir string        // where the user keeps .ascdiipack files
	Templates  []Template
	Players    players.Source
	Binary     bool        // frames are fetched with Raw instead of riding in events
	IDs        guid.Source // asset ids; guid.Default() when zero
	Emit       Emitter
	Now        func() time.Time
}

type Mode string

const (
	Edit   Mode = "edit"
	Play   Mode = "play"
	Paused Mode = "paused"
)

type Session struct {
	mu        sync.Mutex
	d         Deps
	reg       *scene.Registry
	lib       *asset.Library
	proj      *project.Project
	dir       string
	scenePath string
	edit      *scene.Scene
	play      *scene.Scene
	mode      Mode
	live      bool
	pipe      *pipeline.Pipeline
	frame     *pipeline.Frame
	sel       uint64
	dirty     bool
	unsaved   bool
	keys      []input.Key
	sctx      scene.Ctx
	hist      history
	logs      logbook
	raw       []byte
	seq       uint64
	cam       view.Camera // editor camera, separate from the scene's game camera
	viewMode  string
	step      float64
	stopWatch context.CancelFunc
	appCtx    context.Context
}

func New(d Deps) *Session {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.IDs.Rand == nil {
		d.IDs = guid.Default()
	}
	reg, lib := project.Registry(d.Modules)
	s := &Session{d: d, reg: reg, lib: lib, edit: scene.New("untitled"), mode: Edit, live: false, dirty: true, viewMode: SceneView, appCtx: context.Background()}
	s.cam = s.edit.Camera
	s.pipe = app.NewPipeline(s.current, d.Now)
	s.pipe.Add(pipeline.Func("grid", pipeline.Geometry, s.grid))
	s.frame = pipeline.NewFrame(120, 40)
	s.resetCtx()
	return s
}

// resetCtx starts a run from scratch: no Lua state or time scale carried over.
func (s *Session) resetCtx() {
	s.sctx = scene.Ctx{Pipeline: s.pipe, Input: input.NewState(), Log: func(level, source, msg string) { s.Log(s.appCtx, level, source, msg) }}
}

func (s *Session) current() *scene.Scene {
	if s.play != nil {
		return s.play
	}
	return s.edit
}

func (s *Session) touch() { s.dirty = true }

func (s *Session) changed() { s.dirty, s.unsaved = true, true }

type Info struct {
	Open      bool     `json:"open"`
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Engine    string   `json:"engine"`
	Dir       string   `json:"dir"`
	Manifest  string   `json:"manifest"`
	Scene     string   `json:"scene"`
	Scenes    []string `json:"scenes"`
	Unsaved   bool     `json:"unsaved"`
	Mode      Mode     `json:"mode"`
	View      string   `json:"view"`
	Live      bool     `json:"live"`
	Templates []string `json:"templates"`
	CanUndo   bool     `json:"canUndo"`
	CanRedo   bool     `json:"canRedo"`
}

func (s *Session) Info(ctx context.Context) (Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	in := Info{Engine: version.Engine, Mode: s.mode, View: s.viewMode, Live: s.live, Unsaved: s.unsaved, CanUndo: len(s.hist.undo) > 0, CanRedo: len(s.hist.redo) > 0}
	for _, t := range s.d.Templates {
		in.Templates = append(in.Templates, t.Name)
	}
	if s.proj == nil {
		return in, nil
	}
	m := s.proj.Manifest
	in.Open, in.Name, in.Version, in.Dir, in.Scene = true, m.Name, m.Version, s.dir, s.scenePath
	in.Manifest = filepath.Join(s.dir, s.proj.File)
	entries, err := asset.List(ctx, os.DirFS(s.dir), project.BuildDir)
	if err != nil {
		return in, err
	}
	for _, e := range entries {
		if e.Kind == asset.Scene {
			in.Scenes = append(in.Scenes, e.Path)
		}
	}
	return in, nil
}

// Open loads the project whose manifest is at path and its start scene.
func (s *Session) Open(ctx context.Context, manifest string) error {
	dir := filepath.Dir(manifest)
	made, err := project.EnsureMeta(ctx, dir, s.d.IDs)
	if err != nil {
		return err
	}
	p, err := project.Open(ctx, os.DirFS(dir), s.d.Modules)
	if err != nil {
		return err
	}
	sc, err := p.StartScene(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proj, s.dir, s.reg, s.lib, s.appCtx = p, dir, p.Registry, p.Library, ctx
	s.load(sc, p.Manifest.Start)
	s.migrate(ctx, len(made))
	s.startWatch(ctx)
	return nil
}

// migrate moves path references of the loaded scene to GUIDs and says so.
func (s *Session) migrate(ctx context.Context, metas int) {
	n := project.Migrate(s.edit, s.lib)
	if n > 0 {
		s.changed()
	}
	if metas > 0 || n > 0 {
		go s.Log(ctx, "info", "assets", fmt.Sprintf("%d new .meta files, %d references now use GUIDs", metas, n))
	}
}

func (s *Session) load(sc *scene.Scene, path string) {
	s.edit, s.play, s.scenePath, s.cam = sc, nil, path, sc.Camera
	s.mode, s.sel, s.unsaved, s.keys = Edit, 0, false, nil
	s.hist = history{}
	s.touch()
}

func (s *Session) Create(ctx context.Context, dir, name, template string) (string, error) {
	var src fs.FS
	found := template == ""
	for _, t := range s.d.Templates {
		if t.Name == template {
			src, found = t.FS, true
		}
	}
	if !found {
		return "", fmt.Errorf("%w: %s", ErrUnknownTemplate, template)
	}
	manifest, err := project.Create(ctx, dir, name, src)
	if err != nil {
		return "", err
	}
	return manifest, s.Open(ctx, manifest)
}

func (s *Session) need() error {
	if s.proj == nil {
		return ErrNoProject
	}
	return nil
}

func (s *Session) OpenScene(ctx context.Context, rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	p := *s.proj
	p.FS = os.DirFS(s.dir)
	sc, err := p.Scene(ctx, rel)
	if err != nil {
		return err
	}
	s.load(sc, rel)
	s.migrate(ctx, 0)
	return nil
}

// writeAsset writes a project file and gives it a .meta when it has none.
func (s *Session) writeAsset(ctx context.Context, rel string, b []byte) error {
	abs := filepath.Join(s.dir, filepath.FromSlash(rel))
	if err := writeFile(ctx, abs, b); err != nil {
		return err
	}
	if _, err := os.Stat(abs + asset.MetaExt); err == nil {
		return nil
	}
	id, err := s.d.IDs.New()
	if err != nil {
		return err
	}
	if err := asset.WriteMeta(ctx, s.dir, rel, id); err != nil {
		return err
	}
	s.lib.SetGUID(rel, id)
	return nil
}

func writeFile(ctx context.Context, path string, b []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Save writes the edited scene and stamps the manifest with this engine version.
func (s *Session) Save(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(ctx, s.scenePath)
}

func (s *Session) SaveAs(ctx context.Context, rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(ctx, rel)
}

func (s *Session) save(ctx context.Context, rel string) error {
	if err := s.need(); err != nil {
		return err
	}
	if locked(rel) {
		return lockedErr(rel)
	}
	var b bytes.Buffer
	if err := scene.Encode(&b, s.edit); err != nil {
		return err
	}
	if err := writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(rel)), b.Bytes()); err != nil {
		return err
	}
	s.proj.Manifest.Engine = version.Engine
	if err := project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), s.proj.Manifest); err != nil {
		return err
	}
	s.scenePath, s.unsaved = rel, false
	return nil
}

// NewScene writes a starter scene to rel and opens it.
func (s *Session) NewScene(ctx context.Context, rel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	if locked(rel) {
		return lockedErr(rel)
	}
	var b bytes.Buffer
	if err := scene.Encode(&b, project.Starter(s.proj.Manifest.Name)); err != nil {
		return err
	}
	if err := s.writeAsset(ctx, rel, b.Bytes()); err != nil {
		return err
	}
	sc, err := scene.Decode(&b, s.reg, s.lib)
	if err != nil {
		return err
	}
	s.load(sc, rel)
	return nil
}

// SetStart makes the open scene the one a build launches.
func (s *Session) SetStart(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	s.proj.Manifest.Start = s.scenePath
	return project.SaveManifest(ctx, filepath.Join(s.dir, s.proj.File), s.proj.Manifest)
}

// CloseProject returns the editor to its start state without a project.
func (s *Session) CloseProject() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopWatch != nil {
		s.stopWatch()
		s.stopWatch = nil
	}
	s.reg, s.lib = project.Registry(s.d.Modules)
	s.proj, s.dir = nil, ""
	s.load(scene.New("untitled"), "")
}
