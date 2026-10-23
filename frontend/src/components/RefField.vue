<script setup lang="ts">
import { computed, ref } from 'vue'
import { kindLabel } from '../lib/actions'
import { openMenu, type MenuItem } from '../lib/menu'
import { allNodes, assetPath, store } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import Icon from './Icon.vue'

// RefField is a reference slot, as in Unity: drop an object from the
// hierarchy, a component card or a prefab from the Project window on it.
const props = defineProps<{ p: prop.Prop }>()
const emit = defineEmits<{ ref: [name: string, target: number, key: string]; set: [name: string, value: unknown] }>()
const over = ref(false)
const kind = computed(() => props.p.options?.[0] ?? '')
// asset slots take files from the Project window instead of scene objects.
const asset = computed(() => ['prefab', 'texture', 'material', 'controller', 'audio'].includes(props.p.type))
const empty = computed(() => !props.p.value)
const label = computed(() => {
  if (asset.value) return assetPath(String(props.p.value)).split('/').pop()!.replace(/\.(prefab|material|animator)\.json$/, '')
  return props.p.display || String(props.p.value)
})
const names: Record<string, string> = { object: 'Object', prefab: 'Prefab', texture: 'Texture', material: 'Material', controller: 'Animator', audio: 'Sound' }
const what = computed(() => names[props.p.type] ?? (kind.value ? kindLabel(kind.value) : 'Component'))
const icon = computed(() => ({ object: 'box', prefab: 'package', texture: 'image', material: 'sun', controller: 'graph', audio: 'music' })[props.p.type] ?? 'sliders')
const missing = computed(() => props.p.display === 'Missing')

function accepts(e: DragEvent): boolean {
  const types = e.dataTransfer?.types ?? []
  if (asset.value) return types.includes('text/asset')
  return types.includes('text/entity') || (props.p.type === 'component' && types.includes('text/component'))
}

function enter(e: DragEvent) {
  if (!accepts(e) || props.p.locked) return
  e.preventDefault()
  over.value = true
}

function drop(e: DragEvent) {
  over.value = false
  if (props.p.locked) return
  const dt = e.dataTransfer
  if (!dt) return
  if (asset.value) {
    const [k, ...rest] = dt.getData('text/asset').split(':')
    if (k === props.p.type || (k === 'animator' && props.p.type === 'controller')) emit('set', props.p.name, rest.join(':'))
    return
  }
  const comp = dt.getData('text/component')
  if (comp) {
    const [id, key] = comp.split('|')
    emit('ref', props.p.name, Number(id), props.p.type === 'object' ? '' : key)
    return
  }
  const id = Number(dt.getData('text/entity'))
  if (id) emit('ref', props.p.name, id, '')
}

// pick lists what fits the slot, for when dragging is not handy.
function pick(e: MouseEvent) {
  const items: MenuItem[] = [{ label: 'None', icon: 'x', run: () => (asset.value ? emit('set', props.p.name, '') : emit('ref', props.p.name, 0, '')) }, { label: '-' }]
  if (asset.value) {
    const want = props.p.type === 'controller' ? 'animator' : props.p.type
    for (const a of store.assets.filter((x) => x.kind === want)) {
      items.push({ label: a.path.split('/').pop()!.replace(/\.(prefab|material|animator)\.json$/, ''), icon: icon.value, run: () => emit('set', props.p.name, a.guid || a.path) })
    }
  } else {
    for (const n of allNodes()) {
      if (n.id === store.selected && props.p.type === 'object') continue
      if (props.p.type === 'component' && kind.value && !(n.components ?? []).includes(kind.value)) continue
      if (props.p.type === 'component' && !(n.components ?? []).length) continue
      items.push({ label: n.name, icon: n.prefab ? 'package' : 'box', run: () => emit('ref', props.p.name, n.id, '') })
    }
  }
  if (items.length === 2) items.push({ label: 'Nothing fits', disabled: true })
  openMenu(e, items)
}
</script>

<template>
  <div
    class="flex h-7 min-w-0 items-center gap-1.5 rounded-md bg-ink-900 pl-2 pr-0.5 ring-1 transition"
    :class="[over ? 'ring-amber-glow bg-amber-glow/10' : 'ring-ink-700', p.locked ? 'opacity-60' : '']"
    @dragenter="enter"
    @dragover="enter"
    @dragleave="over = false"
    @drop.prevent.stop="drop"
  >
    <Icon :name="icon" :size="12" :class="empty ? 'text-ink-600' : missing ? 'text-bad' : 'text-amber-soft'" />
    <span class="min-w-0 flex-1 truncate text-[12px]" :class="empty ? 'text-ink-500' : missing ? 'text-bad' : 'text-ink-200'">{{ empty ? `None (${what})` : label }}</span>
    <button class="icon-btn size-6 shrink-0" title="Pick" :disabled="p.locked" @click="pick"><Icon name="crosshair" :size="12" /></button>
  </div>
</template>
