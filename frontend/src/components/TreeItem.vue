<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { Scene } from '../lib/api'
import { dropAsset, entityMenu, kindIcon } from '../lib/actions'
import { openMenu } from '../lib/menu'
import { call, refresh, select, store } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'

const props = defineProps<{ node: session.Node; depth: number }>()
const open = ref(props.depth < 1)
const renaming = ref(false)
const draft = ref('')
const input = ref<HTMLInputElement | null>(null)
const over = ref(false)
const kids = computed(() => props.node.children ?? [])
const active = computed(() => store.selected === props.node.id)

async function toggleVisible() {
  await call(() => Scene.SetProp(props.node.id, 'visible', !props.node.visible))
  await refresh()
}

async function startRename() {
  if (props.node.inherited) return
  draft.value = props.node.name
  renaming.value = true
  await nextTick()
  input.value?.select()
}

async function commit() {
  renaming.value = false
  if (draft.value.trim() && draft.value !== props.node.name) {
    await call(() => Scene.SetProp(props.node.id, 'name', draft.value.trim()))
    await refresh()
  }
}

function dragStart(e: DragEvent) {
  e.dataTransfer?.setData('text/entity', String(props.node.id))
}

async function drop(e: DragEvent) {
  over.value = false
  if (await dropAsset(e, props.node.id)) {
    open.value = true
    return
  }
  const id = Number(e.dataTransfer?.getData('text/entity'))
  if (!id || id === props.node.id) return
  await call(() => Scene.Reparent(id, props.node.id, -1))
  open.value = true
  await refresh()
}
</script>

<template>
  <div>
    <div
      class="group flex h-7 cursor-default items-center gap-1 rounded-md pr-1 transition-colors"
      :class="[active ? 'bg-amber-glow/12 text-ink-100' : 'text-ink-300 hover:bg-ink-800', over ? 'ring-1 ring-amber-soft' : '']"
      :style="{ paddingLeft: depth * 14 + 4 + 'px' }"
      :draggable="!node.inherited"
      @dragstart="dragStart"
      @dragover.prevent="over = true"
      @dragleave="over = false"
      @drop.prevent.stop="drop"
      @click="select(node.id)"
      @dblclick="startRename"
      @contextmenu.prevent="(e) => { select(node.id); openMenu(e, entityMenu(node.id)) }"
    >
      <button class="grid size-4 place-items-center text-ink-500" :class="kids.length ? '' : 'invisible'" @click.stop="open = !open">
        <Icon :name="open ? 'chevdown' : 'chevright'" :size="12" />
      </button>
      <Icon :name="node.prefab ? 'package' : kindIcon(node.kind)" :size="13" :class="node.prefab ? 'text-amber-soft' : active ? 'text-amber-glow' : 'text-ink-500'" />
      <input v-if="renaming" ref="input" v-model="draft" class="field h-5 flex-1 px-1" @keydown.enter="commit" @keydown.esc="renaming = false" @blur="commit" @click.stop />
      <span v-else class="flex-1 truncate text-[12.5px]" :class="[node.visible ? '' : 'opacity-40', node.inherited ? 'italic text-ink-400' : '']">{{ node.name }}</span>
      <span v-if="node.overrides" class="size-1.5 rounded-full bg-amber-glow" title="Has overrides" />
      <span v-if="node.behaviours?.length" class="font-mono text-[10px] text-ink-500">{{ node.behaviours.length }}</span>
      <button class="icon-btn size-5 opacity-0 group-hover:opacity-100" :class="node.visible ? '' : 'opacity-100'" @click.stop="toggleVisible">
        <Icon :name="node.visible ? 'eye' : 'eyeoff'" :size="12" />
      </button>
    </div>
    <div v-if="open && kids.length">
      <TreeItem v-for="c in kids" :key="c.id" :node="c" :depth="depth + 1" />
    </div>
  </div>
</template>
