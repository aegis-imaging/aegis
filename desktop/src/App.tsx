// Desktop shell App.
//
// This is a thin wrapper that hosts the AEGIS upload flow with desktop-only
// enhancements layered on top:
//
//   - Native folder picker (Tauri dialog plugin) — no browser file dialog quirks
//   - Watch-folder mode — auto-upload anything that lands in a configured directory
//   - "Reveal in Finder/Explorer" links on completed studies
//   - Native menu bar (set up in src-tauri/src/lib.rs)
//
// The actual upload-portal React code lives in `frontend/upload-portal/`.
// This shell intentionally stays minimal so it can keep up with upload-portal
// changes without divergence; future iterations may factor the shared bits
// into `frontend/upload-shared/` as a real package.

import { useEffect, useState } from 'react'
import { bridge, type BridgeAvailable } from './desktop-bridge'
import { WatchFolderPanel } from './WatchFolderPanel'
import { bulkUpload, type BulkProgress } from '@aegis/client'

interface BootstrapState {
  apiBaseUrl: string
  projectSlug: string
  uploaderEmail: string
}

export function App() {
  const [available, setAvailable] = useState<BridgeAvailable>({ available: false, reason: 'init' })
  const [config, setConfig] = useState<BootstrapState>({
    apiBaseUrl: 'https://api.aegisimaging.ai',
    projectSlug: 'default',
    uploaderEmail: '',
  })
  const [bulkProgress, setBulkProgress] = useState<BulkProgress | null>(null)

  useEffect(() => {
    let cancelled = false
    bridge.probe().then(r => { if (!cancelled) setAvailable(r) })
    return () => { cancelled = true }
  }, [])

  return (
    <div style={{
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      maxWidth: 900, margin: '0 auto', padding: '32px 24px',
    }}>
      <header style={{ marginBottom: 32 }}>
        <h1 style={{ margin: '0 0 4px', fontSize: 28, fontWeight: 700 }}>AEGIS Desktop</h1>
        <p style={{ margin: 0, color: '#6b7280' }}>
          Anonymization &amp; Exchange Gateway for Imaging Studies — native edition
        </p>
        {!available.available && (
          <div style={{
            marginTop: 12, padding: '8px 12px', background: '#fef3c7',
            border: '1px solid #fde68a', borderRadius: 6, fontSize: 13, color: '#92400e',
          }}>
            Running in browser preview mode (no Tauri bridge available). Native filesystem
            features (watch folder, native file picker) are disabled. To get the full app,
            run via <code>npm run tauri:dev</code> or install the released binary.
          </div>
        )}
      </header>

      <ConfigPanel value={config} onChange={setConfig} />

      <DropZone
        bridgeAvailable={available.available}
        onFilesSelected={files => runBulkUpload(files, config, setBulkProgress)}
      />

      {available.available && (
        <WatchFolderPanel
          config={config}
          onProgress={setBulkProgress}
        />
      )}

      {bulkProgress && (
        <ProgressView progress={bulkProgress} />
      )}
    </div>
  )
}

// ── Subcomponents ────────────────────────────────────────────────────────

function ConfigPanel({ value, onChange }: {
  value: BootstrapState
  onChange: (next: BootstrapState) => void
}) {
  return (
    <section style={{
      border: '1px solid #e5e7eb', borderRadius: 8, padding: 16, marginBottom: 20,
    }}>
      <h2 style={{ margin: '0 0 12px', fontSize: 16 }}>Connection</h2>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <label>
          <div style={{ fontSize: 13, color: '#374151', marginBottom: 4 }}>AEGIS API URL</div>
          <input
            type="url"
            value={value.apiBaseUrl}
            onChange={e => onChange({ ...value, apiBaseUrl: e.target.value })}
            style={inputStyle()}
          />
        </label>
        <label>
          <div style={{ fontSize: 13, color: '#374151', marginBottom: 4 }}>Project slug</div>
          <input
            type="text"
            value={value.projectSlug}
            onChange={e => onChange({ ...value, projectSlug: e.target.value })}
            style={inputStyle()}
          />
        </label>
        <label style={{ gridColumn: '1 / span 2' }}>
          <div style={{ fontSize: 13, color: '#374151', marginBottom: 4 }}>
            Your email (for upload-confirmation messages)
          </div>
          <input
            type="email"
            placeholder="you@institution.edu"
            value={value.uploaderEmail}
            onChange={e => onChange({ ...value, uploaderEmail: e.target.value })}
            style={inputStyle()}
          />
        </label>
      </div>
    </section>
  )
}

