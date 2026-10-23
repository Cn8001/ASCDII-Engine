// Package players finds the binaries a build starts from. ascdii is itself
// the player, so the ascdii next to the editor serves its own platform and an
// ascdii-<os>-<arch> beside it serves another one.
package players

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/editor/launch"
)

var ErrNoPlayer = errors.New("no player for this platform; place its ascdii next to the editor")

func Name(goos, arch string) string {
	n := "ascdii-" + goos + "-" + arch
	if goos == "windows" {
		n += ".exe"
	}
	return n
}

// Source is the folders players are looked up in, first match wins.
type Source struct {
	Dirs []fs.FS
}

// Dir is an extra folder to look in, for editors not started from a release
// folder such as wails dev.
const Dir = "ASCDII_PLAYERS"

// Default looks next to the editor, next to the ascdii that started it, and
// in the folder named by ASCDII_PLAYERS.
func Default() Source {
	var dirs []fs.FS
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, os.DirFS(filepath.Dir(exe)))
	}
	if cli := os.Getenv(launch.Env); filepath.IsAbs(cli) {
		dirs = append(dirs, os.DirFS(filepath.Dir(cli)))
	}
	if dir := os.Getenv(Dir); dir != "" {
		dirs = append(dirs, os.DirFS(dir))
	}
	return Source{Dirs: dirs}
}

// names are the files that can be the player for a target.
func names(goos, arch string) []string {
	out := []string{Name(goos, arch)}
	if goos == runtime.GOOS && arch == runtime.GOARCH {
		self := launch.CLI
		if goos == "windows" {
			self += ".exe"
		}
		out = append(out, self)
	}
	return out
}

func (s Source) read(goos, arch string) ([]byte, error) {
	for _, d := range s.Dirs {
		for _, n := range names(goos, arch) {
			if b, err := fs.ReadFile(d, n); err == nil {
				return b, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoPlayer, Name(goos, arch))
}

// Extract writes the player for a target into dir and returns its path.
func (s Source) Extract(ctx context.Context, goos, arch, dir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	b, err := s.read(goos, arch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("extract player: %w", err)
	}
	path := filepath.Join(dir, Name(goos, arch))
	if err := os.WriteFile(path, b, 0o755); err != nil {
		return "", fmt.Errorf("extract player: %w", err)
	}
	return path, nil
}

func (s Source) Available() []string {
	var out []string
	for _, t := range [][2]string{{"linux", "amd64"}, {"windows", "amd64"}} {
		for _, d := range s.Dirs {
			found := false
			for _, n := range names(t[0], t[1]) {
				if _, err := fs.Stat(d, n); err == nil {
					found = true
					break
				}
			}
			if found {
				out = append(out, t[0])
				break
			}
		}
	}
	return out
}
