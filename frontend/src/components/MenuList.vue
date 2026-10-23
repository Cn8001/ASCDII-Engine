<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { closeMenu, closeTo, hold, itemsAt, later, menu, openSub, type MenuItem } from '../lib/menu'
import Icon from './Icon.vue'

const props = defineProps<{ depth: number }>()
const items = computed(() => itemsAt(props.depth))
const subOpen = computed(() => menu.sub.length > props.depth && !!items.value[menu.sub[props.depth]]?.children)
const el = ref<HTMLElement | null>(null)
const shift = ref({ top: 0, flip: false })

function enter(i: MenuItem, n: number) {
  hold()
  const d = props.depth
  menu.hl[d] = n
  menu.hl.length = d + 1
  if (menu.sub[d] === n) return
  const next = () => (i.children && !i.disabled ? openSub(d, n) : closeTo(d))
  if (menu.sub.length > d) later(next, 260)
  else if (i.children && !i.disabled) later(next, 120)
}

function click(i: MenuItem, n: number) {
  if (i.disabled || i.label === '-') return
  if (i.children) {
    if (menu.sub[props.depth] === n) closeTo(props.depth)
    else openSub(props.depth, n)
    return
  }
  closeMenu()
  i.run?.()
}

// Submenus open beside their row; they flip left or move up when the
// window edge would cut them.
watch(
  () => [subOpen.value, menu.sub[props.depth]],
  async () => {
    const row = el.value?.querySelector<HTMLElement>(`:scope > div > [data-row="${menu.sub[props.depth]}"]`)
    shift.value = { top: (row?.offsetTop ?? 0) - 5, flip: false }
    await nextTick()
    const sub = el.value?.querySelector<HTMLElement>(':scope > [data-sub]')
    if (!sub) return
    const r = sub.getBoundingClientRect()
    shift.value = { top: shift.value.top + Math.min(0, window.innerHeight - 8 - r.bottom), flip: r.right > window.innerWidth - 4 }
  },
)
</script>

<template>
  <div ref="el" class="relative">
    <div class="min-w-48 animate-pop rounded-lg border border-ink-700 bg-ink-850 p-1 shadow-2xl shadow-black/70" @pointerenter="hold">
      <template v-for="(i, n) in items" :key="n">
        <div v-if="i.label === '-'" class="my-1 h-px bg-ink-700" />
        <button
          v-else
          :data-row="n"
          class="flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-[12px] transition-colors"
          :class="[
            i.disabled ? 'text-ink-500' : i.danger ? 'text-bad' : 'text-ink-200',
            (menu.hl[depth] === n || menu.sub[depth] === n) && !i.disabled ? (i.danger ? 'bg-bad/10' : 'bg-ink-750 text-ink-100') : '',
          ]"
          @pointerenter="enter(i, n)"
          @click="click(i, n)"
        >
          <Icon v-if="i.icon" :name="i.icon" :size="14" class="shrink-0 text-ink-400" />
          <span v-else class="w-[14px] shrink-0" />
          <span class="flex-1 truncate">{{ i.label }}</span>
          <span v-if="i.hint" class="font-mono text-[10px] text-ink-500">{{ i.hint }}</span>
          <Icon v-if="i.children" name="chevright" :size="12" class="text-ink-500" />
        </button>
      </template>
      <div v-if="!items.length" class="px-2 py-1.5 text-[12px] text-ink-500">Empty</div>
    </div>
    <div
      v-if="subOpen"
      data-sub
      class="absolute z-10"
      :class="shift.flip ? 'right-full pr-0.5' : 'left-full pl-0.5'"
      :style="{ top: shift.top + 'px' }"
    >
      <MenuList :depth="depth + 1" />
    </div>
  </div>
</template>
