package raster

import "github.com/Cn8001/ASCDII-Engine/internal/engine/core/par"

// queue records writes while the buffer is deferred so they can be
// rasterized on every core. Each row band replays all writes in call order,
// so every dot sees the same depth tests as a serial draw and the frame comes
// out identical.
type queue struct {
	on    bool
	ops   []op
	plots []plotOp
	tris  []tri
}

type op struct {
	tri    bool
	i      int32 // index into plots or tris
	y0, y1 int32 // dot rows touched, inclusive
}

type plotOp struct {
	x, y int
	f    Frag
	tag  uint32
	part uint16
}

func (q *queue) plot(x, y int, f Frag, tag uint32, part uint16) {
	q.ops = append(q.ops, op{i: int32(len(q.plots)), y0: int32(y), y1: int32(y)})
	q.plots = append(q.plots, plotOp{x, y, f, tag, part})
}

func (q *queue) tri(t tri) {
	q.ops = append(q.ops, op{tri: true, i: int32(len(q.tris)), y0: int32(t.y0), y1: int32(t.y1)})
	q.tris = append(q.tris, t)
}

func (q *queue) reset() {
	q.ops, q.plots, q.tris = q.ops[:0], q.plots[:0], q.tris[:0]
}

// Defer queues the following writes until Flush or the next read. The caller
// must not touch the buffer from other goroutines meanwhile.
func (b *Buffer) Defer() { b.q.on = true }

// Flush rasterizes the queued writes and ends deferral.
func (b *Buffer) Flush() {
	b.sync()
	b.q.on = false
}

func (b *Buffer) sync() {
	if len(b.q.ops) == 0 {
		return
	}
	q := &b.q
	par.Rows(b.H, func(lo, hi int) {
		for _, o := range q.ops {
			if int(o.y1) < lo || int(o.y0) >= hi {
				continue
			}
			if o.tri {
				b.triangle(&q.tris[o.i], lo, hi)
				continue
			}
			p := &q.plots[o.i]
			b.plot(p.x, p.y, p.f, p.tag, p.part)
		}
	})
	q.reset()
}
