import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

// Go embeds dist/, which must never be empty; vite clears it on every build.
const keepDist = (): Plugin => ({
  name: 'keep-dist',
  closeBundle() {
    writeFileSync(resolve(__dirname, 'dist/.gitkeep'), '')
  },
})

export default defineConfig({
  plugins: [vue(), tailwindcss(), keepDist()],
  build: { target: 'es2022', chunkSizeWarningLimit: 1200 },
})
