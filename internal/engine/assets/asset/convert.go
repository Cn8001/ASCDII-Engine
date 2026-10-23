package asset

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
)

// SpriteFromImage turns a picture into glyph art cols cells wide. Each cell
// shows two pixels stacked with a half block, which matches the roughly
// 1:2 shape of a terminal cell, so the picture keeps its proportions.
func SpriteFromImage(img image.Image, cols int) (Surface, error) {
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return Surface{}, fmt.Errorf("%w: empty image", ErrInvalidSprite)
	}
	cols = max(1, min(cols, 256))
	px := max(2, int(float64(cols)*float64(b.Dy())/float64(b.Dx())+.5))
	rows := min(256, (px+1)/2)
	s, err := NewSurface(cols, rows, cols/2, rows-1)
	if err != nil {
		return s, err
	}
	sample := func(x, y int) (color.Color, bool) {
		// Average the source block under one art pixel.
		x0, x1 := b.Min.X+x*b.Dx()/cols, b.Min.X+max((x+1)*b.Dx()/cols, x*b.Dx()/cols+1)
		y0, y1 := b.Min.Y+y*b.Dy()/px, b.Min.Y+max((y+1)*b.Dy()/px, y*b.Dy()/px+1)
		var r, g, bl, a, n uint64
		for sy := y0; sy < min(y1, b.Max.Y); sy++ {
			for sx := x0; sx < min(x1, b.Max.X); sx++ {
				cr, cg, cb, ca := img.At(sx, sy).RGBA()
				r, g, bl, a, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca), n+1
			}
		}
		if n == 0 || a/n < 0x8000 {
			return color.Color{}, false
		}
		q := func(c uint64) uint8 { return uint8(c*255/a) &^ 7 } // fewer distinct colors, a smaller palette
		return color.Color{R: q(r), G: q(g), B: q(bl)}, true
	}
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			top, okT := sample(x, y*2)
			bot, okB := sample(x, y*2+1)
			p := Pixel{Rune: '▀', FG: top, BG: bot}
			switch {
			case !okT && !okB:
				p = Pixel{Transparent: true}
			case !okT:
				p.FG = bot
			case !okB:
				p.BG = top
			}
			if err := s.Set(x, y, p); err != nil {
				return s, err
			}
		}
	}
	return s, nil
}

// EncodeJSON writes art in the format LoadJSON reads. Palette keys come
// from the private use area, so any number of colors gets a key.
func EncodeJSON(w io.Writer, name string, s Surface) error {
	keys := map[Pixel]rune{}
	file := surfaceFile{Name: name, Pivot: [2]int{s.PivotX, s.PivotY}, Palette: map[string]swatchFile{}}
	for y := 0; y < s.Height; y++ {
		var row strings.Builder
		for x := 0; x < s.Width; x++ {
			p, _ := s.At(x, y)
			if p.Transparent {
				p = Pixel{Transparent: true}
			}
			k, ok := keys[p]
			if !ok {
				k = rune(0xE000 + len(keys))
				keys[p] = k
				sw := swatchFile{Transparent: p.Transparent}
				if !p.Transparent {
					sw.Glyph, sw.FG, sw.BG = string(p.Rune), p.FG.Hex(), p.BG.Hex()
				}
				file.Palette[string(k)] = sw
			}
			row.WriteRune(k)
		}
		file.Rows = append(file.Rows, row.String())
	}
	b, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode art: %w", err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("encode art: %w", err)
	}
	return nil
}
