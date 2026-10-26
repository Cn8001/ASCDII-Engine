import { Console, EventsOn } from './api'
import { call, store } from './store'
import type { session } from '../wailsjs/go/models'

// add merges a repeat of the last line, as the backend's logbook does. An
// error opens the console, so a failing script or update is seen even when
// the console was never opened.
function add(e: session.LogEntry) {
  const last = store.logs[store.logs.length - 1]
  if (last && last.level === e.level && last.source === e.source && last.text === e.text) {
    Object.assign(last, e)
  } else {
    store.logs.push(e)
    if (store.logs.length > 300) store.logs.splice(0, store.logs.length - 300)
  }
  if (e.level === 'error' && !store.dockOpen) {
    store.dockOpen = true
    store.dock = 'console'
  }
}

// listenLogs loads the console lines so far and follows new ones for as long
// as a project is open; it returns the function that stops listening.
export function listenLogs(): () => void {
  const off = EventsOn('log', add)
  call(() => Console.Logs()).then((l) => {
    if (l) store.logs = l
  })
  return off
}
