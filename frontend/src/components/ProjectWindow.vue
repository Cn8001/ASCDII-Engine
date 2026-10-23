<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Assets, EventsOn, Packages, Project, Scene } from '../lib/api'
import { openMenu, type MenuItem } from '../lib/menu'
import { ask, confirmAction } from '../lib/dialog'
import { call, refresh, refreshAssets, select, store, toast } from '../lib/store'
import { previewSound } from '../lib/audio'
import type { FileDrop } from '../lib/filedrop'
import type { asset } from '../wailsjs/go/models'
import SpriteThumb from './SpriteThumb.vue'
import TexThumb from './TexThumb.vue'
import Icon from './Icon.vue'

const kinds = [
  { id: 'all', label: 'All types' },
  { id: 'scene', label: 'Scenes' },
  { id: 'prefab', label: 'Prefabs' },
  { id: 'model', label: 'Models' },
  { id: 'texture', label: 'Textures' },
  { id: 'material', label: 'Materials' },
  { id: 'audio', label: 'Audio' },
  { id: 'animator', label: 'Animators' },
  { id: 'sprite', label: 'Sprites' },
  { id: 'script', label: 'Scripts' },
] as const
const kind = ref<string>('all')
const search = ref('')
const picked = ref('')
const cwd = computed({ get: () => store.folder, set: (v: string) => (store.folder = v) }) // "" is the project folder
const folders = ref<string[]>([])
const open = ref<Record<string, boolean>>({})
const over = ref('')

// Installed packages live under packages/ and are read only.
const isPkg = (p: string) => p === 'packages' || p.startsWith('packages/')
const pkgOf = (p: string) => p.split('/')[1] ?? ''
const locked = computed(() => isPkg(cwd.value))
const dirOf = (p: string) => (p.includes('/') ? p.slice(0, p.lastIndexOf('/')) : '')
const base = (p: string) => p.split('/').pop() ?? p
// searching or filtering looks through every folder.
const flat = computed(() => search.value !== '' || kind.value !== 'all')
const subfolders = computed(() => (flat.value ? [] : folders.value.filter((f) => dirOf(f) === cwd.value && f !== 'packages')))
const items = computed(() =>
  store.assets.filter(
    (a) => (flat.value || dirOf(a.path) === cwd.value) && (kind.value === 'all' || a.kind === kind.value) && name(a.path).toLowerCase().includes(search.value.toLowerCase()),
  ),
)
const crumbs = computed(() => {
  const parts = cwd.value ? cwd.value.split('/') : []
  return parts.map((p, i) => ({ label: p, path: parts.slice(0, i + 1).join('/') }))
})
// tree shows a folder when every folder above it is open.
const visible = (f: string) => {
  const parts = f.split('/')
  return parts.slice(0, -1).every((_, i) => open.value[parts.slice(0, i + 1).join('/')])
}
const tree = computed(() => folders.value.filter((f) => !isPkg(f) && visible(f)))
const pkgTree = computed(() => folders.value.filter((f) => isPkg(f) && f !== 'packages' && visible(f)))
let off: (() => void) | null = null

async function reloadAll() {
  await refreshAssets()
  folders.value = (await call(() => Assets.Folders())) ?? []
}

onMounted(async () => {
  await reloadAll()
  off = EventsOn('assets', async () => {
    await reloadAll()
    await refresh()
  })
})
onBeforeUnmount(() => off?.())
watch(() => store.assetsVersion, reloadAll)
// Another project has other folders: start at its top and read them again.
watch(
  () => store.info?.dir,
  async () => {
    store.folder = ''
    open.value = {}
    await reloadAll()
  },
)

function name(p: string) {
  return base(p).replace(/\.(prefab|scene|material|animator|sprite)\.json$/, '').replace(/\.(json|xp|lua|obj|png|jpe?g|gif|wav|mp3)$/i, '')
}

function icon(k: string) {
  return { scene: 'scene', prefab: 'package', model: 'box', texture: 'image', material: 'sun', sprite: 'image', script: 'file', audio: 'music', animator: 'graph' }[k] ?? 'file'
}

