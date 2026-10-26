// Package appdir is the one folder the editor keeps its own files in, inside
// the user's config folder: %APPDATA%\ASCDIIEngine on Windows,
// ~/.config/ASCDIIEngine on Linux.
package appdir

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const Name = "ASCDIIEngine"

// What the editor keeps in the folder.
const (
	Recent   = "recent.json" // recently opened projects
	Packages = "packages"    // .ascdiipack files the user collects
	Webview  = "webview"     // the editor window's WebView2 data on Windows
)

// Default opens the folder in the user's config folder.
func Default(ctx context.Context) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return Open(ctx, base)
}

// Open returns base/ASCDIIEngine, creating it.
func Open(ctx context.Context, base string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dir := filepath.Join(base, Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("app folder: %w", err)
	}
	return dir, nil
}
