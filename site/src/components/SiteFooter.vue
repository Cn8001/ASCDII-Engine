<script setup lang="ts">
import { repo, version } from '../content'
</script>

<template>
  <footer>
    <div class="wrap row">
      <span>ASCDII Engine {{ version }} is open source under the MIT licence.</span>
      <nav aria-label="Project links">
        <a href="usage/index.html">Docs</a>
        <a href="usage/credits.html">Credits</a>
        <a :href="`${repo}/blob/master/LICENSE`">Licence</a>
        <a :href="repo">GitHub</a>
      </nav>
    </div>
  </footer>
</template>

<style scoped>
footer { border-top: 1px solid var(--line); padding-block: 24px 32px; color: var(--muted); font-size: 14px; }
.row { display: flex; justify-content: space-between; gap: 16px 28px; flex-wrap: wrap; }
nav { display: flex; gap: 18px; flex-wrap: wrap; }
</style>
