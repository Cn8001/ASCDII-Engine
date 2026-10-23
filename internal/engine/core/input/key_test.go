package input

import "testing"

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
