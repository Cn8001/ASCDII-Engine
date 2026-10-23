package session

import (
	"bytes"
	"time"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

const historyLimit = 100

// history stores whole scene snapshots; scenes are small and this keeps undo
// correct for every kind of edit without per operation inverse code.
type history struct {
	undo, redo [][]byte
	label      string
	at         time.Time
}

// checkpoint records the state before an edit. Repeated edits with the same
// label in quick succession, such as dragging a slider, merge into one step.
func (s *Session) checkpoint(label string) {
	now := s.d.Now()
	if label != "" && label == s.hist.label && now.Sub(s.hist.at) < 700*time.Millisecond {
		s.hist.at = now
		return
	}
	var b bytes.Buffer
	if err := scene.Encode(&b, s.edit); err != nil {
		return
	}
	s.hist.undo = append(s.hist.undo, b.Bytes())
	if len(s.hist.undo) > historyLimit {
		s.hist.undo = s.hist.undo[1:]
	}
	s.hist.redo, s.hist.label, s.hist.at = nil, label, now
}

func (s *Session) restore(snap []byte) error {
	t := s.edit.Time
	sc, err := scene.Decode(bytes.NewReader(snap), s.reg, s.lib)
	if err != nil {
		return err
	}
	sc.Time = t
	s.edit = sc
	if _, err := sc.ByID(s.sel); err != nil {
		s.sel = 0
	}
	s.hist.label = ""
	s.changed()
	return nil
}

func (s *Session) Undo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.hist.undo)
	if n == 0 || s.play != nil {
		return nil
	}
	var cur bytes.Buffer
	if err := scene.Encode(&cur, s.edit); err != nil {
		return err
	}
	snap := s.hist.undo[n-1]
	s.hist.undo = s.hist.undo[:n-1]
	s.hist.redo = append(s.hist.redo, cur.Bytes())
	return s.restore(snap)
}

func (s *Session) Redo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.hist.redo)
	if n == 0 || s.play != nil {
		return nil
	}
	var cur bytes.Buffer
	if err := scene.Encode(&cur, s.edit); err != nil {
		return err
	}
	snap := s.hist.redo[n-1]
	s.hist.redo = s.hist.redo[:n-1]
	s.hist.undo = append(s.hist.undo, cur.Bytes())
	return s.restore(snap)
}
