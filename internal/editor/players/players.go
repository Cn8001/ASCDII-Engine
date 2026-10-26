// Package players finds the binaries a build starts from: the ascdii-player
// next to the editor serves its own platform, and an
// ascdii-player-<os>-<arch> beside it serves another one. Games shown in a
// window start from ascdii-player-window, found the same way.
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
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/project"
)

var ErrNoPlayer = errors.New("no player for this platform; place its ascdii-player next to the editor")

// For is the player a game starts from: the window player for a game shown
// in a window, which only it links the window library for.
func For(d project.Display) string {
	if d.Window() {
		return launch.Window
	}
	return launch.Player
}

// Name is the file name of another platform's player.
func Name(player, goos, arch string) string {
	n := player + "-" + goos + "-" + arch
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

// Default looks next to the editor, next to the program that started it, and
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

// names are the files that can be the player for a target. <os>/ is the
// layout of bin/ in the source tree, which ASCDII_PLAYERS points wails dev at.
// The plain name serves the editor's own platform, and the other one too when
// only the .exe tells the two apart, so a copied ascdii-player.exe works
// next to a Linux editor without renaming.
func names(player, goos, arch string) []string {
	own := player
	if goos == "windows" {
		own += ".exe"
	}
	out := []string{Name(player, goos, arch), goos + "/" + own}
	native := goos == runtime.GOOS && arch == runtime.GOARCH
	if native || (goos == "windows") != (runtime.GOOS == "windows") {
		out = append(out, own)
	}
	return out
}

func (s Source) read(player, goos, arch string) ([]byte, error) {
	for _, d := range s.Dirs {
		for _, n := range names(player, goos, arch) {
			if b, err := fs.ReadFile(d, n); err == nil {
				return b, nil
			}
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoPlayer, Name(player, goos, arch))
}

// Extract writes player, launch.Player or launch.Window, for a target into
// dir and returns its path.
func (s Source) Extract(ctx context.Context, player, goos, arch, dir string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	b, err := s.read(player, goos, arch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("extract player: %w", err)
	}
	path := filepath.Join(dir, Name(player, goos, arch))
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
			for _, n := range names(launch.Player, t[0], t[1]) {
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
