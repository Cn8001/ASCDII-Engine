// Command ascdii is both the developer CLI and the game player. A build is
// this binary with a project appended; when it finds one it just plays it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/launch"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/speaker"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
)

const usage = `usage:
  ascdii                      the editor: templates and recent projects
  ascdii new <dir> [-name N] [-template dragonfall]
  ascdii run <dir> [--blocks]
  ascdii preview <dir> [-size 160x50] [-time 2] [-hour 22] [-plain] [-blocks]
  ascdii build <dir> [-out path] [-player path] [-single]
  ascdii pack list|install|remove|export ...
  ascdii version
  ascdii help`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func modules() []project.Module { return []project.Module{dragonfall.Module()} }

func run(ctx context.Context, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	if fsys, closer, err := bundle.OpenFile(ctx, self); err == nil {
		defer closer.Close()
		return play(ctx, fsys, args)
	} else if !errors.Is(err, bundle.ErrNoBundle) {
		return err
	}
	if fsys, err := bundle.OpenData(self); err == nil {
		return play(ctx, fsys, args)
	}
	if len(args) == 0 {
		return edit(ctx, "")
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Println(usage)
		return nil
	case "new":
		return cmdNew(ctx, args[1:])
	case "run":
		if len(args) < 2 {
			return errors.New(usage)
		}
		if _, err := project.RenameSprites(ctx, args[1]); err != nil {
			return err
		}
		return play(ctx, os.DirFS(args[1]), args[2:])
	case "preview":
		return cmdPreview(ctx, args[1:])
	case "build":
		return cmdBuild(ctx, self, args[1:])
	case "pack":
		return cmdPack(ctx, args[1:])
	case "version":
		fmt.Println("ASCDII Engine " + version.Engine)
		return nil
	}
	return errors.New(usage)
}

// edit starts the editor, which opens its start screen without a manifest;
// the editor only opens from here.
func edit(ctx context.Context, manifest string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	editor, err := launch.Find(self, launch.Editor)
	if err != nil {
		return err
	}
	return launch.StartEditor(ctx, editor, manifest)
}

func play(ctx context.Context, fsys fs.FS, args []string) error {
	if len(args) > 0 && args[0] == "--preview" {
		return preview(ctx, fsys, args[1:])
	}
	e, err := app.Open(ctx, fsys, app.Options{Modules: modules(), Speaker: speaker.Open})
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

func split(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", args
	}
	return args[0], args[1:]
}

func cmdNew(ctx context.Context, args []string) error {
	dir, rest := split(args)
	fl := flag.NewFlagSet("new", flag.ContinueOnError)
	name := fl.String("name", "", "game name")
	tpl := fl.String("template", "", "template: dragonfall")
	if err := fl.Parse(rest); err != nil || dir == "" {
		return errors.New(usage)
	}
	if *name == "" {
		*name = filepath.Base(dir)
	}
	var src fs.FS
	if *tpl == "dragonfall" {
		src = dragonfall.Template()
	} else if *tpl != "" {
		return fmt.Errorf("unknown template %q", *tpl)
	}
	file, err := project.Create(ctx, dir, *name, src)
	if err != nil {
		return err
	}
	fmt.Println(file)
	return nil
}

func cmdPreview(ctx context.Context, args []string) error {
	dir, rest := split(args)
	if dir == "" {
		return errors.New(usage)
	}
	if _, err := project.RenameSprites(ctx, dir); err != nil {
		return err
	}
	return preview(ctx, os.DirFS(dir), rest)
}

func preview(ctx context.Context, fsys fs.FS, args []string) error {
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
	e, err := app.Open(ctx, fsys, app.Options{Cols: cols, Rows: rows, Modules: modules()})
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

func parseSize(s string) (int, int, error) {
	a, b, ok := strings.Cut(strings.ToLower(s), "x")
	cols, e1 := strconv.Atoi(a)
	rows, e2 := strconv.Atoi(b)
	if !ok || e1 != nil || e2 != nil || cols < 20 || rows < 8 || cols > 400 || rows > 200 {
		return 0, 0, fmt.Errorf("size must be COLSxROWS between 20x8 and 400x200, got %q", s)
	}
	return cols, rows, nil
}

func cmdBuild(ctx context.Context, self string, args []string) error {
	dir, rest := split(args)
	fl := flag.NewFlagSet("build", flag.ContinueOnError)
	out := fl.String("out", "", "output folder, or executable with -single")
	player := fl.String("player", self, "player binary for the target platform")
	single := fl.Bool("single", false, "one executable with the project inside, instead of a folder")
	if err := fl.Parse(rest); err != nil || dir == "" {
		return errors.New(usage)
	}
	if _, err := project.RenameSprites(ctx, dir); err != nil {
		return err
	}
	if _, err := project.EnsureMeta(ctx, dir, guid.Default()); err != nil {
		return err
	}
	fsys := os.DirFS(dir)
	p, err := project.Open(ctx, fsys, modules())
	if err != nil {
		return err
	}
	ext := ""
	if runtime.GOOS == "windows" || strings.HasSuffix(*player, ".exe") {
		ext = ".exe"
	}
	name := strings.ReplaceAll(p.Manifest.Name, " ", "-")
	if !*single {
		folder := *out
		if folder == "" {
			folder = filepath.Join(dir, project.BuildDir, name)
		}
		exe, err := bundle.BuildFolder(ctx, folder, name+ext, *player, fsys, project.BuildDir)
		if err != nil {
			return err
		}
		fmt.Println(exe)
		return nil
	}
	if *out == "" {
		*out = filepath.Join(dir, project.BuildDir, name+ext)
	}
	if err := bundle.Build(ctx, *out, *player, fsys, project.BuildDir); err != nil {
		return err
	}
	fmt.Println(*out)
	return nil
}
