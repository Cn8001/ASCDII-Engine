package session

import (
	"errors"
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/fx"
)

var (
	ErrNotParticles  = errors.New("component is not a particle system")
	ErrUnknownPreset = errors.New("unknown particle preset")
	ErrUnknownAction = errors.New("unknown particle action")
)

func (s *Session) ParticlePresets() []string { return fx.Presets() }

// Particles runs play, stop, emit, clear, count or preset on the particle
// system key of id, in the editor too, so effects can be tried without
// playing the game. It returns how many particles are alive.
func (s *Session) Particles(id uint64, key, action, arg string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.current().ByID(id)
	if err != nil {
		return 0, err
	}
	e, ok := o.ComponentByKey(key).(*components.Emitter)
	if !ok {
		return 0, fmt.Errorf("particles %s: %w", key, ErrNotParticles)
	}
	switch action {
	case "count":
	case "preset":
		p, ok := fx.Preset(arg)
		if !ok {
			return 0, fmt.Errorf("particles %s: %w", arg, ErrUnknownPreset)
		}
		if s.play == nil {
			s.checkpoint("")
		}
		p.Seed = e.FX.Seed
		e.FX = p
		if _, err := e.Call(&s.sctx, o, "clear", nil); err != nil {
			return 0, err
		}
		if _, err := e.Call(&s.sctx, o, "play", nil); err != nil {
			return 0, err
		}
		s.changed()
	case "play", "stop", "emit", "clear":
		if _, err := e.Call(&s.sctx, o, action, nil); err != nil {
			return 0, err
		}
		s.touch()
	default:
		return 0, fmt.Errorf("particles %s: %w: %s", key, ErrUnknownAction, action)
	}
	n, err := e.Call(&s.sctx, o, "count", nil)
	if err != nil {
		return 0, err
	}
	f, _ := n.(float64)
	return int(f), nil
}
