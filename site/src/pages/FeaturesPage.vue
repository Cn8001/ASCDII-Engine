<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { allFeatures, groups } from '../content'
import { href, route } from '../router'

const feature = computed(() => allFeatures.find((f) => f.id === route.value.sub) ?? allFeatures[0])
const shot = ref(0)
watch(feature, () => { shot.value = 0 })
const current = computed(() => feature.value.shots?.[shot.value] ?? feature.value.shots?.[0])

function pick(e: Event) {
  location.hash = href('features', (e.target as HTMLSelectElement).value)
}
</script>

<template>
  <section class="page">
    <div class="wrap">
      <div class="page-head">
        <h1>Features</h1>
        <p>Everything you need to make a game that runs in a terminal. Pick a topic to see it.</p>
      </div>

      <div class="layout">
        <nav class="side" aria-label="Features">
          <div v-for="g in groups" :key="g.title" class="group">
            <h2>{{ g.title }}</h2>
            <ul class="plain">
              <li v-for="f in g.features" :key="f.id">
                <a :href="href('features', f.id)" :aria-current="f.id === feature.id ? 'page' : undefined">{{ f.title }}</a>
              </li>
            </ul>
          </div>
        </nav>

        <label class="picker">
          <span>Topic</span>
          <select id="feature-picker" :value="feature.id" @change="pick">
            <optgroup v-for="g in groups" :key="g.title" :label="g.title">
              <option v-for="f in g.features" :key="f.id" :value="f.id">{{ f.title }}</option>
            </optgroup>
          </select>
        </label>

        <article class="detail">
          <div class="text">
            <h2>{{ feature.title }}</h2>
            <p v-for="(p, i) in feature.body" :key="i">{{ p }}</p>
            <ul v-if="feature.facts" class="plain facts" aria-label="File types">
              <li v-for="f in feature.facts" :key="f">{{ f }}</li>
            </ul>
            <a class="guide" :href="`usage/${feature.guide}`">Read the guide</a>
          </div>
          <div class="media">
            <template v-if="current">
              <div v-if="feature.shots && feature.shots.length > 1" class="tabs" role="tablist" aria-label="Screenshots">
                <button v-for="(s, i) in feature.shots" :key="s.src" class="tab" role="tab" :aria-selected="i === shot" @click="shot = i">{{ s.label }}</button>
              </div>
              <a class="shot-frame" :href="current.src" target="_blank" rel="noopener">
                <img :key="current.src" :src="current.src" :alt="current.alt">
              </a>
            </template>
            <pre v-else-if="feature.code" class="code" v-html="feature.code" />
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<style scoped>
.layout { display: grid; grid-template-columns: 220px minmax(0, 1fr); gap: 40px; align-items: start; }
.side { position: sticky; top: calc(var(--head-h) + 24px); display: grid; gap: 16px; }
.group h2 { font: 600 13px/1.3 var(--f-body); color: var(--meadow); margin-bottom: 6px; }
.group a { display: block; text-decoration: none; font-size: 15px; line-height: 1.4; padding: 4px 10px; margin-left: -10px; border-radius: 6px; color: var(--muted); border-left: 2px solid transparent; }
.group a:hover { color: var(--ink); background: var(--panel); }
.group a[aria-current="page"] { color: var(--ink); border-left-color: var(--amber); border-radius: 0 6px 6px 0; background: var(--panel); }
.picker { display: none; }

.detail { display: grid; grid-template-columns: minmax(0, 0.75fr) minmax(0, 1.25fr); gap: 36px; align-items: start; }
.text { display: grid; gap: 14px; }
.text h2 { font-size: 34px; color: var(--amber); }
.text p { color: var(--muted); max-width: 52ch; }
.facts { display: flex; flex-wrap: wrap; gap: 8px; }
.facts li { font-family: var(--f-mono); font-size: 12px; padding: 3px 10px; border: 1px solid var(--line); border-radius: 999px; color: var(--river); }
.guide { justify-self: start; color: var(--amber); font-weight: 500; }
.media { display: grid; gap: 12px; min-width: 0; }
.media img { width: 100%; animation: show .25s ease; }
@keyframes show { from { opacity: 0; } }

@media (max-width: 1180px) { .detail { grid-template-columns: minmax(0, 1fr); gap: 24px; } }
@media (max-width: 760px) {
  .layout { grid-template-columns: minmax(0, 1fr); gap: 20px; }
  .side { display: none; }
  .picker { display: grid; gap: 6px; font-size: 14px; color: var(--muted); }
  .picker select { font: 500 16px var(--f-body); color: var(--ink); background: var(--panel); border: 1px solid var(--line); border-radius: 6px; padding: 10px 12px; }
}
</style>
