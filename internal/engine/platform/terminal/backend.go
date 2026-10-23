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
	return t.pr.Enter(ctx)
}

func (t *Terminal) Size() (int, int, error) { return Size(t.fd) }

func (t *Terminal) Present(ctx context.Context, f platform.Frame) error {
	return t.pr.Present(ctx, f.Cells)
}

func (t *Terminal) Close(ctx context.Context) error {
	err := t.pr.Leave(ctx)
	if rerr := Restore(t.fd, t.state); err == nil {
		err = rerr
	}
	return err
}

// Events reports presses only; terminals never say when a key is released.
func (t *Terminal) Events(ctx context.Context) <-chan input.Event {
	keys := Keys(ctx, t.in)
	out := make(chan input.Event, 64)
	go func() {
		defer close(out)
		for k := range keys {
			select {
			case out <- input.Event{Key: k, Down: true}:
			case <-ctx.Done():
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
