<script setup lang="ts">
import { computed, ref } from 'vue'
import { Anim, Viewport } from '../lib/api'
import { call, findNode, refreshClips, refreshProps, store, toast } from '../lib/store'
import Icon from './Icon.vue'

defineProps<{ floating?: boolean }>()
const emit = defineEmits<{ float: [] }>()

const lane = ref<HTMLDivElement | null>(null)
const eases = ['linear', 'in', 'out', 'inout', 'step', 'back', 'elastic']
const span = computed(() => Math.max(4, ...store.clips.map((c) => c.length + 1)))
const live = computed(() => store.info?.live ?? false)
const local = computed(() => {
  const t = store.frame.time
  return live.value ? t % span.value : Math.min(t, span.value)
})
const ticks = computed(() => {
  const step = span.value > 12 ? 2 : span.value > 6 ? 1 : 0.5
  return Array.from({ length: Math.floor(span.value / step) + 1 }, (_, i) => i * step)
})
const selectedKey = computed(() => {
  const k = store.key
  if (!k) return null
  const c = store.clips.find((x) => x.index === k.clip)
  const tr = c?.tracks.find((t) => t.prop === k.prop)
  return tr?.keys.find((x) => Math.abs(x.t - k.t) < 1e-3) ?? null
})
const target = computed(() => (store.selected ? findNode(store.selected)?.name : 'Scene'))
const drag = ref<{ clip: number; prop: string; from: number; to: number } | null>(null)

function timeAt(x: number) {
  const r = lane.value!.getBoundingClientRect()
  return Math.max(0, Math.min(1, (x - r.left) / r.width)) * span.value
}

function snap(t: number) {
  return Math.round(t * 20) / 20
}

async function setLive(on: boolean) {
  Viewport.SetLive(on)
  if (store.info) store.info.live = on
}

async function scrub(e: PointerEvent) {
  await setLive(false)
  await call(() => Viewport.SetTime(timeAt(e.clientX)))
  const el = e.currentTarget as HTMLElement
  el.setPointerCapture(e.pointerId)
  const move = (m: PointerEvent) => Viewport.SetTime(timeAt(m.clientX))
  const up = async () => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', up)
    await refreshProps()
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', up)
}

function grab(e: PointerEvent, clip: number, prop: string, t: number) {
  e.stopPropagation()
  store.key = { clip, prop, t }
  drag.value = { clip, prop, from: t, to: t }
  const el = e.currentTarget as HTMLElement
  el.setPointerCapture(e.pointerId)
  const move = (m: PointerEvent) => {
    if (drag.value) drag.value.to = snap(timeAt(m.clientX))
  }
  const up = async () => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', up)
    const d = drag.value
    drag.value = null
    if (d && Math.abs(d.to - d.from) > 1e-3) {
      await call(() => Anim.MoveKey(d.clip, d.prop, d.from, d.to))
      store.key = { clip: d.clip, prop: d.prop, t: d.to }
      await refreshClips()
    }
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', up)
}

function keyX(clip: number, prop: string, t: number) {
  const d = drag.value
  const at = d && d.clip === clip && d.prop === prop && Math.abs(d.from - t) < 1e-3 ? d.to : t
  return (at / span.value) * 100 + '%'
}

async function addAt(e: MouseEvent, prop: string) {
  if (!store.selected) return
  const t = snap(timeAt(e.clientX))
  await setLive(false)
  await call(() => Viewport.SetTime(t))
  await call(() => Anim.AddKey(store.selected, prop, t))
  await refreshClips()
}

async function setEase(ease: string) {
  const k = store.key
  if (!k) return
  await call(() => Anim.SetEase(k.clip, k.prop, k.t, ease))
  await refreshClips()
}

async function removeKey() {
  const k = store.key
  if (!k) return
  await call(() => Anim.RemoveKey(k.clip, k.prop, k.t))
  store.key = null
  await refreshClips()
}

async function mode(index: number, m: string) {
  const c = store.clips.find((x) => x.index === index)
  await call(() => Anim.SetClip(index, m, c?.speed || 1))
  await refreshClips()
}

async function removeClip(index: number) {
  await call(() => Anim.RemoveClip(index))
  await refreshClips()
}

