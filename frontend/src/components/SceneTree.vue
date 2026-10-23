<script setup lang="ts">
import { ref } from 'vue'
import { Scene } from '../lib/api'
import { add, addMenu, dropAsset } from '../lib/actions'
import { assetKind, type FileDrop } from '../lib/filedrop'
import { openMenu } from '../lib/menu'
import { call, refresh, select, store } from '../lib/store'
import Icon from './Icon.vue'
import TreeItem from './TreeItem.vue'

const filter = ref('')
const over = ref(false)

async function dropRoot(e: DragEvent) {
  over.value = false
  if (await dropAsset(e, 0)) return
  const id = Number(e.dataTransfer?.getData('text/entity'))
  if (!id) return
  await call(() => Scene.Reparent(id, 0, -1))
  await refresh()
}

// placeFiles adds models and prefabs dropped from the OS, under the row
// they were dropped on.
async function placeFiles(e: Event) {
  const d = (e as CustomEvent<FileDrop>).detail
  const row = document.elementFromPoint(d.x, d.y)?.closest<HTMLElement>('[data-node]')
  const parent = Number(row?.dataset.node ?? 0)
  for (const p of d.paths) {
    const kind = assetKind(p)
    if (kind === 'model' || kind === 'prefab') await add(parent, `${kind}:${p}`)
  }
}

function matches(name: string) {
  return !filter.value || name.toLowerCase().includes(filter.value.toLowerCase())
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-1 px-2 pb-2">
      <div class="relative flex-1">
        <input v-model="filter" class="field pl-7" placeholder="Filter" />
        <Icon name="crosshair" :size="13" class="absolute left-2 top-1.5 text-ink-500" />
      </div>
      <button class="icon-btn" title="Add" @click="(e) => openMenu(e, addMenu(0))"><Icon name="plus" :size="15" /></button>
    </div>
    <div
      data-drop="hierarchy"
      class="flex-1 overflow-auto px-1 pb-4"
      :class="over ? 'bg-amber-glow/[.03]' : ''"
      @click.self="select(0)"
      @dragover.prevent="over = true"
      @dragleave.self="over = false"
      @drop.prevent="dropRoot"
      @filedrop="placeFiles"
      @contextmenu.self.prevent="(e) => openMenu(e, addMenu(0))"
    >
      <button class="mb-1 flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12.5px] transition" :class="store.selected === 0 ? 'bg-amber-glow/12 text-ink-100' : 'text-ink-400 hover:bg-ink-800'" @click="select(0)">
        <Icon name="sliders" :size="13" class="text-ink-500" />
        <span>Scene</span>
      </button>
      <TreeItem v-for="n in store.tree.filter((n) => matches(n.name))" :key="n.id" :node="n" :depth="0" />
    </div>
  </div>
</template>
