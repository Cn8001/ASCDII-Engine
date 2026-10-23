<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Build, Project } from '../lib/api'
import { assetPath, call, refreshInfo, store } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'
import TexThumb from './TexThumb.vue'

const all = [
  { id: 'windows', label: 'Windows' },
  { id: 'linux', label: 'Linux' },
]
const available = ref<string[]>([])
const chosen = ref<string[]>([])
const version = ref(store.info?.version ?? '0.1.0')
const result = ref<session.BuildResult | null>(null)
const busy = ref(false)
const single = ref(false) // one executable per platform instead of a folder with data
const icon = ref('') // image asset for Windows executables; '' keeps the player's icon
const canBrand = ref(true)
const images = computed(() => store.assets.filter((a) => a.kind === 'texture' && !a.path.startsWith('mods/')))


onMounted(async () => {
  available.value = (await call(() => Build.Targets())) ?? []
  chosen.value = [...available.value]
  canBrand.value = await Build.CanBrand()
  icon.value = (await call(() => Project.Settings()))?.icon ?? ''

})

function toggle(id: string) {
  if (!available.value.includes(id)) return
  chosen.value = chosen.value.includes(id) ? chosen.value.filter((x) => x !== id) : [...chosen.value, id]
}

// build writes a game folder per chosen platform into build/<platform>/:
// the player, its packed data and SHA256SUMS, ready to hand out as is.
async function build() {
  busy.value = true
  result.value = null
  let r: session.BuildResult | undefined
  for (const t of chosen.value) {
    r = await call(() => Build.Build(t, { single: single.value, version: version.value, icon: icon.value }))
    if (!r) break
  }
  if (r && chosen.value.length > 1) r = { ...r, dir: r.dir.replace(/[\\/][^\\/]+[\\/][^\\/]+$/, '') } as session.BuildResult
  busy.value = false
  if (r) {
    result.value = r
    await refreshInfo()
  }
}
</script>

<template>
  <div class="fixed inset-0 z-40 grid place-items-center bg-black/50 backdrop-blur-[2px] animate-fade" @pointerdown.self="store.buildOpen = false">
    <div class="w-[400px] animate-pop rounded-2xl border border-ink-700 bg-ink-900 p-5 shadow-2xl shadow-black/70">
      <div class="mb-5 flex items-center gap-2">
        <Icon name="hammer" :size="16" class="text-amber-glow" />
        <span class="text-[14px] font-medium">Build {{ store.info?.name }}</span>
        <span class="flex-1" />
        <button class="icon-btn" @click="store.buildOpen = false"><Icon name="x" :size="14" /></button>
      </div>

      <div class="mb-4 grid grid-cols-2 gap-2">
        <button
          v-for="t in all"
          :key="t.id"
          class="flex h-12 items-center justify-center gap-2 rounded-xl border text-[13px] transition"
          :class="!available.includes(t.id) ? 'border-ink-800 text-ink-600' : chosen.includes(t.id) ? 'border-amber-soft bg-amber-glow/[.07] text-ink-100' : 'border-ink-700 text-ink-400 hover:border-ink-600'"
          :title="available.includes(t.id) ? '' : `No player for ${t.id}: place ascdii-player-${t.id}-amd64${t.id === 'windows' ? '.exe' : ''} next to the editor`"
          @click="toggle(t.id)"
        >
          <Icon :name="chosen.includes(t.id) ? 'check' : 'package'" :size="14" :class="chosen.includes(t.id) ? 'text-amber-glow' : ''" />
          {{ t.label }}
        </button>
      </div>

      <p v-if="!available.length" class="mb-3 text-[12px] leading-snug text-amber-soft">No player found. Builds start from the ascdii-player program next to the editor; for another platform, place ascdii-player-linux-amd64 or ascdii-player-windows-amd64.exe beside it.</p>
      <label class="mb-2 flex items-center gap-2 text-[12px] text-ink-300" title="Off: an executable next to a data folder with the assets. On: everything inside one executable.">
        <input v-model="single" type="checkbox" class="accent-[var(--color-amber-glow)]" />
        Single file
        <span class="text-ink-500">{{ single ? 'one executable' : 'executable + data folder' }}</span>
      </label>


      <div class="mt-3 flex items-center gap-3">
        <div class="grid size-12 shrink-0 place-items-center rounded-lg bg-ink-850 ring-1 ring-ink-700">
          <TexThumb v-if="icon" :path="assetPath(icon)" :size="48" />
          <Icon v-else name="image" :size="16" class="text-ink-600" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="mb-1 text-[11px] text-ink-500">Icon of the Windows executable</div>
          <select v-model="icon" class="field" title="A square PNG of 256 pixels or more gives the sharpest result; other sizes are scaled and centred">
            <option value="">Player icon</option>
            <option v-for="a in images" :key="a.path" :value="a.guid || a.path">{{ a.path }}</option>
          </select>
        </div>
      </div>
      <p v-if="!canBrand && icon && chosen.includes('windows')" class="mt-2 text-[11px] leading-snug text-amber-soft">The Linux editor does not write icons into Windows builds; build on Windows to apply this icon. The choice is saved in the project.</p>

      <div class="mt-4 flex items-center gap-2">
        <label class="flex w-28 items-center gap-2 rounded-lg bg-ink-800 px-3 ring-1 ring-ink-700" title="Version of the game; stored in the manifest">
          <span class="text-[11px] text-ink-500">v</span>
          <input v-model="version" class="h-9 w-full bg-transparent font-mono text-[13px] outline-none" />
        </label>
        <span class="flex-1" />
        <button class="btn btn-accent h-9 px-5 text-[13px]" :disabled="busy || !chosen.length" title="A game folder per platform, ready to run and to hand out" @click="build">
          <Icon :name="busy ? 'refresh' : 'hammer'" :size="14" :class="busy ? 'animate-spin' : ''" /> Build
        </button>
      </div>

      <div v-if="result" class="mt-4 flex animate-rise items-center gap-2 rounded-lg bg-ink-850 px-3 py-2.5">
        <Icon name="check" :size="14" class="text-ok" />
        <span class="flex-1 truncate text-[12px] text-ink-300">{{ result.dir.split(/[\\/]/).pop() }}</span>
        <button class="btn h-7" @click="Build.Reveal(result.dir)"><Icon name="folder" :size="13" /> Show</button>
      </div>
    </div>
  </div>
</template>
