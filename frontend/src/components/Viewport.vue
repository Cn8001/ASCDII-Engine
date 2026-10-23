<script setup lang="ts">
import { assetKind, type FileDrop } from '../lib/filedrop'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Assets, EventsOn, Viewport } from '../lib/api'
import { decodeBytes, Painter } from '../lib/cells'
import { engineKey } from '../lib/keys'
import { call, findNode, refresh, refreshProps, select, setFrame, store } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'

const host = ref<HTMLDivElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const cw = computed(() => store.cellPx)
const ch = computed(() => store.cellPx * 2)
let painter: Painter | null = null
let last: { cols: number; rows: number; bytes: Uint8Array } | null = null
let fetching = false
let wanted = 0
let off: (() => void) | null = null
const paintMs = ref(0)
let observer: ResizeObserver | null = null

function paint(cols: number, rows: number, bytes: Uint8Array) {
  if (!canvas.value) return
  painter ??= new Painter(canvas.value)
  last = { cols, rows, bytes }
  painter.draw(cols, rows, bytes, { cw: cw.value, ch: ch.value, style: store.dots })
  paintMs.value = painter.millis
}

// pull fetches the newest raw frame; frames that arrive while a fetch is in
// flight are coalesced so the UI never queues behind the renderer.
async function pull() {
  if (fetching) return
  fetching = true
  try {
    while (wanted) {
      wanted = 0
      const res = await fetch('/frame', { cache: 'no-store' })
      const buf = new Uint8Array(await res.arrayBuffer())
      if (buf.length < 12) continue
      const head = new DataView(buf.buffer)
      paint(head.getUint32(0, true), head.getUint32(4, true), buf.subarray(12))
    }
  } catch {
    // a missed frame is replaced by the next one
  } finally {
    fetching = false
  }
}

function draw(f: session.Frame) {
  setFrame(f)
  if (f.cells) {
    paint(f.cols, f.rows, decodeBytes(f.cells))
    return
  }
  wanted = f.seq
  pull()
}

function fit() {
  const el = host.value
  if (!el) return
  const cols = Math.max(10, Math.floor(el.clientWidth / cw.value))
  const rows = Math.max(6, Math.floor((el.clientHeight - 32) / ch.value))
  Viewport.Resize(cols, rows)
}

watch(() => [store.cellPx, store.dots], () => {
  fit()
  painter?.reset()
  if (last) paint(last.cols, last.rows, last.bytes)
})

onMounted(async () => {
  off = EventsOn('frame', draw)
  observer = new ResizeObserver(fit)
  if (host.value) observer.observe(host.value)
  fit()
  const f = await call(() => Viewport.Snapshot())
  if (f) draw(f)
})

onBeforeUnmount(() => {
  off?.()
  observer?.disconnect()
})

const edges = computed(() => {
  const g = store.frame.gizmo
  if (!g?.corners || g.corners.length !== 8) return []
  const out: number[][] = []
  for (let a = 0; a < 8; a++) {
    for (const b of [1, 2, 4]) {
      if (!(a & b)) out.push([...g.corners[a], ...g.corners[a | b]].map((v, i) => v * (i % 2 ? ch.value : cw.value)))
    }
  }
  return out
})

const anchor = computed(() => {
  const g = store.frame.gizmo
  return g ? { x: g.anchor[0] * cw.value, y: g.anchor[1] * ch.value } : null
})

const axisColor = ['#d06a52', '#8fb573', '#e8b04b']
const axes = computed(() => {
  const g = store.frame.gizmo
  const a = anchor.value
  if (!g?.axes || !a) return []
  return g.axes.map((p, i) => {
    const x = p[0] * cw.value
    const y = p[1] * ch.value
    const len = Math.hypot(x - a.x, y - a.y) || 1
    const ux = (x - a.x) / len
    const uy = (y - a.y) / len
    const tip = { x: a.x + ux * Math.max(len, 34), y: a.y + uy * Math.max(len, 34) }
    return { name: ['x', 'y', 'z'][i], color: axisColor[i], tip, head: `${tip.x + ux * 8},${tip.y + uy * 8} ${tip.x - uy * 4},${tip.y + ux * 4} ${tip.x + uy * 4},${tip.y - ux * 4}` }
  })
})
const hover = ref('')

