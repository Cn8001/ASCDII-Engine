<script setup lang="ts">
import { store } from '../lib/store'
import Icon from './Icon.vue'
</script>

<template>
  <div class="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col items-end gap-2">
    <TransitionGroup enter-from-class="opacity-0 translate-y-2" leave-to-class="opacity-0 translate-x-4" enter-active-class="transition duration-200" leave-active-class="transition duration-200">
      <div v-for="t in store.toasts" :key="t.id" class="pointer-events-auto flex max-w-md items-center gap-2 rounded-lg border border-ink-700 bg-ink-850/95 px-3 py-2 text-[12px] shadow-2xl shadow-black/60 backdrop-blur">
        <Icon :name="t.kind === 'bad' ? 'alert' : t.kind === 'ok' ? 'check' : 'dot'" :class="t.kind === 'bad' ? 'text-bad' : t.kind === 'ok' ? 'text-ok' : 'text-amber-glow'" :size="14" />
        <span class="text-ink-200">{{ t.text }}</span>
      </div>
    </TransitionGroup>
  </div>
</template>
