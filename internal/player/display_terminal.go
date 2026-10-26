//go:build !window

package player

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/launch"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
)

// playWindow hands a project folder to the window player, since the
// terminal player does not link a window library.
func (g Game) playWindow(ctx context.Context, _ *app.Engine, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	bin, err := launch.Find(self, launch.Window)
	if err != nil || g.Dir == "" {
		return fmt.Errorf("%w: the game uses the window display; run it with %s", ErrNoWindow, launch.Window)
	}
	cmd := exec.CommandContext(ctx, bin, append([]string{g.Dir, "--window"}, args...)...) // a repeated --window is harmless
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", launch.Window, err)
	}
	return nil
}
