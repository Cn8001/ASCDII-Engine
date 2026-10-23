package input

import "math"

// Event is one key transition. Most terminals only report presses, so State
// infers how long such a key stays held; Exact events come from sources that
// also report the release, and stay held until it arrives.
type Event struct {
	Key   Key
	Down  bool
	Exact bool
}

// State answers "is this key down" for scripts. An inexact press counts as
// held for a short window that key repeat keeps extending.
type State struct {
	held    map[Key]float64
	pressed map[Key]bool
	hold    float64
	exact   bool
}

func NewState() *State {
	return &State{held: map[Key]float64{}, pressed: map[Key]bool{}, hold: .5}
}

func (s *State) Apply(e Event, now float64) {
	if !e.Down {
		delete(s.held, e.Key)
		return
	}
	if e.Exact {
		s.exact = true
		s.held[e.Key] = math.Inf(1)
		s.pressed[e.Key] = true
		return
	}
	window := s.hold
	if until, ok := s.held[e.Key]; ok && until > now {
		window = .15 // a repeat: the key is clearly still down
	}
	s.held[e.Key] = now + window
	s.pressed[e.Key] = true
}

func (s *State) Down(k Key, now float64) bool { return s.held[k] > now }

// Exact reports whether releases are known, so several keys can be held at
// once; without them only the last key keeps repeating.
func (s *State) Exact() bool { return s.exact }

// Pressed is true only in the frame the key went down.
func (s *State) Pressed(k Key) bool { return s.pressed[k] }

func (s *State) EndFrame() { clear(s.pressed) }
