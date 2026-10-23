<script setup lang="ts">
// Same mark as internal/editor/brand: one Braille cell with extruded lit dots.
withDefaults(defineProps<{ size?: number; plate?: boolean }>(), { size: 20, plate: true })

const lit = new Set(['1,0', '0,1', '1,1', '0,2', '0,3', '1,3'])
const cols = [37, 60]
const rows = [21, 40, 59, 78]
const dots = cols.flatMap((x, c) => rows.map((y, r) => ({ x, y, on: lit.has(`${c},${r}`) })))
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 100 100" aria-label="ASCDII Engine">
    <defs>
      <linearGradient id="plate" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#1a1918" />
        <stop offset="1" stop-color="#0c0c0c" />
      </linearGradient>
      <radialGradient id="dot" cx=".45" cy=".45" r=".6">
        <stop offset="0" stop-color="#f1c06a" />
        <stop offset="1" stop-color="#e8b04b" />
      </radialGradient>
    </defs>
    <rect v-if="plate" width="100" height="100" rx="22" fill="url(#plate)" />
    <template v-for="d in dots" :key="d.x + '-' + d.y">
      <circle v-if="d.on" :cx="d.x + 2.8" :cy="d.y + 2.8" r="7.4" fill="#6b481c" />
      <circle :cx="d.x" :cy="d.y" r="7.4" :fill="d.on ? 'url(#dot)' : '#2e2c29'" />
    </template>
  </svg>
</template>
