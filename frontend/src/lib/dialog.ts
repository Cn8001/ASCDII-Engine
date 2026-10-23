import { reactive } from 'vue'

// Wails' Linux webview has no window.prompt or window.confirm, so the editor
// draws its own dialogs. One is open at a time.
export const dialog = reactive({
  open: false,
  title: '',
  hint: '',
  value: '',
  suffix: '',
  confirm: false,
  danger: false,
  ok: 'OK',
  resolve: null as ((v: string | null) => void) | null,
})

export function ask(title: string, value = '', opts: { hint?: string; suffix?: string; ok?: string } = {}): Promise<string | null> {
  return new Promise((resolve) => {
    Object.assign(dialog, { open: true, title, value, hint: opts.hint ?? '', suffix: opts.suffix ?? '', ok: opts.ok ?? 'Create', confirm: false, danger: false, resolve })
  })
}

export function confirmAction(title: string, hint = '', ok = 'Discard'): Promise<boolean> {
  return new Promise((resolve) => {
    Object.assign(dialog, { open: true, title, hint, value: '', suffix: '', ok, confirm: true, danger: true, resolve: (v: string | null) => resolve(v !== null) })
  })
}

export function close(v: string | null) {
  const r = dialog.resolve
  dialog.open = false
  dialog.resolve = null
  r?.(v)
}
