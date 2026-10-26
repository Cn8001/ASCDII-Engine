// Command player is ascdii-player, the program a game runs on. A build is
// this binary renamed, with the game appended or in data/ beside it; started
// with a project folder instead, it plays that project.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/player"
)

const usage = `usage:
  ascdii-player <project folder> [--blocks] [--window | --terminal]
  ascdii-player <project folder> --preview [-size 160x50] [-time 2] [-blocks]
A built game runs without arguments.`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	modules := []project.Module{dragonfall.Module()}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	g, closer, err := player.Bundled(ctx, self, modules)
	if err == nil {
		defer closer.Close()
		return g.Play(ctx, args)
	}
	if !errors.Is(err, player.ErrNoGame) {
		return err
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		return errors.New(usage)
	}
	return player.Project(args[0], modules).Play(ctx, args[1:])
}
