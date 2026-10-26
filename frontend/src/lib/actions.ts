import { Assets, Scene, Viewport } from './api'
import { ask } from './dialog'
import { adopt, call, findNode, multi, refresh, select, store, toast } from './store'
import type { MenuItem } from './menu'

export function kindIcon(kind: string): string {
  if (!kind) return 'layers'
  if (kind === 'camera') return 'camera'
  if (kind.includes('light')) return 'sun'
  if (kind.includes('audio')) return 'music'
  if (kind === 'animator') return 'graph'
  if (kind === 'paint') return 'brush'
  if (kind.includes('collider') || kind === 'rigidbody') return 'grid'
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

// remove deletes id, or the whole selection when id is part of a selection
// of several objects.
export async function remove(id: number) {
  if (!id) return
  if (multi() && store.frame.selection.includes(id)) {
    await call(() => Scene.RemoveSelection())
    store.selected = 0
  } else {
    await call(() => Scene.Remove(id))
    if (store.selected === id) store.selected = 0
  }
  await refresh()
}

// duplicate copies id, or every selected object when id is one of several.
export async function duplicate(id: number) {
  if (multi() && store.frame.selection.includes(id)) {
    const copies = await call(() => Scene.DuplicateSelection())
    if (copies?.length) {
      await refresh()
      await adopt(copies[0])
    }
    return
  }
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
  const rel = name.includes('/') || !store.folder ? name : `${store.folder}/${name}`
  const ok = await call(() => Scene.SavePrefab(id, rel).then(() => true))
  if (ok) {
    toast(rel, 'ok')
    await refresh()
  }
}

const categories: Record<string, { label: string; icon: string }> = {
  '3d': { label: '3D Object', icon: 'box' },
  light: { label: 'Light', icon: 'sun' },
  effects: { label: 'Effects', icon: 'rocket' },
  ui: { label: 'UI', icon: 'file' },
}

export function categoryLabel(c: string): string {
  return categories[c]?.label ?? (c === 'logic' ? 'Logic' : c.charAt(0).toUpperCase() + c.slice(1))
}

export function kindLabel(kind: string): string {
  return store.kinds.components.find((k) => k.kind === kind)?.label ?? pretty(kind)
}

// addMenu is the GameObject menu: the same list for the menu bar, the
// hierarchy "+" and right click. Objects are created under parent.
export function addMenu(parent: number): MenuItem[] {
  const groups = new Map<string, MenuItem[]>()
  for (const k of store.kinds.components.filter((k) => k.object)) {
    const list = groups.get(k.category) ?? []
    list.push({ label: k.label, icon: kindIcon(k.kind), run: () => add(parent, k.kind) })
    groups.set(k.category, list)
  }
  groups.get('light')?.push(
    { label: 'Spot Light', icon: 'sun', run: () => addLight(parent, 'spot', 'Spot Light') },
    { label: 'Directional Light', icon: 'sun', run: () => addLight(parent, 'directional', 'Directional Light') },
  )
  const order = [...Object.keys(categories), ...[...groups.keys()].filter((c) => !categories[c]).sort()]
  const items: MenuItem[] = [{ label: 'Create Empty', icon: 'layers', run: () => add(parent, 'empty') }, ...(groups.get('camera') ?? []), { label: '-' }]
  groups.delete('camera')
  for (const c of order) {
    const children = groups.get(c)
    if (children) items.push({ label: categoryLabel(c), icon: categories[c]?.icon ?? 'package', children })
  }
  const models = (store.kinds.models ?? []).map((p) => ({ label: p.split('/').pop()!.replace(/\.obj$/i, ''), icon: 'box', run: () => add(parent, 'model:' + p) }))
  const prefabs = store.kinds.prefabs.map((p) => ({ label: p.replace(/^prefabs\//, '').replace('.prefab.json', ''), icon: 'package', run: () => add(parent, 'prefab:' + p) }))
  items.push({ label: '-' }, { label: 'Model', icon: 'box', disabled: !models.length, children: models })
  items.push({ label: 'Prefab', icon: 'package', disabled: !prefabs.length, children: prefabs })
  return items
}

// addLight makes a light of a type the Light component switches between.
async function addLight(parent: number, type: string, name: string) {
  const id = await call(() => Scene.Add(parent, 'light'))
  if (id === undefined) return
  await call(() => Scene.SetProp(id, 'light.type', type))
  await call(() => Scene.SetProp(id, 'name', name))
  if (type === 'spot') await call(() => Scene.SetProp(id, 'pos.z', 4))
  await refresh()
  await select(id)
}

// createParent is where the menu bar creates objects: under the selection,
// unless that is part of a prefab instance.
export function createParent(): number {
  const n = findNode(store.selected)
  return n && !n.prefab && !n.inherited ? n.id : 0
}

export function entityMenu(id: number): MenuItem[] {
  const n = findNode(id)
  if (!n) return []
  const items: MenuItem[] = [
    { label: 'Create', icon: 'plus', disabled: !!n.prefab || n.inherited, children: n.prefab || n.inherited ? undefined : addMenu(id) },
    { label: 'Focus', icon: 'crosshair', hint: 'F', run: () => Viewport.Focus(id) },
    { label: 'Duplicate', icon: 'copy', hint: 'Ctrl+D', disabled: n.inherited, run: () => duplicate(id) },
    { label: 'Save prefab', icon: 'package', disabled: n.inherited, run: () => savePrefab(id) },
  ]
  if (n.prefab) items.push({ label: 'Unpack', icon: 'layers', run: async () => { await call(() => Scene.Unpack(id)); await refresh() } })
  items.push({ label: '-' }, { label: 'Delete', icon: 'trash', hint: 'Del', danger: true, disabled: n.inherited, run: () => remove(id) })
  return items
}

// dropAsset handles a Project window tile dropped on the hierarchy: prefabs
// and sprites become children of parent, scripts attach to it.
export async function dropAsset(e: DragEvent, parent: number): Promise<boolean> {
  const data = e.dataTransfer?.getData('text/asset')
  if (!data) return false
  const [kind, ...rest] = data.split(':')
  const path = rest.join(':')
  if (kind === 'material' && parent) {
    await call(() => Assets.ApplyMaterial(parent, path))
    await refresh()
    await select(parent)
  } else if (kind === 'prefab' || kind === 'model') {
    await add(parent, `${kind}:${path}`)
  } else if (kind === 'script' && parent) {
    const key = await call(() => Scene.AddComponent(parent, 'script'))
    if (key) await call(() => Scene.SetProp(parent, key + '.file', path))
    await refresh()
    await select(parent)
  } else if (kind === 'sprite') {
    const id = await call(() => Assets.AddSprite(path))
    if (id !== undefined) {
      if (parent) await call(() => Scene.Reparent(id, parent, -1))
      await refresh()
      await select(id)
    }
  }
  return true
}
