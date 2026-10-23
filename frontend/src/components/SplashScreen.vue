<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import Logo from './Logo.vue'

const props = defineProps<{ version: string }>()
const emit = defineEmits<{ done: [] }>()
const canvas = ref<HTMLCanvasElement | null>(null)
const showWord = ref(false)
let raf = 0
let finished = false

interface Dot {
  x: number
  y: number
  delay: number
}

// sample rasterises the word once and keeps the lit pixels as dots, so the
// logo assembles from the same 2x4 dot language the engine renders with.
function sample(text: string): { dots: Dot[]; w: number; h: number } {
  const off = document.createElement('canvas')
  off.width = 132
  off.height = 30
  const c = off.getContext('2d')!
  c.fillStyle = '#fff'
  c.font = '900 27px "Inter", "Segoe UI", system-ui, sans-serif'
  c.textBaseline = 'middle'
  c.fillText(text, 2, 16)
  const data = c.getImageData(0, 0, off.width, off.height).data
  const dots: Dot[] = []
  let w = 0
  for (let y = 0; y < off.height; y++) {
    for (let x = 0; x < off.width; x++) {
      if (data[(y * off.width + x) * 4 + 3] > 120) {
        dots.push({ x, y, delay: (x / off.width) * 0.75 + Math.random() * 0.35 })
        w = Math.max(w, x)
      }
    }
  }
  return { dots, w: w + 1, h: off.height }
}

function finish() {
  if (finished) return
  finished = true
  emit('done')
}

onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    finish()
    return
  }
  const cv = canvas.value!
  const { dots, w, h } = sample('ASCDII')
  const cell = 5
  const dpr = window.devicePixelRatio || 1
  cv.width = w * cell * dpr
  cv.height = h * cell * dpr
  cv.style.width = `${w * cell}px`
  cv.style.height = `${h * cell}px`
  const ctx = cv.getContext('2d')!
  ctx.scale(dpr, dpr)
  const start = performance.now()
  setTimeout(() => (showWord.value = true), 1050)
  setTimeout(finish, 2300)
  const frame = (now: number) => {
    const t = (now - start) / 1000
    ctx.clearRect(0, 0, w * cell, h * cell)
    const sweep = (t - 1.2) * 1.6
    for (const d of dots) {
      const k = Math.min(1, Math.max(0, (t - d.delay) / 0.28))
      if (k <= 0) continue
      const glow = Math.max(0, 1 - Math.abs(d.x / w - sweep) * 6)
      const r = (cell * 0.38) * (0.4 + 0.6 * k)
      const light = 58 + glow * 22
      ctx.fillStyle = `hsl(38 78% ${light}% / ${k})`
      ctx.beginPath()
      ctx.arc(d.x * cell + cell / 2, d.y * cell + cell / 2, r, 0, Math.PI * 2)
      ctx.fill()
    }
    raf = requestAnimationFrame(frame)
  }
  raf = requestAnimationFrame(frame)
})

onBeforeUnmount(() => cancelAnimationFrame(raf))
</script>

<template>
  <div class="fixed inset-0 z-40 grid place-items-center bg-ink-950" @click="finish">
    <div class="flex flex-col items-center gap-5">
      <div style="animation: pop .5s .05s both cubic-bezier(.2,.9,.3,1.4)"><Logo :size="64" /></div>
      <canvas ref="canvas" />
      <div class="flex h-6 items-baseline gap-3 font-mono text-[13px] tracking-[.5em] text-ink-300">
        <template v-if="showWord">
          <span v-for="(l, i) in 'ENGINE'.split('')" :key="i" class="inline-block" :style="{ animation: `letter .5s ${i * 0.06}s both cubic-bezier(.2,.8,.2,1)` }">{{ l }}</span>
          <span class="ml-2 tracking-normal text-ink-500" style="animation: fade .6s .45s both">{{ props.version }}</span>
        </template>
      </div>
    </div>
  </div>
</template>
