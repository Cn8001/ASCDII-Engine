package anim

import (
	"math"

	"gitlab.com/rapsodoinc/rapsodo-studios/ascii-engine/internal/engine/core/geom"
)

// Bone pivots are absolute rest coordinates in sprite art space, so part art can
// be authored exactly where it sits in the rest pose and simply pushed per bone.
type Bone struct {
	Name   string    `json:"name"`
	Parent string    `json:"parent,omitempty"`
	Pivot  geom.Vec2 `json:"pivot"`
}

type Rig struct {
	Bones []Bone `json:"bones"` // parents before children
}

type BonePose struct {
	Rot float64 `json:"rot"` // degrees
	DX  float64 `json:"dx,omitempty"`
	DY  float64 `json:"dy,omitempty"`
}

type Pose map[string]BonePose

type PoseKey struct {
	T    float64 `json:"t"`
	Pose Pose    `json:"pose"`
}

// Motion is a pose clip such as walk or flap, eased with smoothstep between keys.
type Motion struct {
	Name string    `json:"name"`
	Loop bool      `json:"loop"`
	Keys []PoseKey `json:"keys"`
}

func (m Motion) Length() float64 {
	if len(m.Keys) == 0 {
		return 0
	}
	return m.Keys[len(m.Keys)-1].T
}

func (m Motion) Sample(t float64) Pose {
	n := len(m.Keys)
	if n == 0 {
		return Pose{}
	}
	l := m.Length()
	if m.Loop && l > 0 {
		t = math.Mod(math.Mod(t, l)+l, l)
	}
	if t <= m.Keys[0].T || n == 1 {
		return m.Keys[0].Pose
	}
	for i := 0; i+1 < n; i++ {
		a, b := m.Keys[i], m.Keys[i+1]
		if t <= b.T {
			return Blend(a.Pose, b.Pose, geom.Smoothstep(0, 1, (t-a.T)/(b.T-a.T)))
		}
	}
	return m.Keys[n-1].Pose
}

// Blend crossfades two poses; bones missing on one side blend from rest.
func Blend(a, b Pose, w float64) Pose {
	out := Pose{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		x := out[k]
		out[k] = BonePose{Rot: geom.Lerp(x.Rot, v.Rot, w), DX: geom.Lerp(x.DX, v.DX, w), DY: geom.Lerp(x.DY, v.DY, w)}
	}
	for k, v := range a {
		if _, ok := b[k]; !ok {
			out[k] = BonePose{Rot: v.Rot * (1 - w), DX: v.DX * (1 - w), DY: v.DY * (1 - w)}
		}
	}
	return out
}

// Solve returns, per bone, the transform from rest art space to posed art space.
func (r Rig) Solve(p Pose) map[string]geom.Affine {
	out := make(map[string]geom.Affine, len(r.Bones))
	for _, b := range r.Bones {
		parent, ok := out[b.Parent]
		if !ok {
			parent = geom.Identity()
		}
		bp := p[b.Name]
		local := geom.Translate(b.Pivot.X+bp.DX, b.Pivot.Y+bp.DY).
			Mul(geom.Rotate(bp.Rot * math.Pi / 180)).
			Mul(geom.Translate(-b.Pivot.X, -b.Pivot.Y))
		out[b.Name] = parent.Mul(local)
	}
	return out
}
