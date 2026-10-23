<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { categoryLabel, kindIcon } from '../lib/actions'
import { Assets, Scene } from '../lib/api'
import { ask } from '../lib/dialog'
import { call, findNode, refresh, refreshAssets, store, toast } from '../lib/store'
import Icon from './Icon.vue'

interface Option {
  label: string
  kind: string
  script: string
  category: string
}

const open = ref(false)
const query = ref('')
const at = ref(0)
const input = ref<HTMLInputElement | null>(null)
const node = computed(() => findNode(store.selected))
const options = computed<Option[]>(() => {
  const q = query.value.toLowerCase()
  const scripts = store.assets
    .filter((a) => a.kind === 'script')
    .map((a) => ({ label: a.path.split('/').pop()!.replace('.lua', ''), kind: 'script', script: a.guid || a.path, category: 'scripts' }))
  const kinds = store.kinds.components.filter((k) => k.kind !== 'script').map((k) => ({ label: k.label, kind: k.kind, script: '', category: k.category }))
  const all = [...scripts, ...kinds].filter((o) => !q || o.label.toLowerCase().includes(q) || o.kind.includes(q))
  const rank = (c: string) => ['scripts', 'logic', '3d', 'light', 'effects', 'ui'].indexOf(c) + 1 || 9
  return all.sort((a, b) => rank(a.category) - rank(b.category) || a.category.localeCompare(b.category))
})

function heading(i: number): string {
  const o = options.value[i]
  if (query.value || (i > 0 && options.value[i - 1].category === o.category)) return ''
  return o.category === 'scripts' ? 'Scripts' : categoryLabel(o.category)
}

async function show() {
  open.value = true
  query.value = ''
  at.value = 0
  await refreshAssets()
  await nextTick()
  input.value?.focus()
}

async function pick(o: Option) {
  open.value = false
  const key = await call(() => Scene.AddComponent(store.selected, o.kind))
  if (key && o.script) await call(() => Scene.SetProp(store.selected, key + '.file', o.script))
  await refresh()
}

async function createScript() {
  open.value = false
  const suggested = query.value.trim() || node.value?.name.replace(/\s+/g, '-').toLowerCase() || 'script'
  const name = await ask('New script', suggested, { hint: 'Saved in scripts/ and opened in your code editor.', suffix: '.lua' })
  if (!name) return
  const rel = await call(() => Assets.NewScript(name))
  if (!rel) return
  await pick({ label: name, kind: 'script', script: rel, category: 'scripts' })
  toast(rel, 'ok')
  await call(() => Assets.OpenExternal(rel, 0))
}

function nav(e: KeyboardEvent) {
  const n = options.value.length
  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    at.value = n ? (at.value + (e.key === 'ArrowDown' ? 1 : n - 1)) % n : 0
    document.querySelector(`[data-option="${at.value}"]`)?.scrollIntoView({ block: 'nearest' })
  } else if (e.key === 'Enter') {
    const o = options.value[at.value]
    if (o) pick(o)
    else createScript()
  } else if (e.key === 'Escape') {
    open.value = false
  }
}
</script>

<template>
  <div class="relative pt-3">
    <button class="btn h-8 w-full justify-center border border-ink-700 text-ink-300" @click="show"><Icon name="plus" :size="13" /> Add Component</button>
    <div v-if="open" class="fixed inset-0 z-40" @pointerdown="open = false" />
    <div v-if="open" class="absolute inset-x-0 top-12 z-50 animate-pop rounded-lg border border-ink-700 bg-ink-850 p-1.5 shadow-2xl shadow-black/70">
      <input ref="input" v-model="query" class="field mb-1" placeholder="Search" @input="at = 0" @keydown="nav" />
      <div class="max-h-72 overflow-auto">
        <template v-for="(o, i) in options" :key="o.kind + o.script">
          <div v-if="heading(i)" class="px-2 pb-0.5 pt-2 text-[10px] font-semibold uppercase tracking-wider text-ink-500">{{ heading(i) }}</div>
          <button
            :data-option="i"
            class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] text-ink-200"
            :class="at === i ? 'bg-ink-750 text-ink-100' : ''"
            @pointerenter="at = i"
            @click="pick(o)"
          >
            <Icon :name="o.script ? 'file' : o.category === 'logic' ? 'sliders' : kindIcon(o.kind)" :size="12" :class="o.script ? 'text-ok' : 'text-ink-400'" />
            <span class="flex-1 truncate">{{ o.label }}</span>
            <span v-if="o.script" class="font-mono text-[10px] text-ink-500">lua</span>
          </button>
        </template>
        <button class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] text-amber-glow hover:bg-ink-750" @click="createScript">
          <Icon name="plus" :size="12" /> New script {{ query ? `"${query}"` : '' }}
        </button>
      </div>
    </div>
  </div>
</template>
