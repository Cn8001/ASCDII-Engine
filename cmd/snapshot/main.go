// Command snapshot renders a project scene to PNG, drawing Braille dots as
// they would appear in a terminal. Used for docs and visual review.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	stdcolor "image/color"
	"image/png"
	"os"
	"os/signal"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/dragonfall"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/app"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/braille"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/scene"
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
	e, err := app.Open(ctx, os.DirFS(*dir), app.Options{Cols: *cols, Rows: *rows, Packs: []project.Pack{dragonfall.Pack()}})
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
		return s.Camera.Set(name, v)
	}
	e := s.Find(target)
	if e == nil {
		return fmt.Errorf("set %q: %w", kv, scene.ErrNodeNotFound)
	}
	return e.Set(name, v)
}

func write(ctx context.Context, path string, g *cell.Grid) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const cw, ch = 8, 16
	img := image.NewRGBA(image.Rect(0, 0, g.W*cw, g.H*ch))
	for cy := 0; cy < g.H; cy++ {
		for cx := 0; cx < g.W; cx++ {
			c, _ := g.At(cx, cy)
			bg := stdcolor.RGBA{c.BG.R, c.BG.G, c.BG.B, 255}
			fg := stdcolor.RGBA{c.FG.R, c.FG.G, c.FG.B, 255}
			for y := 0; y < ch; y++ {
				for x := 0; x < cw; x++ {
					img.SetRGBA(cx*cw+x, cy*ch+y, bg)
				}
			}
			switch {
			case c.Rune >= 0x2800 && c.Rune <= 0x28ff:
				mask := uint8(c.Rune - 0x2800)
				for dy := 0; dy < 4; dy++ {
					for dx := 0; dx < 2; dx++ {
						if mask&braille.Bit(dx, dy) == 0 {
							continue
						}
						for y := 0; y < 3; y++ {
							for x := 0; x < 3; x++ {
								img.SetRGBA(cx*cw+dx*4+x, cy*ch+dy*4+y, fg)
							}
						}
					}
				}
			case c.Rune != ' ':
				for y := 4; y < 12; y++ {
					for x := 2; x < 6; x++ {
						img.SetRGBA(cx*cw+x, cy*ch+y, fg)
					}
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return fmt.Errorf("encode png: %w", err)
	}
	return f.Close()
}