function record() {
  store.autoKey = !store.autoKey
  if (store.autoKey) toast('Record: changes become keys', 'info')
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex h-8 shrink-0 items-center gap-1 border-b border-ink-800 px-2">
      <button class="icon-btn size-6" title="Start" @click="setLive(false); Viewport.SetTime(0)"><Icon name="undo" :size="12" /></button>
      <button class="icon-btn size-6" :class="live ? 'text-amber-glow' : ''" title="Preview" @click="setLive(!live)"><Icon :name="live ? 'pause' : 'play'" :size="12" /></button>
      <button class="flex h-6 items-center gap-1.5 rounded-md px-2 text-[11px] transition" :class="store.autoKey ? 'bg-bad/15 text-bad' : 'text-ink-400 hover:text-ink-200'" title="Record" @click="record">
        <span class="size-2 rounded-full" :class="store.autoKey ? 'animate-pulse bg-bad' : 'bg-ink-500'" /> REC
      </button>
      <span class="ml-2 font-mono text-[11px] text-amber-glow">{{ local.toFixed(2) }}s</span>
      <span class="ml-2 truncate text-[11px] text-ink-500">{{ target }}</span>
      <span class="flex-1" />
      <template v-if="selectedKey && store.key">
        <span class="font-mono text-[10.5px] text-ink-400">{{ store.key.prop }} @ {{ store.key.t.toFixed(2) }}</span>
        <select class="h-6 rounded-md bg-ink-800 px-1.5 text-[11px] text-ink-200 outline-none" :value="selectedKey.ease || 'linear'" @change="setEase(($event.target as HTMLSelectElement).value)">
          <option v-for="e in eases" :key="e" :value="e">{{ e }}</option>
        </select>
        <button class="icon-btn size-6 hover:text-bad" title="Delete key" @click="removeKey"><Icon name="trash" :size="12" /></button>
      </template>
      <button v-if="!floating" class="icon-btn size-6" title="Float" @click="emit('float')"><Icon name="copy" :size="12" /></button>
    </div>

    <div class="relative min-h-0 flex-1 overflow-auto">
      <div class="grid grid-cols-[180px_1fr]">
        <div class="sticky top-0 z-10 h-6 border-b border-ink-800 bg-ink-900" />
        <div ref="lane" class="sticky top-0 z-10 h-6 cursor-ew-resize border-b border-ink-800 bg-ink-900" @pointerdown="scrub">
          <span v-for="t in ticks" :key="t" class="absolute top-1 -translate-x-1/2 font-mono text-[9.5px] text-ink-500" :style="{ left: (t / span) * 100 + '%' }">{{ t }}</span>
        </div>
        <template v-for="c in store.clips" :key="c.index">
          <div class="flex h-7 items-center gap-1 border-b border-ink-800/60 bg-ink-850/60 px-2 text-[11.5px] text-ink-200">
            <Icon name="film" :size="11" class="text-ink-500" />
            <span class="flex-1 truncate">{{ c.name }}</span>
            <select class="h-5 rounded bg-ink-800 px-1 text-[10.5px] text-ink-300 outline-none" :value="c.mode" @change="mode(c.index, ($event.target as HTMLSelectElement).value)">
              <option v-for="m in ['once', 'loop', 'pingpong']" :key="m" :value="m">{{ m }}</option>
            </select>
            <button class="icon-btn size-5" @click="removeClip(c.index)"><Icon name="x" :size="11" /></button>
          </div>
          <div class="h-7 border-b border-ink-800/60 bg-ink-850/60" />
          <template v-for="tr in c.tracks" :key="c.index + tr.prop">
            <div class="flex h-7 items-center pl-6 pr-2 font-mono text-[10.5px] text-ink-400">{{ tr.prop }}</div>
            <div class="relative h-7 border-b border-ink-800/40" @dblclick="addAt($event, tr.prop)">
              <div class="absolute top-1/2 h-px -translate-y-1/2 bg-ink-700" :style="{ left: (tr.keys[0]?.t / span) * 100 + '%', width: ((tr.keys[tr.keys.length - 1]?.t - tr.keys[0]?.t) / span) * 100 + '%' }" />
              <button
                v-for="k in tr.keys"
                :key="k.t"
                class="absolute top-1/2 size-3 -translate-x-1/2 -translate-y-1/2 rotate-45 cursor-grab rounded-[2px] shadow shadow-black transition-transform hover:scale-125 active:cursor-grabbing"
                :class="store.key && store.key.clip === c.index && store.key.prop === tr.prop && Math.abs(store.key.t - k.t) < 1e-3 ? 'bg-ink-100 ring-2 ring-amber-glow' : 'bg-amber-glow'"
                :style="{ left: keyX(c.index, tr.prop, k.t) }"
                :title="`${k.t.toFixed(2)}s  ${JSON.stringify(k.v)}  ${k.ease || 'linear'}`"
                @pointerdown="grab($event, c.index, tr.prop, k.t)"
              />
            </div>
          </template>
        </template>
      </div>
      <div v-if="!store.clips.length" class="grid h-[calc(100%-24px)] place-items-center px-6 text-center text-[12px] text-ink-500">
        <div class="flex flex-col items-center gap-2">
          <Icon name="diamond" :size="14" class="text-ink-600" />
          <span>Select an object, press REC, change a value.</span>
        </div>
      </div>
      <div class="pointer-events-none absolute bottom-0 top-0 w-px bg-amber-glow" :style="{ left: `calc(180px + (100% - 180px) * ${local / span})` }" />
    </div>
  </div>
</template>
