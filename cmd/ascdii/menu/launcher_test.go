package menu

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/recent"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/cell"
)

func TestUnit_MenuMovesWrapsAndChooses(t *testing.T) {
	// Arrange
	m := &Menu{Items: []Item{{Value: "a"}, {Value: "b"}, {Value: "c"}}}

	// Act
	m.Key(input.Up)
	wrapped := m.Selected().Value
	m.Key(input.Down)
	res := m.Key(input.Enter)

	// Assert
	if wrapped != "c" || m.Selected().Value != "a" || res != Chosen {
		t.Fatalf("wrapped %s selected %s res %v", wrapped, m.Selected().Value, res)
	}
}

func TestUnit_PromptEditsText(t *testing.T) {
	// Arrange
	m := &Menu{Prompt: true, Text: "ab"}

	// Act
	m.Key('c')
	m.Key(input.Backspace)
	m.Key(input.Backspace)
	m.Key('q')

	// Assert
	if m.Text != "aq" || m.Key(input.Escape) != Back {
		t.Fatalf("text %q", m.Text)
	}
}

func TestUnit_RenderShowsTitleAndSelection(t *testing.T) {
	// Arrange
	g := cell.New(100, 34)
	m := &Menu{Title: "MAIN MENU", Items: []Item{{Label: "Play Dragonfall"}, {Label: "Quit"}}}

	// Act
	Render(g, m, "0.0.1", "", 2)

	// Assert
	found := map[string]bool{}
	for y := 0; y < g.H; y++ {
		row := ""
		for x := 0; x < g.W; x++ {
			c, _ := g.At(x, y)
			row += string(c.Rune)
		}
		for _, want := range []string{"A S C D I I", "> Play Dragonfall", "MAIN MENU"} {
			if contains(row, want) {
				found[want] = true
			}
		}
	}
	if len(found) != 3 {
		t.Fatalf("found %v", found)
	}
}

func contains(s, sub string) bool { return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0) }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestUnit_NewProjectFlowCreatesAndPlays(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	var played fs.FS
	l := &launcher{d: Deps{
		Templates: map[string]fs.FS{"empty": nil, "dragonfall": fstest.MapFS{}},
		Recent:    recent.Store{Path: filepath.Join(dir, "recent.json")},
		Play:      func(_ context.Context, f fs.FS) error { played = f; return nil },
	}}
	ctx := context.Background()
	l.push(home, l.homeMenu(ctx))

	// Act
	l.choose(ctx, home, Item{Value: "new"})
	l.choose(ctx, newName, Item{Value: "Space Game"})
	l.choose(ctx, newDir, Item{Value: filepath.Join(dir, "sg")})
	action, _ := l.choose(ctx, newTemplate, Item{Value: "empty"})
	err := action(ctx)

	// Assert
	if err != nil || played == nil || l.status != "" || len(l.stack) != 1 {
		t.Fatalf("err %v status %q stack %d", err, l.status, len(l.stack))
	}
	if m := l.stack[0]; m.Items[0].Label != "Continue Space-Game" {
		t.Fatalf("home %+v", m.Items[0])
	}
}
