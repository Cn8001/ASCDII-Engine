import { reactive } from 'vue'

export interface MenuItem {
  label: string
  icon?: string
  hint?: string
  danger?: boolean
  disabled?: boolean
  run?: () => void
  children?: MenuItem[]
}

// hl is the highlighted row per depth and sub the row whose submenu is open.
// They differ while the pointer crosses other rows on its way to a submenu.
export const menu = reactive({ open: false, x: 0, y: 0, items: [] as MenuItem[], hl: [] as number[], sub: [] as number[] })

let timer = 0

export function openMenuAt(x: number, y: number, items: MenuItem[]) {
  menu.x = Math.max(4, Math.min(x, window.innerWidth - 220))
  menu.y = Math.max(4, Math.min(y, window.innerHeight - items.length * 28 - 16))
  menu.items = items
  menu.hl = []
  menu.sub = []
  menu.open = true
}

export function openMenu(e: MouseEvent, items: MenuItem[]) {
  openMenuAt(e.clientX, e.clientY, items)
}

// openBelow anchors a menu under a button, the way a menu bar opens.
export function openBelow(e: MouseEvent, items: MenuItem[]) {
  const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
  openMenuAt(r.left, r.bottom + 4, items)
}

export function closeMenu() {
  clearTimeout(timer)
  menu.open = false
}

export function itemsAt(d: number): MenuItem[] {
  let items = menu.items
  for (let i = 0; i < d; i++) items = items[menu.sub[i]]?.children ?? []
  return items
}

export function openSub(d: number, n: number) {
  clearTimeout(timer)
  menu.sub.length = d
  menu.sub[d] = n
  menu.hl.length = d + 1
  menu.hl[d] = n
}

export function closeTo(d: number) {
  clearTimeout(timer)
  menu.sub.length = d
  menu.hl.length = Math.min(menu.hl.length, d + 1)
}

// later changes the open submenu after a pause, so moving the pointer
// diagonally across other rows towards a submenu does not switch it.
export function later(fn: () => void, ms: number) {
  clearTimeout(timer)
  timer = window.setTimeout(fn, ms)
}

export function hold() {
  clearTimeout(timer)
}

export function selectable(items: MenuItem[], from: number, step: number): number {
  for (let k = 1; k <= items.length; k++) {
    const i = (from + step * k + items.length * 2) % items.length
    if (items[i].label !== '-' && !items[i].disabled) return i
  }
  return -1
}
