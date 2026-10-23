// Package input turns raw terminal bytes into keys and names them so key
// bindings can live in scene files.
package input

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

var ErrUnknownKey = errors.New("unknown key")

type Key rune

const (
	Up Key = -1 - iota
	Down
	Left
	Right
	Enter
	Escape
	Tab
	Backspace
	CtrlC Key = 3
)

func names() map[Key]string {
	return map[Key]string{
		Up: "up", Down: "down", Left: "left", Right: "right", Enter: "enter", Escape: "esc",
		Tab: "tab", Backspace: "backspace", CtrlC: "ctrl+c", ' ': "space",
	}
}

func (k Key) String() string {
	if n, ok := names()[k]; ok {
		return n
	}
	return string(rune(k))
}

func ParseName(s string) (Key, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	for k, n := range names() {
		if n == s {
			return k, nil
		}
	}
	if utf8.RuneCountInString(s) == 1 {
		r, _ := utf8.DecodeRuneInString(s)
		return Key(r), nil
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownKey, s)
}

// Parse decodes one read; terminals deliver an escape sequence in a single write.
func Parse(b []byte) []Key {
	var out []Key
	for len(b) > 0 {
		if b[0] == 27 {
			if len(b) >= 3 && (b[1] == '[' || b[1] == 'O') {
				if k, ok := map[byte]Key{'A': Up, 'B': Down, 'C': Right, 'D': Left}[b[2]]; ok {
					out = append(out, k)
					b = b[3:]
					continue
				}
				b = skipSequence(b)
				continue
			}
			out = append(out, Escape)
			b = b[1:]
			continue
		}
		r, n := utf8.DecodeRune(b)
		b = b[n:]
		switch r {
		case '\r', '\n':
			out = append(out, Enter)
		case '\t':
			out = append(out, Tab)
		case 127, 8:
			out = append(out, Backspace)
		case utf8.RuneError:
		default:
			out = append(out, Key(r))
		}
	}
	return out
}

// skipSequence drops unsupported CSI sequences such as mouse reports.
func skipSequence(b []byte) []byte {
	for i := 2; i < len(b); i++ {
		if b[i] >= 0x40 && b[i] <= 0x7e {
			return b[i+1:]
		}
	}
	return nil
}
