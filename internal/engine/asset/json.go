package asset

import (
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/color"
)

type swatchFile struct {
	Glyph, FG, BG string
	Transparent   bool
}

type surfaceFile struct {
	Name    string                `json:"name"`
	Pivot   [2]int                `json:"pivot"`
	Palette map[string]swatchFile `json:"palette"`
	Rows    []string              `json:"rows"`
}

// LoadJSON reads art with one palette key per cell; the key is independent
// of the glyph shown.
func LoadJSON(r io.Reader) (Surface, error) {
	d := json.NewDecoder(io.LimitReader(r, 1<<20))
	d.DisallowUnknownFields()
	var file surfaceFile
	if err := d.Decode(&file); err != nil {
		return Surface{}, fmt.Errorf("%w: %w", ErrInvalidArt, err)
	}
	if len(file.Rows) == 0 {
		return Surface{}, fmt.Errorf("%w: empty artwork", ErrInvalidArt)
	}
	width := utf8.RuneCountInString(file.Rows[0])
	s, err := NewSurface(width, len(file.Rows), file.Pivot[0], file.Pivot[1])
	if err != nil {
		return s, err
	}
	palette := map[rune]Pixel{}
	for key, sw := range file.Palette {
		if utf8.RuneCountInString(key) != 1 {
			return s, fmt.Errorf("%w: palette key must be one rune: %q", ErrInvalidArt, key)
		}
		k, _ := utf8.DecodeRuneInString(key)
		p := Pixel{Transparent: sw.Transparent}
		if !p.Transparent {
			if utf8.RuneCountInString(sw.Glyph) != 1 {
				return s, fmt.Errorf("%w: glyph must be one rune: %q", ErrInvalidArt, sw.Glyph)
			}
			p.Rune, _ = utf8.DecodeRuneInString(sw.Glyph)
			if p.FG, err = color.ParseHex(sw.FG); err != nil {
				return s, fmt.Errorf("key %s foreground: %w", key, err)
			}
			if p.BG, err = color.ParseHex(sw.BG); err != nil {
				return s, fmt.Errorf("key %s background: %w", key, err)
			}
		}
		palette[k] = p
	}
	for y, row := range file.Rows {
		if utf8.RuneCountInString(row) != width {
			return s, fmt.Errorf("%w: row %d width mismatch", ErrInvalidArt, y)
		}
		for x, k := range []rune(row) {
			p, ok := palette[k]
			if !ok {
				return s, fmt.Errorf("%w: row %d undefined palette key %q", ErrInvalidArt, y, string(k))
			}
			if err := s.Set(x, y, p); err != nil {
				return s, err
			}
		}
	}
	var tail json.RawMessage
	if err := d.Decode(&tail); err != io.EOF {
		return s, fmt.Errorf("%w: unexpected trailing JSON", ErrInvalidArt)
	}
	return s, nil
}
