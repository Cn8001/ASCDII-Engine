package session

import (
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
)

func (s *Session) Passes() []pipeline.Info {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pipe.Infos()
}

func (s *Session) SetPassEnabled(name string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touch()
	return s.pipe.SetEnabled(name, on)
}

func (s *Session) MovePass(name string, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.touch()
	return s.pipe.Move(name, index)
}

func (s *Session) editable(name string) (prop.Editable, error) {
	p, err := s.pipe.Pass(name)
	if err != nil {
		return nil, err
	}
	e, ok := p.(prop.Editable)
	if !ok {
		return nil, fmt.Errorf("pass %s: %w", name, ErrNotEditable)
	}
	return e, nil
}

func (s *Session) PassProps(name string) ([]prop.Prop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.editable(name)
	if err != nil {
		return nil, err
	}
	return e.Props(), nil
}

func (s *Session) SetPassProp(name, p string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.editable(name)
	if err != nil {
		return err
	}
	s.touch()
	return e.Set(p, v)
}
