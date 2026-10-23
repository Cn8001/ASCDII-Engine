// Package bundle turns a project into a game: the player executable with
// the game data packed beside it, or appended to it for a single file.
package bundle

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

const magic = "ASCDIIPK"

var ErrNoBundle = errors.New("no bundled project")

// Write emits player followed by the packed game data and the footer.
func Write(ctx context.Context, w io.Writer, player io.Reader, fsys fs.FS, skip ...string) error {
	if _, err := io.Copy(w, player); err != nil {
		return fmt.Errorf("copy player: %w", err)
	}
	var z bytes.Buffer
	if _, err := Pack(ctx, &z, fsys, skip...); err != nil {
		return err
	}
	if _, err := w.Write(z.Bytes()); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}
	var foot [16]byte
	binary.LittleEndian.PutUint64(foot[:8], uint64(z.Len()))
	copy(foot[8:], magic)
	if _, err := w.Write(foot[:]); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}
	return nil
}

// Read returns the project bundled inside r, or ErrNoBundle.
func Read(ctx context.Context, r io.ReaderAt, size int64) (fs.FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if size < 16 {
		return nil, ErrNoBundle
	}
	var foot [16]byte
	if _, err := r.ReadAt(foot[:], size-16); err != nil {
		return nil, fmt.Errorf("read footer: %w", err)
	}
	if string(foot[8:]) != magic {
		return nil, ErrNoBundle
	}
	n := int64(binary.LittleEndian.Uint64(foot[:8]))
	if n <= 0 || n > size-16 {
		return nil, fmt.Errorf("%w: corrupt footer", ErrNoBundle)
	}
	b := make([]byte, n)
	if _, err := r.ReadAt(b, size-16-n); err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}
	return Unpack(b)
}

// OpenFile reads the bundle of an executable; the caller closes the file.
func OpenFile(ctx context.Context, path string) (fs.FS, io.Closer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("stat %s: %w", path, err)
	}
	fsys, err := Read(ctx, f, info.Size())
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return fsys, f, nil
}
