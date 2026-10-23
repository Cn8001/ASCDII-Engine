package loop

import (
	"context"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
)

func TestUnit_StepperCapsCatchUp(t *testing.T) {
	// Arrange
	s := &Stepper{Step: 10 * time.Millisecond, MaxSteps: 6}
	n := 0

	// Act
	steps := s.Advance(time.Second, func(time.Duration) { n++ })

	// Assert
	if steps != 6 || n != 6 {
		t.Fatalf("steps %d", steps)
	}
}

func TestUnit_RunDeliversKeysAndStopsOnQuit(t *testing.T) {
	// Arrange
	ticks := make(chan time.Time)
	keys := make(chan input.Event, 1)
	var seen []input.Event
	draws := 0
	d := Deps{
		Ticks: ticks, Input: keys,
		Step: func(_ float64, k []input.Event) (bool, error) {
			seen = append(seen, k...)
			return len(seen) > 0, nil
		},
		Draw: func(context.Context) error { draws++; return nil },
	}
	done := make(chan error)
	go func() { done <- Run(context.Background(), d) }()

	// Act
	keys <- input.Event{Key: 'q', Down: true}
	start := time.Now()
	ticks <- start
	ticks <- start.Add(20 * time.Millisecond)
	err := <-done

	// Assert
	if err != nil || len(seen) != 1 || seen[0].Key != 'q' || draws != 1 {
		t.Fatalf("err %v seen %v draws %d", err, seen, draws)
	}
}
