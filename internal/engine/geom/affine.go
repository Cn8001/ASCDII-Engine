package geom

import "math"

// Affine is a 2D transform for sprite rigs: x' = A x + C y + TX, y' = B x + D y + TY.
type Affine struct{ A, B, C, D, TX, TY float64 }

func Identity() Affine { return Affine{A: 1, D: 1} }

func Translate(x, y float64) Affine { return Affine{A: 1, D: 1, TX: x, TY: y} }

func Rotate(rad float64) Affine {
	s, c := math.Sincos(rad)
	return Affine{A: c, B: s, C: -s, D: c}
}

func ScaleXY(x, y float64) Affine { return Affine{A: x, D: y} }

// Mul composes so that n is applied first, then m.
func (m Affine) Mul(n Affine) Affine {
	return Affine{
		A:  m.A*n.A + m.C*n.B,
		B:  m.B*n.A + m.D*n.B,
		C:  m.A*n.C + m.C*n.D,
		D:  m.B*n.C + m.D*n.D,
		TX: m.A*n.TX + m.C*n.TY + m.TX,
		TY: m.B*n.TX + m.D*n.TY + m.TY,
	}
}

func (m Affine) Apply(p Vec2) Vec2 {
	return Vec2{m.A*p.X + m.C*p.Y + m.TX, m.B*p.X + m.D*p.Y + m.TY}
}

// Scale is the mean axis scale, used to size round shapes under a transform.
func (m Affine) Scale() float64 { return math.Sqrt(math.Abs(m.A*m.D - m.B*m.C)) }
