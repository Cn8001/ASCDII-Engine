import { EventsOn, Viewport } from './api'

interface AudioEvent {
  op: 'play' | 'stop' | 'volume' | 'stopall'
  voice: number
  path?: string
  volume: number
  loop?: boolean
}

// Play mode sounds: the backend sends what to play and the web view plays
// the file itself, so the editor needs no audio code of its own.
const voices = new Map<number, HTMLAudioElement>()

function stop(voice: number) {
  const a = voices.get(voice)
  if (!a) return
  a.pause()
  voices.delete(voice)
}

// listenAudio returns the function that stops listening.
export function listenAudio(): () => void {
  return EventsOn('audio', (ev: AudioEvent) => {
    switch (ev.op) {
      case 'play': {
        const a = new Audio('/asset/' + encodeURI(ev.path ?? ''))
        a.volume = Math.max(0, Math.min(1, ev.volume))
        a.loop = !!ev.loop
        a.addEventListener('ended', () => {
          voices.delete(ev.voice)
          Viewport.AudioEnded(ev.voice)
        })
        voices.set(ev.voice, a)
        a.play().catch(() => voices.delete(ev.voice))
        break
      }
      case 'stop':
        stop(ev.voice)
        break
      case 'volume': {
        const a = voices.get(ev.voice)
        if (a) a.volume = Math.max(0, Math.min(1, ev.volume))
        break
      }
      case 'stopall':
        for (const v of [...voices.keys()]) stop(v)
        break
    }
  })
}

let preview: HTMLAudioElement | null = null

// previewSound plays a sound asset once, from the Project window.
export function previewSound(path: string) {
  preview?.pause()
  preview = new Audio('/asset/' + encodeURI(path))
  void preview.play()
}
