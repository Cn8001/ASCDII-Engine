<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Project } from '../lib/api'
import { call, refresh, store, toast } from '../lib/store'
import Icon from './Icon.vue'
import Logo from './Logo.vue'

const name = ref('My Game')
const parent = ref('')
const template = ref('empty')
const templates = computed(() => store.info?.templates ?? ['empty'])

onMounted(async () => {
  store.recent = (await call(() => Project.Recent())) ?? []
})

async function pick() {
  const dir = await call(() => Project.PickFolder())
  if (dir) parent.value = dir
}

async function create() {
  if (!parent.value) {
    await pick()
    if (!parent.value) return
  }
  store.busy = 'create'
  const info = await call(() => Project.Create(parent.value, name.value.trim() || 'My Game', template.value))
  store.busy = ''
  if (info) {
    store.info = info
    toast(info.name, 'ok')
    await refresh()
  }
}

async function open(path?: string) {
  const info = await call(() => (path ? Project.Open(path) : Project.OpenDialog()))
  if (info) {
    store.info = info
    await refresh()
  }
}

function base(p: string) {
  return p.split(/[\\/]/).pop()?.replace('.ascdiiproj', '') ?? p
}

function folder(p: string) {
  const parts = p.split(/[\\/]/)
  return parts.slice(0, -1).join('/')
}
</script>

<template>
  <div class="grid h-full place-items-center bg-ink-950 animate-fade">
    <div class="flex flex-col items-center gap-8">
    <header class="flex flex-col items-center gap-3">
      <Logo :size="84" animate />
      <div class="font-mono text-[15px] tracking-[.5em] text-ink-100">ASCDII ENGINE</div>
      <div class="font-mono text-[11px] text-ink-500">{{ store.info?.engine }}</div>
    </header>
    <div class="grid w-[860px] grid-cols-[1fr_320px] gap-6">
      <section class="animate-rise rounded-2xl border border-ink-750 bg-ink-900 p-6">
        <div class="label mb-2">New</div>
        <div class="mb-4 grid grid-cols-2 gap-3">
          <button
            v-for="t in templates"
            :key="t"
            class="group rounded-xl border p-4 text-left transition"
            :class="template === t ? 'border-amber-soft bg-amber-glow/[.06]' : 'border-ink-750 hover:border-ink-600'"
            @click="template = t"
          >
            <Icon :name="t === 'empty' ? 'box' : 'scene'" :size="18" :class="template === t ? 'text-amber-glow' : 'text-ink-400'" />
            <div class="mt-3 text-[13px] font-medium capitalize">{{ t }}</div>
          </button>
        </div>
        <div class="grid grid-cols-[1fr_auto] gap-2">
          <input v-model="name" class="field col-span-2" placeholder="Name" @keydown.enter="create" />
          <div class="flex min-w-0 items-center gap-2">
            <button class="field flex items-center gap-2 text-left text-ink-300" @click="pick">
              <Icon name="folder" :size="14" />
              <span class="truncate">{{ parent || 'Location' }}</span>
            </button>
          </div>
          <button class="btn btn-accent" :disabled="store.busy === 'create'" @click="create">
            <Icon name="plus" :size="14" /> Create
          </button>
        </div>
      </section>

      <section class="animate-rise flex flex-col rounded-2xl border border-ink-750 bg-ink-900 p-4" style="animation-delay: .06s">
        <div class="mb-3 flex items-center justify-between">
          <div class="label">Recent</div>
          <button class="btn" @click="open()"><Icon name="open" :size="14" /> Open</button>
        </div>
        <div class="-mx-2 flex-1 overflow-auto">
          <button v-for="p in store.recent" :key="p" class="flex w-full flex-col rounded-lg px-2 py-2 text-left transition hover:bg-ink-800" @click="open(p)">
            <span class="text-[13px] text-ink-100">{{ base(p) }}</span>
            <span class="truncate font-mono text-[10.5px] text-ink-500">{{ folder(p) }}</span>
          </button>
          <div v-if="!store.recent.length" class="px-2 py-6 text-center text-ink-500">-</div>
        </div>
      </section>
    </div>
    </div>
  </div>
</template>
