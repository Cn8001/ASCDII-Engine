// Package window is the desktop display: an Ebitengine window that draws
// every dot as a square of screen pixels and reports key presses and
// releases. Frames come from the Pixels pass; text overlays are drawn at
// screen resolution on top of the scaled dots.
package window

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	stdcolor "image/color"
	"math"
	"sync"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/sfnt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/input"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/platform/pixels"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
)

// targetDots is about how many dots wide the view is when Options.Scale is
// 0: the renderer runs on the CPU, and this keeps a frame near the cost of a
// full screen terminal.
const targetDots = 480

type Options struct {
	Title         string
	Icon          image.Image // nil keeps the default
	Width, Height int         // first window size; 0 is 1280x720
	Scale         int         // screen pixels per dot; 0 picks one from the window width
	Fullscreen    bool
}

// Window is a platform.Display and platform.Input. Present and Events are
// called by the engine's goroutine, Update, Draw and Layout by Ebitengine's.
type Window struct {
	opts   Options
	events chan input.Event
	done   chan struct{}
	held   map[ebiten.Key]input.Key
	keys   []ebiten.Key

	mu      sync.Mutex
	screenW int
	screenH int
	front   *image.RGBA
	glyphs  []pixels.Glyph
	fresh   bool

	back      *image.RGBA // the engine's side of the double buffer
	backText  []pixels.Glyph
	img       *ebiten.Image
	shown     []pixels.Glyph
	source    *text.GoTextFaceSource
	face      *text.GoTextFace
	faceScale int
	font      *sfnt.Font
	glyphOf   sfnt.Buffer
	has       map[rune]bool // whether the font draws a rune, filled as runes appear
}

func New(o Options) *Window {
	if o.Width <= 0 || o.Height <= 0 {
		o.Width, o.Height = 1280, 720
	}
	return &Window{opts: o, events: make(chan input.Event, 64), done: make(chan struct{}), held: map[ebiten.Key]input.Key{}, has: map[rune]bool{}, screenW: o.Width, screenH: o.Height}
}

// Run opens the window and plays on it. Ebitengine must own the calling
// goroutine, the program's main one, so play runs on another; closing the
// window cancels play's context, and play returning closes the window.
func Run(ctx context.Context, o Options, play func(context.Context, platform.Display, platform.Input) error) error {
	w := New(o)
	src, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return fmt.Errorf("window font: %w", err)
	}
	if w.font, err = sfnt.Parse(gomono.TTF); err != nil {
		return fmt.Errorf("window font: %w", err)
	}
	w.source = src
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- play(ctx, w, w)
		close(w.done)
	}()
	ebiten.SetWindowTitle(o.Title)
	ebiten.SetWindowSize(w.opts.Width, w.opts.Height)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(o.Fullscreen)
	if o.Icon != nil {
		ebiten.SetWindowIcon([]image.Image{o.Icon})
	}
	runErr := ebiten.RunGame(w)
	cancel()
	err = <-result
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		err = nil // the window was closed
	}
	if runErr != nil {
		return errors.Join(fmt.Errorf("window: %w", runErr), err)
	}
	return err
}

func (w *Window) Open(context.Context) error  { return nil }
func (w *Window) Close(context.Context) error { return nil }

// Size is how many cells fit at the current scale.
func (w *Window) Size() (int, int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := w.scale()
	return max(1, w.screenW/(raster.SubW*s)), max(1, w.screenH/(raster.SubH*s)), nil
}

func (w *Window) scale() int {
	if w.opts.Scale > 0 {
		return w.opts.Scale
	}
	return max(1, int(math.Round(float64(w.screenW)/targetDots)))
}

// Present composes the frame on the engine's goroutine and hands it to the
// next Draw; a frame Draw has not shown yet is replaced.
func (w *Window) Present(ctx context.Context, f platform.Frame) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.back, w.backText = pixels.Compose(w.back, f, w.backText[:0])
	w.mu.Lock()
	w.front, w.back = w.back, w.front
	w.glyphs, w.backText = w.backText, w.glyphs
	w.fresh = true
	w.mu.Unlock()
	return nil
}

