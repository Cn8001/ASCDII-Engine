// Package asset loads glyph art and indexes a project's files: art, prefabs
// and scenes, classified by suffix.
package asset

import (
	"errors"
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
)

var (
	ErrInvalidArt = errors.New("invalid art")
	ErrNotFound   = errors.New("asset not found")
)

// Pixel is one terminal cell of art; a space that is not transparent is an
// opaque background block.
type Pixel struct {
	Rune        rune
	FG, BG      color.Color
	Transparent bool
}

type Surface struct {
	Width, Height  int
	PivotX, PivotY int
	pixels         []Pixel
}

func NewSurface(w, h, px, py int) (Surface, error) {
	if w <= 0 || h <= 0 || w > 256 || h > 256 || px < 0 || px >= w || py < 0 || py >= h {
		return Surface{}, fmt.Errorf("%w: size %dx%d pivot %d,%d", ErrInvalidArt, w, h, px, py)
	}
	s := Surface{Width: w, Height: h, PivotX: px, PivotY: py, pixels: make([]Pixel, w*h)}
	for i := range s.pixels {
		s.pixels[i].Transparent = true
	}
	return s, nil
}

func (s *Surface) Set(x, y int, p Pixel) error {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return fmt.Errorf("%w: pixel %d,%d outside", ErrInvalidArt, x, y)
	}
	if !p.Transparent && p.Rune == 0 {
		p.Rune = ' '
	}
	s.pixels[y*s.Width+x] = p
	return nil
}

func (s Surface) At(x, y int) (Pixel, bool) {
	if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
		return Pixel{}, false
	}
	return s.pixels[y*s.Width+x], true
}

func (s Surface) Clone() Surface {
	out := s
	out.pixels = append([]Pixel(nil), s.pixels...)
	return out
}

func Flatten(layers []Surface) (Surface, error) {
	if len(layers) == 0 {
		return Surface{}, fmt.Errorf("%w: no layers", ErrInvalidArt)
	}
	out := layers[0].Clone()
	for _, s := range layers[1:] {
		if s.Width != out.Width || s.Height != out.Height {
			return out, fmt.Errorf("%w: mismatched layer dimensions", ErrInvalidArt)
		}
		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				if p, _ := s.At(x, y); !p.Transparent {
					_ = out.Set(x, y, p)
				}
			}
		}
	}
	return out, nil
}
