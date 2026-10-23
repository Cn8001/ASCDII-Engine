package asset

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"image"
	stdcolor "image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const tree = `{"name":"t","pivot":[1,0],"palette":{".":{"transparent":true},"x":{"glyph":"X","fg":"#00ff22","bg":"#000000"},"b":{"glyph":" ","fg":"#aaaaaa","bg":"#884422"}},"rows":[".xb"]}`

func TestUnit_JSONPaletteKeepsOpaqueSpace(t *testing.T) {
	// Arrange
	r := strings.NewReader(tree)

	// Act
	s, err := LoadJSON(r)

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := s.At(1, 0); p.Rune != 'X' || p.FG.G != 255 {
		t.Fatalf("wrong pixel %#v", p)
	}
	if p, _ := s.At(2, 0); p.Transparent || p.Rune != ' ' || p.BG.R != 136 {
		t.Fatalf("opaque blank lost %#v", p)
	}
}

func TestUnit_JSONRejectsUndefinedKey(t *testing.T) {
	// Arrange
	r := strings.NewReader(`{"pivot":[0,0],"palette":{},"rows":["no"]}`)

	// Act
	_, err := LoadJSON(r)

	// Assert
	if !errors.Is(err, ErrInvalidSprite) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_XPIsColumnMajorWithMagentaKey(t *testing.T) {
	// Arrange
	var raw bytes.Buffer
	for _, v := range []int32{-1, 1, 2, 2} {
		_ = binary.Write(&raw, binary.LittleEndian, v)
	}
	for _, c := range []struct {
		glyph uint32
		bg    [3]byte
	}{{'A', [3]byte{}}, {'B', [3]byte{}}, {'C', [3]byte{255, 0, 255}}, {'D', [3]byte{}}} {
		_ = binary.Write(&raw, binary.LittleEndian, c.glyph)
		raw.Write([]byte{12, 34, 56})
		raw.Write(c.bg[:])
	}
	var zipped bytes.Buffer
	z := gzip.NewWriter(&zipped)
	_, _ = z.Write(raw.Bytes())
	_ = z.Close()

	// Act
	layers, err := LoadXP(&zipped, CP437())

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	at := func(x, y int) rune {
		p, _ := layers[0].At(x, y)
		if p.Transparent {
			return 0
		}
		return p.Rune
	}
	if at(0, 0) != 'A' || at(0, 1) != 'B' || at(1, 0) != 0 || at(1, 1) != 'D' {
		t.Fatalf("bad order")
	}
}

func TestUnit_ListClassifiesAndSkips(t *testing.T) {
	// Arrange
	fsys := fstest.MapFS{
		"sprites/tree.sprite.json":  {Data: []byte(tree)},
		"prefabs/tower.prefab.json": {Data: []byte(`{}`)},
		"scenes/main.scene.json":    {Data: []byte(`{}`)},
		"build/game":                {Data: []byte("bin")},
		"build/x.json":              {Data: []byte("{}")},
		"notes.txt":                 {Data: []byte("x")},
	}

	// Act
	got, err := List(context.Background(), fsys, "build")

	// Assert
	if err != nil || len(got) != 3 {
		t.Fatalf("err %v got %+v", err, got)
	}
	if got[0].Kind != Prefab || got[1].Kind != Scene || got[2].Kind != Sprite {
		t.Fatalf("kinds %+v", got)
	}
}

func TestUnit_LibraryServesArtAndPrefabs(t *testing.T) {
	// Arrange
	fsys := fstest.MapFS{
		"sprites/tree.sprite.json":  {Data: []byte(tree)},
		"prefabs/empty.prefab.json": {Data: []byte(`{"version":1,"root":{"name":"e","pos":[0,0,0]}}`)},
	}
	l := NewLibrary(scene.NewRegistry())

	// Act
	err := l.Load(context.Background(), fsys)
	art, artErr := l.Sprite("sprites/tree.sprite.json")
	pf, pfErr := l.Prefab("prefabs/empty.prefab.json")
	_, missing := l.Sprite("nope.json")

	// Assert
	if err != nil || artErr != nil || pfErr != nil || art.Width != 3 || pf.Name != "e" {
		t.Fatalf("err %v %v %v", err, artErr, pfErr)
	}
	if !errors.Is(missing, ErrNotFound) {
		t.Fatalf("got %v", missing)
	}
}

func TestUnit_ListHonoursCancelledContext(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	_, err := List(ctx, fstest.MapFS{"a.json": {Data: []byte("{}")}})

	// Assert
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestUnit_PutScriptVersionsOnlyOnChange(t *testing.T) {
	// Arrange
	l := NewLibrary(scene.NewRegistry())

	// Act
	l.PutScript("scripts/a.lua", "x = 1")
	l.PutScript("scripts/a.lua", "x = 1")
	_, v1, _ := l.Script("scripts/a.lua")
	l.PutScript("scripts/a.lua", "x = 2")
	text, v2, err := l.Script("scripts/a.lua")

	// Assert
	if err != nil || v1 != 1 || v2 != 2 || text != "x = 2" {
		t.Fatalf("v1 %d v2 %d text %q err %v", v1, v2, text, err)
	}
}

func TestUnit_WatchReportsChangedFiles(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "a.lua")
	if err := os.WriteFile(path, []byte("x = 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got := make(chan []string, 1)

	// Act
	go func() {
		_ = Watch(ctx, os.DirFS(dir), 20*time.Millisecond, func(c []string) {
			select {
			case got <- c:
			default:
			}
		})
	}()
	time.Sleep(60 * time.Millisecond)
	if err := os.WriteFile(path, []byte("x = 22"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Assert
	select {
	case c := <-got:
		if len(c) != 1 || c[0] != "a.lua" {
			t.Fatalf("changed %v", c)
		}
	case <-ctx.Done():
		t.Fatal("no change reported")
	}
}

func TestUnit_EnsureMetaThenResolveByGUID(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "sprites"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "sprites", "tree.sprite.json"), []byte(tree), 0o644)
	ctx := context.Background()

	// Act
	made, err := EnsureMeta(ctx, dir, guid.Default())
	again, _ := EnsureMeta(ctx, dir, guid.Default())
	l := NewLibrary(scene.NewRegistry())
	errLoad := l.Load(ctx, os.DirFS(dir))
	id := l.Ref("sprites/tree.sprite.json")
	_, errArt := l.Sprite(id)

	// Assert
	if err != nil || errLoad != nil || errArt != nil || len(made) != 1 || len(again) != 0 || !guid.Valid(id) {
		t.Fatalf("made %v again %v id %s err %v %v %v", made, again, id, err, errLoad, errArt)
	}
}

func TestUnit_MoveKeepsGUID(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.lua"), []byte("x = 1"), 0o644)
	ctx := context.Background()
	_, _ = EnsureMeta(ctx, dir, guid.Default())
	before, _ := ReadMeta(os.DirFS(dir), "a.lua")

	// Act
	err := Move(ctx, dir, "a.lua", "scripts/b.lua")
	after, errMeta := ReadMeta(os.DirFS(dir), "scripts/b.lua")
	l := NewLibrary(scene.NewRegistry())
	_ = l.Load(ctx, os.DirFS(dir))
	text, _, errScript := l.Script(before.GUID)

	// Assert
	if err != nil || errMeta != nil || errScript != nil || after.GUID != before.GUID || text != "x = 1" {
		t.Fatalf("err %v %v %v before %s after %s", err, errMeta, errScript, before.GUID, after.GUID)
	}
}

func TestUnit_ImageBecomesGlyphArtAndLoadsBack(t *testing.T) {
	// Arrange
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x >= 4 {
				img.Set(x, y, stdcolor.NRGBA{R: 255, A: 255})
			}
		}
	}

	// Act
	s, err := SpriteFromImage(img, 8)
	var b bytes.Buffer
	errEnc := EncodeJSON(&b, "half", s)
	back, errLoad := LoadJSON(&b)

	// Assert
	if err != nil || errEnc != nil || errLoad != nil {
		t.Fatalf("errs %v %v %v", err, errEnc, errLoad)
	}
	left, _ := back.At(0, 0)
	right, _ := back.At(7, 0)
	if back.Width != 8 || back.Height != 4 || !left.Transparent || right.FG.R < 240 || right.Rune != '▀' {
		t.Fatalf("size %dx%d left %+v right %+v", back.Width, back.Height, left, right)
	}
}

func TestUnit_LoadUnderPrefixesModAssetsAndKeepsGameIDs(t *testing.T) {
	// Arrange
	game := fstest.MapFS{
		"a.sprite.json":      {Data: []byte(tree)},
		"a.sprite.json.meta": {Data: MetaBytes("01a160e8-5000-7000-8000-000000000001")},
	}
	mod := fstest.MapFS{
		"coin.sprite.json":      {Data: []byte(tree)},
		"coin.sprite.json.meta": {Data: MetaBytes("01a160e8-5000-7000-8000-000000000001")},
		"gem.sprite.json":       {Data: []byte(tree)},
		"gem.sprite.json.meta":  {Data: MetaBytes("01a160e8-5000-7000-8000-000000000002")},
	}
	l := NewLibrary(scene.NewRegistry())
	ctx := context.Background()

	// Act
	errGame := l.Load(ctx, game)
	errMod := l.LoadUnder(ctx, mod, "mods/shiny")
	_, errCoin := l.Sprite("mods/shiny/coin.sprite.json")

	// Assert
	if errGame != nil || errMod != nil || errCoin != nil {
		t.Fatal(errGame, errMod, errCoin)
	}
	if l.Resolve("01a160e8-5000-7000-8000-000000000001") != "a.sprite.json" || l.Resolve("01a160e8-5000-7000-8000-000000000002") != "mods/shiny/gem.sprite.json" {
		t.Fatalf("ids %s %s", l.Resolve("01a160e8-5000-7000-8000-000000000001"), l.Resolve("01a160e8-5000-7000-8000-000000000002"))
	}
}
