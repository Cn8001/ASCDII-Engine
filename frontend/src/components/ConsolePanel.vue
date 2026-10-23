<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { Assets, Console, EventsOn } from '../lib/api'
import { call, store } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'

const show = ref({ info: true, warn: true, error: true })
const list = ref<HTMLDivElement | null>(null)
const counts = computed(() => {
  const c = { info: 0, warn: 0, error: 0 } as Record<string, number>
  for (const l of store.logs) c[l.level] = (c[l.level] ?? 0) + 1
  return c
})
const visible = computed(() => store.logs.filter((l) => show.value[l.level as 'info' | 'warn' | 'error'] ?? true))
let off: (() => void) | null = null

async function add(e: session.LogEntry) {
  const last = store.logs[store.logs.length - 1]
  if (last && last.level === e.level && last.source === e.source && last.text === e.text) {
    Object.assign(last, e)
  } else {
    store.logs.push(e)
    if (store.logs.length > 300) store.logs.splice(0, store.logs.length - 300)
  }
  if (e.level === 'error' && !store.dockOpen) store.dockOpen = true
  await nextTick()
  list.value?.scrollTo({ top: list.value.scrollHeight })
}

onMounted(async () => {
  store.logs = (await call(() => Console.Logs())) ?? []
  off = EventsOn('log', add)
})
onBeforeUnmount(() => off?.())

async function clear() {
  await call(() => Console.Clear())
  store.logs = []
}

// open jumps to the script line that "file.lua:12: ..." errors carry.
function open(l: session.LogEntry) {
  if (!l.source.endsWith('.lua')) return
  const m = l.text.match(/\.lua:(\d+)/)
  call(() => Assets.OpenExternal(l.source, m ? Number(m[1]) : 0))
}

const tone: Record<string, string> = { info: 'text-ink-400', warn: 'text-amber-glow', error: 'text-bad' }
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex h-8 shrink-0 items-center gap-1 border-b border-ink-800 px-2">
      <button class="btn h-6" @click="clear"><Icon name="trash" :size="12" /> Clear</button>
      <span class="flex-1" />
      <button v-for="lv in ['info', 'warn', 'error'] as const" :key="lv" class="flex h-6 items-center gap-1 rounded-md px-2 font-mono text-[11px] transition" :class="show[lv] ? 'bg-ink-800 ' + tone[lv] : 'text-ink-600'" @click="show[lv] = !show[lv]">
        <Icon :name="lv === 'error' ? 'alert' : lv === 'warn' ? 'alert' : 'dot'" :size="12" /> {{ counts[lv] ?? 0 }}
      </button>
    </div>
    <div ref="list" class="flex-1 overflow-auto py-1 font-mono text-[11.5px]">
      <div v-for="(l, i) in visible" :key="i" class="flex items-start gap-2 px-3 py-0.5 hover:bg-ink-850" :class="l.source.endsWith('.lua') ? 'cursor-pointer' : ''" @dblclick="open(l)">
        <span class="w-16 shrink-0 text-ink-600">{{ l.time }}</span>
        <span class="w-3 shrink-0" :class="tone[l.level]">{{ l.level === 'error' ? '!' : l.level === 'warn' ? '?' : '·' }}</span>
        <span class="shrink-0 text-ink-500">{{ l.source }}</span>
        <span class="min-w-0 flex-1 break-words" :class="l.level === 'error' ? 'text-bad' : 'text-ink-200'">{{ l.text }}</span>
        <span v-if="l.count > 1" class="shrink-0 rounded-full bg-ink-750 px-1.5 text-[10px] text-ink-300">{{ l.count }}</span>
      </div>
      <div v-if="!visible.length" class="grid h-full place-items-center text-ink-600">-</div>
    </div>
  </div>
</template>
