package input

import "testing"

func TestUnit_StateHoldsAndReleases(t *testing.T) {
	// Arrange
	s := NewState()

	// Act
	s.Apply(Event{Key: 'w', Down: true}, 0)
	held, pressed := s.Down('w', .3), s.Pressed('w')
	s.EndFrame()
	later := s.Down('w', .8)
	s.Apply(Event{Key: 'a', Down: true}, 1)
	s.Apply(Event{Key: 'a', Down: false}, 1.1)

	// Assert
	if !held || !pressed || later || s.Pressed('w') || s.Down('a', 1.2) {
		t.Fatalf("held %v pressed %v later %v", held, pressed, later)
	}
}
