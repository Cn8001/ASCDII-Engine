// Package input turns raw terminal bytes into keys and names them so key
// bindings can live in scene files.
package input

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
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

// ParseEvents decodes one read into key transitions. Sequences of the kitty
// keyboard protocol and of Windows Terminal's win32 input mode carry releases
// and become exact events; everything else is a press as in Parse.
func ParseEvents(b []byte) []Event {
	var out []Event
	for len(b) > 0 {
		if b[0] == 27 && len(b) > 2 && b[1] == '[' {
			end := 2
			for end < len(b) && (b[end] < 0x40 || b[end] > 0x7e) {
				end++
			}
			if end < len(b) {
				if e, ok := csiEvent(string(b[2:end]), b[end]); ok {
					if e.Key != 0 {
						out = append(out, e)
					}
					b = b[end+1:]
					continue
				}
			}
		}
		j := 1
		for j < len(b) && b[j] != 27 {
			j++
		}
		for _, k := range Parse(b[:j]) {
			out = append(out, Event{Key: k, Down: true})
		}
		b = b[j:]
	}
	return out
}

// csiEvent reads the parameters and final byte of a CSI sequence that
// reports a key with its press or release; ok is false for other sequences.
func csiEvent(params string, final byte) (Event, bool) {
	fields := strings.Split(params, ";")
	num := func(i, sub int) int {
		if i >= len(fields) {
			return 0
		}
		parts := strings.Split(fields[i], ":")
		if sub >= len(parts) {
			return 0
		}
		n, err := strconv.Atoi(parts[sub])
		if err != nil {
			return 0
		}
		return n
	}
	switch {
	case final == 'u' && params != "" && params[0] != '?' && params[0] != '>':
		code, mods, ev := num(0, 0), num(1, 0), num(1, 1)
		k := kittyKey(code, mods)
		return Event{Key: k, Down: ev != 3, Exact: true}, true
	case final == '_' && len(fields) >= 4:
		return Event{Key: winKey(num(0, 0), num(2, 0)), Down: num(3, 0) == 1, Exact: true}, true
	case final >= 'A' && final <= 'D' && strings.Contains(params, ":"):
		k := map[byte]Key{'A': Up, 'B': Down, 'C': Right, 'D': Left}[final]
		return Event{Key: k, Down: num(1, 1) != 3, Exact: true}, true
	}
	return Event{}, false
}

// kittyKey maps a kitty protocol key code; modifiers count only for Ctrl+C.
func kittyKey(code, mods int) Key {
	ctrl := mods > 0 && (mods-1)&4 != 0
	switch {
	case ctrl && (code == 'c' || code == 'C'):
		return CtrlC
	case code == 13:
		return Enter
	case code == 27:
		return Escape
	case code == 9:
		return Tab
	case code == 127 || code == 8:
		return Backspace
	case code >= 32 && code < 57344:
		return Key(unicode.ToLower(rune(code)))
	}
	return 0 // modifier and function keys the engine does not name
}

// winKey maps a Windows virtual key code, or the character it typed.
func winKey(vk, char int) Key {
	switch {
	case vk >= 'A' && vk <= 'Z':
		if char == 3 {
			return CtrlC
		}
		return Key(unicode.ToLower(rune(vk)))
	case vk >= '0' && vk <= '9':
		return Key(vk)
	}
	if k, ok := map[int]Key{0x25: Left, 0x26: Up, 0x27: Right, 0x28: Down, 0x0d: Enter, 0x1b: Escape, 0x09: Tab, 0x08: Backspace, 0x20: ' '}[vk]; ok {
		return k
	}
	if char >= 32 {
		return Key(unicode.ToLower(rune(char)))
	}
	return 0
}
