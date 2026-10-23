<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Anim, Scene, Viewport } from '../lib/api'
import { kindIcon, kindLabel } from '../lib/actions'
import { openMenu } from '../lib/menu'
import { animatable, assetPath, call, findNode, refresh, refreshClips, refreshProps, store, toast } from '../lib/store'
import type { prop } from '../wailsjs/go/models'
import AddComponent from './AddComponent.vue'
import Icon from './Icon.vue'
import PropField from './PropField.vue'

interface Card {
  key: string
  kind: string
  index: number // position in the object's component list, -1 for transform and scene groups
  props: prop.Prop[]
  enabled?: prop.Prop
}

const header = ['name', 'tag', 'visible', 'solid']
const node = computed(() => findNode(store.selected))
const own = computed(() => !!node.value && !node.value.prefab && !node.value.inherited)
const field = (name: string) => store.props.find((p) => p.name === name)
const cards = computed(() => {
  const list = store.selected ? store.props : store.sceneProps
  const out: Card[] = []
  let index = 0
  for (const p of list) {
    if (store.selected && header.includes(p.name)) continue
    const key = p.group || 'props'
    let c = out.find((x) => x.key === key)
    if (!c) {
      const component = store.selected && key !== 'transform'
      out.push((c = { key, kind: key.split('#')[0], index: component ? index++ : -1, props: [] }))
    }
    if (p.name === `${key}.enabled`) c.enabled = p
    else c.props.push(p)
  }
  return out
})
const count = computed(() => cards.value.filter((c) => c.index >= 0).length)
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
  if (instance.value || header.includes(name) || name.endsWith('.enabled')) await refresh()
  else await refreshProps()
}

async function setRef(name: string, target: number, key: string) {
  await call(() => Scene.SetRef(store.selected, name, target, key))
  await refreshProps()
}

