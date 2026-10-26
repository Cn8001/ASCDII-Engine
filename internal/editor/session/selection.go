package session

import (
	"slices"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// The selection is the primary object s.sel, which the Inspector and the
// gizmo show, plus s.group, the others selected with it.

// selectOnly makes id the whole selection.
func (s *Session) selectOnly(id uint64) {
	s.sel, s.group = id, nil
}

// selection lists the selected objects that still exist, primary first.
func (s *Session) selection() []uint64 {
	sc := s.current()
	var out []uint64
	for _, id := range append([]uint64{s.sel}, s.group...) {
		if id == 0 || slices.Contains(out, id) {
			continue
		}
		if _, err := sc.ByID(id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// Selection is every selected object, primary first.
func (s *Session) Selection() []uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.selection()
}

// target is what a click at a cell selects. Inside a prefab instance the
// first click takes the instance root and each further click on the same
// spot one level deeper, so a whole lantern is picked before its flame;
// exact takes the object drawn there.
func (s *Session) target(col, row int, exact bool) uint64 {
	raw := uint64(s.frame.Buf.Pick(col, row))
	if raw == 0 || exact {
		return raw
	}
	e, err := s.current().ByID(raw)
	if err != nil {
		return raw
	}
	var chain []*scene.GameObject // from the top level down to e
	for o := e; o != nil && o != s.current().Root; o = o.Parent() {
		chain = append([]*scene.GameObject{o}, chain...)
	}
	root := -1
	for i, o := range chain {
		if o.Prefab != "" && !o.Inherited() {
			root = i
			break
		}
	}
	if root < 0 {
		return raw
	}
	for i := root; i < len(chain)-1; i++ {
		if chain[i].ID == s.sel {
			return chain[i+1].ID
		}
	}
	return chain[root].ID
}

// Pick selects what a click at a cell targets; empty ground clears the
// selection.
func (s *Session) Pick(col, row int, exact bool) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selectOnly(s.target(col, row, exact))
	s.touch()
	return s.sel
}

// TogglePick adds the target of a click to the selection, or takes it out
// when it is already in, as Ctrl or Shift click does. It returns the primary.
func (s *Session) TogglePick(col, row int, exact bool) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.target(col, row, exact)
	if t == 0 {
		return s.sel
	}
	all := s.selection()
	if i := slices.Index(all, t); i >= 0 {
		all = slices.Delete(all, i, i+1)
	} else {
		all = append([]uint64{t}, all...)
	}
	s.setSelection(all)
	s.touch()
	return s.sel
}

// SelectRect selects every object drawn inside a rectangle of cells, each by
// the target rule of a first click; add keeps the current selection. An
// object counts when its position or the center of its bounds lies inside,
// so ground and other large objects reaching into every rectangle stay out.
func (s *Session) SelectRect(col0, row0, col1, row1 int, add bool) []uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	col0, col1 = min(col0, col1), max(col0, col1)
	row0, row1 = min(row0, row1), max(row0, row1)
	var found []uint64
	if add {
		found = s.selection()
	}
	keep := s.sel
	s.sel = 0 // so target picks roots, not one level below the old selection
	seen := map[uint32]bool{}
	for y := row0 * raster.SubH; y < (row1+1)*raster.SubH; y++ {
		for x := col0 * raster.SubW; x < (col1+1)*raster.SubW; x++ {
			d := s.frame.Buf.At(x, y)
			if d == nil || d.Tag == 0 || seen[d.Tag] {
				continue
			}
			seen[d.Tag] = true
			if t := s.target(x/raster.SubW, y/raster.SubH, false); t != 0 && !slices.Contains(found, t) && s.inRect(t, col0, row0, col1, row1) {
				found = append(found, t)
			}
		}
	}
	s.sel = keep
	s.setSelection(found)
	s.touch()
	return s.selection()
}

// inRect reports whether the position or the bounds center of id projects
// into the rectangle of cells.
func (s *Session) inRect(id uint64, col0, row0, col1, row1 int) bool {
	e, err := s.current().ByID(id)
	if err != nil {
		return false
	}
	points := []geom.Vec3{e.World().Pos}
	if b := e.Bounds(); !b.Empty() {
		points = append(points, b.Center())
	}
	for _, p := range points {
		x, y, _ := s.frame.View.Project(p)
		c, r := int(x)/raster.SubW, int(y)/raster.SubH
		if c >= col0 && c <= col1 && r >= row0 && r <= row1 {
			return true
		}
	}
	return false
}

func (s *Session) setSelection(ids []uint64) {
	if len(ids) == 0 {
		s.selectOnly(0)
		return
	}
	s.sel, s.group = ids[0], slices.Clone(ids[1:])
}

// movable is the selection that a transform of id moves: the whole selection
// when id is in it, without objects whose parent is selected too, since they
// already move with it, and without parts of prefab instances.
func (s *Session) movable(id uint64) []*scene.GameObject {
	sc := s.current()
	ids := s.selection()
	if !slices.Contains(ids, id) {
		ids = []uint64{id}
	}
	var out []*scene.GameObject
	for _, i := range ids {
		e, err := sc.ByID(i)
		if err != nil || e.Inherited() {
			continue
		}
		nested := false
		for p := e.Parent(); p != nil; p = p.Parent() {
			if slices.Contains(ids, p.ID) {
				nested = true
				break
			}
		}
		if !nested {
			out = append(out, e)
		}
	}
	return out
}

// RemoveSelection deletes every selected object as one undo step.
func (s *Session) RemoveSelection() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc := s.current()
	ids := s.selection()
	if len(ids) == 0 {
		return nil
	}
	if s.play == nil {
		s.checkpoint("")
	}
	for _, e := range s.movable(ids[0]) {
		if err := sc.Remove(e.ID); err != nil {
			return err
		}
	}
	s.selectOnly(0)
	s.changed()
	return nil
}

// DuplicateSelection copies every selected object next to its original, as
// one undo step, and selects the copies.
func (s *Session) DuplicateSelection() ([]uint64, error) {
	s.mu.Lock()
	var ids []uint64
	if s.sel != 0 {
		for _, e := range s.movable(s.sel) {
			ids = append(ids, e.ID)
		}
	}
	s.mu.Unlock()
	var copies []uint64
	label := "duplicate selection" // one label, so the copies are one undo step
	for _, id := range ids {
		c, err := s.duplicate(id, label)
		if err != nil {
			return copies, err
		}
		copies = append(copies, c)
	}
	s.mu.Lock()
	s.setSelection(copies)
	s.touch()
	s.mu.Unlock()
	return copies, nil
}
