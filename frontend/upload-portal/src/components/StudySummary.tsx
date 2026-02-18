import type { StudySummary as StudySummaryType } from '../types'

interface StudySummaryProps {
  summary: StudySummaryType
}

export function StudySummary({ summary }: StudySummaryProps) {
  return (
    <div style={{
      border: '1px solid #e5e7eb',
      borderRadius: '8px',
      padding: '20px',
      backgroundColor: '#fff',
    }}>
      <h3 style={{ margin: '0 0 16px', fontSize: '16px', fontWeight: 600 }}>
        Study Summary
      </h3>
      <div style={{
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: '12px',
        fontSize: '14px',
      }}>
        <Field label="Patient Name" value={summary.patientName} sensitive />
        <Field label="Patient ID" value={summary.patientId} sensitive />
        <Field label="Study Date" value={summary.studyDate} />
        <Field label="Modality" value={summary.modality} />
        <Field label="Body Part" value={summary.bodyPart} />
        <Field label="Description" value={summary.studyDescription} />
        <Field label="Series" value={String(summary.seriesCount)} />
        <Field label="Images" value={String(summary.imageCount)} />
      </div>
    </div>
  )
}

function Field({ label, value, sensitive }: { label: string; value: string; sensitive?: boolean }) {
  return (
    <div>
      <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '2px' }}>
        {label}
      </div>
      <div style={{
        fontFamily: 'monospace',
        color: sensitive && value ? '#dc2626' : '#111827',
        fontWeight: sensitive && value ? 600 : 400,
      }}>
        {value || '(empty)'}
        {sensitive && value && (
          <span style={{
            marginLeft: '8px',
            fontSize: '11px',
            color: '#dc2626',
            backgroundColor: '#fef2f2',
            padding: '1px 6px',
            borderRadius: '4px',
          }}>
            PHI — will be removed
          </span>
        )}
      </div>
    </div>
  )
}
