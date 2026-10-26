package window

import (
	"github.com/hajimehoshi/ebiten/v2"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
)

// keyOf names a physical key the way the terminal reports it: letters lower
// case, punctuation as the US layout prints it. 0 is a key the engine does
// not name, such as a modifier.
func keyOf(k ebiten.Key, ctrl bool) input.Key {
	switch {
	case ctrl && k == ebiten.KeyC:
		return input.CtrlC
	case k >= ebiten.KeyA && k <= ebiten.KeyZ:
		return input.Key('a' + rune(k-ebiten.KeyA))
	case k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9:
		return input.Key('0' + rune(k-ebiten.KeyDigit0))
	case k >= ebiten.KeyNumpad0 && k <= ebiten.KeyNumpad9:
		return input.Key('0' + rune(k-ebiten.KeyNumpad0))
	}
	switch k {
	case ebiten.KeyArrowUp:
		return input.Up
	case ebiten.KeyArrowDown:
		return input.Down
	case ebiten.KeyArrowLeft:
		return input.Left
	case ebiten.KeyArrowRight:
		return input.Right
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		return input.Enter
	case ebiten.KeyEscape:
		return input.Escape
	case ebiten.KeyTab:
		return input.Tab
	case ebiten.KeyBackspace:
		return input.Backspace
	}
	if r, ok := punctuation(k); ok {
		return input.Key(r)
	}
	return 0
}

func punctuation(k ebiten.Key) (rune, bool) {
	keys := []ebiten.Key{ebiten.KeySpace, ebiten.KeyMinus, ebiten.KeyEqual, ebiten.KeyComma, ebiten.KeyPeriod, ebiten.KeySlash,
		ebiten.KeySemicolon, ebiten.KeyQuote, ebiten.KeyBracketLeft, ebiten.KeyBracketRight, ebiten.KeyBackslash, ebiten.KeyBackquote}
	for i, r := range " -=,./;'[]\\`" {
		if keys[i] == k {
			return r, true
		}
	}
	return 0, false
}
