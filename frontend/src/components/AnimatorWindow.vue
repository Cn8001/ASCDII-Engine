<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Anim, Assets } from '../lib/api'
import { call, store, toast } from '../lib/store'
import { animator as ani } from '../wailsjs/go/models'
import Icon from './Icon.vue'

// AnimatorWindow edits an animator controller as a graph: states are boxes,
// transitions arrows. Shift drag from one state to another adds a transition.
type Pick = { kind: 'state'; i: number } | { kind: 'transition'; i: number } | null

const ctrl = ref<ani.Controller | null>(null)
const pick = ref<Pick>(null)
const live = ref('')
const clips = ref<string[]>([])
const link = ref<{ from: number; x: number; y: number } | null>(null)
const W = 120
const H = 36
const readOnly = computed(() => store.animator.startsWith('packages/'))
let timer = 0
let saveTimer = 0

async function load() {
  pick.value = null
  ctrl.value = store.animator ? ((await call(() => Assets.Controller(store.animator))) ?? null) : null
  const c = ctrl.value
  if (c) {
    c.parameters ??= []
    c.transitions ??= []
    c.states.forEach((s, i) => {
      if (!s.x && !s.y) {
        s.x = 40 + (i % 4) * 170
        s.y = 50 + Math.floor(i / 4) * 110
      }
    })
  }
  clips.value = [...new Set(((await call(() => Anim.Clips(0))) ?? []).map((c) => c.name))]
}

watch(() => store.animator, load, { immediate: true })

// While the game runs, the selected object's animator lights its state.
timer = window.setInterval(async () => {
  live.value = store.frame.mode !== 'edit' && store.selected ? ((await Anim.AnimatorState(store.selected)) ?? '') : ''
}, 250)
onBeforeUnmount(() => clearInterval(timer))

// save batches edits so dragging a box writes the file once it settles.
function save() {
  if (readOnly.value || !ctrl.value) return
  clearTimeout(saveTimer)
  saveTimer = window.setTimeout(async () => {
    if (ctrl.value) await call(() => Assets.SetController(store.animator, ctrl.value as ani.Controller))
  }, 250)
}

const states = computed(() => ctrl.value?.states ?? [])
const index = (name: string) => states.value.findIndex((s) => s.name === name)
const center = (i: number) => ({ x: (states.value[i]?.x ?? 0) + W / 2, y: (states.value[i]?.y ?? 0) + H / 2 })

const arrows = computed(() =>
  (ctrl.value?.transitions ?? []).map((t, i) => {
    const to = index(t.to)
    const from = t.from === 'any' ? -1 : index(t.from)
    const b = center(to)
    const a = from < 0 ? { x: b.x - 70, y: b.y - 60 } : center(from)
    // Offset each direction sideways so a pair of opposite arrows stays apart.
    const dx = b.x - a.x
    const dy = b.y - a.y
    const len = Math.hypot(dx, dy) || 1
    const ox = (-dy / len) * 6
    const oy = (dx / len) * 6
    // Stop at the target box's edge so the arrowhead shows.
    const k = Math.min(dx ? W / 2 / Math.abs(dx) : Infinity, dy ? H / 2 / Math.abs(dy) : Infinity)
    const end = { x: b.x - dx * k - (dx / len) * 3 + ox, y: b.y - dy * k - (dy / len) * 3 + oy }
    return { i, x1: a.x + ox, y1: a.y + oy, x2: end.x, y2: end.y, any: from < 0, ok: to >= 0 }
  }),
)

