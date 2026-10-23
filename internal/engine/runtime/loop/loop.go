// Package loop drives simulation on a fixed step and presentation on ticks.
package loop

import (
	"context"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
)

// Stepper decouples simulation from the refresh rate and limits catch up
// work after a stall.
type Stepper struct {
	Step     time.Duration
	MaxSteps int
	acc      time.Duration
}

func (s *Stepper) Advance(elapsed time.Duration, fn func(time.Duration)) int {
	elapsed = max(0, min(elapsed, 250*time.Millisecond))
	s.acc += elapsed
	n := 0
	for s.acc >= s.Step && n < s.MaxSteps {
		fn(s.Step)
		s.acc -= s.Step
		n++
	}
	if n == s.MaxSteps && s.acc >= s.Step {
		s.acc = 0
	}
	return n
}

type Deps struct {
	Ticks <-chan time.Time
	Input <-chan input.Event
	Step  func(dt float64, events []input.Event) (quit bool, err error)
	Draw  func(ctx context.Context) error
	Rate  time.Duration // simulation step, 60 Hz when zero
}

func Run(ctx context.Context, d Deps) error {
	rate := d.Rate
	if rate <= 0 {
		rate = time.Second / 60
	}
	st := &Stepper{Step: rate, MaxSteps: 6}
	var pending []input.Event
	var last time.Time
	events := d.Input
	for {
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			pending = append(pending, e)
		case now := <-d.Ticks:
			if last.IsZero() {
				last = now
			}
			var quit bool
			var err error
			st.Advance(now.Sub(last), func(step time.Duration) {
				if quit || err != nil {
					return
				}
				quit, err = d.Step(step.Seconds(), pending)
				pending = pending[:0]
			})
			last = now
			if err != nil || quit {
				return err
			}
			if err := d.Draw(ctx); err != nil {
				return err
			}
		}
	}
}
