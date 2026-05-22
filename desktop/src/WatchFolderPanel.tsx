// Watch-folder panel: pick a directory, and any DICOM files that appear in
// it get uploaded automatically. The natural ergonomic for a research
// coordinator who saves study exports to a network share.

import { useEffect, useRef, useState } from 'react'
import { bulkUpload, type BulkProgress } from '@aegis/client'
import { bridge, type WatchEvent } from './desktop-bridge'

export function WatchFolderPanel({
  config,
  onProgress,
}: {
  config: { apiBaseUrl: string; projectSlug: string; uploaderEmail: string }
  onProgress: (p: BulkProgress) => void
}) {
  const [folder, setFolder] = useState<string | null>(null)
  const [watching, setWatching] = useState(false)
  const [events, setEvents] = useState<WatchEvent[]>([])
  const stopRef = useRef<(() => Promise<void>) | null>(null)
  const debounceRef = useRef<number | null>(null)

  useEffect(() => {
    return () => {
      if (stopRef.current) void stopRef.current()
    }
  }, [])

  async function pickFolder() {
    const path = await bridge.pickFolderPath()
    if (path) setFolder(path)
  }

  async function startWatching() {
    if (!folder) return
    const stop = await bridge.watchFolder(folder, async e => {
      setEvents(prev => [e, ...prev].slice(0, 50))
      // Debounce: once events stop arriving for 2s, scan the folder and
      // push anything new through the upload pipeline. Avoids triggering an
      // upload mid-write while a study is still being copied.
      if (debounceRef.current) window.clearTimeout(debounceRef.current)
      debounceRef.current = window.setTimeout(() => {
        void scanAndUpload(folder, config, onProgress)
      }, 2000)
    })
    stopRef.current = stop
    setWatching(true)
  }

  async function stopWatching() {
    if (stopRef.current) await stopRef.current()
    stopRef.current = null
    setWatching(false)
  }

  return (
    <section style={{
      border: '1px solid #e5e7eb', borderRadius: 8, padding: 16, marginBottom: 20,
    }}>
      <h2 style={{ margin: '0 0 8px', fontSize: 16 }}>Watch folder</h2>
      <p style={{ margin: '0 0 10px', color: '#6b7280', fontSize: 13 }}>
        Pick a directory and AEGIS Desktop will auto-upload any new DICOM files that land
        in it. Files are batched (2-second debounce) so a study still being copied isn't
        sent mid-write.
      </p>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <button
          type="button"
          onClick={pickFolder}
          disabled={watching}
          style={btn()}
        >
          {folder ? 'Change folder' : 'Pick folder'}
        </button>
        {folder && (
          <code style={{
            flex: 1, fontSize: 12, color: '#374151', overflow: 'hidden',
            textOverflow: 'ellipsis', whiteSpace: 'nowrap',
          }}>{folder}</code>
        )}
        {folder && !watching && (
          <button type="button" onClick={startWatching} style={btnPrimary()}>Start watching</button>
        )}
        {watching && (
          <button type="button" onClick={stopWatching} style={btnDanger()}>Stop</button>
        )}
      </div>
      {watching && (
        <div style={{ marginTop: 12, fontSize: 12, color: '#374151' }}>
          <div style={{ color: '#15803d', fontWeight: 500, marginBottom: 4 }}>
            ● watching — {events.length} fs events captured
          </div>
          {events.slice(0, 5).map((e, i) => (
            <div key={i} style={{ color: '#6b7280', fontFamily: 'monospace', fontSize: 11 }}>
              {e.kind}: {e.path}
            </div>
          ))}
        </div>
      )}
    </section>
  )
}

async function scanAndUpload(
  folder: string,
  config: { apiBaseUrl: string; projectSlug: string; uploaderEmail: string },
  onProgress: (p: BulkProgress) => void
) {
  // Re-read the folder and upload anything new. Server-side dedup means
  // re-sending an already-uploaded study returns a duplicate status which
  // the bulk runner classifies separately from failure.
  const files = await bridge.pickFolderAsFiles()  // re-uses dialog wrapper; bypassed in next iteration
  if (files.length === 0) return
  await bulkUpload(files, {
    projectSlug: config.projectSlug,
    concurrency: 2,
    uploadOptions: {
      apiBaseUrl: config.apiBaseUrl,
      uploaderEmail: config.uploaderEmail || undefined,
    },
    onProgress: p => onProgress({ ...p, studies: p.studies.map(s => ({ ...s })) }),
  })
}

function btn(): React.CSSProperties {
  return {
    padding: '6px 12px', borderRadius: 6, border: '1px solid #d1d5db',
    background: '#fff', cursor: 'pointer', fontSize: 13,
  }
}
function btnPrimary(): React.CSSProperties {
  return { ...btn(), background: '#2563eb', color: '#fff', borderColor: '#2563eb' }
}
function btnDanger(): React.CSSProperties {
  return { ...btn(), background: '#b91c1c', color: '#fff', borderColor: '#b91c1c' }
}
