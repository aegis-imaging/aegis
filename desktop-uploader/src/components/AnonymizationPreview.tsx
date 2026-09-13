import type { ParsedStudy } from '../lib/anon'

interface Props {
  studies: ParsedStudy[]
  onConfirm: () => void
  onCancel: () => void
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

/**
 * Shows a per-study summary card with key DICOM tags. Anonymization happens
 * during upload — this preview reflects what the operator will be sending
 * once "Anonymize & upload" is clicked.
 *
 * The summary intentionally mirrors the web upload portal's StudySummary
 * component but stays inline here to avoid pulling in CSS files.
 */
export function AnonymizationPreview({ studies, onConfirm, onCancel }: Props) {
  const totalBytes = studies.reduce(
    (acc, s) => acc + s.files.reduce((a, f) => a + f.arrayBuffer.byteLength, 0),
    0,
  )
  const totalFiles = studies.reduce((acc, s) => acc + s.files.length, 0)

  return (
    <div style={{ marginTop: 16 }}>
      <h2 style={{ fontSize: 16, marginBottom: 8 }}>
        {studies.length} {studies.length === 1 ? 'study' : 'studies'} ready
        <span style={{ color: '#6b7280', fontWeight: 400, marginLeft: 8 }}>
          ({totalFiles} files, {formatBytes(totalBytes)})
        </span>
      </h2>
      <div style={{ display: 'grid', gap: 12 }}>
        {studies.map(study => (
          <div
            key={study.uid}
            style={{
              border: '1px solid #e5e7eb',
              borderRadius: 6,
              padding: 12,
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
              <strong>{study.summary.studyDescription || '(no description)'}</strong>
              <span style={{ color: '#6b7280', fontSize: 13 }}>
                {study.summary.modality} {study.summary.bodyPart}
              </span>
            </div>
            <dl style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', columnGap: 12, rowGap: 4, fontSize: 13, margin: 0 }}>
              <dt style={{ color: '#6b7280' }}>Study UID</dt>
              <dd style={{ margin: 0, fontFamily: 'monospace', fontSize: 12 }}>{study.summary.studyInstanceUid}</dd>
              <dt style={{ color: '#6b7280' }}>Study date</dt>
              <dd style={{ margin: 0 }}>{study.summary.studyDate || '(unknown)'}</dd>
              <dt style={{ color: '#6b7280' }}>Series / images</dt>
              <dd style={{ margin: 0 }}>{study.summary.seriesCount} / {study.summary.imageCount}</dd>
            </dl>
          </div>
        ))}
      </div>
      <div style={{ marginTop: 16, padding: 12, background: '#ccfbf1', color: '#0f766e', borderRadius: 6, fontSize: 13 }}>
        Patient-identifying tags will be removed locally per PS3.15 Annex E Basic Profile
        before any bytes leave this machine.
      </div>
      <div style={{ marginTop: 16, display: 'flex', gap: 8 }}>
        <button onClick={onConfirm} style={{ padding: '8px 16px', background: '#0d9488', color: 'white', border: 'none', borderRadius: 4 }}>
          Anonymize and upload
        </button>
        <button onClick={onCancel} style={{ padding: '8px 16px' }}>
          Pick a different folder
        </button>
      </div>
    </div>
  )
}
