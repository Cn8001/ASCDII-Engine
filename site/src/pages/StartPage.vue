<script setup lang="ts">
import { computed } from 'vue'
import CliTerminal from '../components/CliTerminal.vue'
import { goExample } from '../content'
import { href, route } from '../router'

const tabs = [
  { id: '', label: 'In the editor' },
  { id: 'cli', label: 'From the command line' },
  { id: 'go', label: 'From Go' },
]
const tab = computed(() => (tabs.some((t) => t.id === route.value.sub) ? route.value.sub : ''))

const steps = [
  { title: 'Install', text: 'Extract the archive for Windows or Linux into any folder. The programs must stay together in that folder.', link: 'usage/install.html' },
  { title: 'Open the editor', text: 'Run ascdii, for example with a double click. The editor opens at its start screen.', link: 'usage/editor.html' },
  { title: 'Create a project', text: 'Pick a template, enter a name and a location, and press Create. The Dragonfall template gives you a whole sample game to take apart.', link: 'usage/first-game.html' },
  { title: 'Play it', text: 'Press Space to play the scene inside the editor and Esc to stop.', link: 'usage/first-game.html' },
  { title: 'Build it', text: 'Press Ctrl+B. The build folder holds the player and your game, ready to share.', link: 'usage/building.html' },
]
</script>

<template>
  <section class="page">
    <div class="wrap">
      <div class="page-head">
        <h1>Get started</h1>
        <p>Make your first game in the editor, from the command line or straight from Go. The <a href="usage/first-game.html">first game guide</a> walks through every step.</p>
      </div>
      <nav class="tabs" aria-label="Ways to start">
        <a v-for="t in tabs" :key="t.id" class="tab" :href="href('start', t.id)" :aria-selected="tab === t.id">{{ t.label }}</a>
      </nav>

      <div class="panel">
        <div v-if="tab === ''" class="split">
          <ol class="plain steps">
            <li v-for="(s, i) in steps" :key="s.title">
              <span class="n">{{ i + 1 }}</span>
              <div>
                <h3>{{ s.title }}</h3>
                <p>{{ s.text }} <a :href="s.link">More</a></p>
              </div>
            </li>
          </ol>
          <a class="shot-frame" href="usage/img/editor-start.png" target="_blank" rel="noopener">
            <img :src="'usage/img/editor-start.png'" alt="The start screen: templates for a new project on the left, recent projects on the right">
          </a>
        </div>

        <div v-else-if="tab === 'cli'" class="split">
          <div class="copy">
            <h2>One tool for everything</h2>
            <p><code>ascdii</code> opens the editor when started without arguments. With arguments it creates, plays, previews and builds projects, and installs packages.</p>
            <p>Games need a terminal with 24 bit colour and a font with Braille characters, such as Windows Terminal, WezTerm, Kitty or iTerm2.</p>
            <a class="more" href="usage/cli.html">CLI reference</a>
          </div>
          <CliTerminal />
        </div>

        <div v-else class="split">
          <div class="copy">
            <h2>Or write it in Go</h2>
            <p><code>pkg/ascdii</code> is the engine's one public package. A scene, a camera, some ground, a crate that bounces and a hero you can walk around, in a dozen lines.</p>
            <p>Run <code>make hello</code> in the repository for the smallest complete program.</p>
            <a class="more" href="usage/go-api.html">Go API</a>
          </div>
          <pre class="code" v-html="goExample" />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.page-head a { color: var(--amber); }
.panel { padding-top: 28px; }
.split { display: grid; grid-template-columns: minmax(0, 0.85fr) minmax(0, 1.15fr); gap: 40px; align-items: start; }
@media (max-width: 920px) { .split { grid-template-columns: minmax(0, 1fr); } }
.steps { display: grid; gap: 18px; }
.steps li { display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 14px; }
.n { width: 30px; height: 30px; border-radius: 50%; display: grid; place-items: center; font: 600 16px/1 var(--f-display); background: var(--amber); color: var(--amber-ink); }
.steps h3 { font-size: 18px; margin-bottom: 4px; }
.steps p { color: var(--muted); font-size: 15px; }
.steps a, .more { color: var(--amber); }
.copy { display: grid; gap: 14px; }
.copy h2 { font-size: 30px; color: var(--meadow); }
.copy p { color: var(--muted); max-width: 52ch; }
.copy code { color: var(--river); }
.more { justify-self: start; font-weight: 500; }
</style>
