<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const shots = [
  { src: 'usage/img/game-night.png', label: 'Night', alt: 'Dragonfall at night, lit by torches, with a dragon breathing fire' },
  { src: 'usage/img/game-day.png', label: 'Day', alt: 'Dragonfall in the afternoon: a castle gate, torches, a road, a river and a bridge' },
  { src: 'usage/img/game-window.png', label: 'Window', alt: 'Dragonfall in a desktop window, where every dot keeps its own colour' },
  { src: 'usage/img/game-blocks.png', label: 'Blocks', alt: 'Dragonfall drawn with half blocks instead of Braille' },
]

const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches
const cur = ref(0)
const playing = ref(!reduce)
let timer = 0

function show(i: number) {
  cur.value = i
  restart()
}
function restart() {
  clearInterval(timer)
  if (playing.value) timer = window.setInterval(() => { cur.value = (cur.value + 1) % shots.length }, 5000)
}
function toggle() {
  playing.value = !playing.value
  restart()
}

onMounted(restart)
onBeforeUnmount(() => clearInterval(timer))
</script>

<template>
  <figure class="show" :class="{ still: !playing }">
    <div class="frame">
      <img v-for="(s, i) in shots" :key="s.src" :src="s.src" :alt="i === cur ? s.alt : ''" :class="{ on: i === cur }" width="1496" height="928" :loading="i === 0 ? 'eager' : 'lazy'">
      <div class="loupe" aria-hidden="true" :style="{ backgroundImage: `url(${shots[cur].src})` }" />
    </div>
    <figcaption>
      <div class="tabs" role="tablist" aria-label="Renders">
        <button v-for="(s, i) in shots" :key="s.src" class="tab" role="tab" :aria-selected="i === cur" @click="show(i)">{{ s.label }}</button>
      </div>
      <button v-if="!reduce" class="btn btn-sm" type="button" :aria-pressed="!playing" @click="toggle">{{ playing ? 'Pause' : 'Play' }}</button>
    </figcaption>
  </figure>
</template>

<style scoped>
.show { margin: 0; display: grid; gap: 10px; min-width: 0; }
.frame { position: relative; display: grid; border-radius: 10px; overflow: hidden; border: 1px solid var(--line); background: #000; box-shadow: 0 30px 60px -30px rgb(0 0 0 / 0.8), 0 0 0 1px rgb(242 179 61 / 0.06); }
.frame img { grid-area: 1 / 1; width: 100%; opacity: 0; transition: opacity .7s ease; }
.frame img.on { opacity: 1; }

/* A drifting magnifier: the same render at 7 times its size, kept pixelated so
   the Braille dots stay sharp instead of blurring into a photo. */
.loupe {
  position: absolute; width: 26%; aspect-ratio: 1; border-radius: 50%;
  border: 2px solid var(--amber); box-shadow: 0 0 0 4px rgb(12 18 20 / 0.7), 0 12px 30px rgb(0 0 0 / 0.6);
  background-repeat: no-repeat; background-size: 700% auto; image-rendering: pixelated;
  animation: drift 18s ease-in-out infinite alternate;
}
.still .loupe { animation-play-state: paused; }
@keyframes drift {
  0%   { left: 8%;  top: 30%; background-position: 14% 42%; }
  25%  { left: 38%; top: 40%; background-position: 46% 52%; }
  50%  { left: 64%; top: 18%; background-position: 78% 24%; }
  75%  { left: 30%; top: 58%; background-position: 38% 78%; }
  100% { left: 66%; top: 56%; background-position: 82% 76%; }
}

figcaption { display: flex; align-items: center; justify-content: space-between; gap: 12px; min-width: 0; }
figcaption .tabs { flex: 1; min-width: 0; border-bottom: 0; }
.tab { font-size: 14px; padding: 6px 10px 8px; }
</style>
