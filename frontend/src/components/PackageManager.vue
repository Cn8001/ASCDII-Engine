<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Packages } from '../lib/api'
import { confirmAction } from '../lib/dialog'
import { call, refresh, refreshAssets, store, toast } from '../lib/store'
import type { session } from '../wailsjs/go/models'
import Icon from './Icon.vue'

// PackageManager lists the packages of the project and those that can be
// installed; packages are only ever installed or removed, never edited.
const list = ref<session.Packages | null>(null)
const pick = ref('')
const busy = ref(false)
const rows = computed(() => {
  const l = list.value
  if (!l) return []
  const out: { section: string; p: session.Package }[] = l.installed.map((p) => ({ section: 'In Project', p }))
  for (const p of l.available) if (!p.installed) out.push({ section: 'Available', p })
  return out
})
const current = computed(() => {
  const l = list.value
  if (!l) return null
  return l.installed.find((p) => p.name === pick.value) ?? l.available.find((p) => p.name === pick.value) ?? null
})
const source = computed(() => list.value?.available.find((p) => p.name === pick.value)?.source ?? '')
const update = computed(() => {
  const a = list.value?.available.find((p) => p.name === pick.value)
  return !!a && !!a.installed && a.version !== a.installed
})

async function load() {
  list.value = (await call(() => Packages.List())) ?? null
  if (!current.value) pick.value = rows.value[0]?.p.name ?? ''
}

watch(() => store.packagesOpen, (on) => on && load())

async function changed(text: string) {
  toast(text, 'ok')
  store.assetsVersion++
  await Promise.all([refresh(), refreshAssets(), load()])
}

async function install(src = source.value) {
  if (!src) return
  busy.value = true
  const home = await call(() => Packages.Install(src))
  busy.value = false
  if (home) await changed(home)
}

async function installFile() {
  busy.value = true
  const home = await call(() => Packages.InstallFile())
  busy.value = false
  if (home) {
    pick.value = home.split('/').pop() ?? ''
    await changed(home)
  }
}

async function remove() {
  const p = current.value
  if (!p) return
  if (!(await confirmAction(`Remove ${p.name}?`, 'Its files leave the project. Anything that uses them shows as missing.', 'Remove'))) return
  busy.value = true
  const ok = await call(() => Packages.Remove(p.name).then(() => true))
  busy.value = false
  if (ok) await changed(`${p.name} removed`)
}
</script>

<template>
  <div v-if="store.packagesOpen" class="fixed inset-0 z-[55] grid place-items-center bg-black/50 backdrop-blur-[2px] animate-fade" @pointerdown.self="store.packagesOpen = false" @keydown.esc="store.packagesOpen = false">
    <div class="flex h-[520px] w-[780px] animate-pop overflow-hidden rounded-2xl border border-ink-700 bg-ink-900 shadow-2xl shadow-black/70">
      <div class="flex w-64 shrink-0 flex-col border-r border-ink-800">
        <div class="flex h-11 items-center gap-2 border-b border-ink-800 px-3 text-[13px] font-medium"><Icon name="package" :size="14" class="text-amber-glow" /> Packages</div>
        <div class="flex-1 overflow-auto p-1.5">
          <template v-for="(r, i) in rows" :key="r.section + r.p.name">
            <div v-if="i === 0 || rows[i - 1].section !== r.section" class="px-2 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-ink-500">{{ r.section }}</div>
            <button class="flex h-9 w-full items-center gap-2 rounded-md px-2 text-left text-[12.5px] transition" :class="pick === r.p.name ? 'bg-ink-800 text-ink-100' : 'text-ink-300 hover:bg-ink-850'" @click="pick = r.p.name">
              <Icon name="package" :size="13" :class="r.p.installed ? 'text-amber-soft' : 'text-ink-500'" />
              <span class="flex-1 truncate">{{ r.p.name }}</span>
              <span class="font-mono text-[10.5px] text-ink-500">{{ r.p.installed || r.p.version }}</span>
            </button>
          </template>
          <div v-if="!rows.length" class="px-2 py-6 text-center text-[12px] text-ink-500">No packages</div>
        </div>
        <div class="border-t border-ink-800 p-2">
          <button class="btn h-8 w-full justify-center border border-ink-700 text-ink-300" :disabled="busy" @click="installFile"><Icon name="open" :size="13" /> Install from file</button>
        </div>
      </div>
      <div class="flex min-w-0 flex-1 flex-col">
        <div class="flex h-11 items-center justify-end border-b border-ink-800 px-2">
          <button class="icon-btn" title="Close" @click="store.packagesOpen = false"><Icon name="x" :size="14" /></button>
        </div>
        <div v-if="current" class="flex-1 overflow-auto p-5">
          <div class="text-[18px] font-semibold tracking-tight">{{ current.name }}</div>
          <div class="mt-0.5 font-mono text-[11.5px] text-ink-400">
            {{ current.installed ? `installed ${current.installed}` : `version ${current.version}` }}<span v-if="update"> · {{ list?.available.find((p) => p.name === current?.name)?.version }} available</span>
            <span v-if="current.author"> · {{ current.author }}</span>
          </div>
          <p class="mt-4 text-[13px] leading-relaxed text-ink-200">{{ current.description || 'No description.' }}</p>
          <div class="mt-4 grid grid-cols-[110px_1fr] gap-y-1.5 text-[12px]">
            <span class="text-ink-500">Files</span><span class="text-ink-300">{{ current.files }}</span>
            <span class="text-ink-500">Modules</span><span class="text-ink-300">{{ current.modules?.length ? current.modules.join(', ') : 'none, assets only' }}</span>
            <span class="text-ink-500">Location</span><span class="truncate font-mono text-ink-400">{{ current.installed ? `packages/${current.name}` : current.source }}</span>
          </div>
          <div v-if="current.missing?.length" class="mt-4 flex items-center gap-2 rounded-lg bg-bad/10 px-3 py-2 text-[12px] text-bad">
            <Icon name="alert" :size="13" /> Needs code this editor was built without: {{ current.missing.join(', ') }}
          </div>
          <div class="mt-6 flex gap-2">
            <button v-if="!current.installed" class="btn btn-accent" :disabled="busy || !!current.missing?.length" @click="install()"><Icon name="plus" :size="13" /> Install</button>
            <button v-if="update" class="btn btn-accent" :disabled="busy" @click="install()"><Icon name="refresh" :size="13" /> Update</button>
            <button v-if="current.installed" class="btn border border-ink-700 text-bad" :disabled="busy" @click="remove"><Icon name="trash" :size="13" /> Remove</button>
          </div>
        </div>
        <div v-else class="grid flex-1 place-items-center text-[12px] text-ink-500">Pick a package</div>
      </div>
    </div>
  </div>
</template>
