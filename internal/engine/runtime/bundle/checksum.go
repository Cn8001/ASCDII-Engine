package bundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SumsFile lists the SHA-256 of every file of a folder build, so players
// can check what they downloaded.
const SumsFile = "SHA256SUMS"

type Target struct {
	OS, Arch string
}

func (t Target) Ext() string {
	if t.OS == "windows" {
		return ".exe"
	}
	return ""
}

func sha(ctx context.Context, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("checksum: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("checksum: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeSums writes SHA256SUMS into dir for every file below it, in the
// format sha256sum -c reads.
func writeSums(ctx context.Context, dir string) error {
	var sums strings.Builder
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil || rel == SumsFile {
			return err
		}
		sum, err := sha(ctx, p)
		if err != nil {
			return err
		}
		fmt.Fprintf(&sums, "%s  %s\n", sum, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return fmt.Errorf("checksums: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, SumsFile), []byte(sums.String()), 0o644); err != nil {
		return fmt.Errorf("checksums: %w", err)
	}
	return nil
}

// writeSum writes <file>.sha256 next to a single file build.
func writeSum(ctx context.Context, file string) error {
	sum, err := sha(ctx, file)
	if err != nil {
		return err
	}
	if err := os.WriteFile(file+".sha256", []byte(sum+"  "+filepath.Base(file)+"\n"), 0o644); err != nil {
		return fmt.Errorf("checksum: %w", err)
	}
	return nil
}
