<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { close, dialog } from '../lib/dialog'

const input = ref<HTMLInputElement | null>(null)

watch(() => dialog.open, async (on) => {
  if (!on || dialog.confirm) return
  await nextTick()
  input.value?.focus()
  input.value?.select()
})

function submit() {
  if (dialog.confirm) close('')
  else if (dialog.value.trim()) close(dialog.value.trim())
}
</script>

<template>
  <div v-if="dialog.open" class="fixed inset-0 z-[60] grid place-items-center bg-black/50 backdrop-blur-[2px] animate-fade" @pointerdown.self="close(null)" @keydown.esc="close(null)">
    <form class="w-[380px] animate-pop rounded-2xl border border-ink-700 bg-ink-900 p-5 shadow-2xl shadow-black/70" @submit.prevent="submit">
      <div class="mb-1 text-[14px] font-medium">{{ dialog.title }}</div>
      <div v-if="dialog.hint" class="mb-3 text-[12px] text-ink-400">{{ dialog.hint }}</div>
      <label v-if="!dialog.confirm" class="mb-4 mt-3 flex items-center rounded-lg bg-ink-800 px-3 ring-1 ring-ink-700 focus-within:ring-amber-soft">
        <input ref="input" v-model="dialog.value" class="h-9 w-full bg-transparent font-mono text-[13px] outline-none" @keydown.esc.prevent="close(null)" />
        <span v-if="dialog.suffix" class="font-mono text-[12px] text-ink-500">{{ dialog.suffix }}</span>
      </label>
      <div class="flex justify-end gap-2" :class="dialog.confirm ? 'mt-4' : ''">
        <button type="button" class="btn" @click="close(null)">Cancel</button>
        <button type="submit" class="btn" :class="dialog.danger ? 'bg-bad/90 text-ink-950 hover:bg-bad hover:text-ink-950' : 'btn-accent'" :disabled="!dialog.confirm && !dialog.value.trim()">{{ dialog.ok }}</button>
      </div>
    </form>
  </div>
</template>
