import { reactive } from 'vue'

export interface MenuItem {
  label: string
  icon?: string
  danger?: boolean
  disabled?: boolean
  run?: () => void
  children?: MenuItem[]
}

export const menu = reactive({ open: false, x: 0, y: 0, items: [] as MenuItem[] })

export function openMenu(e: MouseEvent, items: MenuItem[]) {
  menu.x = Math.min(e.clientX, window.innerWidth - 220)
  menu.y = Math.min(e.clientY, window.innerHeight - items.length * 30 - 16)
  menu.items = items
  menu.open = true
}

export function closeMenu() {
  menu.open = false
}
