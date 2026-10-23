package bundle

import (
	"archive/zip"
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

type Target struct {
	OS, Arch string
	Player   string // player binary built for this target
}

func (t Target) Ext() string {
	if t.OS == "windows" {
		return ".exe"
	}
	return ""
}

type Release struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"`
}

// MakeRelease builds one executable per target, zips each and writes
// SHA256SUMS, producing a folder ready to attach to a GitHub release.
func MakeRelease(ctx context.Context, outDir, name, version string, targets []Target, fsys fs.FS, skip ...string) (Release, error) {
	base := strings.ReplaceAll(strings.TrimSpace(name), " ", "-") + "-" + strings.TrimPrefix(version, "v")
	dir := filepath.Join(outDir, base)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Release{}, fmt.Errorf("release: %w", err)
	}
	rel := Release{Dir: dir}
	var sums strings.Builder
	for _, t := range targets {
		exe := base + "-" + t.OS + "-" + t.Arch + t.Ext()
		path := filepath.Join(dir, exe)
		if err := Build(ctx, path, t.Player, fsys, skip...); err != nil {
			return rel, fmt.Errorf("release %s/%s: %w", t.OS, t.Arch, err)
		}
		archive := path + ".zip"
		if err := zipOne(ctx, archive, path, exe); err != nil {
			return rel, err
		}
		for _, f := range []string{path, archive} {
			sum, err := sha(ctx, f)
			if err != nil {
				return rel, err
			}
			fmt.Fprintf(&sums, "%s  %s\n", sum, filepath.Base(f))
			rel.Files = append(rel.Files, f)
		}
	}
	sumFile := filepath.Join(dir, "SHA256SUMS")
	if err := os.WriteFile(sumFile, []byte(sums.String()), 0o644); err != nil {
		return rel, fmt.Errorf("release: %w", err)
	}
	rel.Files = append(rel.Files, sumFile)
	return rel, nil
}

func zipOne(ctx context.Context, dst, src, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("zip %s: %w", name, err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("zip %s: %w", name, err)
	}
	z := zip.NewWriter(out)
	h := &zip.FileHeader{Name: name, Method: zip.Deflate}
	h.SetMode(0o755)
	w, err := z.CreateHeader(h)
	if err == nil {
		_, err = io.Copy(w, in)
	}
	if err == nil {
		err = z.Close()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("zip %s: %w", name, err)
	}
	return nil
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
