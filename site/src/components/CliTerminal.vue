<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const lines = [
  { note: 'Create a project from the Dragonfall template', cmd: 'ascdii new MyGame -template dragonfall' },
  { note: 'Play it in this terminal', cmd: 'ascdii play MyGame' },
  { note: 'Or in block mode, if the font draws Braille poorly', cmd: 'ascdii play MyGame --blocks' },
  { note: 'Build the player, the game data and checksums', cmd: 'ascdii build MyGame' },
]

const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const typed = ref(lines.map((l) => l.cmd))
const active = ref(-1)
let run = 0
const timers: number[] = []
const later = (ms: number, f: () => void) => timers.push(window.setTimeout(f, ms))

function play() {
  if (reduce) return
  const id = ++run
  timers.splice(0).forEach(clearTimeout)
  typed.value = lines.map(() => '')
  const line = (n: number) => {
    if (id !== run || n >= lines.length) { active.value = -1; return }
    active.value = n
    let i = 0
    const type = () => {
      if (id !== run) return
      typed.value[n] = lines[n].cmd.slice(0, ++i)
      if (i < lines[n].cmd.length) later(28 + Math.random() * 40, type)
      else later(700, () => line(n + 1))
    }
    later(350, type)
  }
  line(0)
}

onMounted(play)
onBeforeUnmount(() => { run++; timers.splice(0).forEach(clearTimeout) })
</script>

<template>
  <div class="term">
    <div class="head"><span>Terminal</span><button v-if="!reduce" type="button" @click="play">Replay</button></div>
    <div class="body">
      <template v-for="(l, i) in lines" :key="l.cmd">
        <p class="c" :class="{ hide: !typed[i] && active < i }"># {{ l.note }}</p>
        <p :class="{ hide: !typed[i] && active < i }"><span class="pr">$ </span>{{ typed[i] }}<span v-if="active === i" class="caret" /></p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.term { background: var(--code); border: 1px solid var(--line); border-radius: 8px; min-width: 0; }
.head { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 10px 14px; border-bottom: 1px solid var(--line); font-size: 13px; color: var(--muted); }
.head button { font: 500 13px var(--f-body); color: var(--ink); background: none; border: 1px solid var(--line); border-radius: 6px; padding: 4px 10px; cursor: pointer; }
.body { padding: 16px; font: 13px/1.75 var(--f-mono); display: grid; gap: 0; min-height: 250px; align-content: start; overflow-x: auto; }
p { white-space: pre; transition: opacity .2s; }
.c { color: #6f8a82; margin-top: 10px; }
.c:first-child { margin-top: 0; }
.pr { color: var(--meadow); }
.hide { opacity: 0; }
.caret { display: inline-block; width: 8px; height: 1.1em; background: var(--amber); vertical-align: text-bottom; animation: blink 1s steps(1) infinite; }
@keyframes blink { 50% { opacity: 0; } }
</style>