// dragCard lets a component card be dropped on a reference slot.
function dragCard(e: DragEvent, c: Card) {
  e.dataTransfer?.setData('text/component', `${store.selected}|${c.key}`)
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

async function edit(fn: () => Promise<unknown>) {
  await call(fn)
  await refresh()
}

function cardMenu(e: MouseEvent, c: Card) {
  const id = store.selected
  openMenu(e, [
    { label: 'Reset', icon: 'refresh', run: () => edit(() => Scene.ResetComponent(id, c.index)) },
    { label: 'Move Up', icon: 'arrowup', disabled: c.index === 0, run: () => edit(() => Scene.MoveComponent(id, c.index, c.index - 1)) },
    { label: 'Move Down', icon: 'arrowdown', disabled: c.index === count.value - 1, run: () => edit(() => Scene.MoveComponent(id, c.index, c.index + 1)) },
    { label: '-' },
    { label: 'Remove Component', icon: 'trash', danger: true, run: () => edit(() => Scene.RemoveComponent(id, c.index)) },
  ])
}

function controllerOf(c: Card): string {
  const v = c.props.find((p) => p.name.endsWith('.controller'))?.value
  return v ? assetPath(String(v)) : ''
}

function title(c: Card) {
  if (c.key === 'transform') return 'Transform'
  if (c.kind === 'script') {
    const file = c.props.find((p) => p.name.endsWith('.file'))?.value
    if (file) return assetPath(String(file)).split('/').pop()!.replace('.lua', '')
    return 'Script'
  }
  if (c.index < 0) return c.key.charAt(0).toUpperCase() + c.key.slice(1)
  const n = c.key.includes('#') ? ' ' + c.key.split('#')[1] : ''
  return kindLabel(c.kind) + n
}

function icon(c: Card) {
  if (c.key === 'transform') return 'move'
  if (c.kind === 'script') return 'file'
  if (c.key === 'camera') return 'crosshair'
  if (c.key === 'environment') return 'sun'
  return store.kinds.components.find((k) => k.kind === c.kind)?.category === 'logic' ? 'sliders' : kindIcon(c.kind)
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="border-b border-ink-750 px-3 py-2.5">
      <div class="flex items-center gap-2">
        <div class="grid size-8 shrink-0 place-items-center rounded-lg bg-ink-800" :class="node?.prefab ? 'text-amber-soft' : 'text-amber-glow'">
          <Icon :name="store.selected ? (node?.prefab ? 'package' : kindIcon(node?.kind ?? '')) : 'sliders'" :size="15" />
        </div>
        <template v-if="store.selected && field('name')">
          <input type="checkbox" class="accent-[var(--color-amber-glow)]" title="Visible" :checked="!!field('visible')?.value" @change="set('visible', ($event.target as HTMLInputElement).checked)" />
          <input class="field h-7 min-w-0 flex-1 text-[13px] font-medium" :class="field('name')?.overridden ? 'text-amber-glow' : ''" :value="String(field('name')?.value ?? '')" :disabled="field('name')?.locked" @change="set('name', ($event.target as HTMLInputElement).value)" />
        </template>
        <div v-else class="min-w-0 flex-1">
          <div class="truncate text-[13px] font-medium">Scene</div>
          <div class="truncate font-mono text-[10.5px] text-ink-500">{{ store.info?.scene }}</div>
        </div>
        <button v-if="store.selected" class="icon-btn" title="Focus  F" @click="Viewport.Focus(store.selected)"><Icon name="crosshair" :size="14" /></button>
      </div>
      <div v-if="store.selected && field('tag')" class="mt-2 flex items-center gap-2 pl-10 text-[11.5px] text-ink-400">
        <span>Tag</span>
        <input class="field h-6 min-w-0 flex-1 text-[11.5px]" placeholder="Untagged" :value="String(field('tag')?.value ?? '')" @change="set('tag', ($event.target as HTMLInputElement).value)" />
        <label class="flex items-center gap-1.5">
          <input type="checkbox" class="accent-[var(--color-amber-glow)]" :checked="!!field('solid')?.value" @change="set('solid', ($event.target as HTMLInputElement).checked)" />
          Solid
        </label>
      </div>
    </div>

    <div v-if="instance" class="flex items-center gap-2 border-b border-ink-750 bg-amber-glow/[.06] px-3 py-2 text-[12px]">
      <Icon name="package" :size="13" class="text-amber-soft" />
      <span class="min-w-0 flex-1 truncate text-ink-300">{{ assetPath(instance.prefab ?? '').replace('prefabs/', '').replace('.prefab.json', '') }}<span v-if="instance.overrides" class="ml-1.5 text-amber-glow">{{ instance.overrides }} override{{ instance.overrides === 1 ? '' : 's' }}</span></span>
      <button class="btn h-6 px-2" :disabled="!instance.overrides" @click="apply('', instance.id)">Apply</button>
      <button class="btn h-6 px-2" :disabled="!instance.overrides" @click="revert('', instance.id)">Revert</button>
    </div>
    <div class="flex-1 overflow-auto px-2 pb-6 pt-2">
      <section v-for="c in cards" :key="c.key" class="mb-2 animate-fade overflow-hidden rounded-lg border border-ink-800 bg-ink-850/40" @contextmenu.prevent="(e) => c.index >= 0 && own && cardMenu(e, c)">
        <div class="flex h-8 items-center gap-1.5 bg-ink-850 px-2" :draggable="c.index >= 0" @dragstart="dragCard($event, c)">
          <button class="text-ink-500" @click="collapsed[c.key] = !collapsed[c.key]"><Icon :name="collapsed[c.key] ? 'chevright' : 'chevdown'" :size="12" /></button>
          <input v-if="c.enabled" type="checkbox" class="accent-[var(--color-amber-glow)]" :checked="!!c.enabled.value" :disabled="c.enabled.locked" @change="set(c.enabled.name, ($event.target as HTMLInputElement).checked)" />
          <Icon :name="icon(c)" :size="12" class="text-ink-400" />
          <button class="flex-1 truncate text-left text-[12px] font-medium" :class="c.enabled && !c.enabled.value ? 'text-ink-500 line-through' : 'text-ink-200'" @click="collapsed[c.key] = !collapsed[c.key]">{{ title(c) }}</button>
          <button v-if="c.kind === 'emitter'" class="icon-btn size-6" title="Particle Editor" @click="store.particles = true"><Icon name="sparkle" :size="13" /></button>
          <button v-if="c.kind === 'animator' && controllerOf(c)" class="icon-btn size-6" title="Open Animator" @click="store.animator = controllerOf(c)"><Icon name="graph" :size="13" /></button>
          <button v-if="c.index >= 0 && own" class="icon-btn size-6" title="More" @click="(e) => cardMenu(e, c)"><Icon name="more" :size="14" /></button>
        </div>
        <div v-show="!collapsed[c.key]" class="px-2 py-1">
          <PropField v-for="p in c.props" :key="p.name" :p="p" :keyable="!!store.selected" @set="set" @ref="setRef" @key="key" @menu="propMenu" />
        </div>
      </section>
      <AddComponent v-if="store.selected && own" />
    </div>
  </div>
</template>