const selectedName = computed(() => findNode(store.selected)?.name ?? '')
const playing = computed(() => store.frame.mode !== 'edit')

type DragMode = 'none' | 'pan' | 'orbit' | 'move' | 'paint'
let mode: DragMode = 'none'
let axis = 'free'
let prev = { x: 0, y: 0 }
let moved = 0
let pending: (() => void) | null = null

function local(e: PointerEvent | MouseEvent) {
  const r = canvas.value!.getBoundingClientRect()
  return { x: e.clientX - r.left, y: e.clientY - r.top }
}

function queue(fn: () => void) {
  if (!pending) requestAnimationFrame(() => { pending?.(); pending = null })
  pending = fn
}

// paintAt sends one brush dab; a slow backend drops dabs instead of queueing them.
let painting = false
async function paintAt(p: { x: number; y: number }) {
  if (painting || !store.selected) return
  painting = true
  const b = store.brush
  await Viewport.Paint(store.selected, p.x / cw.value, p.y / ch.value, b.color, b.radius, b.erase).catch(() => {})
  painting = false
}

function down(e: PointerEvent) {
  host.value?.focus()
  const p = local(e)
  prev = p
  moved = 0
  if (store.brush.on && e.button === 0 && store.selected && store.view === 'scene') {
    mode = 'paint'
    void paintAt(p)
    ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
    return
  }
  const hit = hitAxis(p)
  if (e.button === 2) mode = 'orbit'
  else if (e.button === 1) mode = 'pan'
  else if (hit && store.view === 'scene') {
    mode = 'move'
    axis = hit
  } else mode = 'pan'
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
}

// hitAxis returns which gizmo handle is under p: an arrow tip or the free centre.
function hitAxis(p: { x: number; y: number }): string {
  const a = anchor.value
  if (!a) return ''
  for (const ax of axes.value) {
    const dx = ax.tip.x - a.x
    const dy = ax.tip.y - a.y
    const t = Math.max(0, Math.min(1.25, ((p.x - a.x) * dx + (p.y - a.y) * dy) / (dx * dx + dy * dy)))
    if (t > 0.3 && Math.hypot(a.x + dx * t - p.x, a.y + dy * t - p.y) < 6) return ax.name
  }
  return Math.hypot(p.x - a.x, p.y - a.y) < 10 ? 'free' : ''
}

function move(e: PointerEvent) {
  if (mode === 'none') {
    hover.value = hitAxis(local(e))
    return
  }
  const p = local(e)
  const dx = p.x - prev.x
  const dy = p.y - prev.y
  moved += Math.abs(dx) + Math.abs(dy)
  if (mode === 'pan' && moved > 3) {
    const from = prev
    queue(() => Viewport.Pan((p.x - from.x) / cw.value, (p.y - from.y) / ch.value))
  } else if (mode === 'orbit') {
    queue(() => Viewport.Orbit(dx * 0.4, -dy * 0.25))
  } else if (mode === 'paint') {
    void paintAt(p)
  } else if (mode === 'move' && store.selected) {
    const from = prev
    const id = store.selected
    const ax = axis
    queue(() => Viewport.Drag(id, ax, from.x / cw.value, from.y / ch.value, p.x / cw.value, p.y / ch.value))
  }
  prev = p
}

async function up(e: PointerEvent) {
  const was = mode
  mode = 'none'
  if (was === 'pan' && moved <= 3 && e.button === 0) {
    const p = local(e)
    const id = await call(() => Viewport.Pick(Math.floor(p.x / cw.value), Math.floor(p.y / ch.value)))
    await select(id ?? 0)
  }
  if (was === 'move' || was === 'paint') await refreshProps()
}

function wheel(e: WheelEvent) {
  Viewport.Zoom(e.deltaY > 0 ? 1.08 : 1 / 1.08)
}

