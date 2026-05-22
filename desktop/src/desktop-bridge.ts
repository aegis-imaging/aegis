// Wrapper around Tauri's JS API. Falls back to "unavailable" when running
// in a regular browser (Vite dev mode without Tauri shell), so the app can
// still be developed without launching the full Tauri shell every time.

export interface BridgeAvailable {
  available: boolean
  reason?: string
}

export interface WatchEvent {
  /** Absolute path on disk that changed. */
  path: string
  /** kind: 'create' | 'modify' | 'remove' — matches notify-rs's event kinds. */
  kind: string
}

interface BridgeFns {
  probe(): Promise<BridgeAvailable>
  pickFolderAsFiles(): Promise<File[]>
  watchFolder(path: string, onEvent: (e: WatchEvent) => void): Promise<() => Promise<void>>
  openInFinder(path: string): Promise<void>
  /** Pop the OS-native folder picker; returns the absolute path or null. */
  pickFolderPath(): Promise<string | null>
}

// Detect at import time whether we're inside Tauri. Tauri injects
// `window.__TAURI_INTERNALS__` (v2) or `window.__TAURI__` (v1).
function isTauri(): boolean {
  if (typeof window === 'undefined') return false
  return (
    '__TAURI_INTERNALS__' in window ||
    '__TAURI__' in window
  )
}

// Dynamic imports routed through string variables so TS doesn't try to
// resolve the module specifiers at compile time. Lets this file type-check
// even when the Tauri packages aren't installed in the current env (e.g.
// CI matrix runners that only validate non-desktop targets).
// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function loadTauri(): Promise<any> {
  const corePath = '@tauri-apps/api/core'
  const dialogPath = '@tauri-apps/plugin-dialog'
  const fsPath = '@tauri-apps/plugin-fs'
  const [core, dialog, fs] = await Promise.all([
    import(/* @vite-ignore */ corePath),
    import(/* @vite-ignore */ dialogPath),
    import(/* @vite-ignore */ fsPath),
  ])
  return { invoke: core.invoke, dialog, fs }
}

async function probe(): Promise<BridgeAvailable> {
  if (!isTauri()) return { available: false, reason: 'not-in-tauri' }
  try {
    await loadTauri()
    return { available: true }
  } catch (e) {
    return { available: false, reason: (e as Error).message }
  }
}

async function pickFolderPath(): Promise<string | null> {
  if (!isTauri()) return null
  const { dialog } = await loadTauri()
  const selected = await dialog.open({ directory: true, multiple: false })
  return typeof selected === 'string' ? selected : null
}

async function pickFolderAsFiles(): Promise<File[]> {
  if (!isTauri()) return []
  const path = await pickFolderPath()
  if (!path) return []
  return readDirAsFiles(path)
}

async function readDirAsFiles(folder: string): Promise<File[]> {
  const { fs } = await loadTauri()
  // Recursively read every regular file under `folder` and turn each into a
  // browser-style `File` object so the rest of the upload pipeline doesn't
  // need to special-case desktop mode.
  const files: File[] = []
  async function walk(dir: string): Promise<void> {
    const entries = await fs.readDir(dir)
    for (const entry of entries) {
      const fullPath = `${dir}/${entry.name}`
      if (entry.isDirectory) {
        await walk(fullPath)
      } else if (entry.isFile) {
        const bytes = await fs.readFile(fullPath)
        const blob = new Blob([new Uint8Array(bytes)])
        const file = new File([blob], entry.name, { lastModified: Date.now() })
        // Annotate with the full path so downstream can show "reveal in finder".
        ;(file as unknown as { __aegisAbsPath?: string }).__aegisAbsPath = fullPath
        files.push(file)
      }
    }
  }
  await walk(folder)
  return files
}

async function watchFolder(
  path: string,
  onEvent: (e: WatchEvent) => void
): Promise<() => Promise<void>> {
  if (!isTauri()) {
    return async () => {}
  }
  const { invoke } = await loadTauri()
  const eventPath = '@tauri-apps/api/event'
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const apiEvent: any = await import(/* @vite-ignore */ eventPath)
  const id: number = await invoke('start_watch', { path })
  const unlisten = await apiEvent.listen(`fs-event:${id}`, (e: { payload: WatchEvent }) => onEvent(e.payload))
  return async () => {
    await invoke('stop_watch', { id })
    unlisten()
  }
}

async function openInFinder(path: string): Promise<void> {
  if (!isTauri()) return
  const shellPath = '@tauri-apps/plugin-shell'
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const shell: any = await import(/* @vite-ignore */ shellPath)
  await shell.open(path)
}

export const bridge: BridgeFns = {
  probe,
  pickFolderAsFiles,
  pickFolderPath,
  watchFolder,
  openInFinder,
}
