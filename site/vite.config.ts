import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { readFileSync } from "node:fs";

// The engine version is set only in version.go (see the Makefile's version target).
function engineVersion(): string {
  const src = readFileSync("../internal/engine/runtime/version/version.go", "utf8");
  return src.match(/Engine = "([^"]+)"/)?.[1] ?? "dev";
}

export default defineConfig({
  // GitHub Pages serves docs/ under /<repo>/; relative URLs work there and from
  // a local file alike.
  base: "./",
  plugins: [vue()],
  // docs/usage, its images and logo are the tutorial's own files; the site
  // links to them and `make site` only replaces its own index and assets.
  publicDir: false,
  build: { outDir: "../docs", emptyOutDir: false },
  define: { __ENGINE_VERSION__: JSON.stringify(engineVersion()) },
});