function down(e: PointerEvent, i: number) {
  pick.value = { kind: 'state', i }
  if (readOnly.value) return
  const s = states.value[i]
  const start = { x: e.clientX, y: e.clientY, sx: s.x ?? 0, sy: s.y ?? 0 }
  const el = e.currentTarget as SVGGElement
  el.setPointerCapture(e.pointerId)
  const shift = e.shiftKey
  const host = (el.closest('svg') as SVGSVGElement).getBoundingClientRect()
  const move = (m: PointerEvent) => {
    if (shift) {
      link.value = { from: i, x: m.clientX - host.left, y: m.clientY - host.top }
      return
    }
    s.x = Math.round(start.sx + m.clientX - start.x)
    s.y = Math.round(start.sy + m.clientY - start.y)
  }
  const up = (u: PointerEvent) => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', up)
    if (shift && link.value) {
      const x = u.clientX - host.left
      const y = u.clientY - host.top
      const to = states.value.findIndex((t) => x >= (t.x ?? 0) && x <= (t.x ?? 0) + W && y >= (t.y ?? 0) && y <= (t.y ?? 0) + H)
      if (to >= 0 && to !== i) {
        ctrl.value!.transitions.push(ani.Transition.createFrom({ from: s.name, to: states.value[to].name, when: [] }))
        pick.value = { kind: 'transition', i: ctrl.value!.transitions.length - 1 }
      }
      link.value = null
    }
    save()
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', up)
}

function addState() {
  const c = ctrl.value
  if (!c) return
  let n = c.states.length + 1
  while (c.states.some((s) => s.name === `state ${n}`)) n++
  c.states.push(ani.State.createFrom({ name: `state ${n}`, clip: clips.value[0] ?? '', speed: 1, x: 40 + (c.states.length % 4) * 170, y: 50 + Math.floor(c.states.length / 4) * 110 }))
  pick.value = { kind: 'state', i: c.states.length - 1 }
  save()
}

function addParam() {
  const c = ctrl.value
  if (!c) return
  c.parameters.push(ani.Param.createFrom({ name: `param${c.parameters.length + 1}`, type: 'float', value: 0 }))
  save()
}

function remove() {
  const c = ctrl.value
  const p = pick.value
  if (!c || !p) return
  if (p.kind === 'state') {
    const name = c.states[p.i].name
    c.states.splice(p.i, 1)
    c.transitions = c.transitions.filter((t) => t.from !== name && t.to !== name)
    if (c.start === name) c.start = c.states[0]?.name ?? ''
  } else {
    c.transitions.splice(p.i, 1)
  }
  pick.value = null
  save()
}

// rename keeps transitions and the start state pointing at the state.
function rename(i: number, name: string) {
  const c = ctrl.value
  if (!c || !name || c.states.some((s, j) => j !== i && s.name === name)) return
  const old = c.states[i].name
  c.states[i].name = name
  for (const t of c.transitions) {
    if (t.from === old) t.from = name
    if (t.to === old) t.to = name
  }
  if (c.start === old) c.start = name
  save()
}

function addCond(t: ani.Transition) {
  const p = ctrl.value?.parameters[0]
  t.when = [...(t.when ?? []), ani.Cond.createFrom({ param: p?.name ?? '', op: p?.type === 'float' ? '>' : '', value: 0 })]
  save()
}

const picked = computed(() => {
  const p = pick.value
  const c = ctrl.value
  if (!p || !c) return null
  return p.kind === 'state' ? { state: c.states[p.i] } : { transition: c.transitions[p.i] }
})

function close() {
  store.animator = ''
  toast('Animator closed', 'info')
}
</script>

