//go:build window

package player

import (
	"context"
	"fmt"
	"os"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/window"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/winbrand"
)

// playWindow shows the game in a desktop window. Only players built with the
// window tag link Ebitengine; the terminal player stays free of it.
func (g Game) playWindow(ctx context.Context, e *app.Engine, _ []string) error {
	if err := pipeline.UsePixels(e.Pipeline()); err != nil {
		return err
	}
	m := e.Project().Manifest
	o := window.Options{Title: m.Name, Width: m.Display.Width, Height: m.Display.Height, Scale: m.Display.Scale, Fullscreen: m.Display.Fullscreen}
	if b, err := winbrand.For(m, g.FS, e.Library()); err == nil {
		o.Icon = b.Icon
	} else {
		fmt.Fprintln(os.Stderr, err) // the default icon beats no game
	}
	return window.Run(ctx, o, e.RunOn)
}
