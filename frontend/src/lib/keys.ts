// Maps browser key events to the engine's key names.
export function engineKey(e: KeyboardEvent): string | null {
  switch (e.key) {
    case 'ArrowUp': return 'up'
    case 'ArrowDown': return 'down'
    case 'ArrowLeft': return 'left'
    case 'ArrowRight': return 'right'
    case 'Enter': return 'enter'
    case 'Tab': return 'tab'
    case 'Backspace': return 'backspace'
    case ' ': return 'space'
  }
  return e.key.length === 1 ? e.key.toLowerCase() : null
}

export function typing(e: Event): boolean {
  const t = e.target as HTMLElement | null
  return !!t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable)
}
