// Package speaker plays a 16 bit stereo stream on the system's sound
// device. It is the only package that needs cgo (ALSA) on Linux, so only the
// terminal player imports it.
package speaker

import (
	"fmt"
	"io"

	"github.com/ebitengine/oto/v3"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/audio"
)

type closer struct{ p *oto.Player }

func (c closer) Close() error { return c.p.Close() }

// Open starts playing r, read as audio.Rate stereo signed 16 bit samples,
// until the returned closer is closed. A process opens one speaker at most.
func Open(r io.Reader) (io.Closer, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{SampleRate: audio.Rate, ChannelCount: 2, Format: oto.FormatSignedInt16LE})
	if err != nil {
		return nil, fmt.Errorf("open speaker: %w", err)
	}
	<-ready
	p := ctx.NewPlayer(r)
	p.Play()
	return closer{p}, nil
}
