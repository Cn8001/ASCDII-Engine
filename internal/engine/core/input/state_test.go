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

func TestUnit_MouseReportsBecomeMouseState(t *testing.T) {
	// Arrange
	s := NewState()
	evs := ParseEvents([]byte("\x1b[<0;11;6M\x1b[<64;11;6M"))

	// Act
	for _, e := range evs {
		s.Apply(e, 0)
	}
	m := s.Mouse()

	// Assert
	if !m.Seen || m.X != 10 || m.Y != 5 || !m.Held[ButtonLeft] || !m.Clicked[ButtonLeft] || m.Wheel != 1 {
		t.Fatalf("mouse %+v from %+v", m, evs)
	}
}

func TestUnit_MouseReleaseAndMoveKeepPosition(t *testing.T) {
	// Arrange
	s := NewState()
	s.Apply(Event{Mouse: true, X: 3, Y: 4, Button: ButtonRight, Down: true}, 0)
	s.EndFrame()

	// Act
	for _, e := range ParseEvents([]byte("\x1b[<2;8;9m\x1b[<35;9;9M")) {
		s.Apply(e, 0)
	}
	m := s.Mouse()

	// Assert
	if m.Held[ButtonRight] || m.Clicked[ButtonRight] || m.X != 8 || m.Y != 8 {
		t.Fatalf("mouse %+v", m)
	}
}

func TestUnit_MouseEventsAreNotKeys(t *testing.T) {
	// Arrange
	b := []byte("\x1b[<0;1;1Ma")

	// Act
	evs := ParseEvents(b)

	// Assert
	if len(evs) != 2 || !evs[0].Mouse || evs[1].Key != 'a' {
		t.Fatalf("events %+v", evs)
	}
}
