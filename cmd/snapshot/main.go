// Command snapshot renders a project scene to PNG, drawing Braille dots as
// they would appear in a terminal. Used for docs and visual review.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/headless"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	dir := flag.String("project", ".", "project folder")
	out := flag.String("o", "snapshot.png", "output png")
	cols := flag.Int("cols", 160, "columns")
	rows := flag.Int("rows", 50, "rows")
	secs := flag.Float64("time", 1, "seconds to simulate")
	hour := flag.Float64("hour", -1, "override time of day")
	var sets, off []string
	flag.Func("off", "render pass to disable, repeatable", func(v string) error { off = append(off, v); return nil })
	flag.Func("set", "entity:prop=json, repeatable", func(v string) error { sets = append(sets, v); return nil })
	flag.Parse()
	e, err := app.Open(ctx, os.DirFS(*dir), app.Options{Cols: *cols, Rows: *rows, Modules: []project.Module{dragonfall.Module()}})
	if err != nil {
		return err
	}
	e.Resize(*cols, *rows)
	if *hour >= 0 {
		e.Scene().Env.Hour = *hour
	}
	for _, name := range off {
		if err := e.Pipeline().SetEnabled(name, false); err != nil {
			return err
		}
	}
	for _, kv := range sets {
		if err := apply(e.Scene(), kv); err != nil {
			return err
		}
	}
	for t := 0.0; t < *secs; t += 1.0 / 30 {
		if _, err := e.Update(1.0/30, nil); err != nil {
			return err
		}
	}
	return write(ctx, *out, e.Render())
}

func apply(s *scene.Scene, kv string) error {
	target, rest, ok := strings.Cut(kv, ":")
	name, raw, ok2 := strings.Cut(rest, "=")
	if !ok || !ok2 {
		return fmt.Errorf("set %q: want entity:prop=json", kv)
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		v = raw
	}
	if target == "camera" {
		return s.CameraProps().Set(name, v)
	}
	e := s.Find(target)
	if e == nil {
		return fmt.Errorf("set %q: %w", kv, scene.ErrNodeNotFound)
	}
	return e.Set(name, v)
}

func write(ctx context.Context, path string, g *cell.Grid) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := headless.WritePNG(ctx, f, g); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
