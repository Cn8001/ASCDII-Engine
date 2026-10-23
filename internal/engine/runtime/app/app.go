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

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/audio"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/terminal"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/loop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/script/vm"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type Options struct {
	Cols, Rows int
	FPS        int
	Modules    []project.Module
	Now        func() time.Time
	// Speaker plays the game's sound stream; nil runs silent. Injected so
	// only the terminal player links the sound device.
	Speaker func(r io.Reader) (io.Closer, error)
	Mods    fs.FS                           // the folder holding mod folders; nil loads none
	Log     func(level, source, msg string) // engine and script messages; nil drops them
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
	input *input.State
	mods  *pendingMods // loaded at Open, started by the first Update
}

type pendingMods struct {
	api     string
	sources []vm.ModSource
	errs    []error
}

// NewPipeline is the standard chain plus the scene passes bound to get, so the
// caller can swap scenes without rebuilding the pipeline.
func NewPipeline(get func() *scene.Scene, now func() time.Time) *pipeline.Pipeline {
	passes := append(pipeline.Standard(), &scene.Geometry{Scene: get}, &scene.Painted{Scene: get}, &scene.Cells{Scene: get})
	return pipeline.New(now, passes...)
}

func build(o Options, reg *scene.Registry, lib *asset.Library, s *scene.Scene) *Engine {
	e := &Engine{opts: o, reg: reg, lib: lib, scn: s, input: input.NewState()}
	e.pipe = NewPipeline(e.Scene, o.Now)
	e.ctx.Pipeline = e.pipe
	e.ctx.Log = o.Log
	e.frame = pipeline.NewFrame(o.Cols, o.Rows)
	return e
}

// New starts with an empty scene and every built in kind registered.
func New(o Options) *Engine {
	o = o.withDefaults()
	reg, lib := project.Registry(o.Modules)
	s := scene.New("untitled")
	s.UsePrefabs(lib)
	return build(o, reg, lib, s)
}

// Open loads a project and its start scene.
func Open(ctx context.Context, fsys fs.FS, o Options) (*Engine, error) {
	p, err := project.Open(ctx, fsys, o.Modules)
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
	if m.Mods.Enabled && o.Mods != nil {
		sources, errs := project.LoadMods(ctx, p.Library, o.Mods)
		e.mods = &pendingMods{api: m.Mods.API, sources: sources, errs: errs}
	}
	return e, nil
}

// loadSound decodes a sound asset of the open project.
func (e *Engine) loadSound(ref string) (audio.Sound, error) {
	if e.proj == nil {
		return audio.Sound{}, fmt.Errorf("sound %s: %w", ref, asset.ErrNotFound)
	}
	p := e.lib.Resolve(ref)
	f, err := e.proj.FS.Open(p)
	if err != nil {
		return audio.Sound{}, fmt.Errorf("sound %s: %w", p, err)
	}
	defer f.Close()
	return audio.Decode(f, p)
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
	if m := e.mods; m != nil {
		e.mods = nil
		// Mods start once the scene runs, so the API script sees the game.
		for _, err := range append(m.errs, vm.StartMods(&e.ctx, e.lib, e.reg, m.api, m.sources)...) {
			e.ctx.Logf("error", "mods", "%v", err)
		}
	}
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
	return terminal.WritePreview(ctx, w, e.Render(), plain)
}

// Step applies input events and advances the game; Down events become the
// key presses behaviours receive.
func (e *Engine) Step(dt float64, events []input.Event) (bool, error) {
	var keys []input.Key
	for _, ev := range events {
		e.input.Apply(ev, e.scn.Time)
		if ev.Down {
			keys = append(keys, ev.Key)
		}
	}
	e.ctx.Input = e.input
	quit, err := e.Update(dt, keys)
	e.input.EndFrame()
	return quit, err
}

// Run plays in the terminal until quit, Ctrl+C or ctx ends.
func (e *Engine) Run(ctx context.Context, in *os.File, out io.Writer) error {
	t := terminal.New(in, out)
	return e.RunOn(ctx, t, t)
}

// RunOn plays on any platform backend: a terminal, a headless writer or a
// future pixel window.
func (e *Engine) RunOn(ctx context.Context, d platform.Display, in platform.Input) error {
	if err := d.Open(ctx); err != nil {
		return err
	}
	defer d.Close(context.WithoutCancel(ctx))
	if e.opts.Speaker != nil {
		mix := audio.NewMixer(e.loadSound)
		if sp, err := e.opts.Speaker(mix); err == nil {
			e.ctx.Audio = mix
			defer sp.Close()
			defer mix.StopAll()
		} else {
			e.ctx.Logf("error", "audio", "%v", err) // a game without sound beats no game
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ticker := time.NewTicker(time.Second / time.Duration(e.opts.FPS))
	defer ticker.Stop()
	return loop.Run(ctx, loop.Deps{
		Ticks: ticker.C,
		Input: in.Events(ctx),
		Step:  e.Step,
		Draw: func(ctx context.Context) error {
			if w, h, err := d.Size(); err == nil {
				e.Resize(w, h)
			}
			cells := e.Render()
			return d.Present(ctx, platform.Frame{Cells: cells, Dots: e.frame.Buf, Time: e.scn.Time})
		},
	})
}
