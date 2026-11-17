<script setup lang="ts">
import { computed, watchEffect } from 'vue'
import { route } from './router'
import SiteHeader from './components/SiteHeader.vue'
import SiteFooter from './components/SiteFooter.vue'
import HomePage from './pages/HomePage.vue'
import StartPage from './pages/StartPage.vue'

const pages = { home: HomePage, start: StartPage }
const titles = { home: 'ASCDII Engine', start: 'Get started' }
const current = computed(() => pages[route.value.page])

watchEffect(() => {
  const t = titles[route.value.page]
  document.title = route.value.page === 'home' ? t : `${t}, ASCDII Engine`
})
</script>

<template>
  <SiteHeader />
  <main>
    <component :is="current" />
  </main>
  <SiteFooter />
</template>
