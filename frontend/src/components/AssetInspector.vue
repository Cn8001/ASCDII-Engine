<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Assets } from '../lib/api'
import { call, refreshAssets, store, toast } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import Icon from './Icon.vue'
import PropField from './PropField.vue'
import TexThumb from './TexThumb.vue'

// AssetInspector edits the asset picked in the Project window: material
// properties, or the import settings of textures and models.
const props = ref<prop.Prop[]>([])
const entry = computed(() => store.assets.find((a) => a.path === store.asset))
const kind = computed(() => entry.value?.kind ?? '')
const title = computed(() => store.asset.split('/').pop()?.replace(/\.(material\.json|png|jpe?g|gif|obj)$/i, '') ?? '')
const heading = computed(() => ({ material: 'Material', texture: 'Import Settings', model: 'Import Settings' })[kind.value] ?? '')

async function load() {
  props.value = store.asset && heading.value ? ((await call(() => Assets.Props(store.asset))) ?? []) : []
}

watch(() => store.asset, load, { immediate: true })

async function set(name: string, value: unknown) {
  await call(() => Assets.SetProp(store.asset, name, value))
  await load()
}

async function toArt() {
  const rel = await call(() => Assets.ConvertToArt(store.asset, 48))
  if (rel) {
    toast(rel, 'ok')
    await refreshAssets()
  }
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-2 border-b border-ink-750 px-3 py-2.5">
      <div class="grid size-8 place-items-center overflow-hidden rounded-lg bg-ink-800 text-amber-glow">
        <TexThumb v-if="kind === 'texture'" :path="store.asset" :size="32" />
        <Icon v-else :name="kind === 'material' ? 'sun' : kind === 'model' ? 'box' : 'file'" :size="15" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="truncate text-[13px] font-medium">{{ title }}</div>
        <div class="truncate font-mono text-[10.5px] text-ink-500">{{ store.asset }}</div>
      </div>
      <button class="icon-btn" title="Close" @click="store.asset = ''"><Icon name="x" :size="14" /></button>
    </div>
    <div class="flex-1 overflow-auto px-2 pb-6 pt-2">
      <section v-if="props.length" class="mb-2 overflow-hidden rounded-lg border border-ink-800 bg-ink-850/40">
        <div class="flex h-8 items-center gap-1.5 bg-ink-850 px-2 text-[12px] font-medium text-ink-200">
          <Icon :name="kind === 'material' ? 'sun' : 'sliders'" :size="12" class="text-ink-400" /> {{ heading }}
        </div>
        <div class="px-2 py-1">
          <PropField v-for="p in props" :key="p.name" :p="p" @set="set" />
        </div>
      </section>
      <div v-if="kind === 'texture'" class="flex flex-col gap-2 px-1">
        <div class="grid place-items-center rounded-lg border border-ink-800 bg-ink-950/60 p-3"><TexThumb :path="store.asset" :size="200" /></div>
        <button class="btn h-8 justify-center border border-ink-700 text-ink-300" @click="toArt"><Icon name="image" :size="13" /> Convert to Art</button>
      </div>
      <div v-if="!props.length && kind !== 'texture'" class="px-2 py-6 text-center text-[12px] text-ink-500">Nothing to edit</div>
    </div>
  </div>
</template>
