package term

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/cell"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/input"
)

func TestUnit_PresentWritesOnlyChanges(t *testing.T) {
	// Arrange
	var out bytes.Buffer
	p := NewPresenter(&out)
	g := cell.New(4, 2)
	ctx := context.Background()
	_ = p.Present(ctx, g)
	out.Reset()

	// Act
	g.Set(1, 0, cell.Cell{Rune: 'x', FG: color.RGB(1, 2, 3)})
	err := p.Present(ctx, g)

	// Assert
	s := out.String()
	if err != nil || !strings.Contains(s, "\x1b[1;2H") || strings.Count(s, "x") != 1 || strings.Contains(s, "\x1b[2J") {
		t.Fatalf("err %v out %q", err, s)
	}
	out.Reset()
	_ = p.Present(ctx, g)
	if out.Len() != 0 {
		t.Fatalf("unchanged frame wrote %q", out.String())
	}
}

func TestUnit_PresentRepaintsAfterResize(t *testing.T) {
	// Arrange
	var out bytes.Buffer
	p := NewPresenter(&out)
	ctx := context.Background()
	_ = p.Present(ctx, cell.New(2, 2))
	out.Reset()

	// Act
	err := p.Present(ctx, cell.New(3, 2))

	// Assert
	if err != nil || !strings.HasPrefix(out.String(), "\x1b[2J") {
		t.Fatalf("got %q", out.String())
	}
}

func TestUnit_PreviewPlainHasNoEscapes(t *testing.T) {
	// Arrange
	g := cell.New(3, 2)
	g.Text(0, 0, "abc", color.Color{}, color.Color{})
	var out bytes.Buffer

	// Act
	err := WritePreview(context.Background(), &out, g, true)

	// Assert
	if err != nil || out.String() != "abc\n   \n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestUnit_KeysDecodesStream(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Act
	ch := Keys(ctx, strings.NewReader("a\x1b[B"))
	var got []input.Key
	for k := range ch {
		got = append(got, k)
	}

	// Assert
	if len(got) != 2 || got[0] != 'a' || got[1] != input.Down {
		t.Fatalf("got %v", got)
	}
}
