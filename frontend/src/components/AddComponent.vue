<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { Assets, Scene } from '../lib/api'
import { ask } from '../lib/dialog'
import { call, findNode, refresh, refreshAssets, store, toast } from '../lib/store'
import Icon from './Icon.vue'

const open = ref(false)
const query = ref('')
const input = ref<HTMLInputElement | null>(null)
const node = computed(() => findNode(store.selected))
const have = computed(() => new Set(node.value?.behaviours ?? []))
const options = computed(() => {
  const q = query.value.toLowerCase()
  const kinds = store.kinds.behaviours.filter((b) => !have.value.has(b) && b !== 'script').map((b) => ({ label: b, kind: b, script: '' }))
  const scripts = have.value.has('script') ? [] : store.assets.filter((a) => a.kind === 'script').map((a) => ({ label: a.path.split('/').pop()!.replace('.lua', ''), kind: 'script', script: a.path }))
  return [...scripts, ...kinds].filter((o) => o.label.toLowerCase().includes(q))
})

async function show() {
  open.value = true
  query.value = ''
  await refreshAssets()
  await nextTick()
  input.value?.focus()
}

async function pick(kind: string, script: string) {
  open.value = false
  await call(() => Scene.AddBehaviour(store.selected, kind))
  if (script) await call(() => Scene.SetProp(store.selected, 'script.file', script))
  await refresh()
}

async function createScript() {
  open.value = false
  const suggested = query.value.trim() || node.value?.name.replace(/\s+/g, '-').toLowerCase() || 'script'
  const name = await ask('New script', suggested, { hint: 'Saved in scripts/ and opened in your code editor.', suffix: '.lua' })
  if (!name) return
  const rel = await call(() => Assets.NewScript(name))
  if (!rel) return
  await pick('script', rel)
  toast(rel, 'ok')
  await call(() => Assets.OpenExternal(rel, 0))
}
</script>

<template>
  <div class="relative pt-3">
    <button class="btn h-8 w-full justify-center border border-ink-700 text-ink-300" @click="show"><Icon name="plus" :size="13" /> Add Component</button>
    <div v-if="open" class="fixed inset-0 z-40" @pointerdown="open = false" />
    <div v-if="open" class="absolute inset-x-0 top-12 z-50 animate-pop rounded-lg border border-ink-700 bg-ink-850 p-1.5 shadow-2xl shadow-black/70">
      <input ref="input" v-model="query" class="field mb-1" placeholder="Search" @keydown.esc="open = false" @keydown.enter="options[0] ? pick(options[0].kind, options[0].script) : createScript()" />
      <div class="max-h-60 overflow-auto">
        <button v-for="o in options" :key="o.kind + o.script" class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] text-ink-200 hover:bg-ink-750" @click="pick(o.kind, o.script)">
          <Icon :name="o.script ? 'file' : 'sliders'" :size="12" :class="o.script ? 'text-ok' : 'text-ink-400'" />
          <span class="flex-1">{{ o.label }}</span>
          <span v-if="o.script" class="font-mono text-[10px] text-ink-500">lua</span>
        </button>
        <button v-if="!have.has('script')" class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] text-amber-glow hover:bg-ink-750" @click="createScript">
          <Icon name="plus" :size="12" /> New script {{ query ? `"${query}"` : '' }}
        </button>
      </div>
    </div>
  </div>
</template>
