// Per-file progress + findings summary shown during upload when heavy mode
// is enabled. Lives next to the existing "Uploading 3/10" indicator.

import type { PixelScrubResult } from '@aegis/client'

export interface PixelScrubProgressState {
  currentFileIndex: number
  totalFiles: number
  currentFileName: string
  results: { filename: string; result: PixelScrubResult }[]
}

export interface PixelScrubProgressProps {
  state: PixelScrubProgressState
}

export function PixelScrubProgress({ state }: PixelScrubProgressProps) {
  const completed = state.results.length
  const totalFindings = state.results.reduce((n, r) => n + r.result.totalFindings, 0)
  const filesWithFindings = state.results.filter(r => r.result.totalFindings > 0).length
  const filesUnsupported = state.results.filter(r => r.result.status === 'unsupported_transfer_syntax').length
  const filesFailed = state.results.filter(r => r.result.status === 'failed').length

  return (
    <div style={{
      padding: '12px 14px',
      background: '#f0fdf4',
      border: '1px solid #bbf7d0',
      borderRadius: '8px',
      fontSize: '13px',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <span style={{ fontWeight: 600 }}>Pixel PHI scrub</span>
        <span style={{ color: '#15803d' }}>
          {completed} / {state.totalFiles} files
        </span>
      </div>
      {state.currentFileName && completed < state.totalFiles && (
        <div style={{ color: '#15803d', fontSize: '12px', marginBottom: 6 }}>
          Scrubbing <code>{state.currentFileName}</code>…
        </div>
      )}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8 }}>
        <Stat label="PHI regions burned" value={totalFindings} good={totalFindings > 0} />
        <Stat label="Files with PHI" value={filesWithFindings} good={filesWithFindings > 0} />
        <Stat label="Unsupported" value={filesUnsupported} bad={filesUnsupported > 0} />
        <Stat label="Failed" value={filesFailed} bad={filesFailed > 0} />
      </div>
      {filesUnsupported > 0 && (
        <div style={{ marginTop: 10, color: '#a16207', fontSize: '12px' }}>
          Some files use compressed transfer syntaxes (JPEG 2000, JPEG-LS, RLE) that
          the in-browser scrub doesn&apos;t handle yet. Those files were uploaded
          unmodified — server-side scrub will handle them.
        </div>
      )}
    </div>
  )
}

function Stat({ label, value, good, bad }: { label: string; value: number; good?: boolean; bad?: boolean }) {
  const color = bad ? '#b91c1c' : good ? '#15803d' : '#6b7280'
  return (
    <div>
      <div style={{ fontSize: '20px', fontWeight: 700, color }}>{value}</div>
      <div style={{ fontSize: '11px', color: '#6b7280' }}>{label}</div>
    </div>
  )
}
