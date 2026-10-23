package components

import (
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// AudioSource plays a sound asset from its object while the game runs.
type AudioSource struct {
	Clip        string
	Volume      float64
	Loop        bool
	PlayOnStart bool
	voice       uint64
	heard       float64 // the volume the output last got
	started     bool
}

func NewAudioSource() *AudioSource { return &AudioSource{Volume: 1, PlayOnStart: true} }

func (*AudioSource) Kind() string { return "audio_source" }
func (a *AudioSource) bind() *prop.Binder {
	return prop.Bind().Audio("clip", &a.Clip).Float("volume", &a.Volume, 0, 1).Bool("loop", &a.Loop).Bool("playOnStart", &a.PlayOnStart)
}
func (a *AudioSource) Props() []prop.Prop        { return a.bind().Props() }
func (a *AudioSource) Set(n string, v any) error { return a.bind().Set(n, v) }

func (a *AudioSource) play(c *scene.Ctx) {
	if c.Audio == nil || a.Clip == "" {
		return
	}
	if a.voice != 0 {
		c.Audio.Stop(a.voice)
	}
	v, err := c.Audio.Play(a.Clip, a.Volume, a.Loop)
	if err != nil {
		c.Logf("error", "audio", "%v", err)
		return
	}
	a.voice, a.heard = v, a.Volume
}

// Update starts the sound on the first frame and passes volume changes on,
// so scripts and the inspector can fade a playing sound.
func (a *AudioSource) Update(c *scene.Ctx, _ *scene.GameObject, _ float64) {
	if !a.started {
		a.started = true
		if a.PlayOnStart {
			a.play(c)
		}
	}
	if a.voice != 0 && c.Audio != nil && a.Volume != a.heard {
		c.Audio.SetVolume(a.voice, a.Volume)
		a.heard = a.Volume
	}
}

func (*AudioSource) Methods() []string { return []string{"play", "stop", "playing"} }

func (a *AudioSource) Call(c *scene.Ctx, _ *scene.GameObject, name string, _ []any) (any, error) {
	switch name {
	case "play":
		a.play(c)
	case "stop":
		if c.Audio != nil && a.voice != 0 {
			c.Audio.Stop(a.voice)
		}
		a.voice = 0
	case "playing":
		return c.Audio != nil && a.voice != 0 && c.Audio.Playing(a.voice), nil
	default:
		return nil, fmt.Errorf("no method %s", name)
	}
	return nil, nil
}
