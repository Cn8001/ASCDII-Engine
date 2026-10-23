package winbrand

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"testing/fstest"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

func TestUnit_ForReadsNameVersionAndIcon(t *testing.T) {
	// Arrange
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 32, 16))); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"art/icon.png": {Data: b.Bytes()}}
	lib := asset.NewLibrary(scene.NewRegistry())
	m := project.Manifest{Name: "Quest", Version: "1.2.0", Icon: "art/icon.png"}

	// Act
	got, err := For(m, fsys, lib)
	_, missing := For(project.Manifest{Icon: "art/none.png"}, fsys, lib)

	// Assert
	if err != nil || got.Name != "Quest" || got.Version != "1.2.0" || got.Icon.Bounds().Dx() != 32 || missing == nil {
		t.Fatalf("got %+v err %v missing %v", got, err, missing)
	}
}

func TestUnit_BrandLeavesNonWindowsPlayersAlone(t *testing.T) {
	// Arrange
	player := t.TempDir() + "/ascdii-player"
	if err := os.WriteFile(player, []byte("\x7fELF not a windows program"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Act
	got, err := Brand(context.Background(), player, t.TempDir(), Branding{Name: "Quest"})

	// Assert
	if err != nil || got != player {
		t.Fatalf("got %s err %v", got, err)
	}
}
