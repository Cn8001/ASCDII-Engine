<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import Icon from './Icon.vue'

const props = defineProps<{ id: string; title: string; icon?: string }>()
const emit = defineEmits<{ dock: []; close: [] }>()
const key = `ascdii.window.${props.id}`
const box = reactive({ x: 120, y: 120, w: 760, h: 300 })

onMounted(() => {
  try {
    Object.assign(box, JSON.parse(localStorage.getItem(key) ?? '{}'))
  } catch {
    // a broken saved layout just falls back to the default
  }
  box.x = Math.min(box.x, window.innerWidth - 120)
  box.y = Math.min(box.y, window.innerHeight - 80)
})

function remember() {
  try {
    localStorage.setItem(key, JSON.stringify(box))
  } catch {
    // storage can be unavailable; the window still works for this session
  }
}

function drag(e: PointerEvent, mode: 'move' | 'size') {
  const from = { cx: e.clientX, cy: e.clientY, x: box.x, y: box.y, w: box.w, h: box.h }
  const el = e.currentTarget as HTMLElement
  el.setPointerCapture(e.pointerId)
  const move = (m: PointerEvent) => {
    const dx = m.clientX - from.cx
    const dy = m.clientY - from.cy
    if (mode === 'move') {
      box.x = Math.max(0, Math.min(window.innerWidth - 80, from.x + dx))
      box.y = Math.max(0, Math.min(window.innerHeight - 40, from.y + dy))
    } else {
      box.w = Math.max(420, from.w + dx)
      box.h = Math.max(180, from.h + dy)
    }
  }
  const up = () => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', up)
    remember()
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', up)
}
</script>

<template>
  <div class="fixed z-30 flex animate-pop flex-col overflow-hidden rounded-xl border border-ink-700 bg-ink-900 shadow-2xl shadow-black/70" :style="{ left: box.x + 'px', top: box.y + 'px', width: box.w + 'px', height: box.h + 'px' }">
    <div class="flex h-8 shrink-0 cursor-move items-center gap-2 border-b border-ink-750 bg-ink-850 px-3" @pointerdown.self="drag($event, 'move')">
      <Icon v-if="icon" :name="icon" :size="13" class="pointer-events-none text-ink-400" />
      <span class="pointer-events-none label">{{ title }}</span>
      <span class="flex-1" @pointerdown="drag($event, 'move')" />
      <button class="icon-btn size-6" title="Dock" @click="emit('dock')"><Icon name="chevdown" :size="12" /></button>
      <button class="icon-btn size-6" title="Close" @click="emit('close')"><Icon name="x" :size="12" /></button>
    </div>
    <div class="min-h-0 flex-1"><slot /></div>
    <div class="absolute bottom-0 right-0 size-4 cursor-nwse-resize" @pointerdown="drag($event, 'size')">
      <svg viewBox="0 0 16 16" class="size-4 text-ink-600"><path d="M14 6v8H6M14 10v4h-4" fill="none" stroke="currentColor" stroke-width="1.5" /></svg>
    </div>
  </div>
</template>
