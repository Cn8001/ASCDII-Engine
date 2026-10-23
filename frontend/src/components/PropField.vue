<script setup lang="ts">
import { computed } from 'vue'
import { Assets } from '../lib/api'
import { assetPath, call, store } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import Icon from './Icon.vue'

const props = defineProps<{ p: prop.Prop; keyable?: boolean }>()
const emit = defineEmits<{ set: [name: string, value: unknown]; key: [name: string]; menu: [e: MouseEvent, name: string] }>()

const label = computed(() => {
  const n = props.p.name
  const i = n.lastIndexOf('.')
  return i >= 0 ? n.slice(i + 1) : n
})
const animatable = computed(() => props.keyable && ['float', 'int', 'vec3', 'color', 'bool'].includes(props.p.type))
const lo = computed(() => props.p.min ?? 0)
const hi = computed(() => props.p.max ?? 0)
const slider = computed(() => props.p.type === 'float' && hi.value > lo.value && hi.value - lo.value <= 10)
const vec = computed(() => (Array.isArray(props.p.value) ? (props.p.value as number[]) : [0, 0, 0]))
const refs = (kind: string) => store.assets.filter((a) => a.kind === kind).map((a) => ({ value: a.guid || a.path, label: a.path }))
const arts = computed(() => refs('art'))
const scripts = computed(() => refs('script'))

function num(v: unknown): number {
  return typeof v === 'number' ? v : Number(v) || 0
}

function fmt(v: unknown): string {
  const n = num(v)
  return Number.isInteger(n) ? String(n) : n.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')
}

function editScript() {
  const file = String(props.p.value ?? '')
  if (file) call(() => Assets.OpenExternal(file, 0))
}

function set(v: unknown) {
  emit('set', props.p.name, v)
}

function setAxis(i: number, v: string) {
  const out = [...vec.value]
  out[i] = Number(v) || 0
  set(out)
}

// scrub lets the label act as a horizontal drag handle, like most 3D tools.
function scrub(e: PointerEvent, axis = -1) {
  if (props.p.locked) return
  const startX = e.clientX
  const base = axis >= 0 ? vec.value[axis] : num(props.p.value)
  const step = props.p.type === 'int' ? 1 : hi.value > lo.value ? (hi.value - lo.value) / 300 : 0.05
  const target = e.target as HTMLElement
  target.setPointerCapture(e.pointerId)
  const move = (m: PointerEvent) => {
    const d = (m.clientX - startX) * step * (m.shiftKey ? 0.1 : 1)
    let v = base + d
    if (hi.value > lo.value) v = Math.min(hi.value, Math.max(lo.value, v))
    if (props.p.type === 'int') v = Math.round(v)
    if (axis >= 0) {
      const out = [...vec.value]
      out[axis] = v
      set(out)
    } else set(v)
  }
  const up = () => {
    target.removeEventListener('pointermove', move)
    target.removeEventListener('pointerup', up)
  }
  target.addEventListener('pointermove', move)
  target.addEventListener('pointerup', up)
}
</script>

