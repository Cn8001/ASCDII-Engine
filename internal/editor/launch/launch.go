// Package launch is the handshake between ascdii and the editor: the editor
// is always opened from ascdii, which marks the process it starts. Opened
// any other way, the editor hands over to ascdii instead.
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

// OpenCLI opens ascdii in a new terminal window, for an editor that was
// started directly.
func OpenCLI(ctx context.Context, cli string) error {
	ctx = context.WithoutCancel(ctx)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", "", cli)
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", "-a", "Terminal", cli)
	default:
		// Tried in order: the program and the flag the command follows.
		terminals := [][2]string{
			{"x-terminal-emulator", "-e"}, {"gnome-terminal", "--"}, {"konsole", "-e"}, {"xfce4-terminal", "-x"},
			{"wezterm", "start"}, {"kitty", ""}, {"alacritty", "-e"}, {"foot", ""}, {"xterm", "-e"},
		}
		if t := os.Getenv("TERMINAL"); t != "" {
			cmd = exec.CommandContext(ctx, t, "-e", cli)
			break
		}
		for _, t := range terminals {
			if p, err := exec.LookPath(t[0]); err == nil {
				args := []string{cli}
				if t[1] != "" {
					args = []string{t[1], cli}
				}
				cmd = exec.CommandContext(ctx, p, args...)
				break
			}
		}
	}
	if cmd == nil {
		return fmt.Errorf("terminal: %w", ErrNotFound)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %s: %w", CLI, err)
	}
	return cmd.Process.Release()
}