function key(e: KeyboardEvent) {
  if (!playing.value) return
  if (e.key === 'Escape') {
    Viewport.Stop()
    return
  }
  const k = engineKey(e)
  if (k) {
    e.preventDefault()
    e.stopPropagation()
    Viewport.Key(k)
  }
}

function keyUp(e: KeyboardEvent) {
  const k = playing.value ? engineKey(e) : null
  if (k) Viewport.KeyUp(k)
}

async function drop(e: DragEvent) {
  const data = e.dataTransfer?.getData('text/asset')
  if (!data) return
  const [kind, ...rest] = data.split(':')
  const path = rest.join(':')
  if (kind === 'material') {
    const p = local(e)
    const id = await call(() => Assets.MaterialAt(path, Math.floor(p.x / cw.value), Math.floor(p.y / ch.value)))
    if (id !== undefined) {
      await refresh()
      await select(id)
    }
    return
  }
  if (kind !== 'prefab' && kind !== 'art' && kind !== 'model') return
  await place(`${kind}:${path}`, local(e))
}

async function place(kind: string, p: { x: number; y: number }) {
  const id = await call(() => Viewport.Drop(kind, p.x / cw.value, p.y / ch.value))
  if (id !== undefined) {
    await refresh()
    await select(id)
  }
}

// fileDrop places files dropped from the OS where they land.
async function fileDrop(e: Event) {
  const d = (e as CustomEvent<FileDrop>).detail
  for (const p of d.paths) {
    const kind = assetKind(p)
    if (kind === 'model' || kind === 'prefab' || kind === 'art') await place(`${kind}:${p}`, local({ clientX: d.x, clientY: d.y } as MouseEvent))
  }
}

function setView(v: 'scene' | 'game') {
  store.view = v
  Viewport.SetView(v)
}

watch(playing, (on) => setView(on ? 'game' : 'scene'))

function setLive() {
  const on = !(store.info?.live ?? true)
  Viewport.SetLive(on)
  if (store.info) store.info.live = on
}
</script>

