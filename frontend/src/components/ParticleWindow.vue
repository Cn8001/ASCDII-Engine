<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Scene, Viewport } from '../lib/api'
import { call, findNode, refreshProps, store, toast } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import Icon from './Icon.vue'
import PropField from './PropField.vue'

// ParticleWindow edits the selected object's particle system: presets, a
// preview of its colors and size over a particle's life, and the settings
// grouped the way an effect is usually built up.
const sections = [
  { title: 'Emission', names: ['active', 'duration', 'loop', 'prewarm', 'rate', 'burst', 'max'] },
  { title: 'Shape', names: ['shape', 'radius', 'box', 'dir', 'spread'] },
  { title: 'Particles', names: ['life', 'speed', 'size', 'sizeEnd', 'start', 'end', 'emissive', 'glow'] },
  { title: 'Forces', names: ['gravity', 'drag', 'wind', 'turbulence', 'collide', 'bounce'] },
  { title: 'Render', names: ['space', 'render', 'seed'] },
]

const presets = ref<string[]>([])
const count = ref(0)
let timer = 0

// key is the first particle system on the selection, "emitter" or "emitter#2".
const key = computed(() => store.props.find((p) => p.group?.split('#')[0] === 'emitter')?.group ?? '')
const props = computed(() => store.props.filter((p) => p.group === key.value))
const short = (p: prop.Prop) => p.name.slice(key.value.length + 1)
const field = (n: string) => props.value.find((p) => short(p) === n)
const value = (n: string) => field(n)?.value
const groups = computed(() => sections.map((s) => ({ title: s.title, props: s.names.map(field).filter((p): p is prop.Prop => !!p) })))
const locked = computed(() => {
  const n = findNode(store.selected)
  return !n || !!n.inherited
})

const gradient = computed(() => `linear-gradient(90deg, ${value('start') ?? '#fff'}, ${value('end') ?? '#000'})`)
const sizePath = computed(() => {
  const a = Number(value('size') ?? 0)
  const b = Number(value('sizeEnd') ?? 0)
  const top = Math.max(a, b, 0.01)
  return `M0 ${30 - (a / top) * 26} L120 ${30 - (b / top) * 26}`
})

onMounted(async () => {
  presets.value = (await call(() => Viewport.ParticlePresets())) ?? []
  if (store.info && !store.info.live) {
    Viewport.SetLive(true)
    store.info.live = true
  }
  timer = window.setInterval(async () => {
    if (store.selected && key.value) count.value = (await Viewport.Particles(store.selected, key.value, 'count', '').catch(() => 0)) ?? 0
  }, 300)
})
onBeforeUnmount(() => clearInterval(timer))

async function act(action: string, arg = '') {
  if (!store.selected || !key.value) return
  const n = await call(() => Viewport.Particles(store.selected, key.value, action, arg))
  if (n !== undefined) count.value = n
  if (action === 'preset') {
    await refreshProps()
    toast(arg, 'ok')
  }
}

async function set(name: string, v: unknown) {
  await call(() => Scene.SetProp(store.selected, name, v))
  await refreshProps()
}
</script>

<template>
  <div v-if="!key" class="grid h-full place-items-center p-6 text-center text-[12px] text-ink-500">
    Select an object with a Particle System, or add one from Add Component.
  </div>
  <div v-else class="flex h-full min-h-0">
    <aside class="flex w-56 shrink-0 flex-col gap-3 overflow-auto border-r border-ink-800 p-3">
      <div>
        <div class="label mb-1.5">Preview</div>
        <div class="h-5 rounded-md border border-ink-700" :style="{ background: gradient }" title="Color over life" />
        <svg viewBox="0 0 120 32" class="mt-1.5 h-10 w-full rounded-md border border-ink-700 bg-ink-950/60" preserveAspectRatio="none">
          <path :d="sizePath" stroke="var(--color-amber-glow)" stroke-width="2" fill="none" vector-effect="non-scaling-stroke" />
        </svg>
        <div class="mt-1 flex justify-between font-mono text-[10px] text-ink-500"><span>birth</span><span>size over life</span><span>death</span></div>
      </div>
      <div class="flex flex-wrap gap-1">
        <button class="btn h-7" title="Restart" @click="act('play')"><Icon name="play" :size="12" /> Play</button>
        <button class="btn h-7" title="Stop emitting; live particles finish" @click="act('stop')"><Icon name="stop" :size="12" /> Stop</button>
        <button class="btn h-7" title="Emit one burst now" @click="act('emit')"><Icon name="plus" :size="12" /> Burst</button>
        <button class="btn h-7" title="Remove every particle" @click="act('clear')"><Icon name="trash" :size="12" /> Clear</button>
      </div>
      <div class="font-mono text-[11px] text-ink-400">{{ count }} particles alive</div>
      <div>
        <div class="label mb-1.5">Presets</div>
        <div class="flex flex-wrap gap-1">
          <button v-for="p in presets" :key="p" class="h-6 rounded-md border border-ink-700 px-2 text-[11px] capitalize text-ink-300 transition hover:border-amber-soft hover:text-ink-100 disabled:opacity-40" :disabled="locked" @click="act('preset', p)">{{ p }}</button>
        </div>
        <p class="mt-1.5 text-[11px] leading-snug text-ink-500">A preset replaces every setting; undo brings the old ones back.</p>
      </div>
    </aside>
    <div class="grid min-w-0 flex-1 auto-rows-min grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-2 overflow-auto p-2">
      <section v-for="g in groups" :key="g.title" class="rounded-lg border border-ink-800 bg-ink-850/40 px-2 py-1">
        <div class="label py-1">{{ g.title }}</div>
        <PropField v-for="p in g.props" :key="p.name" :p="p" @set="set" />
      </section>
    </div>
  </div>
</template>
