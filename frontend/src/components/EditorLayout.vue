<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Project, Viewport } from '../lib/api'
import { duplicate, remove } from '../lib/actions'
import { listenFileDrops } from '../lib/filedrop'
import { typing } from '../lib/keys'
import { call, refresh, refreshAssets, store, toast } from '../lib/store'
import AnimationPanel from './AnimationPanel.vue'
import BuildDialog from './BuildDialog.vue'
import ConsolePanel from './ConsolePanel.vue'
import FloatingWindow from './FloatingWindow.vue'
import Icon from './Icon.vue'
import AssetInspector from './AssetInspector.vue'
import NewProjectDialog from './NewProjectDialog.vue'
import PackageManager from './PackageManager.vue'
import Inspector from './Inspector.vue'
import PipelinePanel from './PipelinePanel.vue'
import ProjectWindow from './ProjectWindow.vue'
import SceneTree from './SceneTree.vue'
import TopBar from './TopBar.vue'
import ViewportView from './Viewport.vue'

const bar = ref<InstanceType<typeof TopBar> | null>(null)
const tabs = [
  { id: 'project', label: 'Project', icon: 'folder' },
  { id: 'console', label: 'Console', icon: 'file' },
  { id: 'animation', label: 'Animation', icon: 'film' },
] as const

async function shortcut(e: KeyboardEvent) {
  if (typing(e)) return
  const mod = e.ctrlKey || e.metaKey
  const k = e.key.toLowerCase()
  if (mod && k === 's') {
    e.preventDefault()
    const ok = await call(() => Project.Save().then(() => true))
    if (ok) toast('Saved', 'ok')
    await refresh()
  } else if (mod && k === 'z' && !e.shiftKey) {
    e.preventDefault()
    await bar.value?.undo()
  } else if (mod && (k === 'y' || (k === 'z' && e.shiftKey))) {
    e.preventDefault()
    await bar.value?.redo()
  } else if (mod && k === 'd' && store.selected) {
    e.preventDefault()
    await duplicate(store.selected)
  } else if (mod && k === 'n') {
    e.preventDefault()
    store.newProject = true
  } else if (mod && k === 'b') {
    e.preventDefault()
    store.buildOpen = true
  } else if (mod && k === '6') {
    e.preventDefault()
    openAnimation()
  } else if ((k === 'delete' || k === 'backspace') && store.selected && store.frame.mode === 'edit') {
    await remove(store.selected)
  } else if (k === 'f' && store.selected) {
    Viewport.Focus(store.selected)
  } else if (k === ' ' && !mod) {
    e.preventDefault()
    await bar.value?.play()
  } else if (k === 'escape' && store.frame.mode !== 'edit') {
    await bar.value?.stop()
  }
}

function openAnimation() {
  if (store.animFloating) return
  store.dock = 'animation'
  store.dockOpen = true
}

function resize(e: PointerEvent) {
  const start = { y: e.clientY, h: store.dockHeight }
  const el = e.currentTarget as HTMLElement
  el.setPointerCapture(e.pointerId)
  const move = (m: PointerEvent) => (store.dockHeight = Math.max(120, Math.min(window.innerHeight - 260, start.h + start.y - m.clientY)))
  const up = () => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', up)
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', up)
}

function float() {
  store.animFloating = true
  if (store.dock === 'animation') store.dock = 'project'
}

function dock() {
  store.animFloating = false
  store.dock = 'animation'
  store.dockOpen = true
}

onMounted(async () => {
  window.addEventListener('keydown', shortcut)
  listenFileDrops()
  await Promise.all([refresh(), refreshAssets()])
})
onBeforeUnmount(() => window.removeEventListener('keydown', shortcut))

function tab(active: boolean) {
  return active ? 'text-ink-100 after:absolute after:inset-x-2 after:-bottom-px after:h-px after:bg-amber-glow' : 'text-ink-500 hover:text-ink-200'
}
</script>

<template>
  <div class="flex h-full flex-col animate-fade">
    <TopBar ref="bar" />
    <div class="flex min-h-0 flex-1">
      <aside class="flex w-64 shrink-0 flex-col border-r border-ink-750 bg-ink-900 animate-rise">
        <div class="flex h-9 shrink-0 items-center gap-1.5 px-3 text-[12px] text-ink-100"><Icon name="layers" :size="13" /> Hierarchy</div>
        <div class="min-h-0 flex-1"><SceneTree /></div>
      </aside>

      <main class="flex min-w-0 flex-1 flex-col">
        <div class="min-h-0 flex-1"><ViewportView /></div>
        <div class="relative shrink-0 border-t border-ink-750 bg-ink-900" :style="{ height: store.dockOpen ? store.dockHeight + 'px' : '32px' }">
          <div class="absolute inset-x-0 -top-1 h-2 cursor-ns-resize" @pointerdown="resize" />
          <nav class="flex h-8 items-center gap-1 border-b border-ink-750 px-2">
            <template v-for="t in tabs" :key="t.id">
              <button v-if="t.id !== 'animation' || !store.animFloating" class="relative flex h-8 items-center gap-1.5 px-2 text-[12px] transition" :class="tab(store.dock === t.id && store.dockOpen)" @click="store.dock = t.id; store.dockOpen = true">
                <Icon :name="t.icon" :size="12" /> {{ t.label }}
                <span v-if="t.id === 'console' && store.logs.some((l) => l.level === 'error')" class="size-1.5 rounded-full bg-bad" />
              </button>
            </template>
            <span class="flex-1" />
            <button class="icon-btn size-6" @click="store.dockOpen = !store.dockOpen"><Icon :name="store.dockOpen ? 'chevdown' : 'chevright'" :size="12" /></button>
          </nav>
          <div v-if="store.dockOpen" class="h-[calc(100%-32px)]">
            <KeepAlive>
              <ProjectWindow v-if="store.dock === 'project'" />
              <ConsolePanel v-else-if="store.dock === 'console'" />
              <AnimationPanel v-else-if="!store.animFloating" @float="float" />
            </KeepAlive>
          </div>
        </div>
      </main>

      <aside class="flex w-[340px] shrink-0 flex-col border-l border-ink-750 bg-ink-900 animate-rise" style="animation-delay: .05s">
        <nav class="flex h-9 shrink-0 items-center gap-1 px-2">
          <button class="relative flex h-9 items-center gap-1.5 px-2 text-[12px] transition" :class="tab(store.right === 'inspector')" @click="store.right = 'inspector'"><Icon name="sliders" :size="13" /> Inspector</button>
          <button class="relative flex h-9 items-center gap-1.5 px-2 text-[12px] transition" :class="tab(store.right === 'pipeline')" @click="store.right = 'pipeline'"><Icon name="layers" :size="13" /> Pipeline</button>
        </nav>
        <div class="min-h-0 flex-1">
          <AssetInspector v-if="store.right === 'inspector' && store.asset" />
          <Inspector v-else-if="store.right === 'inspector'" />
          <PipelinePanel v-else />
        </div>
      </aside>
    </div>
    <FloatingWindow v-if="store.animFloating" id="animation" title="Animation" icon="film" @dock="dock" @close="store.animFloating = false">
      <AnimationPanel floating />
    </FloatingWindow>
    <BuildDialog v-if="store.buildOpen" />
    <NewProjectDialog />
  <PackageManager />
</div>
</template>
