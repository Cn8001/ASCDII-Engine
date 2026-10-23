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
	"strconv"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/cmd/launcher/menu"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
)

const usage = `usage:
  ascdii                      main menu
  ascdii new <dir> [-name N] [-template dragonfall]
  ascdii run <dir>
  ascdii preview <dir> [-size 160x50] [-time 2] [-plain]
  ascdii build <dir> [-out path] [-player path]
  ascdii release <dir> [-version v] [-players dir]
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

func packs() []project.Pack { return []project.Pack{dragonfall.Pack()} }

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
	if len(args) == 0 {
		return launch(ctx)
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
		return play(ctx, os.DirFS(args[1]), args[2:])
	case "preview":
		return cmdPreview(ctx, args[1:])
	case "build":
		return cmdBuild(ctx, self, args[1:])
	case "release":
		return cmdRelease(ctx, self, args[1:])
	case "version":
		fmt.Println("ASCDII Engine " + version.Engine)
		return nil
	}
	return errors.New(usage)
}

// launch is what a double click on the binary opens.
func launch(ctx context.Context) error {
	store, err := recent.Default()
	if err != nil {
		return err
	}
	return menu.Run(ctx, os.Stdin, os.Stdout, menu.Deps{
		Version:   "v" + version.Engine,
		Demo:      dragonfall.Template(),
		Templates: map[string]fs.FS{"dragonfall": dragonfall.Template(), "empty": nil},
		Recent:    store,
		Help:      usage,
		Play:      func(ctx context.Context, fsys fs.FS) error { return play(ctx, fsys, nil) },
	})
}

func play(ctx context.Context, fsys fs.FS, args []string) error {
	if len(args) > 0 && args[0] == "--preview" {
		return preview(ctx, fsys, args[1:])
	}
	e, err := app.Open(ctx, fsys, app.Options{Packs: packs()})
	if err != nil {
		return err
	}
	return e.Run(ctx, os.Stdin, os.Stdout)
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
	return preview(ctx, os.DirFS(dir), rest)
}

func preview(ctx context.Context, fsys fs.FS, args []string) error {
	fl := flag.NewFlagSet("preview", flag.ContinueOnError)
	size := fl.String("size", "160x50", "COLSxROWS")
	secs := fl.Float64("time", 1, "seconds to simulate")
	plain := fl.Bool("plain", false, "no colors")
	hour := fl.Float64("hour", -1, "time of day override")
	if err := fl.Parse(args); err != nil {
		return err
	}
	cols, rows, err := parseSize(*size)
	if err != nil {
		return err
	}
	e, err := app.Open(ctx, fsys, app.Options{Cols: cols, Rows: rows, Packs: packs()})
	if err != nil {
		return err
	}
	e.Resize(cols, rows)
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
	out := fl.String("out", "", "output executable")
	player := fl.String("player", self, "player binary for the target platform")
	if err := fl.Parse(rest); err != nil || dir == "" {
		return errors.New(usage)
	}
	if _, err := project.EnsureMeta(ctx, dir, guid.Default()); err != nil {
		return err
	}
	fsys := os.DirFS(dir)
	p, err := project.Open(ctx, fsys, packs())
	if err != nil {
		return err
	}
	if *out == "" {
		ext := ""
		if runtime.GOOS == "windows" || strings.HasSuffix(*player, ".exe") {
			ext = ".exe"
		}
		*out = filepath.Join(dir, project.BuildDir, strings.ReplaceAll(p.Manifest.Name, " ", "-")+ext)
	}
	if err := bundle.Build(ctx, *out, *player, fsys, project.BuildDir); err != nil {
		return err
	}
	fmt.Println(*out)
	return nil
}

// cmdRelease builds every target whose player is found in -players, named
// ascdii-<os>-<arch>[.exe]; the host player is always this binary.
func cmdRelease(ctx context.Context, self string, args []string) error {
	dir, rest := split(args)
	fl := flag.NewFlagSet("release", flag.ContinueOnError)
	ver := fl.String("version", "", "game version, defaults to the manifest")
	players := fl.String("players", "", "folder with ascdii-<os>-<arch> player binaries")
	if err := fl.Parse(rest); err != nil || dir == "" {
		return errors.New(usage)
	}
	if _, err := project.EnsureMeta(ctx, dir, guid.Default()); err != nil {
		return err
	}
	fsys := os.DirFS(dir)
	p, err := project.Open(ctx, fsys, packs())
	if err != nil {
		return err
	}
	if *ver == "" {
		*ver = p.Manifest.Version
	}
	targets := []bundle.Target{{OS: runtime.GOOS, Arch: runtime.GOARCH, Player: self}}
	if *players != "" {
		for _, t := range []bundle.Target{{OS: "linux", Arch: "amd64"}, {OS: "windows", Arch: "amd64"}, {OS: "darwin", Arch: "arm64"}} {
			path := filepath.Join(*players, "ascdii-"+t.OS+"-"+t.Arch+t.Ext())
			if _, err := os.Stat(path); err != nil || (t.OS == runtime.GOOS && t.Arch == runtime.GOARCH) {
				continue
			}
			t.Player = path
			targets = append(targets, t)
		}
	}
	rel, err := bundle.MakeRelease(ctx, filepath.Join(dir, project.BuildDir, "release"), p.Manifest.Name, *ver, targets, fsys, project.BuildDir)
	if err != nil {
		return err
	}
	for _, f := range rel.Files {
		fmt.Println(f)
	}
	return nil
}
