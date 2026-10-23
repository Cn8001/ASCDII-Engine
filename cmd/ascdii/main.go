// Command ascdii is the developer CLI: ascdii run opens the editor, the other
// commands handle projects, builds and packages. Games run on ascdii-player.
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
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/launch"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/winbrand"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/player"
)

const usage = `usage:
  ascdii run [project]        the editor, at its start screen or with a project open
  ascdii new <dir> [-name N] [-template dragonfall]
  ascdii play <dir> [--blocks]
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
	if len(args) == 0 {
		fmt.Println(usage)
		if ownConsole() {
			// A double click opened this window; it would close before the help is read.
			fmt.Print("\nPress Enter to close.")
			_, _ = fmt.Scanln()
		}
		return nil
	}
	switch args[0] {
	case "run":
		hideConsole()
		if err := edit(ctx, self, args[1:]); err != nil {
			report(fmt.Sprintf("ASCDII could not start the editor.\n\n%v", err))
			return err
		}
		return nil
	case "help", "-h", "--help":
		fmt.Println(usage)
		return nil
	case "new":
		return cmdNew(ctx, args[1:])
	case "play":
		if len(args) < 2 {
			return errors.New(usage)
		}
		if _, err := project.RenameSprites(ctx, args[1]); err != nil {
			return err
		}
		return player.Project(args[1], modules()).Play(ctx, args[2:])
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

// edit starts the editor: at its start screen, or with the project given as
// its manifest or its folder.
func edit(ctx context.Context, self string, args []string) error {
	manifest := ""
	if len(args) > 0 {
		m, err := manifestOf(args[0])
		if err != nil {
			return err
		}
		manifest = m
	}
	editor, err := launch.Find(self, launch.Editor)
	if err != nil {
		return err
	}
	return launch.StartEditor(ctx, editor, manifest)
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
	return player.Project(dir, modules()).Preview(ctx, rest)
}

func cmdBuild(ctx context.Context, self string, args []string) error {
	dir, rest := split(args)
	fl := flag.NewFlagSet("build", flag.ContinueOnError)
	out := fl.String("out", "", "output folder, or executable with -single")
	bin := fl.String("player", "", "player binary for the target platform; default the ascdii-player next to ascdii")
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
	if *bin == "" {
		p, err := launch.Find(self, launch.Player)
		if err != nil {
			return err
		}
		*bin = p
	}
	brand, err := winbrand.For(p.Manifest, fsys, p.Library)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "ascdii-player-*")
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	defer os.RemoveAll(tmp)
	if *bin, err = winbrand.Brand(ctx, *bin, tmp, brand); err != nil {
		return err
	}
	if strings.HasSuffix(*bin, ".exe") {
		ext = ".exe"
	}
	name := strings.ReplaceAll(p.Manifest.Name, " ", "-")
	if !*single {
		folder := *out
		if folder == "" {
			folder = filepath.Join(dir, project.BuildDir, name)
		}
		exe, err := bundle.BuildFolder(ctx, folder, name+ext, *bin, fsys, project.BuildDir)
		if err != nil {
			return err
		}
		if p.Manifest.Mods.Enabled {
			if err := bundle.AddModsDir(folder); err != nil {
				return err
			}
		}
		fmt.Println(exe)
		return nil
	}
	if *out == "" {
		*out = filepath.Join(dir, project.BuildDir, name+ext)
	}
	if err := bundle.Build(ctx, *out, *bin, fsys, project.BuildDir); err != nil {
		return err
	}
	fmt.Println(*out)
	return nil
}

func manifestOf(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("project %s: %w", path, err)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("project %s: %w", path, err)
	}
	if !st.IsDir() {
		return abs, nil
	}
	name, err := project.Find(os.DirFS(abs))
	if err != nil {
		return "", fmt.Errorf("project %s: %w", path, err)
	}
	return filepath.Join(abs, name), nil
}
