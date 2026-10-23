// Package anim animates any prop.Editable. Clips are sampled from absolute
// scene time instead of being stepped, so the editor can scrub without drift.
package anim

import "math"

type Ease string

const (
	Linear  Ease = "linear"
	In      Ease = "in"
	Out     Ease = "out"
	InOut   Ease = "inout"
	Step    Ease = "step"
	Back    Ease = "back"
	Elastic Ease = "elastic"
)

func Eases() []string {
	return []string{string(Linear), string(In), string(Out), string(InOut), string(Step), string(Back), string(Elastic)}
}

func (e Ease) Apply(t float64) float64 {
	t = math.Max(0, math.Min(1, t))
	switch e {
	case In:
		return t * t * t
	case Out:
		u := 1 - t
		return 1 - u*u*u
	case InOut:
		if t < .5 {
			return 4 * t * t * t
		}
		u := -2*t + 2
		return 1 - u*u*u/2
	case Step:
		if t < 1 {
			return 0
		}
		return 1
	case Back:
		const c = 1.70158
		u := t - 1
		return 1 + (c+1)*u*u*u + c*u*u
	case Elastic:
		if t == 0 || t == 1 {
			return t
		}
		return math.Pow(2, -10*t)*math.Sin((t*10-.75)*2*math.Pi/3) + 1
	}
	return t
}

// Wave is a sine in cycles per second, the building block for idle motion.
func Wave(t, hz, phase float64) float64 { return math.Sin(2*math.Pi*hz*t + phase) }
