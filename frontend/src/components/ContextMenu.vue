<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { closeMenu, closeTo, itemsAt, menu, openSub, selectable } from '../lib/menu'
import MenuList from './MenuList.vue'

// Keys act on the deepest open list, like native menus.
function key(e: KeyboardEvent) {
  if (!menu.open) return
  const d = menu.sub.length
  const items = itemsAt(d)
  const cur = menu.hl[d] ?? -1
  const item = items[cur]
  const stop = () => {
    e.preventDefault()
    e.stopPropagation()
  }
  switch (e.key) {
    case 'ArrowDown':
    case 'ArrowUp':
      stop()
      menu.hl[d] = selectable(items, cur < 0 && e.key === 'ArrowDown' ? -1 : cur, e.key === 'ArrowDown' ? 1 : -1)
      menu.hl.length = d + 1
      break
    case 'ArrowRight':
    case 'Enter':
    case ' ':
      stop()
      if (item?.children && !item.disabled) {
        openSub(d, cur)
        menu.hl[d + 1] = selectable(item.children, -1, 1)
      } else if (item && !item.disabled && e.key !== 'ArrowRight') {
        closeMenu()
        item.run?.()
      }
      break
    case 'ArrowLeft':
      stop()
      if (d > 0) closeTo(d - 1)
      break
    case 'Escape':
      stop()
      if (d > 0) closeTo(d - 1)
      else closeMenu()
      break
  }
}

onMounted(() => window.addEventListener('keydown', key, true))
onBeforeUnmount(() => window.removeEventListener('keydown', key, true))
</script>

<template>
  <div v-if="menu.open" class="fixed inset-0 z-50" @pointerdown.self="closeMenu" @contextmenu.prevent.self="closeMenu">
    <div class="absolute" :style="{ left: menu.x + 'px', top: menu.y + 'px' }">
      <MenuList :depth="0" />
    </div>
  </div>
</template>
