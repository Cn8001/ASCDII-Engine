// Package geom holds the world-space math used by content and the projection.
package geom

import (
	"encoding/json"
	"fmt"
	"math"
)

type Vec2 struct{ X, Y float64 }

type Vec3 struct{ X, Y, Z float64 }

func V2(x, y float64) Vec2    { return Vec2{x, y} }
func V3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec2) Add(b Vec2) Vec2             { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2             { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Scale(k float64) Vec2        { return Vec2{a.X * k, a.Y * k} }
func (a Vec2) Dot(b Vec2) float64          { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Len() float64                { return math.Hypot(a.X, a.Y) }
func (a Vec2) Lerp(b Vec2, t float64) Vec2 { return a.Add(b.Sub(a).Scale(t)) }

func (a Vec2) Rotate(rad float64) Vec2 {
	s, c := math.Sincos(rad)
	return Vec2{a.X*c - a.Y*s, a.X*s + a.Y*c}
}

func (a Vec2) Norm() Vec2 {
	l := a.Len()
	if l == 0 {
		return Vec2{}
	}
	return a.Scale(1 / l)
}

func (a Vec3) Add(b Vec3) Vec3             { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3             { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(k float64) Vec3        { return Vec3{a.X * k, a.Y * k, a.Z * k} }
func (a Vec3) Dot(b Vec3) float64          { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64                { return math.Sqrt(a.Dot(a)) }
func (a Vec3) XY() Vec2                    { return Vec2{a.X, a.Y} }
func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Scale(t)) }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

func (a Vec2) MarshalJSON() ([]byte, error) { return json.Marshal([2]float64{a.X, a.Y}) }
func (a Vec3) MarshalJSON() ([]byte, error) { return json.Marshal([3]float64{a.X, a.Y, a.Z}) }

func (a *Vec2) UnmarshalJSON(b []byte) error {
	var v [2]float64
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("decode vec2: %w", err)
	}
	*a = Vec2{v[0], v[1]}
	return nil
}

func (a *Vec3) UnmarshalJSON(b []byte) error {
	var v [3]float64
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("decode vec3: %w", err)
	}
	*a = Vec3{v[0], v[1], v[2]}
	return nil
}

// AABB is the single footprint used for drawing bounds, picking and collision.
type AABB struct{ Min, Max Vec3 }

// Footprint builds a box standing on base, centered in x and y.
func Footprint(base, size Vec3) AABB {
	return AABB{
		Min: Vec3{base.X - size.X/2, base.Y - size.Y/2, base.Z},
		Max: Vec3{base.X + size.X/2, base.Y + size.Y/2, base.Z + size.Z},
	}
}

func (b AABB) Empty() bool           { return b.Max.X <= b.Min.X || b.Max.Y <= b.Min.Y }
func (b AABB) Size() Vec3            { return b.Max.Sub(b.Min) }
func (b AABB) Center() Vec3          { return b.Min.Add(b.Max).Scale(.5) }
func (b AABB) Translate(d Vec3) AABB { return AABB{b.Min.Add(d), b.Max.Add(d)} }

func (b AABB) Contains2D(p Vec2) bool {
	return p.X >= b.Min.X && p.X < b.Max.X && p.Y >= b.Min.Y && p.Y < b.Max.Y
}

func (b AABB) Overlaps2D(o AABB) bool {
	return b.Min.X < o.Max.X && o.Min.X < b.Max.X && b.Min.Y < o.Max.Y && o.Min.Y < b.Max.Y
}

func (b AABB) Union(o AABB) AABB {
	if b.Empty() {
		return o
	}
	if o.Empty() {
		return b
	}
	return AABB{
		Min: Vec3{math.Min(b.Min.X, o.Min.X), math.Min(b.Min.Y, o.Min.Y), math.Min(b.Min.Z, o.Min.Z)},
		Max: Vec3{math.Max(b.Max.X, o.Max.X), math.Max(b.Max.Y, o.Max.Y), math.Max(b.Max.Z, o.Max.Z)},
	}
}

func (b AABB) Corners() [8]Vec3 {
	var out [8]Vec3
	for i := range out {
		out[i] = Vec3{pick(i&1, b.Min.X, b.Max.X), pick(i&2, b.Min.Y, b.Max.Y), pick(i&4, b.Min.Z, b.Max.Z)}
	}
	return out
}

func pick(bit int, lo, hi float64) float64 {
	if bit != 0 {
		return hi
	}
	return lo
}

func Lerp(a, b, t float64) float64 { return a + (b-a)*t }

func Clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func Smoothstep(e0, e1, x float64) float64 {
	t := Clamp((x-e0)/(e1-e0), 0, 1)
	return t * t * (3 - 2*t)
}

func (b AABB) Scaled(k float64) AABB { return AABB{b.Min.Scale(k), b.Max.Scale(k)} }
