package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/guid"
)

const MetaExt = ".meta"

var ErrExists = errors.New("asset already exists")

// Meta sits next to every asset, like Unity's .meta files. The id survives
// renames and moves, so references never break; keep both files in git.
type Meta struct {
	Format int             `json:"format"`
	GUID   string          `json:"guid"`
	Import json.RawMessage `json:"import,omitempty"` // per kind import settings, such as texture size
}

// ReadMeta reads an asset's .meta, or its entry in the asset index of a
// built game, which ships no .meta files.
func ReadMeta(fsys fs.FS, asset string) (Meta, error) {
	return readMeta(fsys, asset, nil)
}

// readMeta takes an index already read, so listing a built game parses it once.
func readMeta(fsys fs.FS, asset string, idx Index) (Meta, error) {
	b, err := fs.ReadFile(fsys, asset+MetaExt)
	if errors.Is(err, fs.ErrNotExist) {
		if idx == nil {
			idx, _ = ReadIndex(fsys)
		}
		if m, ok := idx[asset]; ok {
			return m, nil
		}
	}
	if err != nil {
		return Meta{}, err
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return Meta{}, fmt.Errorf("meta %s: %w", asset, err)
	}
	if !guid.Valid(m.GUID) {
		return Meta{}, fmt.Errorf("meta %s: %w", asset, guid.ErrInvalid)
	}
	return m, nil
}

// IndexFile replaces the .meta files of a built game: every asset's id and
// import settings in one file.
const IndexFile = "assets.index"

type Index map[string]Meta

// Write adds the index to a pak being written.
func (idx Index) Write(z interface {
	Create(name string) (io.Writer, error)
}) error {
	w, err := z.Create(IndexFile)
	if err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	if err := json.NewEncoder(w).Encode(idx); err != nil {
		return fmt.Errorf("write index: %w", err)
	}
	return nil
}

func ReadIndex(fsys fs.FS) (Index, error) {
	b, err := fs.ReadFile(fsys, IndexFile)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	return idx, nil
}

// MetaBytes is the file content of a .meta for id.
func MetaBytes(id string) []byte { return encodeMeta(id) }

func encodeMeta(id string) []byte {
	b, _ := json.MarshalIndent(Meta{Format: 1, GUID: id}, "", "  ")
	return append(b, '\n')
}

// EnsureMeta writes a .meta for every asset in dir that has none and returns
// the asset paths that got one.
func EnsureMeta(ctx context.Context, dir string, src guid.Source, skip ...string) ([]string, error) {
	fsys := os.DirFS(dir)
	entries, err := List(ctx, fsys, skip...)
	if err != nil {
		return nil, err
	}
	var made []string
	for _, e := range entries {
		if e.GUID != "" {
			continue
		}
		id, err := src.New()
		if err != nil {
			return made, err
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(e.Path))+MetaExt, encodeMeta(id), 0o644); err != nil {
			return made, fmt.Errorf("write meta: %w", err)
		}
		made = append(made, e.Path)
	}
	return made, nil
}

// WriteMeta records a known id for a new asset, used when creating files.
func WriteMeta(ctx context.Context, dir, asset, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(asset))+MetaExt, encodeMeta(id), 0o644); err != nil {
		return fmt.Errorf("write meta: %w", err)
	}
	return nil
}

// WriteImport stores import settings in an asset's existing .meta.
func WriteImport(ctx context.Context, dir, asset string, settings any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m, err := ReadMeta(os.DirFS(dir), asset)
	if err != nil {
		return err
	}
	if m.Import, err = json.Marshal(settings); err != nil {
		return fmt.Errorf("write import: %w", err)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(asset))+MetaExt, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write import: %w", err)
	}
	return nil
}

// Move renames an asset and its .meta together inside dir.
func Move(ctx context.Context, dir, from, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src := filepath.Join(dir, filepath.FromSlash(from))
	dst := filepath.Join(dir, filepath.FromSlash(to))
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("move %s: %w", to, ErrExists)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("move: %w", err)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("move: %w", err)
	}
	if err := os.Rename(src+MetaExt, dst+MetaExt); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("move meta: %w", err)
	}
	return nil
}
