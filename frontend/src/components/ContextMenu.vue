<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { closeMenu, menu, type MenuItem } from '../lib/menu'
import Icon from './Icon.vue'

const sub = ref<number | null>(null)

function run(i: MenuItem) {
  if (i.disabled || i.children) return
  closeMenu()
  i.run?.()
}

function esc(e: KeyboardEvent) {
  if (e.key === 'Escape') closeMenu()
}

onMounted(() => window.addEventListener('keydown', esc))
onBeforeUnmount(() => window.removeEventListener('keydown', esc))
</script>

<template>
  <div v-if="menu.open" class="fixed inset-0 z-50" @pointerdown.self="closeMenu" @contextmenu.prevent="closeMenu">
    <div class="absolute min-w-48 animate-pop rounded-lg border border-ink-700 bg-ink-850 p-1 shadow-2xl shadow-black/70" :style="{ left: menu.x + 'px', top: menu.y + 'px' }">
      <div v-for="(i, n) in menu.items" :key="n" class="relative" @mouseenter="sub = i.children ? n : null">
        <div v-if="i.label === '-'" class="my-1 h-px bg-ink-700" />
        <button
          v-else
          class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] transition"
          :class="[i.disabled ? 'text-ink-500' : i.danger ? 'text-bad hover:bg-bad/10' : 'text-ink-200 hover:bg-ink-750 hover:text-ink-100']"
          @click="run(i)"
        >
          <Icon v-if="i.icon" :name="i.icon" :size="14" class="text-ink-400" />
          <span class="flex-1">{{ i.label }}</span>
          <Icon v-if="i.children" name="chevright" :size="12" class="text-ink-500" />
        </button>
        <div v-if="i.children && sub === n" class="absolute left-full top-0 ml-1 max-h-80 min-w-44 overflow-auto rounded-lg border border-ink-700 bg-ink-850 p-1 shadow-2xl shadow-black/70">
          <button v-for="(c, m) in i.children" :key="m" class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] text-ink-200 transition hover:bg-ink-750" @click="run(c)">
            <Icon v-if="c.icon" :name="c.icon" :size="13" class="text-ink-400" />
            {{ c.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
