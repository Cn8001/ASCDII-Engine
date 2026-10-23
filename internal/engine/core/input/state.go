package input

// Event is one key transition. Terminals only report presses, so their
// backends send Down events and State infers how long a key stays held.
type Event struct {
	Key  Key
	Down bool
}

// State answers "is this key down" for scripts. A press without a matching
// release counts as held for a short window that key repeat keeps extending.
type State struct {
	held    map[Key]float64
	pressed map[Key]bool
	hold    float64
}

func NewState() *State {
	return &State{held: map[Key]float64{}, pressed: map[Key]bool{}, hold: .5}
}

func (s *State) Apply(e Event, now float64) {
	if !e.Down {
		delete(s.held, e.Key)
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

// Pressed is true only in the frame the key went down.
func (s *State) Pressed(k Key) bool { return s.pressed[k] }

func (s *State) EndFrame() { clear(s.pressed) }
