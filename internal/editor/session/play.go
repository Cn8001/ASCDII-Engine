package session

import "gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"

// Play runs a copy of the edited scene so playing never changes the file.
func (s *Session) Play() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == Paused {
		s.mode = Play
		return nil
	}
	if s.play != nil {
		return nil
	}
	c, err := scene.Clone(s.edit, s.reg)
	if err != nil {
		return err
	}
	s.play, s.mode, s.keys = c, Play, nil
	s.resetCtx()
	s.touch()
	return nil
}

// Step advances a paused game by one simulation frame.
func (s *Session) Step() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == Paused {
		s.step = 1.0 / 60
	}
}

func (s *Session) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == Play {
		s.mode = Paused
	}
}

func (s *Session) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
}

func (s *Session) stop() {
	if a, ok := s.sctx.Audio.(*editorAudio); ok {
		a.emit(AudioEvent{Op: "stopall"})
		clear(a.live)
	}
	s.play, s.mode, s.keys = nil, Edit, nil
	s.touch()
}

func (s *Session) SetLive(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.live = on
	s.touch()
}

// SetGizmos shows or hides collider and camera outlines in the Scene view.
func (s *Session) SetGizmos(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gizmos = on
	s.touch()
}

// SetTime scrubs edit time and applies clips at that instant.
func (s *Session) SetTime(t float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	sc.Time = max(0, t)
	s.touch()
	return sc.ApplyClips()
}
