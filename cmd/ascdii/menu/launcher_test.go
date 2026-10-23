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
	m := &Menu{Title: "PROJECTS", Items: []Item{{Label: "Open project"}, {Label: "Quit"}}}

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
		for _, want := range []string{"A S C D I I", "> Open project", "PROJECTS"} {
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

func TestUnit_NewProjectFlowCreatesAndOpensEditor(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	var edited string
	l := &launcher{d: Deps{
		Templates: map[string]fs.FS{"empty": nil, "dragonfall": fstest.MapFS{}},
		Recent:    recent.Store{Path: filepath.Join(dir, "recent.json")},
		Edit:      func(_ context.Context, m string) error { edited = m; return nil },
	}}
	ctx := context.Background()
	l.push(home, l.homeMenu(ctx))

	// Act
	l.choose(ctx, home, Item{Value: "new"})
	l.choose(ctx, newName, Item{Value: "Space Game"})
	l.choose(ctx, newDir, Item{Value: filepath.Join(dir, "sg")})
	action, _ := l.choose(ctx, newTemplate, Item{Value: "empty"})

	// Assert
	if action != nil || filepath.Base(edited) != "Space-Game.ascdiiproj" || l.status != "editor opened" || len(l.stack) != 1 {
		t.Fatalf("action %v edited %q status %q stack %d", action != nil, edited, l.status, len(l.stack))
	}
	if m := l.stack[0]; m.Items[0].Label != "Space-Game" {
		t.Fatalf("home %+v", m.Items[0])
	}
}