<template>
  <div class="group grid min-h-8 grid-cols-[78px_minmax(0,1fr)_auto] items-center gap-2 py-0.5">
    <span
      class="flex items-center gap-1 truncate text-[12px]"
      :class="[p.overridden ? 'font-semibold text-amber-glow' : 'text-ink-400', ['float', 'int'].includes(p.type) && !p.locked ? 'cursor-ew-resize hover:text-ink-200' : '']"
      :title="p.overridden ? 'Overridden. Right click to apply or revert.' : ''"
      @pointerdown="['float', 'int'].includes(p.type) && $event.button === 0 ? scrub($event) : null"
      @contextmenu.prevent="emit('menu', $event, p.name)"
    >
      <Icon v-if="p.locked" name="lock" :size="11" class="text-ink-500" />
      {{ label }}
    </span>

    <div class="min-w-0">
      <div v-if="p.type === 'bool'" class="flex">
        <button class="relative h-4 w-7 rounded-full transition" :class="p.value ? 'bg-amber-glow' : 'bg-ink-700'" :disabled="p.locked" @click="set(!p.value)">
          <span class="absolute top-0.5 size-3 rounded-full bg-ink-950 transition-all" :class="p.value ? 'left-3.5' : 'left-0.5'" />
        </button>
      </div>

      <div v-else-if="slider" class="flex items-center gap-2">
        <input type="range" class="h-1 min-w-0 flex-1 accent-[var(--color-amber-glow)]" :min="lo" :max="hi" :step="(hi - lo) / 200" :value="num(p.value)" :disabled="p.locked" @input="set(Number(($event.target as HTMLInputElement).value))" />
        <input class="field w-12 shrink-0 px-1 text-right font-mono" :value="fmt(p.value)" :disabled="p.locked" @change="set(Number(($event.target as HTMLInputElement).value))" />
      </div>

      <input v-else-if="p.type === 'float' || p.type === 'int'" class="field font-mono" :value="fmt(p.value)" :disabled="p.locked" @change="set(Number(($event.target as HTMLInputElement).value))" />

      <div v-else-if="p.type === 'vec3'" class="grid grid-cols-3 gap-1">
        <label v-for="(a, i) in ['x', 'y', 'z']" :key="a" class="relative">
          <span class="absolute left-1 top-1.5 cursor-ew-resize font-mono text-[10px] text-ink-500 hover:text-amber-glow" @pointerdown="scrub($event, i)">{{ a }}</span>
          <input class="field pl-4 pr-1 font-mono text-[11px]" :value="fmt(vec[i])" :disabled="p.locked" @change="setAxis(i, ($event.target as HTMLInputElement).value)" />
        </label>
      </div>

      <div v-else-if="p.type === 'color'" class="flex items-center gap-2">
        <input type="color" class="size-7 shrink-0 cursor-pointer rounded-md" :value="String(p.value)" :disabled="p.locked" @input="set(($event.target as HTMLInputElement).value)" />
        <input class="field font-mono" :value="String(p.value)" :disabled="p.locked" @change="set(($event.target as HTMLInputElement).value)" />
      </div>

      <select v-else-if="p.type === 'enum'" class="field" :value="p.value" :disabled="p.locked" @change="set(($event.target as HTMLSelectElement).value)">
        <option v-for="o in p.options" :key="o" :value="o">{{ o }}</option>
      </select>

      <select v-else-if="p.type === 'asset'" class="field" :value="p.value" :disabled="p.locked" @change="set(($event.target as HTMLSelectElement).value)">
        <option value="">-</option>
        <option v-for="a in arts" :key="a.value" :value="a.value">{{ a.label }}</option>
        <option v-if="p.value && !arts.some((a) => a.value === p.value)" :value="p.value">{{ assetPath(String(p.value)) }}</option>
      </select>

      <div v-else-if="p.type === 'script'" class="flex items-center gap-1">
        <select class="field" :value="p.value" :disabled="p.locked" @change="set(($event.target as HTMLSelectElement).value)">
          <option value="">-</option>
          <option v-for="a in scripts" :key="a.value" :value="a.value">{{ a.label.replace('scripts/', '') }}</option>
        </select>
        <button class="icon-btn shrink-0" title="Edit" :disabled="!p.value" @click="editScript"><Icon name="file" :size="13" /></button>
      </div>

      <input v-else class="field" :value="String(p.value ?? '')" :disabled="p.locked" @change="set(($event.target as HTMLInputElement).value)" />
    </div>

    <button v-if="animatable" class="icon-btn size-6 text-ink-600 opacity-0 transition group-hover:opacity-100 hover:text-amber-glow" title="Key" @click="emit('key', p.name)">
      <Icon name="diamond" :size="10" />
    </button>
    <span v-else class="w-6" />
  </div>
</template>
