import { reactive, markRaw } from 'vue'
import { Project, Scene, Assets, Anim, parseError } from './api'
import type { session, prop, pipeline, asset } from '../wailsjs/go/models'
import type { DotStyle } from './cells'

export interface Toast {
  id: number
  kind: 'info' | 'ok' | 'bad'
  text: string
}

export interface FrameMeta {
  cols: number
  rows: number
  time: number
  mode: string
  millis: number
  fps: number
  passes: pipeline.Info[]
  gizmo: session.Gizmo | null
}

export const store = reactive({
  info: null as session.Info | null,
  recent: [] as string[],
  tree: [] as session.Node[],
  kinds: { components: [] as session.Kind[], prefabs: [] as string[], models: [] as string[] },
  selected: 0,
  asset: '', // the Project window pick the inspector shows instead of an object
  props: [] as prop.Prop[],
  sceneProps: [] as prop.Prop[],
  assets: [] as asset.Entry[],
  clips: [] as session.Clip[],
  frame: { cols: 0, rows: 0, time: 0, mode: 'edit', millis: 0, fps: 0, passes: [], gizmo: null } as FrameMeta,
  toasts: [] as Toast[],
  right: 'inspector' as 'inspector' | 'pipeline',
  dock: 'project' as 'project' | 'console' | 'animation',
  dockOpen: true,
  dockHeight: 240,
  animFloating: false,
  autoKey: false,
  view: 'scene' as 'scene' | 'game',
  dots: 'dots' as DotStyle,
  cellPx: 8,
  buildOpen: false,
  newProject: false,
  packagesOpen: false,
  animator: '', // the controller open in the animator window
  particles: false, // the particle editor window
  folder: '', // the folder open in the Project window, where new files go
  tool: 'move' as 'move' | 'rotate' | 'scale',
  brush: { on: false, color: '#d8463a', radius: 0.4, erase: false },
  assetsVersion: 0, // bumped when packages change the project's files
  busy: '',
  logs: [] as session.LogEntry[],
  key: null as { clip: number; prop: string; t: number } | null,
})

let toastSeq = 0

export function toast(text: string, kind: Toast['kind'] = 'info') {
  const t = { id: ++toastSeq, kind, text }
  store.toasts.push(t)
  setTimeout(() => {
    store.toasts = store.toasts.filter((x) => x.id !== t.id)
  }, kind === 'bad' ? 5200 : 2600)
}

// call runs a backend call and turns failures into a toast; cancelled
// dialogs are silent.
export async function call<T>(fn: () => Promise<T>): Promise<T | undefined> {
  try {
    return await fn()
  } catch (e) {
    const err = parseError(e)
    if (err.code !== 'cancelled') toast(err.message, 'bad')
    return undefined
  }
}

export async function refreshInfo() {
  const info = await call(() => Project.Info())
  if (info) store.info = info
}

export async function refreshProps() {
  if (store.selected) {
    const p = await call(() => Scene.Props(store.selected))
    store.props = p ?? []
    if (!p) store.selected = 0
  } else {
    store.props = []
  }
  store.sceneProps = (await call(() => Scene.SceneProps())) ?? []
}

export async function refreshClips() {
  store.clips = (await call(() => Anim.Clips(store.selected))) ?? []
}

export async function refresh() {
  const [tree, kinds] = await Promise.all([call(() => Scene.Tree()), call(() => Scene.Kinds())])
  store.tree = tree ?? []
  if (kinds) store.kinds = kinds
  await Promise.all([refreshInfo(), refreshProps(), refreshClips()])
}

export function animatable(p: prop.Prop): boolean {
  return ['float', 'int', 'vec3', 'color', 'bool'].includes(p.type) && !p.name.endsWith('.enabled')
}

export async function refreshAssets() {
  store.assets = (await call(() => Assets.List())) ?? []
}

export async function select(id: number) {
  store.selected = id
  store.asset = ''
  await call(() => Scene.Select(id))
  await Promise.all([refreshProps(), refreshClips()])
}

export function setFrame(f: session.Frame) {
  const now = performance.now()
  const dt = now - lastFrame
  lastFrame = now
  const m = store.frame
  m.cols = f.cols
  m.rows = f.rows
  m.time = f.time
  m.mode = f.mode
  m.millis = f.millis
  m.fps = dt > 0 ? m.fps * 0.9 + (1000 / dt) * 0.1 : m.fps
  m.passes = markRaw(f.passes ?? [])
  m.gizmo = f.gizmo ? markRaw(f.gizmo) : null
}

let lastFrame = performance.now()

// assetPath shows a GUID reference as the file it points at.
export function assetPath(ref: string): string {
  return store.assets.find((a) => a.guid === ref)?.path ?? ref
}

// allNodes is the hierarchy flattened, depth first.
export function allNodes(nodes = store.tree, out: session.Node[] = []): session.Node[] {
  for (const n of nodes) {
    out.push(n)
    allNodes(n.children ?? [], out)
  }
  return out
}

export function findNode(id: number, nodes = store.tree): session.Node | undefined {
  for (const n of nodes) {
    if (n.id === id) return n
    const c = findNode(id, n.children ?? [])
    if (c) return c
  }
  return undefined
}
