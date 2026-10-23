package bundle

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func ioLimit(f *os.File, n int64) io.Reader { return io.NewSectionReader(f, 0, n) }

// PlayerSize is the length of the executable without any appended bundle.
func PlayerSize(ctx context.Context, f *os.File) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat player: %w", err)
	}
	size := info.Size()
	if size < 16 {
		return size, nil
	}
	var foot [16]byte
	if _, err := f.ReadAt(foot[:], size-16); err != nil {
		return 0, fmt.Errorf("read player: %w", err)
	}
	if string(foot[8:]) != magic {
		return size, nil
	}
	n := int64(binary.LittleEndian.Uint64(foot[:8]))
	if n <= 0 || n > size-16 {
		return size, nil
	}
	return size - 16 - n, nil
}
