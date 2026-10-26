<script setup lang="ts">
import { onMounted, ref, watchEffect } from 'vue'
import { WindowSetTitle } from './wailsjs/runtime/runtime'
import { refreshInfo, store } from './lib/store'
import ContextMenu from './components/ContextMenu.vue'
import EditorLayout from './components/EditorLayout.vue'
import SplashScreen from './components/SplashScreen.vue'
import StartScreen from './components/StartScreen.vue'
import Toasts from './components/Toasts.vue'
import PromptDialog from './components/PromptDialog.vue'

const engine = __ENGINE_VERSION__

const splash = ref(true)

onMounted(refreshInfo)

// The window title names the project and scene; a star marks unsaved changes.
watchEffect(() => {
  const i = store.info
  if (!i?.open) return WindowSetTitle(`ASCDII Engine ${i?.engine ?? ''}`.trim())
  const scene = (i.scene || 'untitled').split('/').pop()
  WindowSetTitle(`${i.unsaved ? '* ' : ''}${scene} - ${i.name} - ASCDII Engine ${i.engine}`)
})
</script>

<template>
  <Transition leave-active-class="transition duration-500" leave-to-class="opacity-0 scale-[1.02]">
    <SplashScreen v-if="splash" :version="store.info?.engine ?? engine" @done="splash = false" />
  </Transition>
  <template v-if="!splash">
    <EditorLayout v-if="store.info?.open" />
    <StartScreen v-else />
  </template>
  <ContextMenu />
  <PromptDialog />
  <Toasts />
</template>
