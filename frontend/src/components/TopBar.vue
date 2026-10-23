<script setup lang="ts">
import { computed } from 'vue'
import { Project, Viewport } from '../lib/api'
import { addMenu, createParent } from '../lib/actions'
import { ask, confirmAction } from '../lib/dialog'
import { openBelow } from '../lib/menu'
import { call, refresh, refreshInfo, store, toast } from '../lib/store'
import Icon from './Icon.vue'
import Logo from './Logo.vue'

const mode = computed(() => store.frame.mode)

async function save() {
  const ok = await call(() => Project.Save().then(() => true))
  if (ok) {
    toast('Saved', 'ok')
    await refresh()
  }
}

async function play() {
  if (mode.value === 'play') {
    Viewport.Pause()
    return
  }
  await call(() => Viewport.Play())
  document.querySelector<HTMLElement>('[tabindex="0"]')?.focus()
}

async function stop() {
  Viewport.Stop()
  setTimeout(refresh, 60)
}

async function undo() {
  await call(() => Project.Undo())
  await refresh()
}

async function redo() {
  await call(() => Project.Redo())
  await refresh()
}

async function openScene(path: string) {
  if (path === '+') {
    const name = await ask('New scene', 'level', { suffix: '.scene.json' })
    if (!name) {
      await refresh()
      return
    }
    const rel = (name.includes('/') ? name : 'scenes/' + name).replace(/\.scene\.json$/, '') + '.scene.json'
    await call(() => Project.NewScene(rel))
  } else {
    if (store.info?.unsaved && !(await confirmAction('Discard unsaved changes?', 'The open scene has changes that are not saved.'))) return
    await call(() => Project.OpenScene(path))
  }
  store.selected = 0
  await refresh()
}

async function openProject() {
  if (store.info?.unsaved && !(await confirmAction('Discard unsaved changes?', 'The open scene has changes that are not saved.'))) return
  const info = await call(() => Project.OpenDialog())
  if (info) {
    store.info = info
    store.selected = 0
    await refresh()
  }
}

// closeProject goes back to the start screen, like File > Close in other tools.
async function closeProject() {
  if (store.info?.unsaved && !(await confirmAction('Close without saving?', 'The open scene has changes that are not saved.', 'Close'))) return
  if (store.frame.mode !== 'edit') Viewport.Stop()
  await call(() => Project.Close())
  store.selected = 0
  store.tree = []
  store.props = []
  store.clips = []
  store.logs = []
  store.recent = (await call(() => Project.Recent())) ?? []
  await refreshInfo()
}

async function setStart() {
  await call(() => Project.SetStart())
  toast('Start scene', 'ok')
}

defineExpose({ save, undo, redo, play, stop })
</script>

<template>
  <header class="flex h-11 shrink-0 items-center gap-3 border-b border-ink-750 bg-ink-900 px-3">
    <div class="flex items-center gap-2">
      <Logo :size="22" />
      <span class="font-semibold tracking-tight">{{ store.info?.name }}</span>
      <span class="font-mono text-[10.5px] text-ink-500">{{ store.info?.version }}</span>
    </div>

    <div class="flex items-center gap-1">
      <button class="icon-btn" title="New project  Ctrl+N" @click="store.newProject = true"><Icon name="plus" :size="15" /></button>
      <button class="icon-btn" title="Open project" @click="openProject"><Icon name="open" :size="15" /></button>
      <button class="icon-btn" title="Close project" @click="closeProject"><Icon name="x" :size="15" /></button>
      <select class="h-7 max-w-56 rounded-md bg-ink-800 px-2 text-[12px] text-ink-200 outline-none ring-1 ring-ink-700" :value="store.info?.scene" @change="openScene(($event.target as HTMLSelectElement).value)">
        <option v-for="s in store.info?.scenes ?? []" :key="s" :value="s">{{ s.replace('scenes/', '') }}</option>
        <option value="+">+ scene</option>
      </select>
      <button class="icon-btn" title="Start scene" @click="setStart"><Icon name="key" :size="11" /></button>
      <button class="icon-btn" title="Packages" @click="store.packagesOpen = true"><Icon name="package" :size="14" /></button>
      <span class="mx-1 h-5 w-px bg-ink-700" />
      <button class="btn h-7 px-2 text-ink-300" @click="(e) => openBelow(e, addMenu(createParent()))"><Icon name="box" :size="13" /> GameObject <Icon name="chevdown" :size="11" class="text-ink-500" /></button>
    </div>

    <div class="flex flex-1 justify-center">
      <div class="flex items-center gap-1 rounded-lg border border-ink-750 bg-ink-850 p-1">
        <button class="icon-btn" :class="mode === 'play' ? 'bg-amber-glow text-ink-950 hover:bg-amber-glow hover:text-ink-950' : 'text-amber-glow'" title="Play  Space" @click="play">
          <Icon :name="mode === 'play' ? 'pause' : 'play'" :size="14" />
        </button>
        <button class="icon-btn" :disabled="mode !== 'paused'" title="Step" @click="Viewport.Step()"><Icon name="chevright" :size="14" /></button>
        <button class="icon-btn" :disabled="mode === 'edit'" title="Stop  Esc" @click="stop"><Icon name="stop" :size="14" /></button>
      </div>
    </div>

    <div class="flex items-center gap-1">
      <button class="icon-btn" :disabled="!store.info?.canUndo" title="Undo  Ctrl+Z" @click="undo"><Icon name="undo" :size="15" /></button>
      <button class="icon-btn" :disabled="!store.info?.canRedo" title="Redo  Ctrl+Y" @click="redo"><Icon name="redo" :size="15" /></button>
      <button class="icon-btn relative" title="Save  Ctrl+S" @click="save">
        <Icon name="save" :size="15" />
        <span v-if="store.info?.unsaved" class="absolute right-1 top-1 size-1.5 rounded-full bg-amber-glow" />
      </button>
      <span class="mx-1 h-5 w-px bg-ink-700" />
      <button class="btn btn-accent" title="Ctrl+B" @click="store.buildOpen = true"><Icon name="hammer" :size="14" /> Build</button>
    </div>
  </header>
</template>
