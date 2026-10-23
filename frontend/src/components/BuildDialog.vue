<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Build } from '../lib/api'
import { call, refreshInfo, store } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'

const all = [
  { id: 'windows', label: 'Windows' },
  { id: 'linux', label: 'Linux' },
]
const available = ref<string[]>([])
const chosen = ref<string[]>([])
const version = ref(store.info?.version ?? '0.1.0')
const result = ref<session.BuildResult | null>(null)
const busy = ref(false)

onMounted(async () => {
  available.value = (await call(() => Build.Targets())) ?? []
  chosen.value = [...available.value]
})

function toggle(id: string) {
  if (!available.value.includes(id)) return
  chosen.value = chosen.value.includes(id) ? chosen.value.filter((x) => x !== id) : [...chosen.value, id]
}

// One action: every chosen platform gets an executable, a zip and a checksum.
async function build() {
  busy.value = true
  result.value = null
  const r = await call(() => Build.Release(version.value, chosen.value))
  busy.value = false
  if (r) {
    result.value = r
    await refreshInfo()
  }
}
</script>

<template>
  <div class="fixed inset-0 z-40 grid place-items-center bg-black/50 backdrop-blur-[2px] animate-fade" @pointerdown.self="store.buildOpen = false">
    <div class="w-[400px] animate-pop rounded-2xl border border-ink-700 bg-ink-900 p-5 shadow-2xl shadow-black/70">
      <div class="mb-5 flex items-center gap-2">
        <Icon name="hammer" :size="16" class="text-amber-glow" />
        <span class="text-[14px] font-medium">Build {{ store.info?.name }}</span>
        <span class="flex-1" />
        <button class="icon-btn" @click="store.buildOpen = false"><Icon name="x" :size="14" /></button>
      </div>

      <div class="mb-4 grid grid-cols-2 gap-2">
        <button
          v-for="t in all"
          :key="t.id"
          class="flex h-12 items-center justify-center gap-2 rounded-xl border text-[13px] transition"
          :class="!available.includes(t.id) ? 'border-ink-800 text-ink-600' : chosen.includes(t.id) ? 'border-amber-soft bg-amber-glow/[.07] text-ink-100' : 'border-ink-700 text-ink-400 hover:border-ink-600'"
          :title="available.includes(t.id) ? '' : 'make players'"
          @click="toggle(t.id)"
        >
          <Icon :name="chosen.includes(t.id) ? 'check' : 'package'" :size="14" :class="chosen.includes(t.id) ? 'text-amber-glow' : ''" />
          {{ t.label }}
        </button>
      </div>

      <div class="flex items-center gap-2">
        <label class="flex flex-1 items-center gap-2 rounded-lg bg-ink-800 px-3 ring-1 ring-ink-700">
          <span class="text-[11px] text-ink-500">v</span>
          <input v-model="version" class="h-9 w-full bg-transparent font-mono text-[13px] outline-none" />
        </label>
        <button class="btn btn-accent h-9 px-5 text-[13px]" :disabled="busy || !chosen.length" @click="build">
          <Icon :name="busy ? 'refresh' : 'hammer'" :size="14" :class="busy ? 'animate-spin' : ''" /> Build
        </button>
      </div>

      <div v-if="result" class="mt-4 flex animate-rise items-center gap-2 rounded-lg bg-ink-850 px-3 py-2.5">
        <Icon name="check" :size="14" class="text-ok" />
        <span class="flex-1 truncate text-[12px] text-ink-300">{{ result.dir.split(/[\\/]/).pop() }}</span>
        <button class="btn h-7" @click="Build.Reveal(result.dir)"><Icon name="folder" :size="13" /> Show</button>
      </div>
    </div>
  </div>
</template>
