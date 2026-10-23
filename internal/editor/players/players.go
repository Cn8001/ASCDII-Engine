// Package players ships the per platform ascdii binaries inside the editor so
// a game can be built without a Go toolchain. "make players" fills bin/.
package players

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:bin
var embedded embed.FS // read-only build artefact

var ErrNoPlayer = errors.New("no player binary for target, run make players")

func Name(goos, arch string) string {
	n := "ascdii-" + goos + "-" + arch
	if goos == "windows" {
		n += ".exe"
	}
	return n
}

// Source finds players embedded in the editor or in a players folder next to it.
type Source struct {
	FS       fs.FS
	Fallback string
}

func Default() Source {
	sub, _ := fs.Sub(embedded, "bin")
	dir := ""
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Join(filepath.Dir(exe), "players")
	}
	return Source{FS: sub, Fallback: dir}
}

// Extract writes the player for a target into dir and returns its path.
func (s Source) Extract(ctx context.Context, goos, arch, dir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	name := Name(goos, arch)
	b, err := fs.ReadFile(s.FS, name)
	if err != nil && s.Fallback != "" {
		b, err = os.ReadFile(filepath.Join(s.Fallback, name))
	}
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrNoPlayer, name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("extract player: %w", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0o755); err != nil {
		return "", fmt.Errorf("extract player: %w", err)
	}
	return path, nil
}

func (s Source) Available() []string {
	var out []string
	for _, t := range [][2]string{{"linux", "amd64"}, {"windows", "amd64"}} {
		n := Name(t[0], t[1])
		if _, err := fs.Stat(s.FS, n); err == nil {
			out = append(out, t[0])
		} else if _, err := os.Stat(filepath.Join(s.Fallback, n)); s.Fallback != "" && err == nil {
			out = append(out, t[0])
		}
	}
	return out
}
