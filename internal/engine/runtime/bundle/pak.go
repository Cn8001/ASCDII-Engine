package bundle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/assets/asset"
)

// DataDir sits next to a folder build's executable and holds its paks.
const DataDir = "data"

const pakMagic = "ASCDIIPAK1"

// pakSecret is mixed with a random salt per pak. The key ships with the
// player, so this keeps a game's files from being browsed or opened as a
// project; it does not stop someone who reverse engineers the player.
const pakSecret = "ascdii engine game data 0.0.1"

var ErrBadPak = errors.New("not a game data pak")

// PakFile is the game data of a folder build, in its data folder.
const PakFile = "game.pak"

// gameFile says whether a game needs p at run time: the manifest, assets the
// engine reads and model materials. Notes, readmes, dotfiles and .meta files
// stay out; the .meta contents go into the asset index instead.
func gameFile(p string) bool {
	l := strings.ToLower(p)
	if strings.HasSuffix(l, ".ascdiiproj") || strings.HasSuffix(l, ".mtl") {
		return true
	}
	if strings.HasSuffix(l, asset.MetaExt) {
		return false
	}
	_, ok := asset.Classify(p)
	return ok
}

func pakStream(salt []byte) (cipher.Stream, error) {
	key := sha256.Sum256(append([]byte(pakSecret), salt...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewCTR(block, salt), nil
}

// Pack writes the game files of fsys and their asset index as one
// compressed, encrypted pak. It reports how many files it packed.
func Pack(ctx context.Context, w io.Writer, fsys fs.FS, skip ...string) (int, error) {
	n := 0
	index := asset.Index{}
	var plain bytes.Buffer
	z := zip.NewWriter(&plain)
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			if p != "." && (slices.Contains(skip, p) || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !gameFile(p) {
			return nil
		}
		n++
		if m, err := asset.ReadMeta(fsys, p); err == nil {
			index[p] = m
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		f, err := z.Create(p)
		if err != nil {
			return err
		}
		_, err = f.Write(b)
		return err
	})
	if err == nil {
		err = index.Write(z)
	}
	if err == nil {
		err = z.Close()
	}
	if err != nil {
		return n, fmt.Errorf("pack game data: %w", err)
	}
	salt := make([]byte, aes.BlockSize)
	if _, err := rand.Read(salt); err != nil {
		return n, fmt.Errorf("pack game data: %w", err)
	}
	s, err := pakStream(salt)
	if err != nil {
		return n, fmt.Errorf("pack game data: %w", err)
	}
	sealed := plain.Bytes()
	s.XORKeyStream(sealed, sealed)
	for _, part := range [][]byte{[]byte(pakMagic), salt, sealed} {
		if _, err := w.Write(part); err != nil {
			return n, fmt.Errorf("pack game data: %w", err)
		}
	}
	return n, nil
}

// Unpack opens a pak written by Pack as a read only file system.
func Unpack(b []byte) (fs.FS, error) {
	head := len(pakMagic) + aes.BlockSize
	if len(b) < head || string(b[:len(pakMagic)]) != pakMagic {
		return nil, ErrBadPak
	}
	s, err := pakStream(b[len(pakMagic):head])
	if err != nil {
		return nil, fmt.Errorf("open game data: %w", err)
	}
	plain := make([]byte, len(b)-head)
	s.XORKeyStream(plain, b[head:])
	z, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		return nil, fmt.Errorf("open game data: %w: %w", ErrBadPak, err)
	}
	return z, nil
}
