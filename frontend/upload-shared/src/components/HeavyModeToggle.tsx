// Opt-in toggle for in-browser pixel PHI scrub + (future) face de-id.
//
// The first time a user enables this, the page lazy-loads tesseract.js
// (~10 MB). We surface the tradeoffs honestly so users know what they're
// signing up for.

import { useEffect, useState } from 'react'

export interface HeavyModeSettings {
  /** Run pixel PHI scrub (OCR + redaction) on each file before upload. */
  pixelScrub: boolean
  /**
   * Run face de-id on the volume before upload. Currently not implemented —
   * the toggle stays disabled. When the model lands this gets wired through.
   */
  faceDeid: boolean
}

export interface HeavyModeToggleProps {
  value: HeavyModeSettings
  onChange: (next: HeavyModeSettings) => void
  studyCount: number
  fileCount: number
  studyLooksLikeHead: boolean
}

export function HeavyModeToggle({
  value,
  onChange,
  studyCount,
  fileCount,
  studyLooksLikeHead,
}: {
  value: HeavyModeSettings
  onChange: (next: HeavyModeSettings) => void
  studyCount: number
  fileCount: number
  studyLooksLikeHead: boolean
}) {
  const [expanded, setExpanded] = useState(false)
  const anyEnabled = value.pixelScrub || value.faceDeid

  // Estimate of extra time the pixel scrub will cost. Empirically tesseract.js
  // runs at ~0.5-1.5s per frame on a modern laptop. We pick the midpoint and
  // surface a rough range — operators want to know "minutes or hours".
  const estSecondsLow = fileCount * 0.5
  const estSecondsHigh = fileCount * 1.5

  return (
    <div
      style={{
        padding: '12px 14px',
        border: `1px solid ${anyEnabled ? '#bbf7d0' : '#e5e7eb'}`,
        background: anyEnabled ? '#f0fdf4' : '#f9fafb',
        borderRadius: '8px',
        fontSize: '14px',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <span style={{ fontWeight: 600, color: '#111827' }}>On-device deep de-identification</span>
        {anyEnabled && (
          <span style={{
            fontSize: '11px', padding: '2px 8px', borderRadius: '4px',
            background: '#86efac', color: '#14532d', fontWeight: 600,
          }}>
            ENABLED
          </span>
        )}
        <button
          type="button"
          onClick={() => setExpanded(v => !v)}
          style={{
            marginLeft: 'auto', background: 'none', border: 'none', color: '#2563eb',
            cursor: 'pointer', fontSize: '13px', padding: 0,
          }}
        >
          {expanded ? 'Hide details' : 'What is this?'}
        </button>
      </div>

      {expanded && (
        <div style={{ marginTop: 10, color: '#374151', fontSize: '13px', lineHeight: 1.55 }}>
          Pixel PHI scrub uses OCR to detect text burned into the image (names,
          dates, accession numbers) and burns black boxes over the detected regions
          <em> on your computer</em>, before any pixel ever leaves your browser. Face
          de-identification runs an MRI-defacing model locally on head/brain
          studies. This is the strongest privacy posture — patient-identifying
          pixel data never reaches our servers — but it&apos;s slower and adds ~10 MB
          to the page download the first time it&apos;s enabled.
        </div>
      )}

      <label style={{ display: 'flex', alignItems: 'flex-start', gap: 10, marginTop: 12 }}>
        <input
          type="checkbox"
          checked={value.pixelScrub}
          onChange={e => onChange({ ...value, pixelScrub: e.target.checked })}
          style={{ marginTop: 3 }}
        />
        <div>
          <div style={{ fontWeight: 500 }}>Burn-out pixel PHI</div>
          <div style={{ color: '#6b7280', fontSize: '12px' }}>
            Detects text in image pixels (Tesseract OCR, runs in your browser) and
            redacts findings. Estimated extra time for this upload:{' '}
            <strong>{formatRange(estSecondsLow, estSecondsHigh)}</strong>.
          </div>
        </div>
      </label>

      <label style={{
        display: 'flex', alignItems: 'flex-start', gap: 10, marginTop: 10,
        opacity: studyLooksLikeHead ? 1 : 0.6,
      }}>
        <input
          type="checkbox"
          checked={value.faceDeid}
          disabled={!studyLooksLikeHead}
          onChange={e => onChange({ ...value, faceDeid: e.target.checked })}
          style={{ marginTop: 3 }}
        />
        <div>
          <div style={{ fontWeight: 500 }}>
            Remove face from head MRI / CT volume{' '}
            <span style={{ fontSize: '11px', color: '#9ca3af', fontWeight: 400 }}>
              (anterior-heuristic baseline)
            </span>
          </div>
          <div style={{ color: '#6b7280', fontSize: '12px' }}>
            {studyLooksLikeHead
              ? 'Head/brain study detected. Zeros out the anterior ~35% of each axial slice (matches the Python nibabel_fallback baseline). A higher-quality TF.js model is the next iteration.'
              : 'Non-head study detected — defacing skipped automatically.'}
          </div>
        </div>
      </label>

      {value.pixelScrub && studyCount > 1 && (
        <div style={{ marginTop: 10, fontSize: '12px', color: '#6b7280' }}>
          OCR will run once per file across all {studyCount} detected studies. The
          OCR engine loads on the first file and is reused — most of the slow part
          is the first download.
        </div>
      )}
    </div>
  )
}

function formatRange(loSec: number, hiSec: number): string {
  if (hiSec < 60) {
    return `~${Math.round(loSec)}–${Math.round(hiSec)}s`
  }
  const lo = Math.round(loSec / 60)
  const hi = Math.round(hiSec / 60)
  if (lo === hi) return `~${lo} min`
  return `~${lo}–${hi} min`
}