function DropZone({ bridgeAvailable, onFilesSelected }: {
  bridgeAvailable: boolean
  onFilesSelected: (files: File[]) => void
}) {
  return (
    <section
      style={{
        border: '2px dashed #6b7280', borderRadius: 12, padding: '36px 24px',
        textAlign: 'center', marginBottom: 20, cursor: 'pointer', background: '#fafafa',
      }}
      onClick={async () => {
        if (bridgeAvailable) {
          const files = await bridge.pickFolderAsFiles()
          if (files.length > 0) onFilesSelected(files)
        } else {
          // Browser fallback — invisible <input> click.
          const input = document.createElement('input')
          input.type = 'file'
          input.multiple = true
          ;(input as unknown as { webkitdirectory: string }).webkitdirectory = ''
          input.onchange = () => {
            if (input.files) onFilesSelected(Array.from(input.files))
          }
          input.click()
        }
      }}
      onDragOver={e => e.preventDefault()}
      onDrop={e => {
        e.preventDefault()
        const files: File[] = []
        for (let i = 0; i < e.dataTransfer.files.length; i++) files.push(e.dataTransfer.files[i])
        if (files.length > 0) onFilesSelected(files)
      }}
    >
      <div style={{ fontSize: 40, marginBottom: 8 }}>📁</div>
      <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 4 }}>
        Drop DICOM files or pick a folder
      </div>
      <div style={{ fontSize: 13, color: '#6b7280' }}>
        Multi-study folders are auto-detected and uploaded in bulk
      </div>
    </section>
  )
}

function ProgressView({ progress }: { progress: BulkProgress }) {
  return (
    <section style={{
      border: '1px solid #e5e7eb', borderRadius: 8, padding: 16, marginTop: 20,
    }}>
      <h2 style={{ margin: '0 0 8px', fontSize: 16 }}>Upload progress</h2>
      <div style={{ fontSize: 13, color: '#374151', marginBottom: 8 }}>
        Phase: <strong>{progress.phase}</strong> ·
        Completed: {progress.studiesCompleted}/{progress.totalStudies} ·
        Failed: {progress.studiesFailed}
      </div>
      <ul style={{ margin: 0, padding: 0, listStyle: 'none', fontSize: 13 }}>
        {progress.studies.map(s => (
          <li key={s.studyInstanceUid} style={{ display: 'flex', gap: 8, padding: '4px 0' }}>
            <span style={{ color: statusColor(s.status), fontWeight: 500, minWidth: 90 }}>
              {s.status}
            </span>
            <span style={{ flex: 1 }}>
              {s.patientId || '—'} · {s.studyDescription} ({s.fileCount} files)
            </span>
            {s.durationMs != null && (
              <span style={{ color: '#9ca3af' }}>{(s.durationMs / 1000).toFixed(1)}s</span>
            )}
          </li>
        ))}
      </ul>
    </section>
  )
}

// ── Helpers ──────────────────────────────────────────────────────────────

async function runBulkUpload(
  files: File[],
  config: BootstrapState,
  setProgress: (p: BulkProgress) => void
) {
  await bulkUpload(files, {
    projectSlug: config.projectSlug,
    concurrency: 2,
    uploadOptions: {
      apiBaseUrl: config.apiBaseUrl,
      uploaderEmail: config.uploaderEmail || undefined,
    },
    onProgress: p => setProgress({ ...p, studies: p.studies.map(s => ({ ...s })) }),
  })
}

function inputStyle(): React.CSSProperties {
  return {
    width: '100%', padding: '6px 8px', border: '1px solid #d1d5db',
    borderRadius: 6, fontSize: 14, boxSizing: 'border-box',
  }
}

function statusColor(status: string): string {
  switch (status) {
    case 'completed': return '#15803d'
    case 'uploading': return '#2563eb'
    case 'duplicate': return '#a16207'
    case 'failed': return '#b91c1c'
    case 'cancelled': return '#6b7280'
    default: return '#9ca3af'
  }
}
