<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Project, Viewport } from '../lib/api'
import { confirmAction } from '../lib/dialog'
import { call, refresh, refreshAssets, store, toast } from '../lib/store'
import Icon from './Icon.vue'

// NewProjectDialog creates and opens a project without leaving the editor.
const name = ref('My Game')
const parent = ref('')
const template = ref('empty')
const templates = computed(() => store.info?.templates ?? ['empty'])
const input = ref<HTMLInputElement | null>(null)

watch(
  () => store.newProject,
  async (on) => {
    if (!on) return
    name.value = 'My Game'
    template.value = 'empty'
    if (!parent.value && store.info?.dir) parent.value = store.info.dir.replace(/[\\/][^\\/]+$/, '')
    await nextTick()
    input.value?.select()
  },
)

async function pick() {
  const dir = await call(() => Project.PickFolder())
  if (dir) parent.value = dir
}

async function create() {
  if (!parent.value) {
    await pick()
    if (!parent.value) return
  }
  if (store.info?.unsaved && !(await confirmAction('Discard unsaved changes?', 'The open scene has changes that are not saved.'))) return
  if (store.frame.mode !== 'edit') Viewport.Stop()
  store.busy = 'create'
  const info = await call(() => Project.Create(parent.value, name.value.trim() || 'My Game', template.value))
  store.busy = ''
  if (!info) return
  store.newProject = false
  store.info = info
  store.selected = 0
  store.asset = ''
  store.logs = []
  toast(info.name, 'ok')
  await Promise.all([refresh(), refreshAssets()])
}
</script>

<template>
  <div v-if="store.newProject" class="fixed inset-0 z-[55] grid place-items-center bg-black/50 backdrop-blur-[2px] animate-fade" @pointerdown.self="store.newProject = false">
    <form class="w-[440px] animate-pop rounded-2xl border border-ink-700 bg-ink-900 p-5 shadow-2xl shadow-black/70" @submit.prevent="create" @keydown.esc="store.newProject = false">
      <div class="mb-4 text-[14px] font-medium">New project</div>
      <div class="mb-4 grid grid-cols-2 gap-3">
        <button
          v-for="t in templates"
          :key="t"
          type="button"
          class="rounded-xl border p-3 text-left transition"
          :class="template === t ? 'border-amber-soft bg-amber-glow/[.06]' : 'border-ink-750 hover:border-ink-600'"
          @click="template = t"
        >
          <Icon :name="t === 'empty' ? 'box' : 'scene'" :size="16" :class="template === t ? 'text-amber-glow' : 'text-ink-400'" />
          <div class="mt-2 text-[12.5px] font-medium capitalize">{{ t }}</div>
        </button>
      </div>
      <input ref="input" v-model="name" class="field mb-2" placeholder="Name" />
      <button type="button" class="field mb-4 flex items-center gap-2 text-left text-ink-300" @click="pick">
        <Icon name="folder" :size="14" />
        <span class="truncate">{{ parent || 'Location' }}</span>
      </button>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn" @click="store.newProject = false">Cancel</button>
        <button type="submit" class="btn btn-accent" :disabled="store.busy === 'create'"><Icon name="plus" :size="14" /> Create</button>
      </div>
    </form>
  </div>
</template>
