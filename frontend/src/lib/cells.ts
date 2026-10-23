// Decodes the backend frame format (10 bytes per cell: rune u32 LE, fg rgb,
// bg rgb) and paints it into an ImageData, drawing Braille dots per pixel so
// the result does not depend on the installed fonts.

export interface Grid {
  cols: number
  rows: number
  bytes: Uint8Array
}

export function decode(cols: number, rows: number, b64: string): Grid {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  return { cols, rows, bytes }
}

export type DotStyle = 'dots' | 'pixels'

const BRAILLE = 0x2800

function bit(x: number, y: number): number {
  return y === 3 ? 0x40 << x : 1 << (y + 3 * x)
}

export interface Paint {
  cw: number
  ch: number
  style: DotStyle
}

// paint writes background and Braille dots into px and returns the text
// cells that still need a glyph drawn on top.
export function paint(g: Grid, img: ImageData, p: Paint): { x: number; y: number; ch: string; fg: string }[] {
  const out: { x: number; y: number; ch: string; fg: string }[] = []
  const px = new Uint32Array(img.data.buffer)
  const W = img.width
  const dw = p.cw / 2
  const dh = p.ch / 4
  const gap = p.style === 'dots' ? Math.max(1, Math.round(dw * 0.28)) : 0
  for (let cy = 0; cy < g.rows; cy++) {
    for (let cx = 0; cx < g.cols; cx++) {
      const o = (cy * g.cols + cx) * 10
      const b = g.bytes
      const rune = b[o] | (b[o + 1] << 8) | (b[o + 2] << 16) | (b[o + 3] << 24)
      const fg = 0xff000000 | (b[o + 6] << 16) | (b[o + 5] << 8) | b[o + 4]
      const bg = 0xff000000 | (b[o + 9] << 16) | (b[o + 8] << 8) | b[o + 7]
      const x0 = cx * p.cw
      const y0 = cy * p.ch
      for (let y = 0; y < p.ch; y++) px.fill(bg, (y0 + y) * W + x0, (y0 + y) * W + x0 + p.cw)
      if (rune > BRAILLE && rune <= BRAILLE + 0xff) {
        const mask = rune - BRAILLE
        for (let dy = 0; dy < 4; dy++) {
          for (let dx = 0; dx < 2; dx++) {
            if (!(mask & bit(dx, dy))) continue
            const sx = x0 + Math.round(dx * dw)
            const sy = y0 + Math.round(dy * dh)
            const ex = x0 + Math.round((dx + 1) * dw) - gap
            const ey = y0 + Math.round((dy + 1) * dh) - gap
            for (let y = sy; y < ey; y++) px.fill(fg, y * W + sx, y * W + ex)
          }
        }
      } else if (rune !== 32 && rune !== 0 && rune !== BRAILLE) {
        out.push({ x: cx, y: cy, ch: String.fromCodePoint(rune), fg: `rgb(${b[o + 4]},${b[o + 5]},${b[o + 6]})` })
      }
    }
  }
  return out
}

// render draws a grid onto a canvas at integer scale for crisp dots.
export function render(canvas: HTMLCanvasElement, g: Grid, p: Paint) {
  const scale = Math.max(1, Math.round(window.devicePixelRatio || 1))
  const cw = p.cw * scale
  const ch = p.ch * scale
  const w = g.cols * cw
  const h = g.rows * ch
  if (canvas.width !== w || canvas.height !== h) {
    canvas.width = w
    canvas.height = h
  }
  canvas.style.width = `${g.cols * p.cw}px`
  canvas.style.height = `${g.rows * p.ch}px`
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const img = ctx.createImageData(w, h)
  const text = paint(g, img, { ...p, cw, ch })
  ctx.putImageData(img, 0, 0)
  if (text.length === 0) return
  ctx.font = `${Math.round(ch * 0.78)}px ${getComputedStyle(document.body).getPropertyValue('--font-mono') || 'monospace'}`
  ctx.textBaseline = 'middle'
  ctx.textAlign = 'center'
  for (const t of text) {
    ctx.fillStyle = t.fg
    ctx.fillText(t.ch, t.x * cw + cw / 2, t.y * ch + ch / 2 + scale)
  }
}
