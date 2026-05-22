import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Vite config for the Tauri shell. Tauri reads beforeDevCommand / devUrl
// from tauri.conf.json which expects this dev server on :5173.
export default defineConfig({
  plugins: [react()],
  clearScreen: false,
  server: {
    port: 5173,
    strictPort: true,
    host: '127.0.0.1',
  },
  // Tauri reads the build from `dist/`; the conf points there.
  build: {
    target: ['es2020', 'safari14'],
    outDir: 'dist',
    minify: !process.env.TAURI_DEBUG ? 'esbuild' : false,
    sourcemap: !!process.env.TAURI_DEBUG,
  },
})
