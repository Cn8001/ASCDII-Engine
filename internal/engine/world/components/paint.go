package components

import (
	"encoding/json"
	"fmt"
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/color"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/prop"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/pipeline"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/render/raster"
	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/world/scene"
)

// Stroke is one dab of the Braille brush, in the object's own space so the
// paint moves, turns and scales with the object.
type Stroke struct {
	P     geom.Vec3   `json:"p"`
	R     float64     `json:"r"`
	C     color.Color `json:"c"`
	Erase bool        `json:"erase,omitempty"` // hides the dots instead of coloring them
}

// Paint is a layer of brush strokes over whatever its object draws, for
// small fixes to a model or primitive without leaving the editor.
type Paint struct {
	Strokes []Stroke
}

func (*Paint) Kind() string { return "paint" }

func (p *Paint) Props() []prop.Prop {
	return []prop.Prop{{Name: "strokes", Type: prop.Data, Value: p.Strokes}}
}

func (p *Paint) Set(n string, v any) error {
	if n != "strokes" {
		return fmt.Errorf("%w: %s", prop.ErrUnknownProp, n)
	}
	if s, ok := v.([]Stroke); ok {
		p.Strokes = s
		return nil
	}
	b, err := json.Marshal(v) // from a file: a list of plain objects
	if err != nil {
		return fmt.Errorf("strokes: %w", err)
	}
	var out []Stroke
	if err := json.Unmarshal(b, &out); err != nil {
		return fmt.Errorf("strokes: %w: %w", prop.ErrInvalidValue, err)
	}
	p.Strokes = out
	return nil
}

func (p *Paint) Add(s Stroke) { p.Strokes = append(p.Strokes, s) }

// PaintDots recolors the object's dots under a stroke; later strokes win.
func (p *Paint) PaintDots(f *pipeline.Frame, o *scene.GameObject, t scene.Transform) {
	if len(p.Strokes) == 0 {
		return
	}
	tag := uint32(o.ID)
	dots := f.Buf.Dots()
	for i := range dots {
		d := &dots[i]
		if d.Tag != tag {
			continue
		}
		local := t.Inward(geom.V3(float64(d.X), float64(d.Y), float64(d.Z)).Sub(t.Pos))
		for k := len(p.Strokes) - 1; k >= 0; k-- {
			s := p.Strokes[k]
			if local.Sub(s.P).Len() > s.R {
				continue
			}
			if s.Erase {
				*d = raster.Dot{Color: f.Env.Sky, Albedo: f.Env.Sky, Flags: raster.FlagEmpty | raster.FlagUnlit, Depth: float32(math.Inf(-1))}
			} else {
				d.Albedo, d.Color = s.C, s.C
			}
			break
		}
	}
}
