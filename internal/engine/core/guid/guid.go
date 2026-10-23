// Package guid makes the stable asset ids stored in .meta files: RFC 9562
// UUIDv7, so ids sort by creation time and stay readable in diffs.
package guid

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid uuid")

type Source struct {
	Now  func() time.Time
	Rand io.Reader
}

func Default() Source { return Source{Now: time.Now, Rand: rand.Reader} }

// New returns a UUIDv7: 48 bits of Unix milliseconds, then random bits.
func (s Source) New() (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(s.Rand, b[6:]); err != nil {
		return "", fmt.Errorf("uuid: %w", err)
	}
	return format(stamp(b, s.Now())), nil
}

// Stable derives a UUIDv7 shaped id from a name, for shipped templates whose
// ids must not change between builds.
func Stable(at time.Time, name string) string {
	sum := sha256.Sum256([]byte(name))
	var b [16]byte
	copy(b[6:], sum[:10])
	return format(stamp(b, at))
}

func stamp(b [16]byte, at time.Time) [16]byte {
	ms := uint64(at.UnixMilli())
	var t [8]byte
	binary.BigEndian.PutUint64(t[:], ms)
	copy(b[0:6], t[2:8])
	b[6] = 0x70 | (b[6] & 0x0f) // version 7
	b[8] = 0x80 | (b[8] & 0x3f) // RFC 9562 variant
	return b
}

func format(b [16]byte) string {
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// Valid reports whether s is a version 7 UUID in canonical form.
func Valid(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	raw, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	return err == nil && raw[6]>>4 == 7 && raw[8]>>6 == 2
}

// Time returns the creation time encoded in a UUIDv7.
func Time(s string) (time.Time, error) {
	if !Valid(s) {
		return time.Time{}, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	raw, _ := hex.DecodeString(strings.ReplaceAll(s, "-", "")[:12])
	var t [8]byte
	copy(t[2:], raw)
	return time.UnixMilli(int64(binary.BigEndian.Uint64(t[:]))), nil
}
