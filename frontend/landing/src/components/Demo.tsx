import { useState, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { parseDicomFile, deidentify } from '@aegis/client'
import type { DicomTag } from '@aegis/client'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

type DemoState = 'idle' | 'parsing' | 'done' | 'error'

const ACTION_LABELS: Record<string, { label: string; color: string }> = {
  X: { label: 'Removed',  color: '#ea580c' },  // orange — colorblind-safe (not red)
  Z: { label: 'Zeroed',   color: '#b45309' },  // amber-700
  D: { label: 'Replaced', color: '#d97706' },
  U: { label: 'New UID',  color: '#7c3aed' },
  K: { label: 'Kept',     color: '#0d9488' },  // teal — colorblind-safe (not green)
  C: { label: 'Cleaned',  color: '#0284c7' },
}

// Representative sample when no real DICOM is available
const SAMPLE_TAGS: DicomTag[] = [
  { tag: '00100010', keyword: 'PatientName',           vr: 'PN', action: 'X', originalValue: 'DOE^JOHN^A',         anonymizedValue: undefined },
  { tag: '00100020', keyword: 'PatientID',             vr: 'LO', action: 'Z', originalValue: 'MRN-12345',          anonymizedValue: '' },
  { tag: '00100030', keyword: 'PatientBirthDate',      vr: 'DA', action: 'X', originalValue: '19681012',           anonymizedValue: undefined },
  { tag: '00100040', keyword: 'PatientSex',            vr: 'CS', action: 'Z', originalValue: 'M',                  anonymizedValue: '' },
  { tag: '00380010', keyword: 'AdmissionID',           vr: 'LO', action: 'X', originalValue: 'ADM-99218',          anonymizedValue: undefined },
  { tag: '00101040', keyword: 'PatientAddress',        vr: 'LO', action: 'X', originalValue: '42 Elm St, Boston',  anonymizedValue: undefined },
  { tag: '00400275', keyword: 'RequestAttributesSequence', vr: 'SQ', action: 'X', originalValue: '[Sequence]',     anonymizedValue: undefined },
  { tag: '00080020', keyword: 'StudyDate',             vr: 'DA', action: 'K', originalValue: '20231205',           anonymizedValue: '20231205' },
  { tag: '00080060', keyword: 'Modality',              vr: 'CS', action: 'K', originalValue: 'MR',                 anonymizedValue: 'MR' },
  { tag: '00081030', keyword: 'StudyDescription',      vr: 'LO', action: 'K', originalValue: 'BRAIN MRI W/O',     anonymizedValue: 'BRAIN MRI W/O' },
  { tag: '00180087', keyword: 'MagneticFieldStrength', vr: 'DS', action: 'K', originalValue: '3',                  anonymizedValue: '3' },
  { tag: '00081090', keyword: 'ManufacturerModelName', vr: 'LO', action: 'K', originalValue: 'MAGNETOM Prisma',   anonymizedValue: 'MAGNETOM Prisma' },
]

function TagRow({ tag, index }: { tag: DicomTag; index: number }) {
  const info = ACTION_LABELS[tag.action] ?? { label: tag.action, color: '#64748b' }
  const isModified = tag.action !== 'K'
  return (
    <tr style={{
      background: index % 2 === 0 ? 'rgba(248,250,252,0.03)' : 'transparent',
      borderLeft: `3px solid ${isModified ? info.color : 'transparent'}`,
    }}>
      <td style={{ padding: '7px 12px', fontFamily: 'monospace', fontSize: '0.72rem', color: '#64748b' }}>
        ({tag.tag.slice(0, 4)},{tag.tag.slice(4)})
      </td>
      <td style={{ padding: '7px 12px', fontWeight: 500, fontSize: '0.83rem', color: '#e2e8f0' }}>
        {tag.keyword}
      </td>
      <td style={{ padding: '7px 12px', fontSize: '0.83rem', color: isModified ? '#fca5a5' : '#94a3b8', fontFamily: 'monospace' }}>
        {tag.originalValue !== undefined
          ? (tag.originalValue || <em style={{ color: '#475569' }}>empty</em>)
          : <em style={{ color: '#475569' }}>—</em>}
      </td>
      <td style={{ padding: '7px 12px' }}>
        <span style={{
          display: 'inline-flex', alignItems: 'center',
          padding: '2px 9px', borderRadius: '12px',
          background: info.color + '20',
          color: info.color,
          fontWeight: 700, fontSize: '0.71rem',
        }}>
          {info.label}
        </span>
      </td>
    </tr>
  )
}

export default function Demo() {
  const navigate = useNavigate()
  const { ref, isVisible } = useScrollAnimation()
  const [state, setState] = useState<DemoState>('idle')
  const [tags, setTags] = useState<DicomTag[]>([])
  const [filename, setFilename] = useState('')
  const [privateRemoved, setPrivateRemoved] = useState(0)
  const [errorMsg, setErrorMsg] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [usingSample, setUsingSample] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const processFile = useCallback(async (file: File) => {
    setState('parsing')
    setFilename(file.name)
    setUsingSample(false)
    try {
      const buf = await file.arrayBuffer()
      // parseDicomFile is synchronous; deidentify is async (WebCrypto UID hashing)
      const { dataset } = parseDicomFile(buf, file.name)
      const result = await deidentify(dataset, {})
      setTags(result.tagChanges)
      setPrivateRemoved(result.privateTagsRemoved)
      setState('done')
    } catch (e: unknown) {
      setErrorMsg(e instanceof Error ? e.message : 'Failed to parse DICOM file')
      setState('error')
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) processFile(file)
  }, [processFile])

  const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) processFile(file)
  }, [processFile])

  const useSampleData = useCallback(() => {
    setTags(SAMPLE_TAGS)
    setFilename('example_brain_mri.dcm')
    setPrivateRemoved(8)
    setUsingSample(true)
    setState('done')
  }, [])

  const reset = useCallback(() => {
    setState('idle')
    setTags([])
    setFilename('')
    setErrorMsg('')
    setUsingSample(false)
    if (inputRef.current) inputRef.current.value = ''
  }, [])

  const modified = tags.filter(t => t.action !== 'K')
  const kept = tags.filter(t => t.action === 'K')

  return (
    <section id="demo" style={{ padding: '80px 0', background: 'var(--gradient-hero)' }} ref={ref}>
      <div style={{ maxWidth: '1100px', margin: '0 auto', padding: '0 24px' }}>

        {/* ── Header ── */}
        <div
          className={`animate--fade-up${isVisible ? ' animate--visible' : ''}`}
          style={{ textAlign: 'center', marginBottom: '48px' }}
        >
          <span style={{
            display: 'inline-block', padding: '4px 16px', borderRadius: '20px',
            background: 'rgba(59,130,246,0.15)', color: '#60a5fa',
            fontSize: '0.77rem', fontWeight: 700, letterSpacing: '0.1em',
            textTransform: 'uppercase', marginBottom: '16px',
          }}>
            Live Demo
          </span>
          <h2 style={{
            fontSize: 'clamp(1.75rem, 4vw, 2.5rem)', fontWeight: 800,
            color: '#f8fafc', marginBottom: '16px', lineHeight: 1.2,
          }}>
            See de-identification happen in real time
          </h2>
          <p style={{ fontSize: '1.05rem', color: '#94a3b8', maxWidth: '600px', margin: '0 auto' }}>
            Drop any{' '}
            <code style={{ background: 'rgba(59,130,246,0.12)', color: '#93c5fd', padding: '1px 6px', borderRadius: '4px' }}>.dcm</code>
            {' '}file below. AEGIS strips all 18 HIPAA identifiers entirely in your browser —
            no data ever transmitted to a server.
          </p>
        </div>

        {/* ── Step guide ── */}
        <div
          className={`animate--fade-up animate--delay-1${isVisible ? ' animate--visible' : ''}`}
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
            gap: '0',
            marginBottom: '28px',
            background: 'rgba(15,23,42,0.7)',
            border: '1px solid #1e293b',
            borderRadius: '12px',
            overflow: 'hidden',
          }}
        >
          {[
            { n: '1', icon: '📂', title: 'Get a DICOM file', body: 'Have a .dcm file from any modality. No DICOM? Hit "Use sample data" below.' },
            { n: '2', icon: '⬇️', title: 'Drop it in the box', body: 'Drag and drop or click to browse. The file is processed entirely in your browser.' },
            { n: '3', icon: '🔍', title: 'See what AEGIS strips', body: 'Every tag the PS3.15 profile touches appears instantly — with the original value alongside.' },
          ].map((step, i) => (
            <div
              key={step.n}
              style={{
                padding: '20px 22px',
                borderRight: i < 2 ? '1px solid #1e293b' : undefined,
                display: 'flex', gap: '14px', alignItems: 'flex-start',
              }}
            >
              <div style={{
                width: '28px', height: '28px', borderRadius: '50%',
                background: 'rgba(59,130,246,0.15)', color: '#60a5fa',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontWeight: 800, fontSize: '0.8rem', flexShrink: 0,
              }}>
                {step.n}
              </div>
              <div>
                <div style={{ fontWeight: 700, fontSize: '0.88rem', color: '#e2e8f0', marginBottom: '4px' }}>
                  {step.icon} {step.title}
                </div>
                <div style={{ fontSize: '0.78rem', color: '#64748b', lineHeight: 1.5 }}>
                  {step.body}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* ── Drop zone ── */}
        {state === 'idle' && (
          <div className={`animate--fade-up animate--delay-1${isVisible ? ' animate--visible' : ''}`}>
            <div
              onDragOver={e => { e.preventDefault(); setDragOver(true) }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
              onClick={() => inputRef.current?.click()}
              style={{
                border: `2px dashed ${dragOver ? '#3b82f6' : '#334155'}`,
                borderRadius: '16px',
                background: dragOver ? 'rgba(59,130,246,0.07)' : 'rgba(15,23,42,0.5)',
                padding: '64px 32px',
                textAlign: 'center',
                cursor: 'pointer',
                transition: 'all 0.2s ease',
              }}
            >
              <input ref={inputRef} type="file" accept=".dcm,application/dicom" style={{ display: 'none' }} onChange={handleChange} />
              <div style={{ fontSize: '2.8rem', marginBottom: '16px' }}>🩻</div>
              <p style={{ color: '#e2e8f0', fontWeight: 600, fontSize: '1.1rem', marginBottom: '8px' }}>
                Drop a .dcm file here, or click to browse
              </p>
              <p style={{ color: '#64748b', fontSize: '0.87rem', marginBottom: '28px' }}>
                100% client-side — your file never leaves your machine
              </p>
              <button
                type="button"
                onClick={e => { e.stopPropagation(); useSampleData() }}
                style={{
                  padding: '8px 22px', borderRadius: '8px',
                  background: 'transparent', border: '1px solid #334155',
                  color: '#94a3b8', fontSize: '0.84rem', cursor: 'pointer',
                }}
                onMouseOver={e => { const b = e.currentTarget; b.style.borderColor = '#3b82f6'; b.style.color = '#60a5fa' }}
                onMouseOut={e => { const b = e.currentTarget; b.style.borderColor = '#334155'; b.style.color = '#94a3b8' }}
              >
                Use sample data instead →
              </button>
            </div>
          </div>
        )}

        {/* ── Parsing ── */}
        {state === 'parsing' && (
          <div style={{ textAlign: 'center', padding: '64px 0' }}>
            <div style={{ fontSize: '2rem', marginBottom: '16px' }}>⚙️</div>
            <p style={{ color: '#e2e8f0', fontWeight: 600, marginBottom: '6px' }}>Parsing {filename}…</p>
            <p style={{ color: '#64748b', fontSize: '0.87rem' }}>Applying PS3.15 Basic Confidentiality Profile in your browser</p>
          </div>
        )}

        {/* ── Error ── */}
        {state === 'error' && (
          <div style={{ background: 'rgba(220,38,38,0.08)', border: '1px solid rgba(220,38,38,0.25)', borderRadius: '12px', padding: '32px', textAlign: 'center' }}>
            <p style={{ color: '#fca5a5', fontWeight: 600, marginBottom: '8px' }}>Could not parse DICOM file</p>
            <p style={{ color: '#f87171', fontSize: '0.83rem', marginBottom: '20px', fontFamily: 'monospace' }}>{errorMsg}</p>
            <button onClick={reset} style={{ padding: '8px 20px', borderRadius: '8px', background: '#dc2626', color: '#fff', border: 'none', cursor: 'pointer', fontWeight: 600 }}>
              Try another file
            </button>
          </div>
        )}

        {/* ── Results ── */}
        {state === 'done' && (
          <div>
            {/* Stats + controls */}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '12px', marginBottom: '18px', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
                {[
                  { val: modified.length, label: 'Tags modified',        color: '#dc2626' },
                  { val: kept.length,     label: 'Tags retained',        color: '#059669' },
                  { val: privateRemoved,  label: 'Private tags removed', color: '#7c3aed' },
                ].map(({ val, label, color }) => (
                  <div key={label} style={{ background: 'rgba(15,23,42,0.7)', border: `1px solid ${color}30`, borderRadius: '10px', padding: '10px 16px', textAlign: 'center' }}>
                    <div style={{ fontSize: '1.4rem', fontWeight: 800, color, lineHeight: 1 }}>{val}</div>
                    <div style={{ fontSize: '0.71rem', color: '#64748b', marginTop: '3px' }}>{label}</div>
                  </div>
                ))}
              </div>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                {usingSample && (
                  <span style={{ padding: '4px 12px', borderRadius: '20px', background: 'rgba(234,179,8,0.12)', color: '#fbbf24', fontSize: '0.72rem', fontWeight: 600 }}>
                    Sample data
                  </span>
                )}
                <span style={{ display: 'flex', alignItems: 'center', gap: '5px', padding: '5px 12px', borderRadius: '20px', background: 'rgba(5,150,105,0.12)', color: '#34d399', fontSize: '0.76rem', fontWeight: 600 }}>
                  🔒 Zero data transmitted
                </span>
                <button onClick={reset} style={{ padding: '5px 14px', borderRadius: '8px', background: 'transparent', border: '1px solid #1e293b', color: '#64748b', fontSize: '0.77rem', cursor: 'pointer' }}>
                  Reset
                </button>
              </div>
            </div>

            {/* Table */}
            <div style={{ background: 'rgba(15,23,42,0.85)', borderRadius: '14px', border: '1px solid #1e293b', overflow: 'hidden' }}>
              <div style={{ padding: '12px 18px', background: 'rgba(30,41,59,0.7)', borderBottom: '1px solid #1e293b', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span>📄</span>
                <span style={{ color: '#cbd5e1', fontWeight: 600, fontFamily: 'monospace', fontSize: '0.87rem' }}>{filename}</span>
                <span style={{ marginLeft: 'auto', color: '#475569', fontSize: '0.74rem' }}>DICOM PS3.15 Annex E — Basic Confidentiality Profile</span>
              </div>
              <div style={{ overflowX: 'auto', maxHeight: '460px', overflowY: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead style={{ position: 'sticky', top: 0, zIndex: 1 }}>
                    <tr style={{ background: 'rgba(15,23,42,0.98)', borderBottom: '1px solid #1e293b' }}>
                      {['Tag', 'Keyword', 'Original value', 'Action'].map(h => (
                        <th key={h} style={{ padding: '10px 12px', textAlign: 'left', fontSize: '0.69rem', fontWeight: 700, color: '#475569', textTransform: 'uppercase', letterSpacing: '0.07em' }}>{h}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {modified.map((tag, i) => <TagRow key={tag.tag} tag={tag} index={i} />)}
                    {kept.length > 0 && modified.length > 0 && (
                      <tr>
                        <td colSpan={4} style={{ padding: '5px 12px', background: 'rgba(5,150,105,0.05)', borderTop: '1px dashed #1e293b', borderBottom: '1px dashed #1e293b' }}>
                          <span style={{ fontSize: '0.69rem', color: '#059669', fontWeight: 700 }}>▼ Retained tags — safe to keep (no direct PHI risk)</span>
                        </td>
                      </tr>
                    )}
                    {kept.map((tag, i) => <TagRow key={tag.tag} tag={tag} index={modified.length + i} />)}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Action legend */}
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '14px', marginTop: '14px', justifyContent: 'center' }}>
              {Object.entries(ACTION_LABELS).map(([, { label, color }]) => (
                <span key={label} style={{ display: 'flex', alignItems: 'center', gap: '5px', fontSize: '0.75rem', color: '#64748b' }}>
                  <span style={{ width: '9px', height: '9px', borderRadius: '2px', background: color, display: 'inline-block' }} />
                  <strong style={{ color }}>{label}</strong>
                </span>
              ))}
            </div>
          </div>
        )}

        {/* ── Bottom CTA ── */}
        <div
          className={`animate--fade-up animate--delay-2${isVisible ? ' animate--visible' : ''}`}
          style={{ textAlign: 'center', marginTop: '52px' }}
        >
          <p style={{ color: '#475569', fontSize: '0.87rem', marginBottom: '20px' }}>
            The same open-source engine powers the upload portal — auditable, DICOM PS3.15 compliant, zero dependency on cloud AI.
          </p>
          <button
            onClick={() => navigate('/contact')}
            style={{
              display: 'inline-block', padding: '14px 32px',
              background: 'var(--gradient-cta)', color: '#fff',
              borderRadius: '8px', fontWeight: 700, fontSize: '1rem',
              border: 'none', cursor: 'pointer', transition: 'transform 0.15s',
            }}
            onMouseOver={e => { e.currentTarget.style.transform = 'translateY(-1px)' }}
            onMouseOut={e => { e.currentTarget.style.transform = '' }}
          >
            Request a full platform demo →
          </button>
        </div>
      </div>
    </section>
  )
}
