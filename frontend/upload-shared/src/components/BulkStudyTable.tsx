// Live table of detected studies during a bulk upload. Each row updates as
// the per-study uploader transitions states. Renders during both 'preview'
// (before upload starts) and 'uploading' (live status).

import type { BulkProgress, BulkStudyState, BulkStudyStatus } from '@aegis/client'

const STATUS_COLOR: Record<BulkStudyStatus, string> = {
  pending: '#9ca3af',
  uploading: '#2563eb',
  completed: '#15803d',
  duplicate: '#a16207',
  failed: '#b91c1c',
  cancelled: '#6b7280',
}

const STATUS_LABEL: Record<BulkStudyStatus, string> = {
  pending: 'queued',
  uploading: '⟳ uploading',
  completed: '✓ uploaded',
  duplicate: '↻ already in cloud',
  failed: '✗ failed',
  cancelled: '⊘ cancelled',
}

export interface BulkStudyTableProps {
  progress: BulkProgress
}

export function BulkStudyTable({ progress }: BulkStudyTableProps) {
  const totalCount = progress.studies.length
  if (totalCount === 0) return null

  const completedCount = progress.studies.filter(s => s.status === 'completed').length
  const dupCount = progress.studies.filter(s => s.status === 'duplicate').length
  const failedCount = progress.studies.filter(s => s.status === 'failed').length

  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ display: 'flex', gap: 16, marginBottom: 8, fontSize: 13, color: '#374151' }}>
        <span><strong>{totalCount}</strong> studies detected</span>
        {progress.phase !== 'parsing' && progress.phase !== 'grouping' && (
          <>
            <span style={{ color: '#15803d' }}>✓ {completedCount}</span>
            {dupCount > 0 && <span style={{ color: '#a16207' }}>↻ {dupCount} duplicate</span>}
            {failedCount > 0 && <span style={{ color: '#b91c1c' }}>✗ {failedCount} failed</span>}
          </>
        )}
      </div>
      <div style={{
        maxHeight: 320, overflowY: 'auto', border: '1px solid #e5e7eb',
        borderRadius: 6, background: '#fff',
      }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead style={{ background: '#f9fafb', position: 'sticky', top: 0 }}>
            <tr>
              <th style={th()}>Subject</th>
              <th style={th()}>Description</th>
              <th style={{ ...th(), textAlign: 'right' }}>Files</th>
              <th style={th()}>Status</th>
              <th style={{ ...th(), textAlign: 'right' }}>Time</th>
            </tr>
          </thead>
          <tbody>
            {progress.studies.map(s => <Row key={s.studyInstanceUid} state={s} />)}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Row({ state }: { state: BulkStudyState }) {
  const color = STATUS_COLOR[state.status]
  const label = STATUS_LABEL[state.status]
  return (
    <tr style={{ borderTop: '1px solid #f3f4f6' }}>
      <td style={td()}>
        <code style={{ fontSize: 12 }}>{state.patientId || '—'}</code>
      </td>
      <td style={td()}>
        {state.studyDescription}
        <div style={{ fontSize: 11, color: '#9ca3af', fontFamily: 'monospace' }}>
          {state.studyInstanceUid.length > 32
            ? `…${state.studyInstanceUid.slice(-30)}`
            : state.studyInstanceUid}
        </div>
      </td>
      <td style={{ ...td(), textAlign: 'right' }}>{state.fileCount}</td>
      <td style={td()}>
        <span style={{ color, fontWeight: 500 }}>{label}</span>
        {state.error && (
          <div style={{ fontSize: 11, color: '#b91c1c' }}>{state.error}</div>
        )}
      </td>
      <td style={{ ...td(), textAlign: 'right', color: '#6b7280' }}>
        {state.durationMs != null ? `${(state.durationMs / 1000).toFixed(1)}s` : ''}
      </td>
    </tr>
  )
}

function th(): React.CSSProperties {
  return {
    padding: '8px 10px', textAlign: 'left', fontWeight: 600,
    color: '#374151', fontSize: 12, borderBottom: '1px solid #e5e7eb',
  }
}

function td(): React.CSSProperties {
  return { padding: '8px 10px', verticalAlign: 'top' }
}
