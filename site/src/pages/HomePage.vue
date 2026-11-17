<script setup lang="ts">
import RenderShowcase from '../components/RenderShowcase.vue'
import { repo, version } from '../content'
import { href } from '../router'
</script>

<template>
  <section class="page">
    <div class="wrap hero">
      <div class="intro">
        <h1>Games drawn in Braille.</h1>
        <p class="lede">ASCDII Engine is a 2.5D game engine that renders into Unicode Braille characters. Build worlds in a desktop editor, script them in Lua, and ship games that run in any terminal with 24 bit colour.</p>
        <ul class="plain specs">
          <li><b>2 × 4</b><span>lit, depth tested dots in every terminal cell</span></li>
          <li><b>2</b><span>colours per cell, picked from those dots</span></li>
          <li><b>Lua</b><span>scripts that reload while the game runs</span></li>
        </ul>
        <div class="actions">
          <a class="btn btn-primary" :href="href('start')">Get started</a>
          <a class="btn" href="usage/index.html">Read the docs</a>
        </div>
        <p class="meta">Version {{ version }} for Windows and Linux. <a :href="repo">Source on GitHub</a>.</p>
      </div>
      <RenderShowcase />
    </div>
  </section>
</template>

<style scoped>
.hero { display: grid; grid-template-columns: minmax(0, 380px) minmax(0, 1fr); gap: 48px; align-items: center; }
@media (max-width: 1060px) { .hero { grid-template-columns: minmax(0, 1fr); gap: 36px; } }
.intro { display: grid; gap: 18px; }
h1 { font-size: clamp(40px, 4.6vw, 60px); color: var(--amber); text-shadow: 0 0 28px rgb(242 179 61 / 0.25); }
.lede { color: var(--muted); font-size: 17px; }
.specs { display: grid; gap: 8px; }
.specs li { display: grid; grid-template-columns: 56px minmax(0, 1fr); gap: 12px; align-items: baseline; }
.specs b { font: 600 22px/1 var(--f-display); color: var(--meadow); }
.specs li:nth-child(2) b { color: var(--river); }
.specs li:nth-child(3) b { color: var(--ember); }
.specs span { color: var(--muted); font-size: 15px; }
.actions { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; max-width: 380px; }
.meta { font-size: 14px; color: var(--muted); }
.meta a { color: var(--ink); }
</style>
