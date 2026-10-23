package audio

import (
	"encoding/binary"
	"math"
	"sync"
)

type voice struct {
	s      Sound
	pos    int // sample index
	volume float32
	loop   bool
}

// Mixer plays sounds as voices and is read as one 16 bit stereo stream.
// load turns an asset reference into a sound; results are cached.
type Mixer struct {
	mu     sync.Mutex
	load   func(ref string) (Sound, error)
	cache  map[string]Sound
	voices map[uint64]*voice
	next   uint64
}

func NewMixer(load func(ref string) (Sound, error)) *Mixer {
	return &Mixer{load: load, cache: map[string]Sound{}, voices: map[uint64]*voice{}}
}

func (m *Mixer) Play(ref string, volume float64, loop bool) (uint64, error) {
	m.mu.Lock()
	s, ok := m.cache[ref]
	m.mu.Unlock()
	if !ok {
		var err error
		if s, err = m.load(ref); err != nil {
			return 0, err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[ref] = s
	m.next++
	m.voices[m.next] = &voice{s: s, volume: float32(volume), loop: loop}
	return m.next, nil
}

func (m *Mixer) Stop(id uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.voices, id)
}

// StopAll silences everything, for when a game stops.
func (m *Mixer) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.voices)
}

func (m *Mixer) SetVolume(id uint64, v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if vc, ok := m.voices[id]; ok {
		vc.volume = float32(v)
	}
}

func (m *Mixer) Playing(id uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.voices[id]
	return ok
}

// Read fills p with the mix as 16 bit little endian stereo; it never ends,
// writing silence when nothing plays, as a speaker expects.
func (m *Mixer) Read(p []byte) (int, error) {
	n := len(p) / 4 * 4
	frames := n / 4
	mix := make([]float32, frames*2)
	m.mu.Lock()
	for id, v := range m.voices {
		for i := 0; i < len(mix); i++ {
			if v.pos >= len(v.s.Samples) {
				if !v.loop || len(v.s.Samples) == 0 {
					delete(m.voices, id)
					break
				}
				v.pos = 0
			}
			mix[i] += v.s.Samples[v.pos] * v.volume
			v.pos++
		}
	}
	m.mu.Unlock()
	for i, s := range mix {
		c := int16(math.Round(float64(max(-1, min(1, s))) * 32767))
		binary.LittleEndian.PutUint16(p[i*2:], uint16(c))
	}
	return n, nil
}
