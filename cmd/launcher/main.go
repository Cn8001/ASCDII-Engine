// Command launcher is the ASCDII terminal main menu on its own: open recent
// projects, create one from a template or play the demo. The ascdii binary
// shows the same menu when it starts without arguments.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/cmd/launcher/menu"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/version"
)

const help = `ASCDII Engine launcher

Arrow keys move, Enter picks, Esc goes back, Q quits.
Use the ascdii command for scripted builds: ascdii help`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	store, err := recent.Default()
	if err != nil {
		return err
	}
	return menu.Run(ctx, os.Stdin, os.Stdout, menu.Deps{
		Version:   "v" + version.Engine,
		Demo:      dragonfall.Template(),
		Templates: map[string]fs.FS{"dragonfall": dragonfall.Template(), "empty": nil},
		Recent:    store,
		Help:      help,
		Play:      play,
	})
}

func play(ctx context.Context, fsys fs.FS) error {
	e, err := app.Open(ctx, fsys, app.Options{Modules: []project.Module{dragonfall.Module()}})
	if err != nil {
		return err
	}
	return e.Run(ctx, os.Stdin, os.Stdout)
}
