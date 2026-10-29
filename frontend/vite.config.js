import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/auth': 'http://localhost:8080',
      '/users': 'http://localhost:8080',
      '/wallet': 'http://localhost:8080',
      '/menu': 'http://localhost:8080',
      '/orders': 'http://localhost:8080',
      '/queue': 'http://localhost:8080',
    },
  },
})
