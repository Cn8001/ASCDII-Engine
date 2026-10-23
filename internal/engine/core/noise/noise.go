// Package noise gives deterministic per-coordinate variation so that procedural
// detail never shimmers when the camera moves.
package noise

import "math"

func Hash(x, y int, seed uint32) uint32 {
	h := uint32(int32(x))*374761393 + uint32(int32(y))*668265263 + seed*2246822519
	h ^= h >> 13
	h *= 1274126177
	return h ^ (h >> 16)
}

// Unit maps a hash to [0, 1).
func Unit(h uint32) float64 { return float64(h%100000) / 100000 }

// Value is smooth value noise in [0, 1).
func Value(x, y float64, seed uint32) float64 {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := fade(x-math.Floor(x)), fade(y-math.Floor(y))
	u := func(x, y int) float64 { return Unit(Hash(x, y, seed)) }
	top := u(ix, iy) + (u(ix+1, iy)-u(ix, iy))*fx
	bottom := u(ix, iy+1) + (u(ix+1, iy+1)-u(ix, iy+1))*fx
	return top + (bottom-top)*fy
}

func fade(t float64) float64 { return t * t * (3 - 2*t) }
