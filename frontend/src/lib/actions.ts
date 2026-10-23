import { Assets, Scene, Viewport } from './api'
import { ask } from './dialog'
import { call, findNode, refresh, select, store, toast } from './store'
import type { MenuItem } from './menu'

export function kindIcon(kind: string): string {
  if (!kind) return 'layers'
  if (kind.includes('light')) return 'sun'
  if (kind.includes('emitter')) return 'rocket'
  if (kind.includes('label') || kind.includes('hud')) return 'file'
  if (kind.includes('sprite')) return 'image'
  if (kind.includes('terrain') || kind.includes('ground') || kind.includes('water')) return 'grid'
  if (kind.startsWith('df.')) return 'scene'
  return 'box'
}

export function pretty(kind: string): string {
  return kind.replace(/^df\./, '')
}

export async function add(parent: number, kind: string) {
  const id = await call(() => Scene.Add(parent, kind))
  if (id !== undefined) {
    await refresh()
    await select(id)
  }
}

export async function remove(id: number) {
  if (!id) return
  await call(() => Scene.Remove(id))
  if (store.selected === id) store.selected = 0
  await refresh()
}

export async function duplicate(id: number) {
  const c = await call(() => Scene.Duplicate(id))
  if (c !== undefined) {
    await refresh()
    await select(c)
  }
}

export async function savePrefab(id: number) {
  const n = findNode(id)
  if (!n) return
  const name = await ask('Save prefab', n.name.replace(/\s+/g, '-').toLowerCase(), { hint: 'Every instance updates when the prefab changes.', suffix: '.prefab.json', ok: 'Save' })
  if (!name) return
  const rel = name.includes('/') ? name : 'prefabs/' + name
  const ok = await call(() => Scene.SavePrefab(id, rel).then(() => true))
  if (ok) {
    toast(rel, 'ok')
    await refresh()
  }
}

export function addMenu(parent: number): MenuItem[] {
  const nodes = store.kinds.nodes.map((k) => ({ label: pretty(k), icon: kindIcon(k), run: () => add(parent, k) }))
  const prefabs = store.kinds.prefabs.map((p) => ({ label: p.replace(/^prefabs\//, '').replace('.prefab.json', ''), icon: 'package', run: () => add(parent, 'prefab:' + p) }))
  const items: MenuItem[] = [{ label: 'Empty', icon: 'layers', run: () => add(parent, 'empty') }, { label: 'Node', icon: 'box', children: nodes }]
  if (prefabs.length) items.push({ label: 'Prefab', icon: 'package', children: prefabs })
  return items
}

export function entityMenu(id: number): MenuItem[] {
  const n = findNode(id)
  if (!n) return []
  const items: MenuItem[] = [
    { label: 'Add', icon: 'plus', disabled: !!n.prefab || n.inherited, children: n.prefab || n.inherited ? undefined : addMenu(id) },
    { label: 'Focus', icon: 'crosshair', run: () => Viewport.Focus(id) },
    { label: 'Duplicate', icon: 'copy', disabled: n.inherited, run: () => duplicate(id) },
    { label: 'Save prefab', icon: 'package', disabled: n.inherited, run: () => savePrefab(id) },
  ]
  if (n.prefab) items.push({ label: 'Unpack', icon: 'layers', run: async () => { await call(() => Scene.Unpack(id)); await refresh() } })
  items.push({ label: '-' }, { label: 'Delete', icon: 'trash', danger: true, disabled: n.inherited, run: () => remove(id) })
  return items
}

// dropAsset handles a Project window tile dropped on the hierarchy: prefabs
// and art become children of parent, scripts attach to it.
export async function dropAsset(e: DragEvent, parent: number): Promise<boolean> {
  const data = e.dataTransfer?.getData('text/asset')
  if (!data) return false
  const [kind, ...rest] = data.split(':')
  const path = rest.join(':')
  if (kind === 'prefab') {
    await add(parent, 'prefab:' + path)
  } else if (kind === 'script' && parent) {
    await call(() => Scene.AddBehaviour(parent, 'script'))
    await call(() => Scene.SetProp(parent, 'script.file', path))
    await refresh()
    await select(parent)
  } else if (kind === 'art') {
    const id = await call(() => Assets.AddSprite(path))
    if (id !== undefined) {
      if (parent) await call(() => Scene.Reparent(id, parent, -1))
      await refresh()
      await select(id)
    }
  }
  return true
}
