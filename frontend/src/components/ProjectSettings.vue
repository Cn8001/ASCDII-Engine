<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Project } from '../lib/api'
import { call, refreshInfo, store, toast } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'

const s = ref<session.ProjectSettings | null>(null)
const busy = ref(false)
// The API script is a project script; scripts inside mods/ belong to mods.
const scripts = computed(() => store.assets.filter((a) => a.kind === 'script' && !a.path.startsWith('mods/')))

onMounted(async () => {
  s.value = (await call(() => Project.Settings())) ?? null
})

function close() {
  store.settingsOpen = false
}

async function save() {
  if (!s.value) return
  busy.value = true
  const ok = await call(() => Project.SetSettings(s.value!).then(() => true))
  busy.value = false
  if (ok) {
    toast('Project settings saved', 'ok')
    await refreshInfo()
    close()
  }
}
</script>

<template>
  <div class="fixed inset-0 z-40 grid place-items-center bg-black/50 backdrop-blur-[2px] animate-fade" @pointerdown.self="close">
    <div v-if="s" class="w-[440px] animate-pop rounded-2xl border border-ink-700 bg-ink-900 p-5 shadow-2xl shadow-black/70">
      <div class="mb-5 flex items-center gap-2">
        <Icon name="sliders" :size="16" class="text-amber-glow" />
        <span class="text-[14px] font-medium">Project Settings</span>
        <span class="flex-1" />
        <button class="icon-btn" @click="close"><Icon name="x" :size="14" /></button>
      </div>

      <div class="grid grid-cols-[96px_1fr] items-center gap-x-3 gap-y-2.5 text-[12px]">
        <span class="text-ink-400">Name</span>
        <input v-model="s.name" class="field" title="Name of the game; builds use it for the executable" />
        <span class="text-ink-400">Version</span>
        <input v-model="s.version" class="field font-mono" title="Version of the game, also set by the build dialog" />
        <span class="text-ink-400">Start scene</span>
        <select v-model="s.start" class="field" title="The scene a built game starts with">
          <option v-for="sc in store.info?.scenes ?? []" :key="sc" :value="sc">{{ sc }}</option>
        </select>
        <span class="text-ink-400">Terminal size</span>
        <div class="flex items-center gap-2" title="Columns and rows the game asks for; 0 uses the defaults of 120 by 40. Players resize with their terminal.">
          <input v-model.number="s.cols" type="number" min="0" class="field w-20 font-mono" />
          <span class="text-ink-500">x</span>
          <input v-model.number="s.rows" type="number" min="0" class="field w-20 font-mono" />
        </div>
        <span class="text-ink-400">Frame rate</span>
        <input v-model.number="s.fps" type="number" min="0" max="240" class="field w-20 font-mono" title="Frames per second; 0 uses the default of 30" />
      </div>

      <div class="mt-5 border-t border-ink-800 pt-4">
        <div class="grid grid-cols-[96px_1fr] items-center gap-x-3 gap-y-2.5 text-[12px]">
          <span class="text-ink-400">Display</span>
          <select v-model="s.display" class="field" title="Where the built game shows: in the terminal as Braille, or in a desktop window with one color per dot">
            <option value="terminal">Terminal</option>
            <option value="window">Window</option>
          </select>
          <template v-if="s.display === 'window'">
            <span class="text-ink-400">Window size</span>
            <div class="flex items-center gap-2" title="Size of the window when the game starts, in pixels; 0 opens 1280 by 720. Players can resize it.">
              <input v-model.number="s.width" type="number" min="0" class="field w-20 font-mono" />
              <span class="text-ink-500">x</span>
              <input v-model.number="s.height" type="number" min="0" class="field w-20 font-mono" />
            </div>
            <span class="text-ink-400">Pixel scale</span>
            <input v-model.number="s.scale" type="number" min="0" max="16" class="field w-20 font-mono" title="Screen pixels per dot; 0 picks the scale that shows about 480 dots across" />
            <span class="text-ink-400">Fullscreen</span>
            <label class="flex items-center gap-2 text-ink-200" title="Start in fullscreen; F11 or Alt+Enter switches while playing">
              <input v-model="s.fullscreen" type="checkbox" class="accent-[var(--color-amber-glow)]" />
              <span class="text-ink-500">{{ s.fullscreen ? 'starts fullscreen' : 'starts in a window' }}</span>
            </label>
          </template>
        </div>
        <p v-if="s.display === 'window'" class="mt-2.5 text-[11px] leading-snug text-ink-500">Builds start from <span class="font-mono">ascdii-player-window</span>. Play mode in the editor still shows Braille cells.</p>
      </div>

      <div class="mt-5 border-t border-ink-800 pt-4">
        <label class="flex items-center gap-2 text-[12px] text-ink-200" title="Load the mods players put in the mods folder next to the game">
          <input v-model="s.mods" type="checkbox" class="accent-[var(--color-amber-glow)]" />
          Mods
          <span class="text-ink-500">{{ s.mods ? 'loaded from mods/ next to the game' : 'off' }}</span>
        </label>
        <div class="mt-2.5 grid grid-cols-[96px_1fr] items-center gap-x-3 text-[12px]" :class="s.mods ? '' : 'opacity-40'">
          <span class="text-ink-400">API script</span>
          <select v-model="s.modApi" class="field" :disabled="!s.mods" title="The script that declares what mods can call, read and handle">
            <option value="">None</option>
            <option v-for="a in scripts" :key="a.path" :value="a.guid || a.path">{{ a.path }}</option>
          </select>
        </div>
        <p class="mt-2.5 text-[11px] leading-snug text-ink-500">Play mode loads the project's own <span class="font-mono">mods/</span> folder for testing; builds leave it out.</p>
      </div>

      <div class="mt-5 flex justify-end gap-2">
        <button class="btn h-9 px-4 text-[13px]" @click="close">Cancel</button>
        <button class="btn btn-accent h-9 px-5 text-[13px]" :disabled="busy" @click="save"><Icon name="check" :size="14" /> Save</button>
      </div>
    </div>
  </div>
</template>
