<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { refreshInfo, store } from './lib/store'
import ContextMenu from './components/ContextMenu.vue'
import EditorLayout from './components/EditorLayout.vue'
import SplashScreen from './components/SplashScreen.vue'
import StartScreen from './components/StartScreen.vue'
import Toasts from './components/Toasts.vue'
import PromptDialog from './components/PromptDialog.vue'

const splash = ref(true)

onMounted(refreshInfo)
</script>

<template>
  <Transition leave-active-class="transition duration-500" leave-to-class="opacity-0 scale-[1.02]">
    <SplashScreen v-if="splash" :version="store.info?.engine ?? '0.0.1'" @done="splash = false" />
  </Transition>
  <template v-if="!splash">
    <EditorLayout v-if="store.info?.open" />
    <StartScreen v-else />
  </template>
  <ContextMenu />
  <PromptDialog />
  <Toasts />
</template>
