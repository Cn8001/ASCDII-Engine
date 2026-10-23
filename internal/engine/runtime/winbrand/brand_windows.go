//go:build windows

package winbrand

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/runtime/bundle"
)

const Supported = true

// Brand writes a copy of player with b applied into dir and returns its
// path. Players that are not Windows executables are returned unchanged:
// Linux executables carry no icon.
func Brand(ctx context.Context, player, dir string, b Branding) (string, error) {
	f, err := os.Open(player)
	if err != nil {
		return "", fmt.Errorf("open player: %w", err)
	}
	defer f.Close()
	size, err := bundle.PlayerSize(ctx, f) // a player carrying a game brands without it
	if err != nil {
		return "", err
	}
	src := make([]byte, size)
	if _, err := io.ReadFull(io.NewSectionReader(f, 0, size), src); err != nil {
		return "", fmt.Errorf("read player: %w", err)
	}
	if !bytes.HasPrefix(src, []byte("MZ")) {
		return player, nil
	}
	rs, err := winres.LoadFromEXE(bytes.NewReader(src))
	if err != nil {
		return "", fmt.Errorf("brand: %w", err)
	}
	if b.Icon != nil {
		sizes := []int{256, 64, 48, 32, 24, 16} // Windows picks from these for lists, tiles and the taskbar
		ic, err := winres.NewIconFromResizedImage(b.Icon, sizes)
		if err != nil {
			return "", fmt.Errorf("brand icon: %w", err)
		}
		if err := rs.SetIcon(winres.Name("APPICON"), ic); err != nil {
			return "", fmt.Errorf("brand icon: %w", err)
		}
	}
	var vi version.Info
	vi.SetFileVersion(b.Version)
	vi.SetProductVersion(b.Version)
	for k, v := range map[string]string{version.ProductName: b.Name, version.FileDescription: b.Name} {
		if err := vi.Set(version.LangDefault, k, v); err != nil {
			return "", fmt.Errorf("brand: %w", err)
		}
	}
	rs.SetVersionInfo(vi)
	var out bytes.Buffer
	if err := rs.WriteToEXE(&out, bytes.NewReader(src)); err != nil {
		return "", fmt.Errorf("brand: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("brand: %w", err)
	}
	path := filepath.Join(dir, filepath.Base(player))
	if err := os.WriteFile(path, out.Bytes(), 0o755); err != nil {
		return "", fmt.Errorf("brand: %w", err)
	}
	return path, nil
}
