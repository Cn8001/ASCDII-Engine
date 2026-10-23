// Package launcher is the terminal main menu shown when the ascdii binary is
// started without arguments: play the demo, open or create a project.
package launcher

import (
	"strings"
	"unicode/utf8"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
)

type Item struct {
	Label string
	Hint  string
	Value string
}

// Menu is a list or a text prompt; the launcher keeps a stack of them.
type Menu struct {
	Title  string
	Items  []Item
	Cursor int
	Prompt bool
	Text   string
}

type Result int

const (
	Pending Result = iota
	Chosen
	Back
)

// Key updates the menu and reports whether the user chose or backed out.
func (m *Menu) Key(k input.Key) Result {
	switch {
	case k == input.Escape:
		return Back
	case k == input.Enter:
		if m.Prompt && strings.TrimSpace(m.Text) == "" {
			return Pending
		}
		return Chosen
	case m.Prompt && k == input.Backspace:
		if _, n := utf8.DecodeLastRuneInString(m.Text); n > 0 {
			m.Text = m.Text[:len(m.Text)-n]
		}
	case m.Prompt && k > 31:
		m.Text += string(rune(k))
	case !m.Prompt && (k == input.Up || k == 'w' || k == 'k'):
		m.Cursor = (m.Cursor - 1 + len(m.Items)) % max(1, len(m.Items))
	case !m.Prompt && (k == input.Down || k == 's' || k == 'j' || k == input.Tab):
		m.Cursor = (m.Cursor + 1) % max(1, len(m.Items))
	case !m.Prompt && k == 'q':
		return Back
	}
	return Pending
}

func (m *Menu) Selected() Item {
	if m.Prompt || len(m.Items) == 0 {
		return Item{Value: strings.TrimSpace(m.Text)}
	}
	return m.Items[m.Cursor]
}