func (w *Window) Events(context.Context) <-chan input.Event { return w.events }

func (w *Window) Update() error {
	select {
	case <-w.done:
		return ebiten.Termination
	default:
	}
	alt := ebiten.IsKeyPressed(ebiten.KeyAlt)
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || alt && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return nil
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	w.keys = inpututil.AppendJustPressedKeys(w.keys[:0])
	for _, k := range w.keys {
		if ik := keyOf(k, ctrl); ik != 0 {
			w.held[k] = ik
			select {
			case w.events <- input.Event{Key: ik, Down: true, Exact: true}:
			default: // the engine is behind; a lost press beats a stalled window
			}
		}
	}
	w.keys = inpututil.AppendJustReleasedKeys(w.keys[:0])
	for _, k := range w.keys {
		if ik, ok := w.held[k]; ok {
			delete(w.held, k)
			select {
			case w.events <- input.Event{Key: ik, Down: false, Exact: true}:
			case <-w.done:
			}
		}
	}
	return nil
}

func (w *Window) Draw(screen *ebiten.Image) {
	w.mu.Lock()
	if w.fresh {
		b := w.front.Rect
		if w.img == nil || w.img.Bounds() != b {
			if w.img != nil {
				w.img.Deallocate()
			}
			w.img = ebiten.NewImage(b.Dx(), b.Dy())
		}
		w.img.WritePixels(w.front.Pix)
		w.shown = append(w.shown[:0], w.glyphs...)
		w.fresh = false
	}
	s := w.scale()
	w.mu.Unlock()
	if w.img == nil {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	iw, ih := w.img.Bounds().Dx()*s, w.img.Bounds().Dy()*s
	ox, oy := float64((sw-iw)/2), float64((sh-ih)/2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(s), float64(s))
	op.GeoM.Translate(ox, oy)
	screen.DrawImage(w.img, op)
	w.drawGlyphs(screen, s, ox, oy)
}

// drawGlyphs writes overlay text one cell at a time so every glyph sits in
// its cell whatever the font's advance. A rune the font lacks is a square in
// the glyph color, as the PNG writer draws glyphs, not a missing glyph box.
func (w *Window) drawGlyphs(screen *ebiten.Image, s int, ox, oy float64) {
	if len(w.shown) == 0 {
		return
	}
	cw, ch := float64(raster.SubW*s), float64(raster.SubH*s)
	if w.face == nil || w.faceScale != s {
		w.face = &text.GoTextFace{Source: w.source, Size: min(ch*.9, cw/.6)} // Go Mono advances .6 em
		w.faceScale = s
	}
	m := w.face.Metrics()
	dy := (ch - (m.HAscent + m.HDescent)) / 2
	var buf [4]byte
	for _, g := range w.shown {
		x, y := ox+float64(g.Col)*cw, oy+float64(g.Row)*ch
		if !w.drawable(g.Rune) {
			vector.FillRect(screen, float32(x+cw/4), float32(y+ch/4), float32(cw/2), float32(ch/2), stdcolor.RGBA{g.FG.R, g.FG.G, g.FG.B, 255}, false)
			continue
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y+dy)
		op.ColorScale.ScaleWithColor(stdcolor.RGBA{g.FG.R, g.FG.G, g.FG.B, 255})
		n := utf8.EncodeRune(buf[:], g.Rune)
		text.Draw(screen, string(buf[:n]), w.face, op)
	}
}

func (w *Window) drawable(r rune) bool {
	ok, seen := w.has[r]
	if !seen {
		i, err := w.font.GlyphIndex(&w.glyphOf, r)
		ok = err == nil && i != 0
		w.has[r] = ok
	}
	return ok
}

// Layout works in device pixels so dots and text stay sharp on high density
// screens.
func (w *Window) Layout(outW, outH int) (int, int) {
	f := ebiten.Monitor().DeviceScaleFactor()
	sw, sh := max(1, int(float64(outW)*f)), max(1, int(float64(outH)*f))
	w.mu.Lock()
	w.screenW, w.screenH = sw, sh
	w.mu.Unlock()
	return sw, sh
}