<template>
  <div class="flex h-full text-[12px]">
    <div class="flex w-44 shrink-0 flex-col border-r border-ink-800">
      <div class="flex h-7 items-center justify-between px-2 text-[10.5px] font-semibold uppercase tracking-wider text-ink-500">
        Parameters <button v-if="!readOnly" class="icon-btn size-5" title="Add parameter" @click="addParam"><Icon name="plus" :size="11" /></button>
      </div>
      <div class="flex-1 overflow-auto px-1.5">
        <div v-for="(p, i) in ctrl?.parameters ?? []" :key="i" class="mb-1 flex items-center gap-1">
          <input class="field h-6 min-w-0 flex-1 px-1.5 text-[11.5px]" :value="p.name" :disabled="readOnly" @change="p.name = ($event.target as HTMLInputElement).value; save()" />
          <select class="h-6 rounded bg-ink-800 text-[10.5px] text-ink-300 outline-none" :value="p.type" :disabled="readOnly" @change="p.type = ($event.target as HTMLSelectElement).value; save()">
            <option>float</option><option>bool</option><option>trigger</option>
          </select>
          <button v-if="!readOnly" class="icon-btn size-5" @click="ctrl!.parameters.splice(i, 1); save()"><Icon name="x" :size="10" /></button>
        </div>
      </div>
      <div class="border-t border-ink-800 p-2 text-[10.5px] leading-relaxed text-ink-500">Drag boxes to arrange. <b class="text-ink-400">Shift</b> drag from a state to another to add a transition.</div>
    </div>

    <div class="relative min-w-0 flex-1 overflow-auto bg-ink-950/60">
      <div class="absolute left-2 top-2 z-10 flex gap-1">
        <button v-if="!readOnly" class="btn h-6" @click="addState"><Icon name="plus" :size="11" /> State</button>
        <button v-if="pick && !readOnly" class="btn h-6 text-bad" @click="remove"><Icon name="trash" :size="11" /> Delete</button>
        <span v-if="readOnly" class="flex items-center gap-1 text-[11px] text-ink-500"><Icon name="lock" :size="11" /> read only</span>
      </div>
      <svg class="h-[600px] w-[1000px]" @pointerdown.self="pick = null">
        <defs>
          <marker id="head" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill="var(--color-ink-300)" /></marker>
        </defs>
        <g v-for="a in arrows" :key="'t' + a.i" class="cursor-pointer" @pointerdown.stop="pick = { kind: 'transition', i: a.i }">
          <line :x1="a.x1" :y1="a.y1" :x2="a.x2" :y2="a.y2" stroke="transparent" stroke-width="10" />
          <line v-if="a.ok" :x1="a.x1" :y1="a.y1" :x2="a.x2" :y2="a.y2" :stroke="pick?.kind === 'transition' && pick.i === a.i ? 'var(--color-amber-glow)' : 'var(--color-ink-300)'" stroke-width="1.5" :stroke-dasharray="a.any ? '4 3' : ''" marker-end="url(#head)" />
        </g>
        <line v-if="link" :x1="center(link.from).x" :y1="center(link.from).y" :x2="link.x" :y2="link.y" stroke="var(--color-amber-glow)" stroke-dasharray="4 3" />
        <g v-for="(s, i) in states" :key="s.name + i" class="cursor-grab" @pointerdown.stop="down($event, i)">
          <rect :x="s.x" :y="s.y" :width="W" :height="H" rx="8"
            :fill="live === s.name ? 'color-mix(in srgb, var(--color-amber-glow) 30%, var(--color-ink-850))' : 'var(--color-ink-850)'"
            :stroke="pick?.kind === 'state' && pick.i === i ? 'var(--color-amber-glow)' : ctrl?.start === s.name ? 'var(--color-amber-soft)' : 'var(--color-ink-600)'"
            :stroke-width="ctrl?.start === s.name ? 2 : 1" />
          <text :x="(s.x ?? 0) + W / 2" :y="(s.y ?? 0) + 15" text-anchor="middle" fill="var(--color-ink-100)" font-size="12">{{ s.name }}</text>
          <text :x="(s.x ?? 0) + W / 2" :y="(s.y ?? 0) + 28" text-anchor="middle" fill="var(--color-ink-500)" font-size="10" font-family="var(--font-mono)">{{ s.clip || 'no clip' }}</text>
        </g>
      </svg>
      <div v-if="!ctrl" class="absolute inset-0 grid place-items-center text-ink-500">Open an animator controller from the Project window</div>
    </div>

    <div class="w-56 shrink-0 overflow-auto border-l border-ink-800 p-2.5">
      <template v-if="picked?.state">
        <div class="mb-2 text-[10.5px] font-semibold uppercase tracking-wider text-ink-500">State</div>
        <label class="mb-1.5 block text-ink-400">Name<input class="field mt-0.5" :value="picked.state.name" :disabled="readOnly" @change="rename(pick!.i, ($event.target as HTMLInputElement).value)" /></label>
        <label class="mb-1.5 block text-ink-400">Clip
          <input class="field mt-0.5" list="clip-names" :value="picked.state.clip" :disabled="readOnly" @change="picked.state.clip = ($event.target as HTMLInputElement).value; save()" />
        </label>
        <label class="mb-2 block text-ink-400">Speed<input type="number" step="0.1" class="field mt-0.5" :value="picked.state.speed ?? 1" :disabled="readOnly" @change="picked.state.speed = Number(($event.target as HTMLInputElement).value); save()" /></label>
        <button v-if="ctrl?.start !== picked.state.name && !readOnly" class="btn h-7 w-full justify-center" @click="ctrl!.start = picked.state.name; save()">Make start state</button>
        <div v-else class="text-[11px] text-amber-soft">Start state</div>
      </template>
      <template v-else-if="picked?.transition">
        <div class="mb-2 text-[10.5px] font-semibold uppercase tracking-wider text-ink-500">Transition</div>
        <label class="mb-1.5 block text-ink-400">From
          <select class="field mt-0.5" :value="picked.transition.from" :disabled="readOnly" @change="picked.transition.from = ($event.target as HTMLSelectElement).value; save()">
            <option value="any">any state</option>
            <option v-for="s in states" :key="s.name" :value="s.name">{{ s.name }}</option>
          </select>
        </label>
        <div class="mb-2 text-ink-400">To <span class="text-ink-200">{{ picked.transition.to }}</span></div>
        <label class="mb-2 flex items-center gap-2 text-ink-300"><input type="checkbox" class="accent-[var(--color-amber-glow)]" :checked="!!picked.transition.exit" :disabled="readOnly" @change="picked.transition.exit = ($event.target as HTMLInputElement).checked; save()" /> After the clip ends</label>
        <div class="mb-1 flex items-center justify-between text-ink-400">When <button v-if="!readOnly" class="icon-btn size-5" @click="addCond(picked.transition)"><Icon name="plus" :size="11" /></button></div>
        <div v-for="(w, k) in picked.transition.when ?? []" :key="k" class="mb-1 flex items-center gap-1">
          <select class="h-6 min-w-0 flex-1 rounded bg-ink-800 text-[11px] outline-none" :value="w.param" :disabled="readOnly" @change="w.param = ($event.target as HTMLSelectElement).value; save()">
            <option v-for="p in ctrl?.parameters ?? []" :key="p.name" :value="p.name">{{ p.name }}</option>
          </select>
          <select class="h-6 w-11 rounded bg-ink-800 text-[11px] outline-none" :value="w.op ?? ''" :disabled="readOnly" @change="w.op = ($event.target as HTMLSelectElement).value; save()">
            <option value="">set</option><option>&gt;</option><option>&lt;</option><option>==</option><option>!=</option>
          </select>
          <input v-if="w.op" type="number" step="0.1" class="field h-6 w-12 px-1" :value="w.value ?? 0" :disabled="readOnly" @change="w.value = Number(($event.target as HTMLInputElement).value); save()" />
          <button v-if="!readOnly" class="icon-btn size-5" @click="picked.transition.when!.splice(k, 1); save()"><Icon name="x" :size="10" /></button>
        </div>
        <div v-if="!(picked.transition.when ?? []).length && !picked.transition.exit" class="text-[11px] text-bad">Needs a condition or After the clip ends</div>
      </template>
      <div v-else class="text-[11.5px] leading-relaxed text-ink-500">
        <div class="mb-1 truncate font-mono text-ink-400">{{ store.animator }}</div>
        Pick a state or an arrow. The start state has an amber border; while playing, the selected object's active state glows.
        <button class="btn mt-3 h-7 w-full justify-center" @click="close">Close</button>
      </div>
    </div>
    <datalist id="clip-names"><option v-for="c in clips" :key="c" :value="c" /></datalist>
  </div>
</template>
