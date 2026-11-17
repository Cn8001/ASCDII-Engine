<script setup lang="ts">
import { href, route, type Page } from '../router'
import { repo } from '../content'

const links: { page: Page; label: string }[] = [
  { page: 'home', label: 'Home' },
  { page: 'features', label: 'Features' },
  { page: 'start', label: 'Get started' },
]
</script>

<template>
  <header class="head">
    <div class="wrap row">
      <a class="brand" :href="href('home')"><img :src="'usage/img/logo.svg'" alt="" width="32" height="32"><span>ASCDII Engine</span></a>
      <nav aria-label="Pages">
        <a v-for="l in links" :key="l.page" :href="href(l.page)" :aria-current="route.page === l.page ? 'page' : undefined">{{ l.label }}</a>
        <a href="usage/index.html">Docs</a>
        <a :href="repo" class="gh">GitHub</a>
      </nav>
    </div>
  </header>
</template>

<style scoped>
.head { position: sticky; top: 0; z-index: 5; background: rgb(12 18 20 / 0.82); backdrop-filter: blur(10px); border-bottom: 1px solid var(--line); }
.row { height: var(--head-h); display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.brand { display: flex; align-items: center; gap: 10px; text-decoration: none; font: 600 21px/1 var(--f-display); flex: none; }
.brand img { width: 32px; height: 32px; }
nav { display: flex; gap: 2px; overflow-x: auto; scrollbar-width: none; }
nav::-webkit-scrollbar { display: none; }
nav a { flex: none; text-decoration: none; color: var(--muted); font-size: 15px; padding: 6px 10px; border-radius: 6px; }
nav a:hover { color: var(--ink); background: var(--panel); }
nav a[aria-current="page"] { color: var(--amber); }
@media (max-width: 600px) { .gh { display: none; } nav a { padding: 6px 7px; font-size: 14px; } }
@media (max-width: 480px) { .brand span { display: none; } }
</style>
