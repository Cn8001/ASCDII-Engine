// Package launch is the handshake between ascdii and the editor: the editor
// is always opened from ascdii, which marks the process it starts. Opened
// any other way, the editor asks ascdii to open it again.
package launch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	Env    = "ASCDII_LAUNCHER" // set by ascdii on the editor it starts, to its own path
	Editor = "ascdii-editor"
	Player = "ascdii-player"
	CLI    = "ascdii"
)

var ErrNotFound = errors.New("program not found")

// FromLauncher reports whether ascdii started this process.
func FromLauncher() bool { return os.Getenv(Env) != "" }

// Find looks for a program next to self first, so an unpacked release works
// without PATH, then on PATH.
func Find(self, name string) (string, error) {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	near := filepath.Join(filepath.Dir(self), name)
	if info, err := os.Stat(near); err == nil && !info.IsDir() {
		return near, nil
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("%s: %w", name, ErrNotFound)
}

// StartEditor runs the editor without waiting for it, opening manifest when
// one is given.
func StartEditor(ctx context.Context, editor, manifest string) error {
	var args []string
	if manifest != "" {
		args = append(args, manifest)
	}
	cli, err := os.Executable()
	if err != nil {
		return fmt.Errorf("start editor: %w", err)
	}
	cmd := exec.CommandContext(context.WithoutCancel(ctx), editor, args...)
	cmd.Env = append(os.Environ(), Env+"="+cli) // also tells the editor where players are
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start editor: %w", err)
	}
	return cmd.Process.Release()
}

// Relaunch starts ascdii, which starts the editor again with its mark, for
// an editor that was opened directly.
func Relaunch(ctx context.Context, cli string) error {
	cmd := exec.CommandContext(context.WithoutCancel(ctx), cli)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", CLI, err)
	}
	return cmd.Process.Release()
}
