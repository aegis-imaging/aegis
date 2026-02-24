import { defineConfig } from 'vite'

export default defineConfig({
  base: '/',
  build: {
    outDir: 'dist',
    target: 'es2020',
  },
  server: {
    port: 3005,
    proxy: {
      '/dicomweb': 'http://localhost:8080',
    },
  },
})
