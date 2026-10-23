<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Anim, Scene, Viewport } from '../lib/api'
import { kindIcon, pretty } from '../lib/actions'
import { openMenu } from '../lib/menu'
import { animatable, assetPath, call, findNode, refresh, refreshClips, refreshProps, store, toast } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import AddComponent from './AddComponent.vue'
import Icon from './Icon.vue'
import PropField from './PropField.vue'

const node = computed(() => findNode(store.selected))
const behaviours = computed(() => new Set(node.value?.behaviours ?? []))
const groups = computed(() => {
  const list = store.selected ? store.props : store.sceneProps
  const out: { name: string; props: prop.Prop[]; enabled?: prop.Prop }[] = []
  for (const p of list) {
    const g = p.group || 'props'
    let entry = out.find((x) => x.name === g)
    if (!entry) out.push((entry = { name: g, props: [] }))
    if (p.name === `${g}.enabled`) entry.enabled = p
    else entry.props.push(p)
  }
  return out
})
const collapsed = ref<Record<string, boolean>>({})
let timer = 0

onMounted(() => {
  timer = window.setInterval(() => {
    if (store.frame.mode === 'play' && store.selected) refreshProps()
  }, 250)
})
onBeforeUnmount(() => clearInterval(timer))

async function set(name: string, value: unknown) {
  if (store.selected) {
    await call(() => Scene.SetProp(store.selected, name, value))
    const p = store.props.find((x) => x.name === name || name.startsWith(x.name + '.'))
    if (store.autoKey && p && animatable(p) && store.frame.mode === 'edit') {
      await call(() => Anim.AddKey(store.selected, name, store.frame.time))
      await refreshClips()
    }
  } else {
    await call(() => Scene.SetSceneProp(name, value))
  }
  if (instance.value || name === 'name' || name === 'visible' || name.endsWith('.enabled')) await refresh()
  else await refreshProps()
}

async function key(name: string) {
  if (!store.selected) return
  await call(() => Anim.AddKey(store.selected, name, store.frame.time))
  toast(`${name} @ ${store.frame.time.toFixed(2)}s`, 'ok')
  await refreshClips()
}

// instance is the prefab instance the selection belongs to, for the banner.
const instance = computed(() => {
  const n = node.value
  if (!n) return null
  if (n.prefab) return n
  if (!n.inherited) return null
  const up = (id: number, nodes = store.tree, parent: typeof n | null = null): typeof n | null => {
    for (const x of nodes) {
      if (x.id === id) return parent
      const found = up(id, x.children ?? [], x.prefab ? x : parent)
      if (found) return found
    }
    return null
  }
  return up(n.id)
})

async function apply(name: string, id = store.selected) {
  const ok = await call(() => Scene.Apply(id, name).then(() => true))
  if (ok) toast('Applied to prefab', 'ok')
  await refresh()
}

async function revert(name: string, id = store.selected) {
  await call(() => Scene.Revert(id, name))
  await refresh()
}

function propMenu(e: MouseEvent, name: string) {
  const p = store.props.find((x) => x.name === name)
  if (!store.selected || !p?.overridden) return
  openMenu(e, [
    { label: 'Apply to prefab', icon: 'package', run: () => apply(name) },
    { label: 'Revert', icon: 'undo', run: () => revert(name) },
  ])
}

async function removeBehaviour(kind: string) {
  await call(() => Scene.RemoveBehaviour(store.selected, kind))
  await refresh()
}

function title(g: string) {
  if (g === 'entity') return 'Transform'
  return pretty(g)
}

function icon(g: string) {
  if (g === 'entity') return 'move'
  if (g === 'script') return 'file'
  if (g === 'camera') return 'crosshair'
  if (g === 'environment') return 'sun'
  return behaviours.value.has(g) ? 'sliders' : kindIcon(g)
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-2 border-b border-ink-750 px-3 py-2.5">
      <div class="grid size-8 place-items-center rounded-lg bg-ink-800" :class="node?.prefab ? 'text-amber-soft' : 'text-amber-glow'">
        <Icon :name="store.selected ? (node?.prefab ? 'package' : kindIcon(node?.kind ?? '')) : 'sliders'" :size="15" />
      </div>
      <div class="min-w-0 flex-1">
        <div class="truncate text-[13px] font-medium">{{ store.selected ? node?.name : 'Scene' }}</div>
        <div class="truncate font-mono text-[10.5px]" :class="node?.prefab ? 'text-amber-soft' : 'text-ink-500'">{{ store.selected ? (node?.prefab ? assetPath(node.prefab) : pretty(node?.kind || 'group')) : store.info?.scene }}</div>
      </div>
      <button v-if="store.selected" class="icon-btn" title="Focus  F" @click="Viewport.Focus(store.selected)"><Icon name="crosshair" :size="14" /></button>
    </div>

    <div v-if="instance" class="flex items-center gap-2 border-b border-ink-750 bg-amber-glow/[.06] px-3 py-2 text-[12px]">
      <Icon name="package" :size="13" class="text-amber-soft" />
      <span class="min-w-0 flex-1 truncate text-ink-300">{{ assetPath(instance.prefab ?? '').replace('prefabs/', '').replace('.prefab.json', '') }}<span v-if="instance.overrides" class="ml-1.5 text-amber-glow">{{ instance.overrides }} override{{ instance.overrides === 1 ? '' : 's' }}</span></span>
      <button class="btn h-6 px-2" :disabled="!instance.overrides" @click="apply('', instance.id)">Apply</button>
      <button class="btn h-6 px-2" :disabled="!instance.overrides" @click="revert('', instance.id)">Revert</button>
    </div>
    <div class="flex-1 overflow-auto px-2 pb-6 pt-2">
      <section v-for="g in groups" :key="g.name" class="mb-2 animate-fade overflow-hidden rounded-lg border border-ink-800 bg-ink-850/40">
        <div class="flex h-8 items-center gap-1.5 bg-ink-850 px-2">
          <button class="text-ink-500" @click="collapsed[g.name] = !collapsed[g.name]"><Icon :name="collapsed[g.name] ? 'chevright' : 'chevdown'" :size="12" /></button>
          <input v-if="g.enabled" type="checkbox" class="accent-[var(--color-amber-glow)]" :checked="!!g.enabled.value" :disabled="g.enabled.locked" @change="set(g.enabled.name, ($event.target as HTMLInputElement).checked)" />
          <Icon :name="icon(g.name)" :size="12" class="text-ink-400" />
          <button class="flex-1 truncate text-left text-[12px] font-medium" :class="g.enabled && !g.enabled.value ? 'text-ink-500 line-through' : 'text-ink-200'" @click="collapsed[g.name] = !collapsed[g.name]">{{ title(g.name) }}</button>
          <button v-if="behaviours.has(g.name) && !node?.prefab && !node?.inherited" class="icon-btn size-6" title="Remove" @click="removeBehaviour(g.name)"><Icon name="x" :size="12" /></button>
        </div>
        <div v-show="!collapsed[g.name]" class="px-2 py-1">
          <PropField v-for="p in g.props" :key="p.name" :p="p" :keyable="!!store.selected" @set="set" @key="key" @menu="propMenu" />
        </div>
      </section>
      <AddComponent v-if="store.selected && !node?.prefab && !node?.inherited" />
    </div>
  </div>
</template>
