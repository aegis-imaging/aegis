import { useState, useRef, useCallback, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import dicomParser from 'dicom-parser'
import { deidentify, parseDicomFile } from '@aegis/client'
import type { DicomTag } from '@aegis/client'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

// ---------------------------------------------------------------------------
// Env vars (baked in at build time by Vite)
// ---------------------------------------------------------------------------

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Stage = 'idle' | 'generating' | 'polling' | 'loading' | 'done'

interface RenderedSlice {
  filename: string
  originalDataURL: string   // canvas toDataURL — raw store (before defacing)
  defacedDataURL: string    // canvas toDataURL — clean store (after defacing)
  rows: number
  cols: number
}

// ---------------------------------------------------------------------------
// DICOM pixel rendering helpers
// ---------------------------------------------------------------------------

/** Parse raw DICOM bytes and render to an offscreen canvas. Returns data URL. */
function renderDicomToCanvas(
  bytes: Uint8Array,
  applyDefaceMask: boolean
): { dataURL: string; rows: number; cols: number } {
  const dataSet = dicomParser.parseDicom(bytes)
  const rows = dataSet.uint16('x00280010') ?? 256
  const cols = dataSet.uint16('x00280011') ?? 256
  const bitsAlloc = dataSet.uint16('x00280100') ?? 16

  // Window: use stored values or sensible defaults for Shepp–Logan phantom
  const wcStr = dataSet.floatString('x00281050')
  const wwStr = dataSet.floatString('x00281051')
  const wc = parseFloat(typeof wcStr === 'string' ? wcStr : '2048')
  const ww = parseFloat(typeof wwStr === 'string' ? wwStr : '4096')
  const winMin = wc - ww / 2
  const winMax = wc + ww / 2

  // Extract raw pixel bytes
  const pixElem = dataSet.elements['x7fe00010']
  const rawBytes = dataSet.byteArray.slice(
    pixElem.dataOffset,
    pixElem.dataOffset + pixElem.length
  )

  // Interpret as 16-bit unsigned values
  const pixels =
    bitsAlloc === 16
      ? new Uint16Array(rawBytes.buffer, rawBytes.byteOffset, rawBytes.byteLength / 2)
      : new Uint8Array(rawBytes.buffer, rawBytes.byteOffset, rawBytes.byteLength)

  // Render to canvas
  const canvas = document.createElement('canvas')
  canvas.width = cols
  canvas.height = rows
  const ctx = canvas.getContext('2d')!
  const imgData = ctx.createImageData(cols, rows)
  const d = imgData.data

  for (let i = 0; i < rows * cols; i++) {
    const raw = pixels[i] ?? 0
    // Windowing: map [winMin, winMax] to [0, 255]
    let gray = ((raw - winMin) / (winMax - winMin)) * 255
    gray = Math.max(0, Math.min(255, gray))

    // Defacing simulation: zero the anterior 28% of the image (top rows in axial view)
    if (applyDefaceMask) {
      const row = Math.floor(i / cols)
      const anteriorThreshold = Math.floor(rows * 0.28)
      if (row < anteriorThreshold) {
        gray = 0 // black = removed tissue
      }
    }

    const gInt = Math.round(gray)
    d[i * 4]     = gInt // R
    d[i * 4 + 1] = gInt // G
    d[i * 4 + 2] = gInt // B
    d[i * 4 + 3] = 255  // A
  }
  ctx.putImageData(imgData, 0, 0)

  return { dataURL: canvas.toDataURL('image/png'), rows, cols }
}

// ---------------------------------------------------------------------------
// Tag diff helpers
// ---------------------------------------------------------------------------

const ACTION_LABELS: Record<string, { label: string; color: string }> = {
  X: { label: 'Removed',  color: '#ea580c' },  // orange — colorblind-safe (not red)
  Z: { label: 'Zeroed',   color: '#b45309' },  // amber-700
  D: { label: 'Replaced', color: '#d97706' },
  U: { label: 'New UID',  color: '#7c3aed' },
  K: { label: 'Kept',     color: '#0d9488' },  // teal — colorblind-safe (not green)
  C: { label: 'Cleaned',  color: '#0284c7' },
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function SliceGrid({ slices, label, maskBadge }: {
  slices: RenderedSlice[]
  label: string
  maskBadge?: boolean
}) {
  return (
    <div style={{ flex: 1, minWidth: 0 }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: '8px',
        marginBottom: '10px',
      }}>
        <span style={{
          fontSize: '0.75rem', fontWeight: 700, letterSpacing: '0.07em',
          textTransform: 'uppercase',
          color: maskBadge ? '#34d399' : '#94a3b8',
        }}>
          {label}
        </span>
        {maskBadge && (
          <span style={{
            fontSize: '0.65rem', fontWeight: 700,
            background: 'rgba(52,211,153,0.12)', color: '#34d399',
            padding: '1px 7px', borderRadius: '10px',
          }}>
            Anonymized + Defaced
          </span>
        )}
      </div>
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(4, 1fr)',
        gap: '6px',
      }}>
        {slices.map((s, i) => (
          <div key={i} style={{
            position: 'relative',
            borderRadius: '6px',
            overflow: 'hidden',
            border: '1px solid #1e293b',
            aspectRatio: '1',
            background: '#000',
          }}>
            <img
              src={maskBadge ? s.defacedDataURL : s.originalDataURL}
              alt={`Slice ${i + 1}`}
              style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
            />
            <div style={{
              position: 'absolute', bottom: '3px', right: '4px',
              fontSize: '0.6rem', color: 'rgba(255,255,255,0.4)',
              fontFamily: 'monospace',
            }}>
              {i + 1}/{slices.length}
            </div>
          </div>
        ))}
      </div>
      <p style={{
        fontSize: '0.69rem', color: '#475569', marginTop: '6px', textAlign: 'center',
        fontFamily: 'monospace',
      }}>
        {slices.length > 0
          ? `${slices[0].rows}×${slices[0].cols} px · 16-bit grayscale · MR`
          : '—'}
      </p>
    </div>
  )
}

