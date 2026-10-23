// Package app ties scene, pipeline and terminal together into a runnable
// engine; pkg/ascdii re-exports it as the public API.
package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/loop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/term"
)

type Options struct {
	Cols, Rows int
	FPS        int
	Packs      []project.Pack
	Now        func() time.Time
}

func (o Options) withDefaults() Options {
	if o.Cols <= 0 {
		o.Cols = 120
	}
	if o.Rows <= 0 {
		o.Rows = 40
	}
	if o.FPS <= 0 {
		o.FPS = 30
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

type Engine struct {
	opts  Options
	reg   *scene.Registry
	lib   *asset.Library
	proj  *project.Project
	scn   *scene.Scene
	pipe  *pipeline.Pipeline
	frame *pipeline.Frame
	ctx   scene.Ctx
}

// NewPipeline is the standard chain plus the scene passes bound to get, so the
// caller can swap scenes without rebuilding the pipeline.
func NewPipeline(get func() *scene.Scene, now func() time.Time) *pipeline.Pipeline {
	passes := append(pipeline.Standard(), &scene.Geometry{Scene: get}, &scene.Cells{Scene: get})
	return pipeline.New(now, passes...)
}

func build(o Options, reg *scene.Registry, lib *asset.Library, s *scene.Scene) *Engine {
	e := &Engine{opts: o, reg: reg, lib: lib, scn: s}
	e.pipe = NewPipeline(e.Scene, o.Now)
	e.frame = pipeline.NewFrame(o.Cols, o.Rows)
	return e
}

// New starts with an empty scene and every built in kind registered.
func New(o Options) *Engine {
	o = o.withDefaults()
	reg, lib := project.Registry(o.Packs)
	s := scene.New("untitled")
	s.UsePrefabs(lib)
	return build(o, reg, lib, s)
}

// Open loads a project and its start scene.
func Open(ctx context.Context, fsys fs.FS, o Options) (*Engine, error) {
	p, err := project.Open(ctx, fsys, o.Packs)
	if err != nil {
		return nil, err
	}
	s, err := p.StartScene(ctx)
	if err != nil {
		return nil, err
	}
	m := p.Manifest
	if o.Cols <= 0 {
		o.Cols = m.Cols
	}
	if o.Rows <= 0 {
		o.Rows = m.Rows
	}
	if o.FPS <= 0 {
		o.FPS = m.FPS
	}
	e := build(o.withDefaults(), p.Registry, p.Library, s)
	e.proj = p
	return e, nil
}

func (e *Engine) Scene() *scene.Scene          { return e.scn }
func (e *Engine) SetScene(s *scene.Scene)      { e.scn = s }
func (e *Engine) Pipeline() *pipeline.Pipeline { return e.pipe }
func (e *Engine) Registry() *scene.Registry    { return e.reg }
func (e *Engine) Library() *asset.Library      { return e.lib }
func (e *Engine) Project() *project.Project    { return e.proj }
func (e *Engine) Frame() *pipeline.Frame       { return e.frame }
func (e *Engine) Resize(cols, rows int)        { e.frame.Resize(cols, rows) }

// Update advances the game; quit is true when a behaviour or Ctrl+C asks for it.
func (e *Engine) Update(dt float64, keys []input.Key) (bool, error) {
	for _, k := range keys {
		if k == input.CtrlC {
			return true, nil
		}
	}
	e.ctx.Scene = e.scn
	if err := e.scn.Update(&e.ctx, dt, keys); err != nil {
		return false, fmt.Errorf("update: %w", err)
	}
	return e.ctx.Quit, nil
}

func (e *Engine) Render() *cell.Grid {
	scene.Render(e.pipe, e.frame, e.scn)
	return e.frame.Cells
}

// Preview simulates seconds of game time and prints one frame.
func (e *Engine) Preview(ctx context.Context, w io.Writer, seconds float64, plain bool) error {
	for t := 0.0; t < seconds; t += 1.0 / 30 {
		if _, err := e.Update(1.0/30, nil); err != nil {
			return err
		}
	}
	return term.WritePreview(ctx, w, e.Render(), plain)
}

// Run plays in the terminal until quit, Ctrl+C or ctx ends.
func (e *Engine) Run(ctx context.Context, in *os.File, out io.Writer) error {
	fd := int(in.Fd())
	if _, _, err := term.Size(fd); err != nil {
		return fmt.Errorf("interactive mode needs a terminal: %w", err)
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)
	pr := term.NewPresenter(out)
	if err := pr.Enter(ctx); err != nil {
		return err
	}
	defer pr.Leave(context.WithoutCancel(ctx))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ticker := time.NewTicker(time.Second / time.Duration(e.opts.FPS))
	defer ticker.Stop()
	return loop.Run(ctx, loop.Deps{
		Ticks: ticker.C,
		Keys:  term.Keys(ctx, in),
		Step:  e.Update,
		Draw: func(ctx context.Context) error {
			if w, h, err := term.Size(fd); err == nil {
				e.Resize(w, h)
			}
			return pr.Present(ctx, e.Render())
		},
	})
}
