import { Assets, OnFileDrop } from './api'
import { call, refreshAssets, store, toast } from './store'

export interface FileDrop {
  x: number
  y: number
  paths: string[] // project paths after import
}

// assetKind mirrors asset.Classify for files that just arrived.
export function assetKind(p: string): string {
  const l = p.toLowerCase()
  if (l.endsWith('.prefab.json')) return 'prefab'
  if (l.endsWith('.scene.json')) return 'scene'
  if (l.endsWith('.lua')) return 'script'
  if (l.endsWith('.obj')) return 'model'
  if (l.endsWith('.material.json')) return 'material'
  if (/\.(png|jpe?g|gif)$/.test(l)) return 'texture'
  if (l.endsWith('.sprite.json') || l.endsWith('.xp')) return 'sprite'
  return '' // a folder or a file the engine does not use
}

// listenFileDrops imports files dropped from the OS and hands the project
// paths to the panel under the pointer, which decides what a drop means
// there: place in the viewport, parent in the hierarchy, file in a folder.
export function listenFileDrops() {
  OnFileDrop(async (x, y, raw) => {
    const files = raw.filter(Boolean) // drags inside the page arrive with no files
    if (!files.length) return
    const zone = document.elementFromPoint(x, y)?.closest<HTMLElement>('[data-drop]')
    const folder = zone?.dataset.folder ?? store.folder
    const paths = await call(() => Assets.Import(files, folder))
    if (!paths?.length) return
    toast(paths.length === 1 ? paths[0] : `${paths.length} files imported`, 'ok')
    await refreshAssets()
    zone?.dispatchEvent(new CustomEvent<FileDrop>('filedrop', { detail: { x, y, paths } }))
  }, true)
}
