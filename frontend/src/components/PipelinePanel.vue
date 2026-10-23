<script setup lang="ts">
import { computed, ref } from 'vue'
import { Pipeline } from '../lib/api'
import { call, store } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import Icon from './Icon.vue'
import PropField from './PropField.vue'

const open = ref('')
const props = ref<prop.Prop[]>([])
const dragging = ref('')
const stages = ['geometry', 'lighting', 'post', 'resolve', 'overlay']
const passes = computed(() => store.frame.passes)
const peak = computed(() => Math.max(1, ...passes.value.map((p) => p.micros)))

async function toggle(name: string, on: boolean) {
  await call(() => Pipeline.SetEnabled(name, on))
}

async function expand(name: string, editable: boolean) {
  if (!editable) return
  if (open.value === name) {
    open.value = ''
    return
  }
  props.value = (await call(() => Pipeline.Props(name))) ?? []
  open.value = name
}

async function set(key: string, value: unknown) {
  await call(() => Pipeline.SetProp(open.value, key, value))
  props.value = (await call(() => Pipeline.Props(open.value))) ?? []
}

async function drop(target: string) {
  const from = dragging.value
  dragging.value = ''
  if (!from || from === target) return
  const index = passes.value.findIndex((p) => p.name === target)
  await call(() => Pipeline.Move(from, index))
}
</script>

<template>
  <div class="h-full overflow-auto px-3 py-2">
    <template v-for="s in stages" :key="s">
      <div v-if="passes.some((p) => p.stage === s)" class="mb-3">
        <div class="label mb-1 px-1">{{ s }}</div>
        <div
          v-for="p in passes.filter((x) => x.stage === s)"
          :key="p.name"
          class="mb-1 rounded-lg border border-transparent bg-ink-850 transition"
          :class="[dragging === p.name ? 'opacity-50' : '', open === p.name ? 'border-ink-700' : '']"
          draggable="true"
          @dragstart="dragging = p.name"
          @dragover.prevent
          @drop.prevent="drop(p.name)"
        >
          <div class="flex h-9 items-center gap-2 px-2">
            <button class="relative h-4 w-7 shrink-0 rounded-full transition" :class="p.enabled ? 'bg-amber-glow' : 'bg-ink-700'" @click="toggle(p.name, !p.enabled)">
              <span class="absolute top-0.5 size-3 rounded-full bg-ink-950 transition-all" :class="p.enabled ? 'left-3.5' : 'left-0.5'" />
            </button>
            <button class="flex flex-1 items-center gap-2 text-left" @click="expand(p.name, p.editable)">
              <span class="text-[12.5px]" :class="p.enabled ? 'text-ink-100' : 'text-ink-500'">{{ p.name }}</span>
              <Icon v-if="p.editable" :name="open === p.name ? 'chevdown' : 'chevright'" :size="11" class="text-ink-500" />
            </button>
            <div class="flex w-24 items-center gap-2">
              <div class="h-1 flex-1 overflow-hidden rounded-full bg-ink-800">
                <div class="h-full rounded-full bg-amber-soft transition-[width] duration-300" :style="{ width: (p.micros / peak) * 100 + '%' }" />
              </div>
              <span class="w-10 text-right font-mono text-[10px] text-ink-500">{{ (p.micros / 1000).toFixed(1) }}</span>
            </div>
          </div>
          <div v-if="open === p.name" class="animate-fade border-t border-ink-750 px-2 py-1">
            <PropField v-for="x in props" :key="x.name" :p="x" @set="set" />
          </div>
        </div>
      </div>
    </template>
    <div class="mt-2 flex items-center justify-between px-1 font-mono text-[10.5px] text-ink-500">
      <span>total</span>
      <span>{{ store.frame.millis.toFixed(2) }} ms</span>
    </div>
  </div>
</template>
