// Command brand writes the app icon PNG that Wails turns into the window and
// Windows executable icons.
package main

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/brand"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	out := "cmd/editor/build/appicon.png"
	if len(args) > 0 {
		out = args[0]
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("brand: %w", err)
	}
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("brand: %w", err)
	}
	if err := png.Encode(f, brand.Icon(1024)); err != nil {
		f.Close()
		return fmt.Errorf("brand: %w", err)
	}
	return f.Close()
}
