<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { Assets } from '../lib/api'

const props = defineProps<{ path: string; size?: number }>()
const src = ref('')

async function load() {
  try {
    src.value = await Assets.Thumb(props.path, props.size ?? 64)
  } catch {
    src.value = '' // unreadable pictures show the plain icon instead
  }
}

onMounted(load)
watch(() => props.path, load)
</script>

<template>
  <img v-if="src" :src="src" class="max-h-12 max-w-16 rounded-sm [image-rendering:pixelated]" alt="" />
</template>
