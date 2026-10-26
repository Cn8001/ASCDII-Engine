package window

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
)

func TestUnit_KeysMapToEngineKeys(t *testing.T) {
	// Arrange
	cases := []struct {
		key  ebiten.Key
		ctrl bool
		want input.Key
	}{
		{ebiten.KeyA, false, 'a'}, {ebiten.KeyZ, false, 'z'}, {ebiten.KeyC, true, input.CtrlC},
		{ebiten.KeyDigit7, false, '7'}, {ebiten.KeyNumpad3, false, '3'}, {ebiten.KeySpace, false, ' '},
		{ebiten.KeyArrowLeft, false, input.Left}, {ebiten.KeyNumpadEnter, false, input.Enter},
		{ebiten.KeyEscape, false, input.Escape}, {ebiten.KeyBackquote, false, '`'}, {ebiten.KeyShift, false, 0},
	}

	for _, c := range cases {
		// Act
		got := keyOf(c.key, c.ctrl)

		// Assert
		if got != c.want {
			t.Errorf("%v ctrl=%v: got %v, want %v", c.key, c.ctrl, got, c.want)
		}
	}
}

func TestUnit_SizeFitsCellsAtScale(t *testing.T) {
	// Arrange
	auto := New(Options{Width: 1920, Height: 1080})
	fixed := New(Options{Width: 1280, Height: 720, Scale: 5})

	// Act
	ac, ar, _ := auto.Size()
	fc, fr, _ := fixed.Size()

	// Assert
	if ac != 240 || ar != 67 {
		t.Fatalf("1920x1080 picks scale 4, 480x270 dots: got %dx%d cells", ac, ar)
	}
	if fc != 128 || fr != 36 {
		t.Fatalf("scale 5: got %dx%d cells", fc, fr)
	}
}
