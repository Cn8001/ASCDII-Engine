package input

import (
	"fmt"
	"testing"
)

func TestUnit_ParseArrowsAndText(t *testing.T) {
	// Arrange
	raw := []byte("w\x1b[Aq\r\x1b")

	// Act
	keys := Parse(raw)

	// Assert
	want := []Key{'w', Up, 'q', Enter, Escape}
	if len(keys) != len(want) {
		t.Fatalf("got %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("got %v", keys)
		}
	}
}

func TestUnit_ParseSkipsMouseReports(t *testing.T) {
	// Arrange
	raw := []byte("\x1b[<35;88;6Mx")

	// Act
	keys := Parse(raw)

	// Assert
	if len(keys) != 1 || keys[0] != 'x' {
		t.Fatalf("got %v", keys)
	}
}

func TestUnit_NamesRoundTrip(t *testing.T) {
	// Arrange
	all := []Key{Up, Space(), 'e', Enter}

	// Act
	for _, k := range all {
		back, err := ParseName(k.String())

		// Assert
		if err != nil || back != k {
			t.Fatalf("%v -> %q -> %v %v", k, k.String(), back, err)
		}
	}
}

func Space() Key { return ' ' }

func TestUnit_ParseEventsReadsReleases(t *testing.T) {
	// Arrange
	kitty := []byte("\x1b[119;1:1u\x1b[100u\x1b[119;1:3u\x1b[1;1:3A")
	win := []byte("\x1b[87;17;119;1;0;1_\x1b[87;17;119;0;0;1_\x1b[67;46;3;1;8;1_")
	plain := []byte("wd\x1b[A")

	// Act
	k, w, p := ParseEvents(kitty), ParseEvents(win), ParseEvents(plain)

	// Assert
	wantK := []Event{{'w', true, true}, {'d', true, true}, {'w', false, true}, {Up, false, true}}
	wantW := []Event{{'w', true, true}, {'w', false, true}, {CtrlC, true, true}}
	wantP := []Event{{'w', true, false}, {'d', true, false}, {Up, true, false}}
	for i, c := range []struct{ got, want []Event }{{k, wantK}, {w, wantW}, {p, wantP}} {
		if fmt.Sprint(c.got) != fmt.Sprint(c.want) {
			t.Fatalf("case %d: got %v want %v", i, c.got, c.want)
		}
	}
}
