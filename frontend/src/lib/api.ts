import * as Project from '../wailsjs/go/service/ProjectService'
import * as Scene from '../wailsjs/go/service/SceneService'
import * as Viewport from '../wailsjs/go/service/ViewportService'
import * as Pipeline from '../wailsjs/go/service/PipelineService'
import * as Assets from '../wailsjs/go/service/AssetService'
import * as Anim from '../wailsjs/go/service/AnimService'
import * as Build from '../wailsjs/go/service/BuildService'
import * as Console from '../wailsjs/go/service/ConsoleService'
import { EventsOn } from '../wailsjs/runtime/runtime'

export { Project, Scene, Viewport, Pipeline, Assets, Anim, Build, Console, EventsOn }

export interface ApiError {
  code: string
  message: string
}

// The backend prefixes every error with a stable code, "not_found: ...".
export function parseError(e: unknown): ApiError {
  const text = String((e as { message?: string })?.message ?? e)
  const i = text.indexOf(': ')
  if (i > 0 && /^[a-z_]+$/.test(text.slice(0, i))) {
    return { code: text.slice(0, i), message: text.slice(i + 2) }
  }
  return { code: 'internal', message: text }
}
