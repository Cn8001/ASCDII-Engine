package bundle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrNoData = errors.New("no game data next to the player")

// ModsDir is the folder next to a built game that players put mods in.
const ModsDir = "mods"

// AddModsDir gives a folder build the empty mods folder, so players of a game
// with mods enabled see where theirs go.
func AddModsDir(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, ModsDir), 0o755); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

// BuildFolder writes a game as dir/<exe> plus dir/data/game.pak, replacing
// what dir held, and a SHA256SUMS of both.
func BuildFolder(ctx context.Context, dir, exe, player string, fsys fs.FS, skip ...string) (string, error) {
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, DataDir), 0o755); err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	var b bytes.Buffer
	if _, err := Pack(ctx, &b, fsys, skip...); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, DataDir, PakFile), b.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	src, err := os.Open(player)
	if err != nil {
		return "", fmt.Errorf("open player: %w", err)
	}
	defer src.Close()
	size, err := playerSize(ctx, src) // a player carrying a bundle copies without it
	if err != nil {
		return "", err
	}
	out := filepath.Join(dir, exe)
	f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	if _, err := io.Copy(f, ioLimit(src, size)); err != nil {
		f.Close()
		return "", fmt.Errorf("build: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("build: %w", err)
	}
	return out, writeSums(ctx, dir)
}

// OpenData opens the game data next to the executable at exe.
func OpenData(exe string) (fs.FS, error) {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), DataDir, PakFile))
	if err != nil {
		return nil, ErrNoData
	}
	return Unpack(b)
}
