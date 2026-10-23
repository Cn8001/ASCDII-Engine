// Package audio decodes sound files and mixes playing voices into one
// stream; backends only have to copy that stream to a speaker.
package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hajimehoshi/go-mp3"
)

// Rate is the mixer's sample rate; every sound is resampled to it.
const Rate = 44100

var ErrFormat = errors.New("unsupported sound")

// Sound is interleaved stereo samples in -1..1 at Rate.
type Sound struct {
	Samples []float32
}

func (s Sound) Frames() int { return len(s.Samples) / 2 }

// Decode reads a .wav or .mp3 file, by its name.
func Decode(r io.Reader, name string) (Sound, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return Sound{}, fmt.Errorf("decode %s: %w", name, err)
	}
	switch l := strings.ToLower(name); {
	case strings.HasSuffix(l, ".wav"):
		return decodeWAV(b)
	case strings.HasSuffix(l, ".mp3"):
		return decodeMP3(b)
	}
	return Sound{}, fmt.Errorf("decode %s: %w", name, ErrFormat)
}

// decodeWAV reads PCM WAV files, 8 or 16 bit, mono or stereo.
func decodeWAV(b []byte) (Sound, error) {
	if len(b) < 12 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return Sound{}, fmt.Errorf("wav: %w", ErrFormat)
	}
	var channels, bits int
	var rate int
	pos := 12
	for pos+8 <= len(b) {
		id, size := string(b[pos:pos+4]), int(binary.LittleEndian.Uint32(b[pos+4:]))
		body := pos + 8
		if body+size > len(b) {
			size = len(b) - body
		}
		switch id {
		case "fmt ":
			if size < 16 || binary.LittleEndian.Uint16(b[body:]) != 1 {
				return Sound{}, fmt.Errorf("wav: only PCM: %w", ErrFormat)
			}
			channels = int(binary.LittleEndian.Uint16(b[body+2:]))
			rate = int(binary.LittleEndian.Uint32(b[body+4:]))
			bits = int(binary.LittleEndian.Uint16(b[body+14:]))
		case "data":
			if channels < 1 || channels > 2 || (bits != 8 && bits != 16) || rate <= 0 {
				return Sound{}, fmt.Errorf("wav: %d channels, %d bit: %w", channels, bits, ErrFormat)
			}
			return Sound{Samples: resample(pcm(b[body:body+size], channels, bits), rate)}, nil
		}
		pos = body + size + size%2
	}
	return Sound{}, fmt.Errorf("wav: no data: %w", ErrFormat)
}

// pcm turns raw samples into interleaved stereo floats.
func pcm(b []byte, channels, bits int) []float32 {
	width := bits / 8
	frames := len(b) / (width * channels)
	out := make([]float32, 0, frames*2)
	sample := func(i int) float32 {
		if bits == 8 {
			return (float32(b[i]) - 128) / 128
		}
		return float32(int16(binary.LittleEndian.Uint16(b[i:]))) / 32768
	}
	for f := 0; f < frames; f++ {
		at := f * width * channels
		l := sample(at)
		r := l
		if channels == 2 {
			r = sample(at + width)
		}
		out = append(out, l, r)
	}
	return out
}

func decodeMP3(b []byte) (Sound, error) {
	d, err := mp3.NewDecoder(bytes.NewReader(b))
	if err != nil {
		return Sound{}, fmt.Errorf("mp3: %w", err)
	}
	raw, err := io.ReadAll(d)
	if err != nil {
		return Sound{}, fmt.Errorf("mp3: %w", err)
	}
	return Sound{Samples: resample(pcm(raw, 2, 16), d.SampleRate())}, nil // go-mp3 always yields 16 bit stereo
}

// resample converts stereo samples to Rate with linear interpolation, plenty
// for game sounds.
func resample(in []float32, rate int) []float32 {
	if rate == Rate || len(in) < 4 {
		return in
	}
	frames := len(in) / 2
	n := int(int64(frames) * Rate / int64(rate))
	out := make([]float32, n*2)
	step := float64(rate) / Rate
	for i := 0; i < n; i++ {
		x := float64(i) * step
		j := int(x)
		f := float32(x - float64(j))
		k := min(j+1, frames-1)
		out[i*2] = in[j*2]*(1-f) + in[k*2]*f
		out[i*2+1] = in[j*2+1]*(1-f) + in[k*2+1]*f
	}
	return out
}
