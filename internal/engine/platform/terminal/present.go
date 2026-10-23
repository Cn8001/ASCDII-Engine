// Package term is the terminal backend: raw mode, size and an ANSI presenter
// that only rewrites changed cells.
package terminal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
)

var ErrNotTerminal = errors.New("not a terminal")

type Presenter struct {
	out   io.Writer
	front []cell.Cell
	w, h  int
	buf   bytes.Buffer
}

func NewPresenter(out io.Writer) *Presenter { return &Presenter{out: out} }

func (p *Presenter) Enter(ctx context.Context) error {
	return p.write(ctx, "\x1b[?1049h\x1b[?25l\x1b[2J")
}

func (p *Presenter) Leave(ctx context.Context) error {
	return p.write(ctx, "\x1b[0m\x1b[?25h\x1b[?1049l")
}

func (p *Presenter) write(ctx context.Context, s string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := io.WriteString(p.out, s); err != nil {
		return fmt.Errorf("terminal write: %w", err)
	}
	return nil
}

func sgr(b *bytes.Buffer, fg, bg color.Color) {
	b.WriteString("\x1b[38;2;")
	writeRGB(b, fg)
	b.WriteString(";48;2;")
	writeRGB(b, bg)
	b.WriteByte('m')
}

func writeRGB(b *bytes.Buffer, c color.Color) {
	b.WriteString(strconv.Itoa(int(c.R)))
	b.WriteByte(';')
	b.WriteString(strconv.Itoa(int(c.G)))
	b.WriteByte(';')
	b.WriteString(strconv.Itoa(int(c.B)))
}

// Present writes runs of changed cells. The bottom right cell is skipped
// because writing it scrolls some terminals.
func (p *Presenter) Present(ctx context.Context, g *cell.Grid) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	full := g.W != p.w || g.H != p.h
	if full {
		p.w, p.h = g.W, g.H
		p.front = make([]cell.Cell, g.W*g.H)
	}
	cells := g.Cells()
	b := &p.buf
	b.Reset()
	if full {
		b.WriteString("\x1b[2J")
	}
	var lastFG, lastBG color.Color
	known := false
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; {
			i := y*g.W + x
			if (x == g.W-1 && y == g.H-1) || (!full && cells[i] == p.front[i]) {
				x++
				continue
			}
			b.WriteString("\x1b[")
			b.WriteString(strconv.Itoa(y + 1))
			b.WriteByte(';')
			b.WriteString(strconv.Itoa(x + 1))
			b.WriteByte('H')
			for ; x < g.W; x++ {
				i = y*g.W + x
				c := cells[i]
				if (x == g.W-1 && y == g.H-1) || (!full && c == p.front[i]) {
					break
				}
				if !known || c.FG != lastFG || c.BG != lastBG {
					sgr(b, c.FG, c.BG)
					lastFG, lastBG, known = c.FG, c.BG, true
				}
				r := c.Rune
				if r == 0 {
					r = ' '
				}
				b.WriteRune(r)
			}
		}
	}
	if b.Len() == 0 {
		return nil
	}
	if _, err := p.out.Write(b.Bytes()); err != nil {
		return fmt.Errorf("terminal write: %w", err)
	}
	copy(p.front, cells)
	return nil
}

// WritePreview prints a grid once, for screenshots and CI without a TTY.
func WritePreview(ctx context.Context, w io.Writer, g *cell.Grid, plain bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var b bytes.Buffer
	for y := 0; y < g.H; y++ {
		known := false
		var fg, bg color.Color
		for x := 0; x < g.W; x++ {
			c, _ := g.At(x, y)
			if !plain && (!known || c.FG != fg || c.BG != bg) {
				sgr(&b, c.FG, c.BG)
				fg, bg, known = c.FG, c.BG, true
			}
			b.WriteRune(c.Rune)
		}
		if !plain {
			b.WriteString("\x1b[0m")
		}
		b.WriteByte('\n')
	}
	if _, err := w.Write(b.Bytes()); err != nil {
		return fmt.Errorf("write preview: %w", err)
	}
	return nil
}

// Keys reads raw input until r fails or ctx ends. Keys are dropped rather than
// blocking when the game falls behind.
func Keys(ctx context.Context, r io.Reader) <-chan input.Key {
	out := make(chan input.Key, 64)
	go func() {
		defer close(out)
		buf := make([]byte, 64)
		for {
			n, err := r.Read(buf)
			for _, k := range input.Parse(buf[:n]) {
				select {
				case out <- k:
				case <-ctx.Done():
					return
				default:
				}
			}
			if err != nil || ctx.Err() != nil {
				return
			}
		}
	}()
	return out
}
