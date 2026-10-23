<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Assets } from '../lib/api'
import { decode, render } from '../lib/cells'

const props = defineProps<{ path: string }>()
const canvas = ref<HTMLCanvasElement | null>(null)

onMounted(async () => {
  try {
    const pv = await Assets.Preview(props.path)
    if (canvas.value) render(canvas.value, decode(pv.cols, pv.rows, pv.cells), { cw: 4, ch: 8, style: 'pixels' })
  } catch {
    // an unreadable sprite shows the plain icon instead
  }
})
</script>

<template>
  <canvas ref="canvas" class="max-h-12 max-w-16 [image-rendering:pixelated]" />
</template>
