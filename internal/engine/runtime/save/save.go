// Package save keeps game save data as files, one per slot, in a folder of
// its own. It implements scene.Store; scripts reach it through save() and
// load() in Lua, never through paths.
package save

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrBadSlot is returned for slot names that could leave the folder.
var ErrBadSlot = errors.New("bad save slot name")

var slotName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)

// Dir stores each slot as <Path>/<slot>.json.
type Dir struct {
	Path string
}

// ForGame is the save folder of a game in the user's config folder, so
// saves survive reinstalls and are kept apart per game.
func ForGame(name string) (Dir, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Dir{}, fmt.Errorf("save folder: %w", err)
	}
	return Dir{Path: filepath.Join(base, "ascdii", "saves", folderName(name))}, nil
}

func folderName(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if clean == "" {
		return "game"
	}
	return clean
}

func (d Dir) file(slot string) (string, error) {
	if !slotName.MatchString(slot) {
		return "", fmt.Errorf("%w: %q", ErrBadSlot, slot)
	}
	return filepath.Join(d.Path, slot+".json"), nil
}

// Load returns the slot's data; an empty slot gives an fs.ErrNotExist error.
func (d Dir) Load(ctx context.Context, slot string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := d.file(slot)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", slot, err)
	}
	return data, nil
}

// Save writes the slot through a temporary file and a rename, so a crash
// while saving leaves the previous save intact.
func (d Dir) Save(ctx context.Context, slot string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := d.file(slot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.Path, 0o755); err != nil {
		return fmt.Errorf("save %s: %w", slot, err)
	}
	tmp, err := os.CreateTemp(d.Path, slot+".*.tmp")
	if err != nil {
		return fmt.Errorf("save %s: %w", slot, err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("save %s: %w", slot, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("save %s: %w", slot, err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("save %s: %w", slot, err)
	}
	return nil
}
