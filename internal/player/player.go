// Package player plays a game in the terminal. ascdii-player runs it for
// built games and project folders; ascdii run and preview share it.
package player

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/speaker"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

var ErrNoGame = errors.New("no game data in or next to the player")

// Game is what to play: the game's files, the folder mods are read from and
// the code modules the player was built with.
type Game struct {
	FS      fs.FS
	Mods    string
	Modules []project.Module
}

// Project plays a project folder, with its own mods folder.
func Project(dir string, modules []project.Module) Game {
	return Game{FS: os.DirFS(dir), Mods: filepath.Join(dir, project.ModsDir), Modules: modules}
}

// Bundled finds the game a build carries: appended to exe, or in data/ next
// to it. Mods come from the mods folder beside exe.
func Bundled(ctx context.Context, exe string, modules []project.Module) (Game, io.Closer, error) {
	g := Game{Mods: filepath.Join(filepath.Dir(exe), bundle.ModsDir), Modules: modules}
	fsys, closer, err := bundle.OpenFile(ctx, exe)
	if err == nil {
		g.FS = fsys
		return g, closer, nil
	}
	if !errors.Is(err, bundle.ErrNoBundle) {
		return Game{}, nil, err
	}
	fsys, err = bundle.OpenData(exe)
	switch {
	case err == nil:
		g.FS = fsys
		return g, noClose{}, nil
	case errors.Is(err, bundle.ErrWrongKey):
		return Game{}, nil, err
	}
	return Game{}, nil, ErrNoGame
}

type noClose struct{}

func (noClose) Close() error { return nil }

// Play runs the game until it quits. --preview prints one frame instead.
func (g Game) Play(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "--preview" {
		return g.Preview(ctx, args[1:])
	}
	o := app.Options{Modules: g.Modules, Speaker: speaker.Open, Mods: modsDir(g.Mods)}
	if o.Mods != nil {
		l := newFileLog(g.Mods)
		defer l.Close()
		o.Log = l.Log
	} else {
		l := &heldLog{}
		defer l.Flush(os.Stderr)
		o.Log = l.Log
	}
	e, err := app.Open(ctx, g.FS, o)
	if err != nil {
		return err
	}
	if slices.Contains(args, "--blocks") || os.Getenv("ASCDII_BLOCKS") != "" {
		if err := blocks(e); err != nil {
			return err
		}
	}
	return e.Run(ctx, os.Stdin, os.Stdout)
}

// Preview simulates a moment of the game and prints the frame.
func (g Game) Preview(ctx context.Context, args []string) error {
	fl := flag.NewFlagSet("preview", flag.ContinueOnError)
	size := fl.String("size", "160x50", "COLSxROWS")
	secs := fl.Float64("time", 1, "seconds to simulate")
	plain := fl.Bool("plain", false, "no colors")
	hour := fl.Float64("hour", -1, "time of day override")
	block := fl.Bool("blocks", false, "half blocks instead of Braille")
	if err := fl.Parse(args); err != nil {
		return err
	}
	cols, rows, err := parseSize(*size)
	if err != nil {
		return err
	}
	e, err := app.Open(ctx, g.FS, app.Options{Cols: cols, Rows: rows, Modules: g.Modules, Mods: modsDir(g.Mods), Log: stderrLog})
	if err != nil {
		return err
	}
	e.Resize(cols, rows)
	if *block {
		if err := blocks(e); err != nil {
			return err
		}
	}
	if *hour >= 0 {
		e.Scene().Env.Hour = *hour
	}
	return e.Preview(ctx, os.Stdout, *secs, *plain)
}

// blocks draws half blocks instead of Braille, for terminals whose fonts
// draw Braille badly.
func blocks(e *app.Engine) error {
	p, err := e.Pipeline().Pass("braille")
	if err != nil {
		return err
	}
	if ed, ok := p.(prop.Editable); ok {
		return ed.Set("mode", pipeline.ModeBlocks)
	}
	return nil
}

func parseSize(s string) (int, int, error) {
	a, b, ok := strings.Cut(strings.ToLower(s), "x")
	cols, e1 := strconv.Atoi(a)
	rows, e2 := strconv.Atoi(b)
	if !ok || e1 != nil || e2 != nil || cols < 20 || rows < 8 || cols > 400 || rows > 200 {
		return 0, 0, fmt.Errorf("size must be COLSxROWS between 20x8 and 400x200, got %q", s)
	}
	return cols, rows, nil
}