function go(path: string) {
  cwd.value = path
  search.value = ''
  kind.value = 'all'
  const parts = path.split('/')
  parts.forEach((_, i) => (open.value[parts.slice(0, i + 1).join('/')] = true))
}

// within prefixes a new file's name with the open folder, so things are
// created where the user is looking.
const within = (n: string) => (cwd.value && !n.includes('/') ? `${cwd.value}/${n}` : n)

async function fileDrop(e: Event) {
  const paths = (e as CustomEvent<FileDrop>).detail.paths
  picked.value = paths[0] ?? ''
  folders.value = (await call(() => Assets.Folders())) ?? []
}

function dragStart(e: DragEvent, a: asset.Entry) {
  e.dataTransfer?.setData('text/asset', `${a.kind}:${a.guid || a.path}`)
  e.dataTransfer?.setData('text/path', a.path)
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'copyMove'
}

function dragFolder(e: DragEvent, f: string) {
  if (isPkg(f)) return
  e.dataTransfer?.setData('text/path', f)
}

function dragOver(e: DragEvent, target: string) {
  if (!e.dataTransfer?.types.includes('text/path') || isPkg(target)) return
  e.preventDefault()
  over.value = target
}

// dropInto moves a dragged asset or folder into target.
async function dropInto(e: DragEvent, target: string) {
  over.value = ''
  const from = e.dataTransfer?.getData('text/path')
  if (!from || dirOf(from) === target || from === target) return
  const to = await call(() => Assets.Move(from, target))
  if (to) {
    toast(to, 'ok')
    await reloadAll()
    await refresh()
  }
}

