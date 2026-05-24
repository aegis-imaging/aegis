import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Tauri serves the frontend on a fixed port so the Rust shell can attach to it.
// Mirror the value in tauri.conf.json `build.devUrl`.
const TAURI_DEV_PORT = 1420

export default defineConfig(async () => ({
  plugins: [react()],
  clearScreen: false,
  server: {
    port: TAURI_DEV_PORT,
    strictPort: true,
    host: '127.0.0.1',
    // Tauri injects its own IPC over the file:// scheme in prod, so HMR only
    // needs to work on the dev port.
    hmr: {
      protocol: 'ws',
      host: '127.0.0.1',
      port: TAURI_DEV_PORT + 1,
    },
    watch: {
      // Tauri rebuilds its own Rust crate; don't have Vite spin on src-tauri changes.
      ignored: ['**/src-tauri/**'],
    },
  },
  build: {
    target: 'es2021',
    // Bigger chunks are fine — this isn't network-served.
    chunkSizeWarningLimit: 4096,
  },
  // dcmjs publishes CommonJS — make sure Vite pre-bundles it cleanly.
  optimizeDeps: {
    include: ['dcmjs', 'dicom-parser'],
  },
}))
