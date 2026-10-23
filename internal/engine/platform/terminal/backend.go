package terminal

import (
	"context"
	"fmt"
	"io"
	"os"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
)

// Terminal implements platform.Display and platform.Input on a TTY.
type Terminal struct {
	in    *os.File
	out   io.Writer
	fd    int
	state *State
	pr    *Presenter
}

func New(in *os.File, out io.Writer) *Terminal {
	return &Terminal{in: in, out: out, fd: int(in.Fd()), pr: NewPresenter(out)}
}

func (t *Terminal) Open(ctx context.Context) error {
	if _, _, err := Size(t.fd); err != nil {
		return fmt.Errorf("interactive mode needs a terminal: %w", err)
	}
	st, err := MakeRaw(t.fd)
	if err != nil {
		return err
	}
	t.state = st
	if err := t.pr.Enter(ctx); err != nil {
		return err
	}
	_, err = io.WriteString(t.out, releasesOn)
	return err
}

// Terminals report key releases only when asked: the kitty keyboard
// protocol (all keys as escape codes, with event types) and Windows
// Terminal's win32 input mode. Terminals that know neither ignore both.
const (
	releasesOn  = "\x1b[>11u\x1b[?9001h"
	releasesOff = "\x1b[<u\x1b[?9001l"
)

func (t *Terminal) Size() (int, int, error) { return Size(t.fd) }

func (t *Terminal) Present(ctx context.Context, f platform.Frame) error {
	return t.pr.Present(ctx, f.Cells)
}

func (t *Terminal) Close(ctx context.Context) error {
	_, werr := io.WriteString(t.out, releasesOff)
	err := t.pr.Leave(ctx)
	if err == nil {
		err = werr
	}
	if rerr := Restore(t.fd, t.state); err == nil {
		err = rerr
	}
	return err
}

// Events reports presses, and releases where the terminal supports one of
// the protocols enabled in Open.
func (t *Terminal) Events(ctx context.Context) <-chan input.Event {
	out := make(chan input.Event, 64)
	go func() {
		defer close(out)
		buf := make([]byte, 256)
		for {
			n, err := t.in.Read(buf)
			for _, e := range input.ParseEvents(buf[:n]) {
				select {
				case out <- e:
				case <-ctx.Done():
					return
				default: // a stalled game drops keys rather than blocking input
				}
			}
			if err != nil || ctx.Err() != nil {
				return
			}
		}
	}()
	return out
}

var (
	_ platform.Display = (*Terminal)(nil)
	_ platform.Input   = (*Terminal)(nil)
)
