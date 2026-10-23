// Package headless is a display without a screen: it writes frames as ANSI
// text or PNG, for previews, screenshots and tests.
package headless

import (
	"context"
	"fmt"
	"image"
	stdcolor "image/color"
	"image/png"
	"io"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/terminal"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/braille"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
)

type Format int

const (
	ANSI Format = iota
	Plain
	PNG
)

// Display writes every presented frame to Out in the chosen format.
type Display struct {
	Out        io.Writer
	Format     Format
	Cols, Rows int
}

func (d *Display) Open(context.Context) error  { return nil }
func (d *Display) Close(context.Context) error { return nil }
func (d *Display) Size() (int, int, error)     { return d.Cols, d.Rows, nil }

func (d *Display) Present(ctx context.Context, f platform.Frame) error {
	if d.Format == PNG {
		return WritePNG(ctx, d.Out, f.Cells)
	}
	return terminal.WritePreview(ctx, d.Out, f.Cells, d.Format == Plain)
}

// Events never fires; headless runs are driven by the caller.
func (d *Display) Events(context.Context) <-chan input.Event {
	ch := make(chan input.Event)
	close(ch)
	return ch
}

var (
	_ platform.Display = (*Display)(nil)
	_ platform.Input   = (*Display)(nil)
)

// WritePNG draws cells as a terminal would: 8x16 pixel cells, Braille dots as
// 3x3 squares, other glyphs as a block in the foreground color.
func WritePNG(ctx context.Context, w io.Writer, g *cell.Grid) error {
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
	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}
	return nil
}
