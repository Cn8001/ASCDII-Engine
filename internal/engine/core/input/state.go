package input

import "math"

// Event is one key transition. Most terminals only report presses, so State
// infers how long such a key stays held; Exact events come from sources that
// also report the release, and stay held until it arrives.
//
// An event with Mouse set is a pointer report instead: the cell under the
// pointer and a button going down or up (ButtonNone for a move).
type Event struct {
	Key    Key
	Down   bool
	Exact  bool
	Mouse  bool
	X, Y   int // cell under the pointer, from the top left
	Button Button
}

// Button is a mouse button; the wheel reports as two buttons that only go
// down.
type Button int

const (
	ButtonNone Button = iota
	ButtonLeft
	ButtonMiddle
	ButtonRight
	WheelUp
	WheelDown
)

// Mouse is the pointer as scripts see it: Seen stays false until the first
// report, since many terminals never send one.
type Mouse struct {
	X, Y    int
	Seen    bool
	Held    [4]bool // indexed by ButtonLeft, ButtonMiddle, ButtonRight
	Clicked [4]bool // went down in this frame
	Wheel   int     // steps this frame, up positive
}

// State answers "is this key down" for scripts. An inexact press counts as
// held for a short window that key repeat keeps extending.
type State struct {
	held    map[Key]float64
	pressed map[Key]bool
	hold    float64
	exact   bool
	mouse   Mouse
	cols    int
	rows    int
}

func NewState() *State {
	return &State{held: map[Key]float64{}, pressed: map[Key]bool{}, hold: .5}
}

func (s *State) Apply(e Event, now float64) {
	if e.Mouse {
		s.applyMouse(e)
		return
	}
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

func (s *State) EndFrame() {
	clear(s.pressed)
	s.mouse.Clicked = [4]bool{}
	s.mouse.Wheel = 0
}

func (s *State) applyMouse(e Event) {
	s.mouse.X, s.mouse.Y, s.mouse.Seen = e.X, e.Y, true
	switch e.Button {
	case ButtonLeft, ButtonMiddle, ButtonRight:
		s.mouse.Held[e.Button] = e.Down
		if e.Down {
			s.mouse.Clicked[e.Button] = true
		}
	case WheelUp:
		s.mouse.Wheel++
	case WheelDown:
		s.mouse.Wheel--
	}
}

// Mouse is the pointer state of the current frame.
func (s *State) Mouse() Mouse { return s.mouse }

// SetScreen records the size of the display in cells, so a pointer cell can
// be turned into a point of the world.
func (s *State) SetScreen(cols, rows int) { s.cols, s.rows = cols, rows }

func (s *State) Screen() (cols, rows int) { return s.cols, s.rows }
