<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Assets, EventsOn, Project, Scene } from '../lib/api'
import { openMenu, type MenuItem } from '../lib/menu'
import { ask, confirmAction } from '../lib/dialog'
import { call, refresh, refreshAssets, select, store, toast } from '../lib/store'
import type { asset } from '../wailsjs/go/models'
import ArtThumb from './ArtThumb.vue'
import Icon from './Icon.vue'

const folder = ref<'all' | 'scene' | 'prefab' | 'art' | 'script'>('all')
const search = ref('')
const picked = ref('')
const folders = [
  { id: 'all', label: 'All', icon: 'grid' },
  { id: 'scene', label: 'Scenes', icon: 'scene' },
  { id: 'prefab', label: 'Prefabs', icon: 'package' },
  { id: 'art', label: 'Art', icon: 'image' },
  { id: 'script', label: 'Scripts', icon: 'file' },
] as const
const items = computed(() =>
  store.assets.filter((a) => (folder.value === 'all' || a.kind === folder.value) && name(a.path).toLowerCase().includes(search.value.toLowerCase())),
)
let off: (() => void) | null = null

onMounted(async () => {
  await refreshAssets()
  off = EventsOn('assets', async () => {
    await refreshAssets()
    await refresh()
  })
})
onBeforeUnmount(() => off?.())

function name(p: string) {
  return p.split('/').pop()?.replace(/\.(prefab|scene)\.json$/, '').replace(/\.(json|xp|lua)$/, '') ?? p
}

function icon(k: string) {
  return { scene: 'scene', prefab: 'package', art: 'image', script: 'file' }[k] ?? 'file'
}

function dragStart(e: DragEvent, a: asset.Entry) {
  e.dataTransfer?.setData('text/asset', `${a.kind}:${a.guid || a.path}`)
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'copy'
}

async function open(a: asset.Entry) {
  if (a.kind === 'scene') {
    if (store.info?.unsaved && !(await confirmAction('Discard unsaved changes?', 'The open scene has changes that are not saved.'))) return
    await call(() => Project.OpenScene(a.path))
    store.selected = 0
    await refresh()
  } else if (a.kind === 'script') {
    await call(() => Assets.OpenExternal(a.path, 0))
  } else if (a.kind === 'prefab') {
    const id = await call(() => Scene.Add(0, 'prefab:' + a.path))
    if (id !== undefined) {
      await refresh()
      await select(id)
    }
  } else {
    const id = await call(() => Assets.AddSprite(a.path))
    if (id !== undefined) {
      await refresh()
      await select(id)
    }
  }
}

async function newScript() {
  const n = await ask('New script', 'new', { hint: 'Saved in scripts/ and opened in your code editor.', suffix: '.lua' })
  if (!n) return
  const rel = await call(() => Assets.NewScript(n))
  if (rel) {
    toast(rel, 'ok')
    await refreshAssets()
    await call(() => Assets.OpenExternal(rel, 0))
  }
}

async function newScene() {
  const n = await ask('New scene', 'level', { suffix: '.scene.json' })
  if (!n) return
  const rel = (n.includes('/') ? n : 'scenes/' + n).replace(/\.scene\.json$/, '') + '.scene.json'
  await call(() => Project.NewScene(rel))
  store.selected = 0
  await Promise.all([refresh(), refreshAssets()])
}

function createMenu(e: MouseEvent) {
  openMenu(e, [
    { label: 'Script', icon: 'file', run: newScript },
    { label: 'Scene', icon: 'scene', run: newScene },
  ])
}

function itemMenu(e: MouseEvent, a: asset.Entry) {
  picked.value = a.path
  const items: MenuItem[] = [{ label: a.kind === 'scene' ? 'Open' : a.kind === 'script' ? 'Edit' : 'Place', icon: 'open', run: () => open(a) }]
  items.push({ label: 'Rename', icon: 'file', run: () => rename(a) })
  if (a.kind === 'script') items.push({ label: 'New script', icon: 'plus', run: newScript })
  openMenu(e, items)
}

// rename moves the file and its .meta; references hold the GUID and keep working.
async function rename(a: asset.Entry) {
  const n = await ask('Rename', name(a.path), { hint: a.guid ? 'References use the asset id, nothing breaks.' : '', ok: 'Rename' })
  if (!n || n === name(a.path)) return
  const to = await call(() => Assets.Rename(a.path, n))
  if (to) {
    picked.value = to
    toast(to, 'ok')
    await Promise.all([refreshAssets(), refresh()])
  }
}

async function reload() {
  await call(() => Assets.Reload())
  await Promise.all([refreshAssets(), refresh()])
}
</script>

<template>
  <div class="flex h-full">
    <nav class="w-36 shrink-0 border-r border-ink-800 p-1.5">
      <button
        v-for="f in folders"
        :key="f.id"
        class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] transition"
        :class="folder === f.id ? 'bg-ink-800 text-ink-100' : 'text-ink-400 hover:text-ink-200'"
        @click="folder = f.id"
      >
        <Icon :name="f.icon" :size="13" :class="folder === f.id ? 'text-amber-glow' : ''" />
        <span class="flex-1">{{ f.label }}</span>
        <span class="font-mono text-[10px] text-ink-600">{{ f.id === 'all' ? store.assets.length : store.assets.filter((a) => a.kind === f.id).length }}</span>
      </button>
    </nav>
    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex h-8 shrink-0 items-center gap-1 border-b border-ink-800 px-2">
        <button class="btn h-6" @click="createMenu"><Icon name="plus" :size="12" /> Create</button>
        <span class="flex-1" />
        <input v-model="search" class="field h-6 w-44" placeholder="Search" />
        <button class="icon-btn size-6" title="Reload" @click="reload"><Icon name="refresh" :size="12" /></button>
      </div>
      <div class="flex-1 overflow-auto p-2" @click.self="picked = ''">
        <div class="grid grid-cols-[repeat(auto-fill,minmax(88px,1fr))] gap-1">
          <button
            v-for="a in items"
            :key="a.path"
            class="group flex h-[88px] flex-col items-center justify-center gap-1.5 rounded-lg p-1.5 transition"
            :class="picked === a.path ? 'bg-amber-glow/12 ring-1 ring-amber-soft' : 'hover:bg-ink-800'"
            draggable="true"
            :title="a.path"
            @dragstart="dragStart($event, a)"
            @click="picked = a.path"
            @dblclick="open(a)"
            @contextmenu.prevent="itemMenu($event, a)"
          >
            <div class="grid h-12 w-16 place-items-center rounded-md bg-ink-950/60">
              <ArtThumb v-if="a.kind === 'art'" :path="a.path" />
              <Icon v-else :name="icon(a.kind)" :size="22" :class="a.kind === 'prefab' ? 'text-amber-soft' : a.kind === 'script' ? 'text-ok' : 'text-ink-400'" />
            </div>
            <span class="w-full truncate text-center text-[11px] text-ink-300 group-hover:text-ink-100">{{ name(a.path) }}</span>
          </button>
        </div>
        <div v-if="!items.length" class="grid h-full place-items-center text-ink-600">-</div>
      </div>
    </div>
  </div>
</template>
