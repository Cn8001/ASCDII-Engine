// Command gen writes the Dragonfall template project from BuildScene, so the
// scene file and the code that authors it never drift apart.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	dir := "internal/dragonfall/template"
	if len(args) > 0 {
		dir = args[0]
	}
	files, err := dragonfall.TemplateFiles()
	if err != nil {
		return err
	}
	for name, b := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("gen: %w", err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return fmt.Errorf("gen: %w", err)
		}
	}
	return nil
}
