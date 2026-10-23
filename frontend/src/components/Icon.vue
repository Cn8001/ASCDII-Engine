<script setup lang="ts">
// Hand drawn 24px stroke icons; a tiny set keeps the bundle dependency free.
defineProps<{ name: string; size?: number }>()

const paths: Record<string, string> = {
  play: 'M7 5v14l11-7z',
  pause: 'M8 5v14M16 5v14',
  stop: 'M7 7h10v10H7z',
  save: 'M5 4h11l3 3v13H5zM8 4v5h7V4M8 20v-6h8v6',
  undo: 'M9 14L4 9l5-5M4 9h10a6 6 0 010 12h-3',
  redo: 'M15 14l5-5-5-5M20 9H10a6 6 0 000 12h3',
  folder: 'M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z',
  plus: 'M12 5v14M5 12h14',
  trash: 'M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13',
  copy: 'M8 8h11v11H8zM5 16V5h11',
  eye: 'M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12zM12 15a3 3 0 100-6 3 3 0 000 6z',
  eyeoff: 'M3 3l18 18M10.6 6.1A10 10 0 0112 6c6 0 10 6 10 6a17 17 0 01-3.2 3.8M6.3 7.6A17 17 0 002 12s4 6 10 6a9.6 9.6 0 004.2-1',
  box: 'M12 3l8 4.5v9L12 21l-8-4.5v-9zM12 12l8-4.5M12 12v9M12 12L4 7.5',
  layers: 'M12 3l9 5-9 5-9-5zM3 13l9 5 9-5',
  image: 'M4 5h16v14H4zM4 15l4-4 5 5 3-3 4 4M15 9h.01',
  film: 'M4 4h16v16H4zM8 4v16M16 4v16M4 9h4M16 9h4M4 15h4M16 15h4',
  hammer: 'M14 4l6 6-3 3-6-6zM11 7L4 14l3 3 7-7',
  chevright: 'M9 6l6 6-6 6',
  chevdown: 'M6 9l6 6 6-6',
  lock: 'M6 11h12v9H6zM8 11V8a4 4 0 018 0v3',
  diamond: 'M12 4l7 8-7 8-7-8z',
  x: 'M6 6l12 12M18 6L6 18',
  sliders: 'M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12M20 18h0M14 4v4M8 10v4M16 16v4',
  sun: 'M12 16a4 4 0 100-8 4 4 0 000 8zM12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
  package: 'M3 7l9-4 9 4v10l-9 4-9-4zM3 7l9 4 9-4M12 11v10',
  crosshair: 'M12 3v4M12 17v4M3 12h4M17 12h4M12 15a3 3 0 100-6 3 3 0 000 6z',
  rocket: 'M5 15c-1 1-2 4-2 6 2 0 5-1 6-2M9 15l-3-3c1-4 5-9 12-9 0 7-5 11-9 12zM15 9h.01',
  check: 'M5 12l5 5 9-10',
  alert: 'M12 4l9 16H3zM12 10v4M12 17h.01',
  refresh: 'M20 12a8 8 0 11-2.3-5.7M20 4v5h-5',
  move: 'M12 3v18M3 12h18M12 3l-3 3M12 3l3 3M12 21l-3-3M12 21l3-3M3 12l3-3M3 12l3 3M21 12l-3-3M21 12l-3 3',
  camera: 'M3 8h3l2-3h8l2 3h3v11H3zM12 17a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
  sparkle: 'M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8zM19 16l.7 2 2 .7-2 .7-.7 2-.7-2-2-.7 2-.7z',
  rotate: 'M21 12a9 9 0 1 1-9-9c2.5 0 4.9 1 6.7 2.7L21 8M21 3v5h-5',
  scale: 'M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7',
  file: 'M6 3h8l4 4v14H6zM14 3v4h4',
  grid: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z',
  scene: 'M3 18l6-8 4 5 3-4 5 7zM16 7a2 2 0 100-4 2 2 0 000 4z',
  key: 'M12 4l7 8-7 8-7-8z',
  dot: 'M12 13a1 1 0 100-2 1 1 0 000 2z',
  music: 'M9 18V6l10-2v12M9 18a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0zM19 16a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z',
  brush: 'M14 4l6 6-8 8-6-6zM6 12l-2 6 6-2',
  graph: 'M6 6h4v4H6zM14 14h4v4h-4zM14 6h4v4h-4zM10 8h4M16 10v4',
  more: 'M5 13.4a1.4 1.4 0 100-2.8 1.4 1.4 0 000 2.8zM12 13.4a1.4 1.4 0 100-2.8 1.4 1.4 0 000 2.8zM19 13.4a1.4 1.4 0 100-2.8 1.4 1.4 0 000 2.8z',
  arrowup: 'M12 19V5M6 11l6-6 6 6',
  arrowdown: 'M12 5v14M6 13l6 6 6-6',
  open: 'M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v1H7l-4 9zM7 10h15l-3 9H3',
}
</script>

<template>
  <svg :width="size ?? 16" :height="size ?? 16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path :d="paths[name] ?? paths.dot" :fill="name === 'play' || name === 'diamond' || name === 'key' || name === 'more' ? 'currentColor' : 'none'" />
  </svg>
</template>
