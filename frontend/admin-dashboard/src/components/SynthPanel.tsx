import { useState, useEffect } from 'react'

// ── Types ─────────────────────────────────────────────────────────────────────

type Project = {
  id: string
  name: string
  slug: string
  archived?: boolean
}

type SynthResult = {
  study_uid: string
  study_ids: string[]
  file_count: number
  tool_used: string
  duration_seconds: number
  message: string
}

// ── Component ─────────────────────────────────────────────────────────────────

export function SynthPanel({
  isAdmin,
  onStudyGenerated,
}: {
  isAdmin: boolean
  onStudyGenerated?: () => void
}) {
  const [projects, setProjects] = useState<Project[]>([])
  const [projectSlug, setProjectSlug] = useState('')

  const [slices, setSlices] = useState(20)
  const [size, setSize] = useState(256)
  const [seed, setSeed] = useState(() => Math.floor(Math.random() * 100000))
  const [withFace, setWithFace] = useState(false)

  const [generating, setGenerating] = useState(false)
  const [result, setResult] = useState<SynthResult | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Load project list once.
  useEffect(() => {
    fetch('/api/projects')
      .then(r => r.json())
      .then((data: Project[]) => {
        const active = (data ?? []).filter(p => !p.archived)
        setProjects(active)
        if (active.length > 0) setProjectSlug(active[0].slug)
      })
      .catch(() => { /* non-fatal */ })
  }, [])

  function randomiseSeed() {
    setSeed(Math.floor(Math.random() * 100000))
  }

  async function generate() {
    setGenerating(true)
    setResult(null)
    setError(null)
    try {
      const res = await fetch('/api/studies/generate-synthetic', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          project_slug: projectSlug || 'default',
          slices,
          size,
          seed,
          with_face: withFace,
        }),
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error ?? `HTTP ${res.status}`)
      } else {
        setResult(data as SynthResult)
        onStudyGenerated?.()
      }
    } catch {
      setError('Network error — please try again')
    } finally {
      setGenerating(false)
    }
  }

  return (
    <div className="tcia-panel">

      {/* Header */}
      <div className="tcia-header">
        <h2>Synthetic MRI Generator</h2>
        <p className="tcia-subtitle">
          Generate a synthetic brain MRI phantom study for testing the pipeline.
          The generated DICOM files are imported directly into AEGIS and run through
          the normal routing and processing workflow.
          No real patient data is used.
        </p>
      </div>

      {/* Controls */}
      <div className="tcia-search-bar">

        {/* Slices slider */}
        <div className="tcia-field">
          <label className="tcia-label">Slices: {slices}</label>
          <input
            type="range"
            min={10}
            max={200}
            step={10}
            value={slices}
            onChange={e => setSlices(Number(e.target.value))}
            disabled={generating}
            style={{ width: 140 }}
          />
        </div>

        {/* Size dropdown */}
        <div className="tcia-field tcia-field--narrow">
          <label className="tcia-label">Size</label>
          <select
            className="form-select"
            value={size}
            onChange={e => setSize(Number(e.target.value))}
            disabled={generating}
          >
            {[64, 128, 256, 512].map(s => (
              <option key={s} value={s}>{s} × {s}</option>
            ))}
          </select>
        </div>

        {/* Seed input */}
        <div className="tcia-field tcia-field--narrow">
          <label className="tcia-label">Seed</label>
          <div style={{ display: 'flex', gap: 4 }}>
            <input
              type="number"
              className="form-input"
              value={seed}
              min={0}
              max={999999}
              onChange={e => setSeed(Math.max(0, Number(e.target.value)))}
              disabled={generating}
              style={{ width: 90 }}
            />
            <button
              type="button"
              className="btn btn--secondary"
              onClick={randomiseSeed}
              disabled={generating}
              title="Pick a random seed"
            >
              ↺
            </button>
          </div>
        </div>

        {/* Project selector */}
        {projects.length > 0 && (
          <div className="tcia-field">
            <label className="tcia-label">Project</label>
            <select
              className="form-select"
              value={projectSlug}
              onChange={e => setProjectSlug(e.target.value)}
              disabled={generating}
            >
              {projects.map(p => (
                <option key={p.id} value={p.slug}>{p.name}</option>
              ))}
            </select>
          </div>
        )}

        {/* With face toggle */}
        <div className="tcia-field" style={{ justifyContent: 'flex-end' }}>
          <label className="tcia-label" style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
            <input
              type="checkbox"
              checked={withFace}
              onChange={e => setWithFace(e.target.checked)}
              disabled={generating}
            />
            Include facial features
          </label>
          <span style={{ fontSize: 11, color: '#64748b' }}>
            Enables defacing demo
          </span>
        </div>

        {/* Generate button */}
        {isAdmin && (
          <div className="tcia-field tcia-field--action">
            <button
              type="button"
              className="btn-primary"
              onClick={generate}
              disabled={generating}
            >
              {generating ? 'Generating…' : 'Generate Study'}
            </button>
          </div>
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="form-error tcia-search-error">{error}</div>
      )}

      {/* Result */}
      {result && (
        <div className="tcia-progress" style={{ marginTop: 16 }}>
          <h3 style={{ marginBottom: 8 }}>Generated study</h3>
          <table className="table tcia-table">
            <tbody>
              <tr>
                <th style={{ width: 160 }}>Study UID</th>
                <td className="tcia-uid" title={result.study_uid}>…{result.study_uid.slice(-30)}</td>
              </tr>
              <tr>
                <th>Files</th>
                <td>{result.file_count} DICOM files</td>
              </tr>
              <tr>
                <th>Tool</th>
                <td><span className="badge">{result.tool_used}</span></td>
              </tr>
              <tr>
                <th>Duration</th>
                <td>{result.duration_seconds.toFixed(1)}s</td>
              </tr>
              <tr>
                <th>Status</th>
                <td>
                  <span className="badge badge--approved">Imported — visible in Studies tab</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      )}

      {/* Notes */}
      <p className="tcia-attribution" style={{ marginTop: 16 }}>
        Synthetic studies are generated by the <strong>synth-service</strong> sidecar using
        nibabel/NumPy. They are indistinguishable from real DICOM studies in the pipeline.
        Set <code>SYNTH_SERVICE_URL</code> to enable this feature.
      </p>
    </div>
  )
}
