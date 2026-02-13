import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: '../auth/ui/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/oauth2': 'http://localhost:8080',
      '/.well-known': 'http://localhost:8080',
      '/admin': 'http://localhost:8080',
    },
  },
})