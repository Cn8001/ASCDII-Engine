package loop

import (
	"context"
	"testing"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
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
	keys := make(chan input.Key, 1)
	var seen []input.Key
	draws := 0
	d := Deps{
		Ticks: ticks, Keys: keys,
		Step: func(_ float64, k []input.Key) (bool, error) {
			seen = append(seen, k...)
			return len(seen) > 0, nil
		},
		Draw: func(context.Context) error { draws++; return nil },
	}
	done := make(chan error)
	go func() { done <- Run(context.Background(), d) }()

	// Act
	keys <- 'q'
	start := time.Now()
	ticks <- start
	ticks <- start.Add(20 * time.Millisecond)
	err := <-done

	// Assert
	if err != nil || len(seen) != 1 || seen[0] != 'q' || draws != 1 {
		t.Fatalf("err %v seen %v draws %d", err, seen, draws)
	}
}
