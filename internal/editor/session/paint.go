package session

import (
	"errors"
	"fmt"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/components"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

var ErrMissed = errors.New("the brush is not over the selected object")

// PaintAt adds a brush stroke where the selected object id shows under a
// viewport cell position. Strokes live in the object's paint component,
// which is added on the first stroke.
func (s *Session) PaintAt(id uint64, col, row float64, hex string, radius float64, erase bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.play != nil {
		return fmt.Errorf("paint: %w", ErrNotEditable)
	}
	d := s.frame.Buf.At(int(col*raster.SubW), int(row*raster.SubH))
	if d == nil || d.Tag != uint32(id) {
		return ErrMissed
	}
	o, err := s.edit.ByID(id)
	if err != nil {
		return err
	}
	if o.Prefab != "" || o.Inherited() {
		return fmt.Errorf("paint %s: %w", o.Name, scene.ErrPrefabLocked)
	}
	c, err := color.ParseHex(hex)
	if err != nil && !erase {
		return fmt.Errorf("paint: %w", err)
	}
	s.checkpoint(fmt.Sprintf("paint:%d", id))
	p, _ := o.Component("paint").(*components.Paint)
	if p == nil {
		p = &components.Paint{}
		o.With(p)
	}
	t := o.World()
	local := t.Inward(geom.V3(float64(d.X), float64(d.Y), float64(d.Z)).Sub(t.Pos))
	k := t.Uniform()
	if k == 0 {
		k = 1
	}
	p.Add(components.Stroke{P: local, R: radius / k, C: c, Erase: erase})
	s.changed()
	return nil
}
