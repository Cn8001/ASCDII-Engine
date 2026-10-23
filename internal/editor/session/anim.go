package session

import (
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/anim"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

type Clip struct {
	Index  int          `json:"index"`
	Name   string       `json:"name"`
	Target string       `json:"target"`
	Mode   anim.Mode    `json:"mode"`
	Speed  float64      `json:"speed"`
	Length float64      `json:"length"`
	Tracks []anim.Track `json:"tracks"`
}

// Clips lists the clips bound to an entity; id 0 lists every clip.
func (s *Session) Clips(id uint64) ([]Clip, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	name := ""
	if id != 0 {
		e, err := sc.ByID(id)
		if err != nil {
			return nil, err
		}
		name = e.Name
	}
	out := []Clip{}
	for i, c := range sc.Clips {
		if name == "" || c.Target == name {
			out = append(out, Clip{Index: i, Name: c.Name, Target: c.Target, Mode: c.Mode, Speed: c.Speed, Length: c.Length(), Tracks: c.Tracks})
		}
	}
	return out, nil
}

// AddKey stores the property's current value at time t on the entity's first
// clip, creating a looping clip when there is none.
func (s *Session) AddKey(id uint64, name string, t float64) error {
	return s.edit1(id, "", func(sc *scene.Scene, e *scene.GameObject) error {
		name := e.Resolve(name)
		p, ok := prop.Get(e, name)
		if !ok {
			return fmt.Errorf("%w: %s", prop.ErrUnknownProp, name)
		}
		for i := range sc.Clips {
			if sc.Clips[i].Target == e.Name {
				sc.Clips[i].SetKey(name, t, prop.Encode(p), anim.InOut)
				return nil
			}
		}
		c := anim.Clip{Name: e.Name, Target: e.Name, Mode: anim.Loop}
		c.SetKey(name, t, prop.Encode(p), anim.InOut)
		sc.Clips = append(sc.Clips, c)
		return nil
	})
}

func (s *Session) clip(index int) (*anim.Clip, error) {
	sc := s.current()
	if index < 0 || index >= len(sc.Clips) {
		return nil, fmt.Errorf("%w: clip %d", scene.ErrNodeNotFound, index)
	}
	return &sc.Clips[index], nil
}

func (s *Session) RemoveKey(index int, name string, t float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.clip(index)
	if err != nil {
		return err
	}
	s.checkpoint("")
	c.RemoveKey(name, t)
	s.changed()
	return nil
}

func (s *Session) SetClip(index int, mode anim.Mode, speed float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.clip(index)
	if err != nil {
		return err
	}
	s.checkpoint("")
	c.Mode, c.Speed = mode, speed
	s.changed()
	return nil
}

func (s *Session) RemoveClip(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.clip(index); err != nil {
		return err
	}
	s.checkpoint("")
	sc := s.current()
	sc.Clips = append(sc.Clips[:index], sc.Clips[index+1:]...)
	s.changed()
	return nil
}

func (s *Session) MoveKey(index int, name string, from, to float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.clip(index)
	if err != nil {
		return err
	}
	s.checkpoint(fmt.Sprintf("key:%d:%s", index, name))
	if !c.MoveKey(name, from, to) {
		return fmt.Errorf("%w: key %s at %.3f", scene.ErrNodeNotFound, name, from)
	}
	s.changed()
	return s.current().ApplyClips()
}

func (s *Session) SetEase(index int, name string, t float64, e anim.Ease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := s.clip(index)
	if err != nil {
		return err
	}
	s.checkpoint("")
	if !c.SetEase(name, t, e) {
		return fmt.Errorf("%w: key %s at %.3f", scene.ErrNodeNotFound, name, t)
	}
	s.changed()
	return nil
}
