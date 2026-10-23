package app

import (
	"bytes"
	"context"
	"os"
	"testing"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
)

func TestUnit_OpenStarterAndPreview(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	ctx := context.Background()
	if _, err := project.Create(ctx, dir, "demo", nil); err != nil {
		t.Fatal(err)
	}
	e, err := Open(ctx, os.DirFS(dir), Options{Cols: 60, Rows: 20})
	if err != nil {
		t.Fatal(err)
	}
	e.Scene().Add("crate", components.NewBox(geom.V3(8, 8, 8), color.RGB(176, 128, 80)))
	var out bytes.Buffer

	// Act
	err = e.Preview(ctx, &out, .5, true)

	// Assert
	braille := 0
	for _, r := range out.String() {
		if r >= 0x2800 && r <= 0x28ff {
			braille++
		}
	}
	if err != nil || braille < 20 {
		t.Fatalf("err %v braille %d\n%s", err, braille, out.String())
	}
}

func TestUnit_UpdateQuitsOnCtrlC(t *testing.T) {
	// Arrange
	e := New(Options{})

	// Act
	quit, err := e.Update(.1, []input.Key{input.CtrlC})

	// Assert
	if err != nil || !quit {
		t.Fatalf("quit %v err %v", quit, err)
	}
}