async function openAsset(a: asset.Entry) {
  if (a.kind === 'scene') {
    if (store.info?.unsaved && !(await confirmAction('Discard unsaved changes?', 'The open scene has changes that are not saved.'))) return
    await call(() => Project.OpenScene(a.path))
    store.selected = 0
    await refresh()
  } else if (a.kind === 'script') {
    await call(() => Assets.OpenExternal(a.path, 0))
  } else if (a.kind === 'texture' || a.kind === 'material') {
    pick(a)
  } else if (a.kind === 'audio') {
    previewSound(a.path)
  } else if (a.kind === 'animator') {
    store.animator = a.path
  } else if (a.kind === 'prefab') {
    const ok = await call(() => Scene.OpenPrefab(a.path).then(() => true))
    if (ok) {
      store.selected = 0
      await refresh()
    }
  } else if (a.kind === 'model') {
    const id = await call(() => Scene.Add(0, `${a.kind}:${a.path}`))
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

async function place(a: asset.Entry) {
  const id = await call(() => Scene.Add(0, `${a.kind}:${a.path}`))
  if (id !== undefined) {
    await refresh()
    await select(id)
  }
}

// pick selects a tile; assets with settings open in the inspector.
function pick(a: asset.Entry) {
  picked.value = a.path
  if (['material', 'texture', 'model'].includes(a.kind)) store.asset = a.path
}

async function newScript() {
  const n = await ask('New script', 'new', { hint: 'Opened in your code editor.', suffix: '.lua' })
  if (!n) return
  const rel = await call(() => Assets.NewScript(within(n)))
  if (rel) {
    toast(rel, 'ok')
    await reloadAll()
    await call(() => Assets.OpenExternal(rel, 0))
  }
}

async function newScene() {
  const n = await ask('New scene', 'level', { suffix: '.scene.json' })
  if (!n) return
  const rel = (n.includes('/') ? n : cwd.value ? `${cwd.value}/${n}` : n).replace(/\.scene\.json$/, '') + '.scene.json'
  await call(() => Project.NewScene(rel))
  store.selected = 0
  await Promise.all([refresh(), reloadAll()])
}

async function newMaterial(texture = '') {
  const n = await ask('New material', texture ? name(texture) : 'material', { suffix: '.material.json' })
  if (!n) return
  const rel = await call(() => Assets.CreateMaterial(within(n)))
  if (!rel) return
  if (texture) await call(() => Assets.SetProp(rel, 'texture', isPkg(texture) ? (store.assets.find((a) => a.path === texture)?.guid ?? texture) : texture))
  toast(rel, 'ok')
  await reloadAll()
  store.asset = rel
}

async function newController() {
  const n = await ask('New animator controller', 'animator', { suffix: '.animator.json' })
  if (!n) return
  const rel = await call(() => Assets.CreateController(within(n)))
  if (!rel) return
  toast(rel, 'ok')
  await reloadAll()
  store.animator = rel
}

async function newFolder(parent = cwd.value) {
  const n = await ask('New folder', 'folder', { ok: 'Create' })
  if (!n) return
  const rel = await call(() => Assets.CreateFolder(parent ? `${parent}/${n}` : n))
  if (rel) {
    await reloadAll()
    open.value[parent] = true
  }
}

async function toSprite(a: asset.Entry) {
  const rel = await call(() => Assets.ConvertToSprite(a.path, 48))
  if (rel) {
    toast(rel, 'ok')
    await reloadAll()
  }
}

async function remove(path: string, folder: boolean) {
  const what = folder ? 'the folder and everything in it' : 'the file and its .meta'
  if (!(await confirmAction(`Delete ${base(path)}?`, `This deletes ${what} from disk. References to it show as missing.`, 'Delete'))) return
  const ok = await call(() => Assets.Delete(path).then(() => true))
  if (!ok) return
  if (store.asset.startsWith(path)) store.asset = ''
  if (cwd.value === path || cwd.value.startsWith(path + '/')) cwd.value = dirOf(path)
  await reloadAll()
  await refresh()
}

async function renameFolder(f: string) {
  const n = await ask('Rename folder', base(f), { ok: 'Rename' })
  if (!n || n === base(f)) return
  const to = await call(() => Assets.RenameFolder(f, n))
  if (!to) return
  if (cwd.value === f || cwd.value.startsWith(f + '/')) cwd.value = to + cwd.value.slice(f.length)
  await reloadAll()
  await refresh()
}

// rename moves the file and its .meta; references hold the GUID and keep working.
async function rename(a: asset.Entry) {
  const n = await ask('Rename', name(a.path), { hint: a.guid ? 'References use the asset id, nothing breaks.' : '', ok: 'Rename' })
  if (!n || n === name(a.path)) return
  const to = await call(() => Assets.Rename(a.path, n))
  if (to) {
    picked.value = to
    toast(to, 'ok')
    await Promise.all([reloadAll(), refresh()])
  }
}

function createItems(): MenuItem[] {
  if (locked.value) return [{ label: 'Packages are read only', icon: 'lock', disabled: true }]
  return [
    { label: 'Folder', icon: 'folder', run: () => newFolder() },
    { label: '-' },
    { label: 'Script', icon: 'file', run: newScript },
    { label: 'Scene', icon: 'scene', run: newScene },
    { label: 'Material', icon: 'sun', run: () => newMaterial() },
    { label: 'Animator Controller', icon: 'graph', run: newController },
  ]
}

async function exportFolder(f: string) {
  const name = await ask('Export package', (f ? base(f) : store.info?.name ?? 'package').toLowerCase().replace(/[^a-z0-9.-]+/g, '-'), { hint: 'Lower case letters, digits, dots and dashes.', ok: 'Next' })
  if (!name) return
  const version = await ask('Version', '1.0.0', { ok: 'Export' })
  if (!version) return
  const out = await call(() => Packages.Export(f, { format: 1, name, version, engine: '', description: '', author: '', modules: [] } as never))
  if (out) toast(out, 'ok')
}

function showPackages() {
  store.packagesOpen = true
}

async function removePackage(f: string) {
  const name = pkgOf(f)
  if (!(await confirmAction(`Remove ${name}?`, 'Its files leave the project. Anything that uses them shows as missing.', 'Remove'))) return
  const ok = await call(() => Packages.Remove(name).then(() => true))
  if (!ok) return
  if (isPkg(cwd.value)) cwd.value = ''
  toast(`${name} removed`, 'ok')
  await reloadAll()
  await refresh()
}

function folderMenu(e: MouseEvent, f: string) {
  if (isPkg(f)) {
    const items: MenuItem[] = [{ label: 'Open', icon: 'open', run: () => go(f) }]
    if (f.split('/').length === 2) items.push({ label: 'Package Manager', icon: 'package', run: showPackages }, { label: '-' }, { label: 'Remove Package', icon: 'trash', danger: true, run: () => removePackage(f) })
    openMenu(e, items)
    return
  }
  openMenu(e, [
    { label: 'Open', icon: 'open', run: () => go(f) },
    { label: 'New Folder', icon: 'folder', run: () => newFolder(f) },
    { label: 'Rename', icon: 'file', run: () => renameFolder(f) },
    { label: 'Export Package', icon: 'package', run: () => exportFolder(f) },
    { label: '-' },
    { label: 'Delete', icon: 'trash', danger: true, run: () => remove(f, true) },
  ])
}

function itemMenu(e: MouseEvent, a: asset.Entry) {
  picked.value = a.path
  const items: MenuItem[] = []
  if (a.kind === 'texture') {
    items.push({ label: 'Create Material', icon: 'sun', run: () => newMaterial(a.path) }, { label: 'Convert to Sprite', icon: 'image', run: () => toSprite(a) })
  } else if (a.kind === 'prefab') {
    items.push({ label: 'Open Prefab', icon: 'open', run: () => openAsset(a) }, { label: 'Place', icon: 'plus', run: () => place(a) })
  } else if (a.kind === 'audio') {
    items.push({ label: 'Play', icon: 'music', run: () => previewSound(a.path) })
  } else if (a.kind === 'animator') {
    items.push({ label: 'Open Animator', icon: 'graph', run: () => (store.animator = a.path) })
  } else if (a.kind === 'material' || a.kind === 'model') {
    items.push({ label: a.kind === 'model' ? 'Place' : 'Edit', icon: 'open', run: () => (a.kind === 'model' ? openAsset(a) : pick(a)) })
    if (a.kind === 'model') items.push({ label: 'Import Settings', icon: 'sliders', run: () => pick(a) })
  } else {
    items.push({ label: a.kind === 'scene' ? 'Open' : a.kind === 'script' ? 'Edit' : 'Place', icon: 'open', run: () => openAsset(a) })
  }
  if (flat.value) items.push({ label: 'Show in Folder', icon: 'folder', run: () => { go(dirOf(a.path)); picked.value = a.path } })
  if (!isPkg(a.path)) {
    items.push({ label: 'Rename', icon: 'file', run: () => rename(a) })
    items.push({ label: '-' }, { label: 'Delete', icon: 'trash', danger: true, run: () => remove(a.path, false) })
  }
  openMenu(e, items)
}

async function reload() {
  await call(() => Assets.Reload())
  await Promise.all([reloadAll(), refresh()])
}
</script>

<template>
  <div class="flex h-full">
    <nav class="w-44 shrink-0 overflow-auto border-r border-ink-800 p-1.5" @contextmenu.self.prevent="(e) => openMenu(e, createItems())">
      <button
        class="flex h-7 w-full items-center gap-1.5 rounded-md px-2 text-left text-[12px] transition"
        :class="[cwd === '' && !flat ? 'bg-ink-800 text-ink-100' : 'text-ink-400 hover:text-ink-200', over === '/' ? 'ring-1 ring-amber-soft' : '']"
        @click="go('')"
        @dragover="(e) => dragOver(e, '/')"
        @dragleave="over = ''"
        @drop.prevent="(e) => dropInto(e, '')"
        @contextmenu.prevent="(e) => openMenu(e, [{ label: 'New Folder', icon: 'folder', run: () => newFolder('') }, { label: 'Export Package', icon: 'package', run: () => exportFolder('') }])"
      >
        <Icon name="folder" :size="13" :class="cwd === '' ? 'text-amber-glow' : ''" />
        <span class="flex-1 truncate">Assets</span>
      </button>
      <button
        v-for="f in tree"
        :key="f"
        class="flex h-7 w-full items-center gap-1 rounded-md pr-2 text-left text-[12px] transition"
        :class="[cwd === f && !flat ? 'bg-ink-800 text-ink-100' : 'text-ink-400 hover:text-ink-200', over === f ? 'ring-1 ring-amber-soft' : '']"
        :style="{ paddingLeft: f.split('/').length * 12 + 'px' }"
        draggable="true"
        @dragstart="(e) => dragFolder(e, f)"
        @click="go(f)"
        @contextmenu.prevent="(e) => folderMenu(e, f)"
        @dragover="(e) => dragOver(e, f)"
        @dragleave="over = ''"
        @drop.prevent="(e) => dropInto(e, f)"
      >
        <span class="grid size-4 place-items-center text-ink-500" :class="folders.some((x) => dirOf(x) === f) ? '' : 'invisible'" @click.stop="open[f] = !open[f]">
          <Icon :name="open[f] ? 'chevdown' : 'chevright'" :size="11" />
        </span>
        <Icon name="folder" :size="13" :class="cwd === f ? 'text-amber-glow' : ''" />
        <span class="flex-1 truncate">{{ base(f) }}</span>
      </button>
      <button
        class="mt-2 flex h-7 w-full items-center gap-1.5 rounded-md px-2 text-left text-[12px] transition"
        :class="cwd === 'packages' && !flat ? 'bg-ink-800 text-ink-100' : 'text-ink-400 hover:text-ink-200'"
        @click="go('packages')"
        @contextmenu.prevent="(e) => openMenu(e, [{ label: 'Package Manager', icon: 'package', run: showPackages }])"
      >
        <Icon name="package" :size="13" :class="cwd === 'packages' ? 'text-amber-glow' : ''" />
        <span class="flex-1 truncate">Packages</span>
      </button>
      <button
        v-for="f in pkgTree"
        :key="f"
        class="flex h-7 w-full items-center gap-1 rounded-md pr-2 text-left text-[12px] transition"
        :class="cwd === f && !flat ? 'bg-ink-800 text-ink-100' : 'text-ink-400 hover:text-ink-200'"
        :style="{ paddingLeft: (f.split('/').length - 1) * 12 + 'px' }"
        @click="go(f)"
        @contextmenu.prevent="(e) => folderMenu(e, f)"
      >
        <span class="grid size-4 place-items-center text-ink-500" :class="folders.some((x) => dirOf(x) === f) ? '' : 'invisible'" @click.stop="open[f] = !open[f]">
          <Icon :name="open[f] ? 'chevdown' : 'chevright'" :size="11" />
        </span>
        <Icon :name="f.split('/').length === 2 ? 'package' : 'folder'" :size="13" :class="cwd === f ? 'text-amber-glow' : 'text-amber-soft/70'" />
        <span class="flex-1 truncate">{{ base(f) }}</span>
      </button>
    </nav>
    <div class="flex min-w-0 flex-1 flex-col">
      <div class="flex h-8 shrink-0 items-center gap-1 border-b border-ink-800 px-2">
        <button class="btn h-6" @click="(e) => openMenu(e, createItems())"><Icon name="plus" :size="12" /> Create</button>
        <div class="ml-1 flex min-w-0 flex-1 items-center gap-0.5 truncate text-[12px] text-ink-400">
          <button class="rounded px-1 hover:text-ink-100" @click="go('')" @dragover="(e) => dragOver(e, '/')" @drop.prevent="(e) => dropInto(e, '')">Assets</button>
          <template v-for="c in crumbs" :key="c.path">
            <Icon name="chevright" :size="10" class="text-ink-600" />
            <Icon v-if="c.path === 'packages'" name="package" :size="11" class="text-amber-soft" />
            <button class="truncate rounded px-1 hover:text-ink-100" @click="go(c.path)" @dragover="(e) => dragOver(e, c.path)" @drop.prevent="(e) => dropInto(e, c.path)">{{ c.label }}</button>
          </template>
        </div>
        <span v-if="locked" class="mr-1 flex items-center gap-1 text-[11px] text-ink-500"><Icon name="lock" :size="11" /> read only</span>
        <select v-model="kind" class="h-6 rounded-md bg-ink-800 px-1.5 text-[11.5px] text-ink-300 outline-none ring-1 ring-ink-700">
          <option v-for="k in kinds" :key="k.id" :value="k.id">{{ k.label }}</option>
        </select>
        <input v-model="search" class="field h-6 w-40" placeholder="Search" />
        <button class="icon-btn size-6" title="Reload" @click="reload"><Icon name="refresh" :size="12" /></button>
      </div>
      <div class="flex-1 overflow-auto p-2" :data-drop="locked ? undefined : 'project'" :data-folder="cwd" @click.self="picked = ''" @filedrop="fileDrop" @contextmenu.self.prevent="(e) => openMenu(e, createItems())">
        <div class="grid grid-cols-[repeat(auto-fill,minmax(88px,1fr))] gap-1" @contextmenu.self.prevent="(e) => openMenu(e, createItems())">
          <button
            v-for="f in subfolders"
            :key="f"
            class="group flex h-[88px] flex-col items-center justify-center gap-1.5 rounded-lg p-1.5 transition"
            :class="over === f ? 'bg-amber-glow/12 ring-1 ring-amber-soft' : 'hover:bg-ink-800'"
            :draggable="!isPkg(f)"
            :title="f"
            @dragstart="(e) => dragFolder(e, f)"
            @dblclick="go(f)"
            @contextmenu.prevent="(e) => folderMenu(e, f)"
            @dragover="(e) => dragOver(e, f)"
            @dragleave="over = ''"
            @drop.prevent.stop="(e) => dropInto(e, f)"
          >
            <div class="grid h-12 w-16 place-items-center"><Icon :name="f.split('/').length === 2 && isPkg(f) ? 'package' : 'folder'" :size="30" class="text-amber-soft/80" /></div>
            <span class="w-full truncate text-center text-[11px] text-ink-300 group-hover:text-ink-100">{{ base(f) }}</span>
          </button>
          <button
            v-for="a in items"
            :key="a.path"
            class="group flex h-[88px] flex-col items-center justify-center gap-1.5 rounded-lg p-1.5 transition"
            :class="picked === a.path ? 'bg-amber-glow/12 ring-1 ring-amber-soft' : 'hover:bg-ink-800'"
            draggable="true"
            :title="a.path"
            @dragstart="dragStart($event, a)"
            @click="pick(a)"
            @dblclick="openAsset(a)"
            @contextmenu.prevent="itemMenu($event, a)"
          >
            <div class="grid h-12 w-16 place-items-center rounded-md bg-ink-950/60">
              <SpriteThumb v-if="a.kind === 'sprite'" :path="a.path" />
              <TexThumb v-else-if="a.kind === 'texture'" :path="a.path" />
              <Icon v-else :name="icon(a.kind)" :size="22" :class="a.kind === 'prefab' ? 'text-amber-soft' : a.kind === 'script' || a.kind === 'audio' ? 'text-ok' : a.kind === 'model' || a.kind === 'animator' ? 'text-ink-200' : 'text-ink-400'" />
            </div>
            <span class="w-full truncate text-center text-[11px] text-ink-300 group-hover:text-ink-100">{{ name(a.path) }}</span>
          </button>
        </div>
        <div v-if="!items.length && !subfolders.length" class="grid h-[calc(100%-1rem)] place-items-center text-[12px] text-ink-600">{{ flat ? 'No matches' : 'Drop files here' }}</div>
      </div>
    </div>
  </div>
</template>
