package bundle

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Build writes an executable game to dst from a player binary and a project.
func Build(ctx context.Context, dst, player string, fsys fs.FS, skip ...string) error {
	src, err := os.Open(player)
	if err != nil {
		return fmt.Errorf("open player: %w", err)
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}
	// A player that already carries a bundle must not nest a second one.
	size, err := playerSize(ctx, src)
	if err != nil {
		out.Close()
		return err
	}
	if err := Write(ctx, out, ioLimit(src, size), fsys, skip...); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return writeSum(ctx, dst)
}
