import { useEffect, useRef, useState, useCallback } from 'react'
import { Niivue, SLICE_TYPE } from '@niivue/niivue'

type AnalyticsFile = {
  path: string
  name: string
  size: number
  tool: string
  file_type: string
}

type SliceMode = 'axial' | 'coronal' | 'sagittal' | 'multiplanar' | 'render'

const SLICE_MODES: { label: string; value: SliceMode }[] = [
  { label: 'Axial', value: 'axial' },
  { label: 'Coronal', value: 'coronal' },
  { label: 'Sagittal', value: 'sagittal' },
  { label: 'Multi', value: 'multiplanar' },
  { label: '3D', value: 'render' },
]

function sliceModeToNiivue(mode: SliceMode): number {
  switch (mode) {
    case 'axial': return SLICE_TYPE.AXIAL
    case 'coronal': return SLICE_TYPE.CORONAL
    case 'sagittal': return SLICE_TYPE.SAGITTAL
    case 'multiplanar': return SLICE_TYPE.MULTIPLANAR
    case 'render': return SLICE_TYPE.RENDER
    default: return SLICE_TYPE.MULTIPLANAR
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function NiivueViewer({ studyId }: { studyId: string }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const nvRef = useRef<Niivue | null>(null)
  const [files, setFiles] = useState<AnalyticsFile[]>([])
  const [niftiFiles, setNiftiFiles] = useState<AnalyticsFile[]>([])
  const [selectedFile, setSelectedFile] = useState<AnalyticsFile | null>(null)
  const [overlayFile, setOverlayFile] = useState<AnalyticsFile | null>(null)
  const [sliceMode, setSliceMode] = useState<SliceMode>('multiplanar')
  const [loading, setLoading] = useState(true)
  const [viewerReady, setViewerReady] = useState(false)
  const [error, setError] = useState('')
  const [opacity, setOpacity] = useState(0.5)

  // Fetch analytics files
  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const res = await fetch(`/api/studies/${studyId}/analytics-files`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const data = await res.json()
        if (cancelled) return
        const allFiles: AnalyticsFile[] = data.files || []
        setFiles(allFiles)
        const niis = allFiles.filter(f => f.file_type === 'nifti')
        setNiftiFiles(niis)
        if (niis.length > 0) setSelectedFile(niis[0])
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load files')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => { cancelled = true }
  }, [studyId])

  // Initialize Niivue
  useEffect(() => {
    if (!canvasRef.current || !selectedFile) return

    const nv = new Niivue({
      backColor: [0.15, 0.15, 0.15, 1],
      show3Dcrosshair: true,
      isColorbar: true,
    })
    nvRef.current = nv
    nv.attachToCanvas(canvasRef.current)

    const fileUrl = `/api/studies/${studyId}/analytics-files/${encodeURIComponent(selectedFile.path)}`
    const volumes: { url: string; colormap?: string; opacity?: number }[] = [{ url: fileUrl }]

    if (overlayFile) {
      const overlayUrl = `/api/studies/${studyId}/analytics-files/${encodeURIComponent(overlayFile.path)}`
      volumes.push({ url: overlayUrl, colormap: 'hot', opacity })
    }

    nv.loadVolumes(volumes).then(() => {
      nv.setSliceType(sliceModeToNiivue(sliceMode))
      setViewerReady(true)
    }).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : 'Failed to load volume')
    })

    return () => {
      nvRef.current = null
      setViewerReady(false)
    }
  }, [studyId, selectedFile, overlayFile, opacity])

  // Update slice mode when changed
  useEffect(() => {
    if (nvRef.current && viewerReady) {
      nvRef.current.setSliceType(sliceModeToNiivue(sliceMode))
    }
  }, [sliceMode, viewerReady])

  const handleFileSelect = useCallback((file: AnalyticsFile) => {
    setSelectedFile(file)
    setOverlayFile(null)
    setError('')
  }, [])

  const handleOverlaySelect = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    const path = e.target.value
    if (!path) {
      setOverlayFile(null)
      return
    }
    const file = niftiFiles.find(f => f.path === path)
    if (file) setOverlayFile(file)
  }, [niftiFiles])

  if (loading) {
    return <div style={{ padding: '1rem', color: '#aaa' }}>Loading analytics files...</div>
  }

  if (error && niftiFiles.length === 0) {
    return <div style={{ padding: '1rem', color: '#e57373' }}>Error: {error}</div>
  }

  if (niftiFiles.length === 0) {
    return (
      <div style={{ padding: '1rem' }}>
        <p style={{ color: '#aaa' }}>No NIfTI files available for this study.</p>
        {files.length > 0 && (
          <p style={{ color: '#888', fontSize: '0.8rem' }}>
            {files.length} analytics file(s) found but none are NIfTI format.
          </p>
        )}
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', gap: '0.75rem', minHeight: 400 }}>
      {/* File list sidebar */}
      <div style={{ width: 220, flexShrink: 0, overflowY: 'auto', maxHeight: 500 }}>
        <div style={{ fontSize: '0.75rem', color: '#888', marginBottom: 6 }}>
          NIfTI Files ({niftiFiles.length})
        </div>
        {niftiFiles.map(f => (
          <button
            key={f.path}
            type="button"
            onClick={() => handleFileSelect(f)}
            style={{
              display: 'block',
              width: '100%',
              textAlign: 'left',
              padding: '6px 8px',
              marginBottom: 2,
              border: selectedFile?.path === f.path ? '1px solid #4db6ac' : '1px solid #333',
              borderRadius: 4,
              background: selectedFile?.path === f.path ? 'rgba(77,182,172,0.1)' : 'transparent',
              color: '#ddd',
              cursor: 'pointer',
              fontSize: '0.75rem',
            }}
          >
            <div style={{ fontWeight: selectedFile?.path === f.path ? 600 : 400, wordBreak: 'break-all' }}>
              {f.name}
            </div>
            <div style={{ color: '#888', fontSize: '0.7rem' }}>
              {f.tool} — {formatSize(f.size)}
            </div>
          </button>
        ))}
      </div>

      {/* Viewer area */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Toolbar */}
        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', marginBottom: 6, flexWrap: 'wrap' }}>
          {SLICE_MODES.map(m => (
            <button
              key={m.value}
              type="button"
              onClick={() => setSliceMode(m.value)}
              style={{
                padding: '3px 10px',
                fontSize: '0.75rem',
                border: sliceMode === m.value ? '1px solid #4db6ac' : '1px solid #555',
                borderRadius: 4,
                background: sliceMode === m.value ? 'rgba(77,182,172,0.15)' : 'transparent',
                color: sliceMode === m.value ? '#4db6ac' : '#aaa',
                cursor: 'pointer',
              }}
            >
              {m.label}
            </button>
          ))}

          {/* Overlay selector */}
          {niftiFiles.length > 1 && (
            <>
              <span style={{ color: '#888', fontSize: '0.75rem', marginLeft: 8 }}>Overlay:</span>
              <select
                value={overlayFile?.path || ''}
                onChange={handleOverlaySelect}
                style={{
                  fontSize: '0.75rem',
                  padding: '2px 6px',
                  background: '#222',
                  color: '#ddd',
                  border: '1px solid #555',
                  borderRadius: 4,
                  maxWidth: 200,
                }}
              >
                <option value="">None</option>
                {niftiFiles.filter(f => f.path !== selectedFile?.path).map(f => (
                  <option key={f.path} value={f.path}>{f.name}</option>
                ))}
              </select>
            </>
          )}

          {/* Overlay opacity */}
          {overlayFile && (
            <label style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: '0.75rem', color: '#888' }}>
              Opacity:
              <input
                type="range"
                min="0"
                max="1"
                step="0.05"
                value={opacity}
                onChange={e => setOpacity(parseFloat(e.target.value))}
                style={{ width: 60 }}
              />
              <span>{Math.round(opacity * 100)}%</span>
            </label>
          )}
        </div>

        {/* Canvas */}
        <div style={{ flex: 1, position: 'relative', minHeight: 350, background: '#1a1a1a', borderRadius: 4 }}>
          <canvas
            ref={canvasRef}
            style={{ width: '100%', height: '100%', display: 'block' }}
          />
          {error && (
            <div style={{
              position: 'absolute', bottom: 8, left: 8,
              background: 'rgba(229,115,115,0.9)', color: '#fff',
              padding: '4px 10px', borderRadius: 4, fontSize: '0.75rem',
            }}>
              {error}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
