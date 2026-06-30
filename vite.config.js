import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [svelte()],
  build: {
    emptyOutDir: true,
    outDir: "frontend/static",
    rollupOptions: {
      input: "frontend/src/main.js",
      output: {
        entryFileNames: "app.js",
        assetFileNames: "app.css",
      },
    },
  },
});
