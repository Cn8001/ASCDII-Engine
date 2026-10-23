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

func TestUnit_ExactKeysStayHeldTogetherUntilReleased(t *testing.T) {
	// Arrange
	s := NewState()

	// Act
	s.Apply(Event{Key: 'w', Down: true, Exact: true}, 0)
	s.Apply(Event{Key: 'd', Down: true, Exact: true}, 0.1)
	both := s.Down('w', 5) && s.Down('d', 5)
	s.Apply(Event{Key: 'w', Exact: true}, 5)

	// Assert
	if !both || s.Down('w', 5.1) || !s.Down('d', 5.1) || !s.Exact() {
		t.Fatalf("both %v w %v d %v exact %v", both, s.Down('w', 5.1), s.Down('d', 5.1), s.Exact())
	}
}
