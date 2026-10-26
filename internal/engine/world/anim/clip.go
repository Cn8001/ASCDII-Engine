package anim

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
)

type Mode string

const (
	Once     Mode = "once"
	Loop     Mode = "loop"
	PingPong Mode = "pingpong"
)

type Key struct {
	T    float64 `json:"t"`
	V    any     `json:"v"`
	Ease Ease    `json:"ease,omitempty"` // easing toward the next key
}

type Track struct {
	Prop string `json:"prop"`
	Keys []Key  `json:"keys"`
}

type Clip struct {
	Name      string  `json:"name"`
	Target    string  `json:"target"` // entity name
	Mode      Mode    `json:"mode"`
	Speed     float64 `json:"speed,omitempty"`
	Offset    float64 `json:"offset,omitempty"`
	Tracks    []Track `json:"tracks"`
	Transient bool    `json:"-"` // created at runtime by scripts, never saved
}

// Tween is the one line way to animate a property between two values.
func Tween(name string, from, to any, seconds float64, e Ease) Clip {
	return Clip{Name: name, Mode: Once, Tracks: []Track{{Prop: name, Keys: []Key{{T: 0, V: from, Ease: e}, {T: seconds, V: to}}}}}
}

func (c Clip) Looping() Clip        { c.Mode = Loop; return c }
func (c Clip) Bouncing() Clip       { c.Mode = PingPong; return c }
func (c Clip) Delay(s float64) Clip { c.Offset = s; return c }

func (c Clip) Length() float64 {
	l := 0.0
	for _, tr := range c.Tracks {
		if n := len(tr.Keys); n > 0 {
			l = math.Max(l, tr.Keys[n-1].T)
		}
	}
	return l
}

// Local maps scene time to clip time according to the play mode.
func (c Clip) Local(t float64) float64 {
	speed := c.Speed
	if speed == 0 {
		speed = 1
	}
	t = (t - c.Offset) * speed
	l := c.Length()
	if l <= 0 || t <= 0 {
		return math.Max(0, t)
	}
	switch c.Mode {
	case Loop:
		return math.Mod(t, l)
	case PingPong:
		m := math.Mod(t, 2*l)
		if m > l {
			return 2*l - m
		}
		return m
	}
	return math.Min(t, l)
}

// Apply samples every track at scene time t and writes the values into target.
// A track that fails is reported and skipped; the others still apply.
func (c Clip) Apply(t float64, target prop.Editable) error {
	lt := c.Local(t)
	var errs []error
	for _, tr := range c.Tracks {
		if err := tr.apply(lt, target); err != nil {
			errs = append(errs, fmt.Errorf("clip %s: %w", c.Name, err))
		}
	}
	return errors.Join(errs...)
}

func (tr Track) apply(t float64, target prop.Editable) error {
	p, ok := prop.Get(target, tr.Prop)
	if !ok {
		return fmt.Errorf("%w: %s", prop.ErrUnknownProp, tr.Prop)
	}
	v, err := tr.Sample(t, p.Type)
	if err != nil || v == nil {
		return err
	}
	return target.Set(tr.Prop, v)
}

func (tr Track) Sample(t float64, typ prop.Type) (any, error) {
	n := len(tr.Keys)
	if n == 0 {
		return nil, nil
	}
	if t <= tr.Keys[0].T || n == 1 {
		return tr.Keys[0].V, nil
	}
	if t >= tr.Keys[n-1].T {
		return tr.Keys[n-1].V, nil
	}
	i := sort.Search(n, func(i int) bool { return tr.Keys[i].T > t }) - 1
	a, b := tr.Keys[i], tr.Keys[i+1]
	k := a.Ease.Apply((t - a.T) / (b.T - a.T))
	return interpolate(a.V, b.V, k, typ)
}

func interpolate(a, b any, k float64, typ prop.Type) (any, error) {
	switch typ {
	case prop.Float, prop.Int:
		x, err := prop.ToFloat(a)
		if err != nil {
			return nil, err
		}
		y, err := prop.ToFloat(b)
		if err != nil {
			return nil, err
		}
		return geom.Lerp(x, y, k), nil
	case prop.Vec3:
		x, err := prop.ToVec3(a)
		if err != nil {
			return nil, err
		}
		y, err := prop.ToVec3(b)
		if err != nil {
			return nil, err
		}
		return x.Lerp(y, k), nil
	case prop.Color:
		x, err := prop.ToColor(a)
		if err != nil {
			return nil, err
		}
		y, err := prop.ToColor(b)
		if err != nil {
			return nil, err
		}
		return color.Mix(x, y, k), nil
	}
	if k < 1 {
		return a, nil
	}
	return b, nil
}

// SetKey inserts or replaces the key at time t on the named track.
func (c *Clip) SetKey(propName string, t float64, v any, e Ease) {
	for i := range c.Tracks {
		if c.Tracks[i].Prop != propName {
			continue
		}
		keys := c.Tracks[i].Keys
		for j := range keys {
			if math.Abs(keys[j].T-t) < 1e-3 {
				keys[j].V, keys[j].Ease = v, e
				return
			}
		}
		keys = append(keys, Key{T: t, V: v, Ease: e})
		sort.Slice(keys, func(a, b int) bool { return keys[a].T < keys[b].T })
		c.Tracks[i].Keys = keys
		return
	}
	c.Tracks = append(c.Tracks, Track{Prop: propName, Keys: []Key{{T: t, V: v, Ease: e}}})
}

// RemoveKey drops a key and the track once it is empty.
func (c *Clip) RemoveKey(propName string, t float64) bool {
	for i := range c.Tracks {
		if c.Tracks[i].Prop != propName {
			continue
		}
		keys := c.Tracks[i].Keys
		for j := range keys {
			if math.Abs(keys[j].T-t) < 1e-3 {
				c.Tracks[i].Keys = append(keys[:j], keys[j+1:]...)
				if len(c.Tracks[i].Keys) == 0 {
					c.Tracks = append(c.Tracks[:i], c.Tracks[i+1:]...)
				}
				return true
			}
		}
	}
	return false
}

func (c *Clip) key(propName string, t float64) (*Track, int) {
	for i := range c.Tracks {
		if c.Tracks[i].Prop != propName {
			continue
		}
		for j := range c.Tracks[i].Keys {
			if math.Abs(c.Tracks[i].Keys[j].T-t) < 1e-3 {
				return &c.Tracks[i], j
			}
		}
	}
	return nil, -1
}

// MoveKey retimes a key, replacing any key already at the destination.
func (c *Clip) MoveKey(propName string, from, to float64) bool {
	tr, j := c.key(propName, from)
	if tr == nil {
		return false
	}
	k := tr.Keys[j]
	tr.Keys = append(tr.Keys[:j], tr.Keys[j+1:]...)
	c.SetKey(propName, math.Max(0, to), k.V, k.Ease)
	return true
}

func (c *Clip) SetEase(propName string, t float64, e Ease) bool {
	tr, j := c.key(propName, t)
	if tr == nil {
		return false
	}
	tr.Keys[j].Ease = e
	return true
}