function TagDiffTable({ tags }: { tags: DicomTag[] }) {
  const modified = tags.filter(t => t.action !== 'K' && t.originalValue !== undefined)
  const kept = tags.filter(t => t.action === 'K' && t.originalValue !== undefined)
  return (
    <div style={{
      background: 'rgba(15,23,42,0.9)',
      borderRadius: '10px',
      border: '1px solid #1e293b',
      overflow: 'hidden',
      marginTop: '20px',
    }}>
      <div style={{
        padding: '10px 14px',
        background: 'rgba(30,41,59,0.7)',
        borderBottom: '1px solid #1e293b',
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
      }}>
        <span style={{ color: '#cbd5e1', fontWeight: 600, fontSize: '0.82rem' }}>
          DICOM Tag De-identification
        </span>
        <span style={{ fontSize: '0.7rem', color: '#475569' }}>
          PS3.15 Annex E Basic Profile
        </span>
      </div>
      <div style={{ maxHeight: '240px', overflowY: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead style={{ position: 'sticky', top: 0, zIndex: 1 }}>
            <tr style={{ background: 'rgba(15,23,42,0.98)', borderBottom: '1px solid #1e293b' }}>
              {['Keyword', 'Original', 'Action'].map(h => (
                <th key={h} style={{
                  padding: '7px 10px', textAlign: 'left',
                  fontSize: '0.65rem', fontWeight: 700,
                  color: '#475569', textTransform: 'uppercase', letterSpacing: '0.06em',
                }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {modified.slice(0, 12).map((tag, i) => {
              const info = ACTION_LABELS[tag.action] ?? { label: tag.action, color: '#64748b' }
              return (
                <tr key={tag.tag} style={{
                  background: i % 2 === 0 ? 'rgba(248,250,252,0.02)' : 'transparent',
                  borderLeft: `2px solid ${info.color}`,
                }}>
                  <td style={{ padding: '5px 10px', fontSize: '0.78rem', color: '#e2e8f0', fontWeight: 500 }}>
                    {tag.keyword}
                  </td>
                  <td style={{ padding: '5px 10px', fontSize: '0.75rem', color: '#fca5a5', fontFamily: 'monospace' }}>
                    {tag.originalValue || <em style={{ color: '#475569' }}>empty</em>}
                  </td>
                  <td style={{ padding: '5px 10px' }}>
                    <span style={{
                      padding: '1px 7px', borderRadius: '10px',
                      background: info.color + '20', color: info.color,
                      fontWeight: 700, fontSize: '0.67rem',
                    }}>
                      {info.label}
                    </span>
                  </td>
                </tr>
              )
            })}
            {kept.slice(0, 4).map((tag, i) => (
              <tr key={tag.tag} style={{
                background: (modified.length + i) % 2 === 0 ? 'rgba(248,250,252,0.02)' : 'transparent',
                borderLeft: '2px solid #059669',
              }}>
                <td style={{ padding: '5px 10px', fontSize: '0.78rem', color: '#e2e8f0', fontWeight: 500 }}>
                  {tag.keyword}
                </td>
                <td style={{ padding: '5px 10px', fontSize: '0.75rem', color: '#94a3b8', fontFamily: 'monospace' }}>
                  {tag.originalValue || '—'}
                </td>
                <td style={{ padding: '5px 10px' }}>
                  <span style={{
                    padding: '1px 7px', borderRadius: '10px',
                    background: '#05966920', color: '#059669',
                    fontWeight: 700, fontSize: '0.67rem',
                  }}>
                    Kept
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

// Slice indices (0-based) to fetch from DICOMweb for the canvas preview.
const CANVAS_SLICE_INDICES = [3, 7, 11, 15]
const POLL_TIMEOUT_MS = 5 * 60 * 1000 // 5 minutes

export default function PipelineDemo() {
  const navigate = useNavigate()
  const { ref, isVisible } = useScrollAnimation()
  const [stage, setStage] = useState<Stage>('idle')
  const [studyID, setStudyID] = useState('')
  const [studyUID, setStudyUID] = useState('')
  const [slices, setSlices] = useState<RenderedSlice[]>([])
  const [tagChanges, setTagChanges] = useState<DicomTag[]>([])
  const [errorMsg, setErrorMsg] = useState('')
  const [pollSeconds, setPollSeconds] = useState(0)
  const abortRef = useRef(false)

  useEffect(() => { return () => { abortRef.current = true } }, [])

  // ---------------------------------------------------------------------------
  // Fetch DICOM slices from DICOMweb for canvas + tag diff, then set done state
  // ---------------------------------------------------------------------------

  const loadDicomData = useCallback(async (uid: string) => {
    setStage('loading')
    try {
      const seriesUID = `${uid}.1`
      const loaded: RenderedSlice[] = []

      for (const idx of CANVAS_SLICE_INDICES) {
        if (abortRef.current) return
        const sopUID = `${uid}.1.${idx}`

        // Fetch the same slice from both stores in parallel
        const [rawRes, cleanRes] = await Promise.all([
          fetch(`${API_BASE}/dicomweb-raw/studies/${uid}/series/${seriesUID}/instances/${sopUID}`),
          fetch(`${API_BASE}/dicomweb/studies/${uid}/series/${seriesUID}/instances/${sopUID}`),
        ])
        if (!rawRes.ok || !cleanRes.ok) continue
        const [rawBuf, cleanBuf] = await Promise.all([rawRes.arrayBuffer(), cleanRes.arrayBuffer()])

        const { dataURL: origDataURL, rows, cols } = renderDicomToCanvas(new Uint8Array(rawBuf), false)
        const { dataURL: defacedDataURL } = renderDicomToCanvas(new Uint8Array(cleanBuf), false)

        loaded.push({ filename: `slice_${idx}.dcm`, originalDataURL: origDataURL, defacedDataURL, rows, cols })
      }

      // Client-side tag diff from first raw slice (shows which HIPAA tags would be removed)
      if (!abortRef.current) {
        const sopUID0 = `${uid}.1.0`
        const firstRes = await fetch(`${API_BASE}/dicomweb-raw/studies/${uid}/series/${uid}.1/instances/${sopUID0}`)
        if (firstRes.ok) {
          const firstBuf = await firstRes.arrayBuffer()
          const { dataset } = parseDicomFile(firstBuf, 'slice_0.dcm')
          const { tagChanges: tc } = await deidentify(dataset, {})
          setTagChanges(tc)
        }
      }

      setSlices(loaded)
      setStage('done')
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : 'Failed to load DICOM data')
      setStage('idle')
    }
  }, [])

  // ---------------------------------------------------------------------------
  // Start the demo — call the real API
  // ---------------------------------------------------------------------------

  const handleGenerate = useCallback(async () => {
    setStage('generating')
    setErrorMsg('')
    setPollSeconds(0)
    abortRef.current = false

    try {
      const res = await fetch(`${API_BASE}/api/demo/generate`, { method: 'POST' })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error ?? `Server error ${res.status}`)
      }
      const data = await res.json()
      setStudyID(data.study_id)
      setStudyUID(data.study_uid)
      setStage('polling')
    } catch (e) {
      setErrorMsg(e instanceof Error ? e.message : 'Failed to start demo')
      setStage('idle')
    }
  }, [])

  // ---------------------------------------------------------------------------
  // Poll study status until defacing is complete
  // ---------------------------------------------------------------------------

  useEffect(() => {
    if (stage !== 'polling' || !studyID) return
    const startTime = Date.now()

    const timer = setInterval(async () => {
      if (abortRef.current) { clearInterval(timer); return }

      const elapsed = Date.now() - startTime
      setPollSeconds(Math.floor(elapsed / 1000))

      if (elapsed > POLL_TIMEOUT_MS) {
        clearInterval(timer)
        setErrorMsg('Demo timed out — defacing took too long. Please try again.')
        setStage('idle')
        return
      }

      try {
        const res = await fetch(`${API_BASE}/api/demo/study/${studyID}`)
        if (!res.ok) return
        const data = await res.json()

        // Defacing complete when status is 'defaced' or dicom_store switched to 'clean'
        if (data.status === 'defaced' || data.dicom_store === 'clean') {
          clearInterval(timer)
          await loadDicomData(data.study_uid || studyUID)
        } else if (data.status === 'rejected') {
          clearInterval(timer)
          setErrorMsg('Pipeline step failed — please try again.')
          setStage('idle')
        }
      } catch { /* ignore transient network errors during polling */ }
    }, 3000)

    return () => clearInterval(timer)
  }, [stage, studyID, studyUID, loadDicomData])

  const handleReset = useCallback(() => {
    abortRef.current = true
    setTimeout(() => { abortRef.current = false }, 0)
    setStage('idle')
    setStudyID('')
    setStudyUID('')
    setSlices([])
    setTagChanges([])
    setErrorMsg('')
    setPollSeconds(0)
  }, [])

  // ---------------------------------------------------------------------------
  // Step indicator config
  // ---------------------------------------------------------------------------

  const steps = [
    {
      n: '1', label: 'Generate', desc: 'Synthetic brain MRI via synth-service',
      done: stage !== 'idle' && stage !== 'generating',
      active: stage === 'idle' || stage === 'generating',
    },
    {
      n: '2', label: 'Deface', desc: 'Server-side facial feature removal',
      done: stage === 'loading' || stage === 'done',
      active: stage === 'polling',
    },
    {
      n: '3', label: 'Verify', desc: 'DICOM tags + before/after review',
      done: stage === 'done',
      active: stage === 'loading' || stage === 'done',
    },
  ]

  return (
    <section
      id="pipeline-demo"
      ref={ref}
      style={{
        padding: '80px 0 60px',
        background: 'linear-gradient(180deg, rgba(15,23,42,0) 0%, rgba(15,23,42,0.4) 100%)',
        borderTop: '1px solid #1e293b',
      }}
    >
      <div style={{ maxWidth: '1100px', margin: '0 auto', padding: '0 24px' }}>

        {/* ── Header ── */}
        <div
          className={`animate--fade-up${isVisible ? ' animate--visible' : ''}`}
          style={{ textAlign: 'center', marginBottom: '48px' }}
        >
          <span style={{
            display: 'inline-block', padding: '4px 16px', borderRadius: '20px',
            background: 'rgba(52,211,153,0.12)', color: '#34d399',
            fontSize: '0.77rem', fontWeight: 700, letterSpacing: '0.1em',
            textTransform: 'uppercase', marginBottom: '16px',
          }}>
            Live Pipeline Demo
          </span>
          <h2 style={{
            fontSize: 'clamp(1.75rem, 4vw, 2.5rem)', fontWeight: 800,
            color: '#f8fafc', marginBottom: '16px', lineHeight: 1.2,
          }}>
            Generate a brain MRI — watch the full pipeline run
          </h2>
          <p style={{ fontSize: '1.05rem', color: '#94a3b8', maxWidth: '640px', margin: '0 auto' }}>
            Click to generate a real synthetic brain MRI on the AEGIS server.
            The platform anonymizes every DICOM tag and runs server-side defacing.
            Compare before and after in the DWV DICOM viewer.
          </p>
        </div>

        {/* ── Pipeline steps indicator ── */}
        <div
          className={`animate--fade-up animate--delay-1${isVisible ? ' animate--visible' : ''}`}
          style={{
            display: 'flex', gap: '0', marginBottom: '36px',
            background: 'rgba(15,23,42,0.7)',
            border: '1px solid #1e293b',
            borderRadius: '12px', overflow: 'hidden',
          }}
        >
          {steps.map((step, i) => (
            <div key={step.n} style={{
              flex: 1, padding: '16px 18px',
              borderRight: i < steps.length - 1 ? '1px solid #1e293b' : undefined,
              background: step.active
                ? 'rgba(59,130,246,0.07)'
                : step.done ? 'rgba(52,211,153,0.05)' : 'transparent',
            }}>
              <div style={{ display: 'flex', gap: '10px', alignItems: 'flex-start' }}>
                <div style={{
                  width: '24px', height: '24px', borderRadius: '50%', flexShrink: 0,
                  background: step.done
                    ? 'rgba(52,211,153,0.2)'
                    : step.active ? 'rgba(59,130,246,0.2)' : 'rgba(100,116,139,0.15)',
                  color: step.done ? '#34d399' : step.active ? '#60a5fa' : '#475569',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontWeight: 800, fontSize: '0.75rem',
                }}>
                  {step.done ? '✓' : step.n}
                </div>
                <div>
                  <div style={{
                    fontWeight: 700, fontSize: '0.82rem',
                    color: step.done ? '#34d399' : step.active ? '#e2e8f0' : '#64748b',
                    marginBottom: '2px',
                  }}>
                    {step.label}
                  </div>
                  <div style={{ fontSize: '0.71rem', color: '#475569', lineHeight: 1.4 }}>
                    {step.desc}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* ── Error ── */}
        {errorMsg && (
          <div style={{
            background: 'rgba(220,38,38,0.08)', border: '1px solid rgba(220,38,38,0.25)',
            borderRadius: '10px', padding: '14px 18px', marginBottom: '18px',
            fontSize: '0.83rem', color: '#fca5a5',
          }}>
            {errorMsg}
          </div>
        )}

        {/* ── Step 1: idle / generating ── */}
        {(stage === 'idle' || stage === 'generating') && (
          <div
            className={`animate--fade-up animate--delay-1${isVisible ? ' animate--visible' : ''}`}
            style={{ textAlign: 'center' }}
          >
            <div style={{
              background: 'rgba(15,23,42,0.8)',
              border: '2px dashed #1e293b',
              borderRadius: '16px', padding: '52px 32px',
            }}>
              <div style={{ fontSize: '3rem', marginBottom: '16px' }}>🧠</div>
              <p style={{ color: '#e2e8f0', fontWeight: 600, fontSize: '1.1rem', marginBottom: '8px' }}>
                Ready to generate a real synthetic brain MRI
              </p>
              <p style={{ color: '#64748b', fontSize: '0.87rem', marginBottom: '32px', maxWidth: '460px', margin: '0 auto 28px' }}>
                Clicks the AEGIS synth-service to produce a fresh Shepp–Logan phantom with
                facial anatomy, then runs the full server-side anonymization and defacing pipeline.
              </p>
              <button
                onClick={handleGenerate}
                disabled={stage === 'generating'}
                style={{
                  padding: '13px 36px', borderRadius: '8px',
                  background: stage === 'generating'
                    ? 'rgba(59,130,246,0.4)'
                    : 'var(--gradient-cta)',
                  color: '#fff', border: 'none',
                  fontSize: '0.97rem', fontWeight: 700, cursor: stage === 'generating' ? 'default' : 'pointer',
                  transition: 'transform 0.15s, opacity 0.15s',
                }}
                onMouseOver={e => { if (stage !== 'generating') e.currentTarget.style.transform = 'translateY(-1px)' }}
                onMouseOut={e => { e.currentTarget.style.transform = '' }}
              >
                {stage === 'generating'
                  ? '⚙️  Generating synthetic MRI…'
                  : '🧠  Generate Synthetic Brain MRI'}
              </button>
              {stage === 'generating' && (
                <p style={{ color: '#475569', fontSize: '0.78rem', marginTop: '12px' }}>
                  Calling synth-service · Importing study · Triggering pipeline
                </p>
              )}
            </div>
          </div>
        )}

        {/* ── Step 2: polling — waiting for defacing ── */}
        {stage === 'polling' && (
          <div style={{ textAlign: 'center', padding: '48px 0' }}>
            <div style={{
              display: 'inline-flex', flexDirection: 'column', alignItems: 'center',
              background: 'rgba(15,23,42,0.8)', border: '1px solid #1e293b',
              borderRadius: '16px', padding: '40px 52px', gap: '16px',
            }}>
              {/* Spinner */}
              <div style={{
                width: '48px', height: '48px', borderRadius: '50%',
                border: '3px solid rgba(59,130,246,0.2)',
                borderTop: '3px solid #3b82f6',
                animation: 'spin 1s linear infinite',
              }} />
              <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>

              <p style={{ color: '#e2e8f0', fontWeight: 600, fontSize: '1rem', margin: 0 }}>
                Defacing in progress…
              </p>
              <p style={{ color: '#64748b', fontSize: '0.85rem', margin: 0 }}>
                Server-side mri_deface removing facial anatomy · {pollSeconds}s elapsed
              </p>
              <div style={{
                display: 'flex', gap: '8px', flexWrap: 'wrap', justifyContent: 'center',
                marginTop: '4px',
              }}>
                {['DICOM imported', 'Routing evaluated', 'Defacing queued'].map(label => (
                  <span key={label} style={{
                    display: 'inline-flex', alignItems: 'center', gap: '5px',
                    padding: '3px 10px', borderRadius: '12px',
                    background: 'rgba(52,211,153,0.10)', color: '#34d399',
                    fontSize: '0.72rem', fontWeight: 600,
                  }}>
                    ✓ {label}
                  </span>
                ))}
              </div>
              <p style={{ color: '#334155', fontSize: '0.74rem', margin: 0 }}>
                Typically completes in 30–90 seconds · Study UID: {studyUID.slice(0, 24)}…
              </p>
            </div>
          </div>
        )}

        {/* ── Step 2.5: loading DICOM data ── */}
        {stage === 'loading' && (
          <div style={{ textAlign: 'center', padding: '48px 0' }}>
            <div style={{ fontSize: '2rem', marginBottom: '16px' }}>📂</div>
            <p style={{ color: '#e2e8f0', fontWeight: 600 }}>Loading DICOM slices…</p>
            <p style={{ color: '#64748b', fontSize: '0.85rem' }}>
              Fetching before/after images from DICOMweb
            </p>
          </div>
        )}

        {/* ── Step 3: done — results ── */}
        {stage === 'done' && (
          <div>
            {/* ── Before / After canvas comparison ── */}
            <div style={{ display: 'flex', gap: '20px', marginBottom: '24px', flexWrap: 'wrap' }}>
              <SliceGrid slices={slices} label="Before — original with facial anatomy" />
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: '1.4rem', color: '#475569', flexShrink: 0,
                padding: '0 4px',
              }}>→</div>
              <SliceGrid slices={slices} label="After — server-side defaced" maskBadge />
            </div>

            {/* Summary badges */}
            <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', marginBottom: '8px', justifyContent: 'center' }}>
              <span style={{
                display: 'inline-flex', alignItems: 'center', gap: '5px',
                padding: '5px 14px', borderRadius: '20px',
                background: 'rgba(5,150,105,0.12)', color: '#34d399',
                fontSize: '0.76rem', fontWeight: 600,
              }}>
                ✓ {tagChanges.filter(t => t.action !== 'K').length} identifiers de-identified
              </span>
              <span style={{
                display: 'inline-flex', alignItems: 'center', gap: '5px',
                padding: '5px 14px', borderRadius: '20px',
                background: 'rgba(52,211,153,0.10)', color: '#34d399',
                fontSize: '0.76rem', fontWeight: 600,
              }}>
                ✓ Server-side defacing complete
              </span>
              <span style={{
                display: 'inline-flex', alignItems: 'center', gap: '5px',
                padding: '5px 14px', borderRadius: '20px',
                background: 'rgba(59,130,246,0.10)', color: '#60a5fa',
                fontSize: '0.76rem', fontWeight: 600,
              }}>
                🩺 DICOMweb-accessible
              </span>
            </div>

            {/* DICOM tag diff */}
            {tagChanges.length > 0 && <TagDiffTable tags={tagChanges} />}


            {/* CTA */}
            <div style={{
              marginTop: '24px', padding: '20px 24px',
              background: 'rgba(59,130,246,0.06)',
              border: '1px solid rgba(59,130,246,0.2)',
              borderRadius: '12px',
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              gap: '16px', flexWrap: 'wrap',
            }}>
              <div>
                <div style={{ fontWeight: 700, color: '#e2e8f0', marginBottom: '4px', fontSize: '0.92rem' }}>
                  This is the real AEGIS platform — not a simulation
                </div>
                <div style={{ color: '#64748b', fontSize: '0.82rem' }}>
                  Your institution's studies would flow through the same pipeline, with per-project
                  routing rules, PHI detection, protocol compliance, and BIDS export.
                </div>
              </div>
              <button
                onClick={() => navigate('/contact')}
                style={{
                  display: 'inline-block', padding: '10px 22px',
                  background: 'var(--gradient-cta)', color: '#fff',
                  borderRadius: '8px', fontWeight: 700, fontSize: '0.87rem',
                  border: 'none', cursor: 'pointer', whiteSpace: 'nowrap',
                  transition: 'transform 0.15s',
                }}
                onMouseOver={e => { e.currentTarget.style.transform = 'translateY(-1px)' }}
                onMouseOut={e => { e.currentTarget.style.transform = '' }}
              >
                Schedule Demo →
              </button>
            </div>

            {/* Reset */}
            <div style={{ textAlign: 'center', marginTop: '20px' }}>
              <button
                onClick={handleReset}
                style={{
                  padding: '7px 18px', borderRadius: '8px',
                  background: 'transparent', border: '1px solid #1e293b',
                  color: '#64748b', fontSize: '0.8rem', cursor: 'pointer',
                }}
              >
                Generate another study
              </button>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}