<template>
  <div
    ref="host"
    tabindex="0"
    class="relative h-full w-full overflow-hidden bg-ink-950 outline-none transition-shadow duration-300"
    :class="playing ? 'shadow-[inset_0_0_0_1px_var(--color-amber-soft)]' : ''"
    @keydown="key"
    @keyup="keyUp"
    @contextmenu.prevent
  >
    <canvas
      ref="canvas"
      class="absolute left-0 top-8 [image-rendering:pixelated]"
      :class="hover ? 'cursor-grab' : 'cursor-crosshair'"
      @pointerdown="down"
      @pointermove="move"
      @pointerup="up"
      @wheel.prevent="wheel"
      data-drop="viewport"
      @dragover.prevent
      @drop.prevent="drop"
      @filedrop="fileDrop"
    />
    <svg v-if="store.view === 'scene' && edges.length" class="pointer-events-none absolute left-0 top-8 h-full w-full overflow-visible">
      <line v-for="(l, i) in edges" :key="i" :x1="l[0]" :y1="l[1]" :x2="l[2]" :y2="l[3]" stroke="var(--color-amber-glow)" stroke-opacity=".45" stroke-width="1" stroke-dasharray="3 3" />
      <g v-for="ax in axes" :key="ax.name">
        <line :x1="anchor!.x" :y1="anchor!.y" :x2="ax.tip.x" :y2="ax.tip.y" :stroke="ax.color" :stroke-width="hover === ax.name ? 3 : 2" />
        <polygon :points="ax.head" :fill="ax.color" />
      </g>
      <g v-if="anchor" :transform="`translate(${anchor.x} ${anchor.y})`">
        <rect x="-5" y="-5" width="10" height="10" rx="2" fill="var(--color-ink-950)" fill-opacity=".7" :stroke="hover === 'free' ? '#fff' : 'var(--color-ink-200)'" stroke-width="1.5" />
        <text x="14" y="-10" fill="var(--color-ink-100)" font-size="11" font-family="var(--font-mono)">{{ selectedName }}</text>
      </g>
    </svg>

    <div class="absolute inset-x-0 top-0 flex h-8 items-center gap-1 border-b border-ink-750 bg-ink-900 px-2">
      <button v-for="v in ['scene', 'game'] as const" :key="v" class="relative flex h-8 items-center gap-1.5 px-2.5 text-[12px] capitalize transition" :class="store.view === v ? 'text-ink-100 after:absolute after:inset-x-2 after:-bottom-px after:h-px after:bg-amber-glow' : 'text-ink-500 hover:text-ink-200'" @click="setView(v)">
        <Icon :name="v === 'scene' ? 'grid' : 'play'" :size="12" /> {{ v }}
      </button>
      <span class="flex-1" />
      <button v-if="store.view === 'scene'" class="btn h-6" title="Align game camera with view" @click="Viewport.AlignGame()"><Icon name="crosshair" :size="12" /> Align</button>
    </div>

    <div class="absolute left-2 top-10 flex items-center gap-1 rounded-lg border border-ink-750 bg-ink-900/85 p-1 backdrop-blur">
      <button class="icon-btn" :class="store.dots === 'dots' ? 'text-amber-glow' : ''" title="Braille dots" @click="store.dots = 'dots'"><Icon name="grid" :size="14" /></button>
      <button class="icon-btn" :class="store.dots === 'pixels' ? 'text-amber-glow' : ''" title="Pixels" @click="store.dots = 'pixels'"><Icon name="image" :size="14" /></button>
      <span class="mx-1 h-4 w-px bg-ink-700" />
      <button v-for="px in [6, 8, 10, 12]" :key="px" class="h-7 rounded-md px-1.5 font-mono text-[11px] transition" :class="store.cellPx === px ? 'bg-ink-750 text-amber-glow' : 'text-ink-400 hover:text-ink-100'" @click="store.cellPx = px">{{ px }}</button>
      <span class="mx-1 h-4 w-px bg-ink-700" />
      <button class="icon-btn" :class="store.info?.live ? 'text-amber-glow' : ''" title="Live" @click="setLive"><Icon name="sun" :size="14" /></button>
      <span class="mx-1 h-4 w-px bg-ink-700" />
      <button class="icon-btn" :class="store.brush.on ? 'text-amber-glow' : ''" title="Braille brush: paint or erase the selected object's dots" @click="store.brush.on = !store.brush.on"><Icon name="brush" :size="14" /></button>
      <template v-if="store.brush.on">
        <input v-model="store.brush.color" type="color" class="size-6 cursor-pointer rounded" title="Color" />
        <input v-model.number="store.brush.radius" type="range" min="0.1" max="3" step="0.05" class="h-1 w-20 accent-[var(--color-amber-glow)]" title="Size" />
        <button class="h-7 rounded-md px-1.5 text-[11px] transition" :class="store.brush.erase ? 'bg-ink-750 text-amber-glow' : 'text-ink-400 hover:text-ink-100'" title="Erase dots" @click="store.brush.erase = !store.brush.erase">erase</button>
        <span v-if="!store.selected" class="px-1 text-[11px] text-ink-500">select an object</span>
      </template>
    </div>

    <div class="pointer-events-none absolute right-2 top-10 rounded-md border border-ink-750 bg-ink-900/85 px-2 py-1 font-mono text-[10.5px] text-ink-400 backdrop-blur">
      {{ store.frame.cols }}x{{ store.frame.rows }} · render {{ store.frame.millis.toFixed(1) }}ms · paint {{ paintMs.toFixed(1) }}ms · {{ Math.round(store.frame.fps) }}fps
    </div>

    <Transition enter-from-class="opacity-0 -translate-y-2" leave-to-class="opacity-0" enter-active-class="transition" leave-active-class="transition">
      <div v-if="playing" class="pointer-events-none absolute left-1/2 top-10 -translate-x-1/2 rounded-full bg-amber-glow px-3 py-0.5 font-mono text-[10.5px] font-semibold tracking-widest text-ink-950">
        {{ store.frame.mode === 'paused' ? 'PAUSED' : 'PLAY' }}
      </div>
    </Transition>
  </div>
</template>
