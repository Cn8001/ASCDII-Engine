package session

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/view"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// stashed is the scene that waits while a prefab is edited on its own.
type stashed struct {
	edit    *scene.Scene
	path    string
	hist    history
	unsaved bool
	sel     uint64
	cam     view.Camera
	active  int
}

// OpenPrefab edits a prefab alone, as Unity's prefab mode does. The open
// scene, its history and unsaved edits wait until ClosePrefab.
func (s *Session) OpenPrefab(ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.need(); err != nil {
		return err
	}
	if s.play != nil {
		return fmt.Errorf("open prefab: %w", ErrNotEditable)
	}
	rel, raw, err := s.lib.PrefabSource(ref)
	if err != nil {
		return err
	}
	root, err := scene.DecodePrefab(bytes.NewReader(raw), s.reg)
	if err != nil {
		return err
	}
	sc := scene.New(strings.TrimSuffix(path.Base(rel), ".prefab.json"))
	sc.UsePrefabs(s.lib)
	sc.Env = s.edit.Env
	if _, err := sc.Graft(nil, root, -1); err != nil {
		return err
	}
	keep := s.stash
	if keep == nil {
		keep = &stashed{edit: s.edit, path: s.scenePath, hist: s.hist, unsaved: s.unsaved, sel: s.sel, cam: s.cam, active: s.active}
	}
	s.load(sc, "")
	s.prefab, s.stash = rel, keep
	s.cam.Target, s.cam.Span = geom.V3(0, 0, 0), 20
	s.sel = root.ID
	return nil
}

func (s *Session) savePrefabMode(ctx context.Context) error {
	if locked(s.prefab) {
		return lockedErr(s.prefab)
	}
	roots := s.edit.Root.Children
	if len(roots) != 1 {
		return fmt.Errorf("save %s: a prefab has exactly one root object, found %d: %w", s.prefab, len(roots), ErrNotEditable)
	}
	scene.Localize(roots[0])
	var b bytes.Buffer
	if err := scene.EncodePrefab(&b, roots[0]); err != nil {
		return err
	}
	if err := writeFile(ctx, filepath.Join(s.dir, filepath.FromSlash(s.prefab)), b.Bytes()); err != nil {
		return err
	}
	s.lib.PutPrefab(s.prefab, b.Bytes())
	s.unsaved = false
	return nil
}

// ClosePrefab goes back to the scene; its instances show the saved prefab.
func (s *Session) ClosePrefab() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stash == nil {
		return nil
	}
	if s.play != nil {
		s.stop()
	}
	st := s.stash
	s.load(st.edit, st.path)
	s.hist, s.unsaved, s.sel, s.cam, s.active = st.hist, st.unsaved, st.sel, st.cam, st.active
	return s.edit.Instantiate()
}
