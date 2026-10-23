package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

// wav builds a mono 16 bit file at rate holding samples.
func wav(rate int, samples ...int16) []byte {
	var b bytes.Buffer
	data := len(samples) * 2
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+data))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16)} {
		_ = binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(data))
	_ = binary.Write(&b, binary.LittleEndian, samples)
	return b.Bytes()
}

func TestUnit_DecodeWAVMonoToStereo(t *testing.T) {
	// Arrange
	file := wav(Rate, 16384, -16384)

	// Act
	s, err := Decode(bytes.NewReader(file), "beep.wav")
	_, errBad := Decode(bytes.NewReader([]byte("nope")), "x.ogg")

	// Assert
	if err != nil || s.Frames() != 2 || s.Samples[0] != .5 || s.Samples[1] != .5 || s.Samples[2] != -.5 {
		t.Fatalf("err %v samples %v", err, s.Samples)
	}
	if !errors.Is(errBad, ErrFormat) {
		t.Fatalf("bad format err %v", errBad)
	}
}

func TestUnit_DecodeResamplesToMixerRate(t *testing.T) {
	// Arrange
	file := wav(Rate/2, 0, 0, 0, 0)

	// Act
	s, err := Decode(bytes.NewReader(file), "low.wav")

	// Assert
	if err != nil || s.Frames() != 8 {
		t.Fatalf("err %v frames %d", err, s.Frames())
	}
}

func TestUnit_MixerMixesLoopsAndEnds(t *testing.T) {
	// Arrange
	one := Sound{Samples: []float32{.25, .25, .25, .25}}
	m := NewMixer(func(string) (Sound, error) { return one, nil })
	looping, _ := m.Play("a", 1, true)
	once, _ := m.Play("b", 2, false)
	p := make([]byte, 4*4)

	// Act
	_, _ = m.Read(p)

	// Assert
	first := int16(binary.LittleEndian.Uint16(p))
	last := int16(binary.LittleEndian.Uint16(p[12:]))
	if first < 24000 || last < 8000 || last > 8400 {
		t.Fatalf("first %d last %d", first, last)
	}
	if !m.Playing(looping) || m.Playing(once) {
		t.Fatal("the loop should keep playing and the one shot end")
	}
}
