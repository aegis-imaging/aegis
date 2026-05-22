// Cloud admin-dashboard wrapper around the shared @aegis/qc-components.
//
// Lays out the QC review surface for the analyst:
//   - Top: analyst throughput card (everyone's stats)
//   - Middle: triage queue (assigned-to-me by default; toggle "all" for
//             supervisors)
//   - Right drawer: when a row is clicked, the QCReviewPane slides in with
//                   an embedded DWV viewer for the selected study.
//
// The desktop app has its own host wrapper for the same components (different
// DICOM viewer integration). Look in `desktop/src/QCDesktopPanel.tsx`.

import { useState } from 'react'
import {
  AnalystThroughputCard,
  QCReviewPane,
  QCTriageQueue,
  type QCTriageItem,
} from '@aegis/qc-components'

export function QCDashboardPanel({ isAdmin }: { isAdmin: boolean }) {
  const [selected, setSelected] = useState<QCTriageItem | null>(null)

  return (
    <div style={{ padding: '16px 24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <h2 style={{ margin: 0, flex: 1 }}>QC Review</h2>
      </div>

      <div style={{ marginBottom: 20 }}>
        <AnalystThroughputCard />
      </div>

      <div style={{ marginBottom: 20 }}>
        <QCTriageQueue onSelectStudy={setSelected} />
      </div>

      {selected && (
        <ReviewDrawer
          study={selected}
          isAdmin={isAdmin}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  )
}

function ReviewDrawer({ study, isAdmin, onClose }: {
  study: QCTriageItem
  isAdmin: boolean
  onClose: () => void
}) {
  return (
    <div
      style={{
        position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)', zIndex: 1000,
        display: 'flex', alignItems: 'stretch', justifyContent: 'flex-end',
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: 'min(1200px, 95vw)', background: '#fff', boxShadow: '-4px 0 24px rgba(0,0,0,0.2)',
          padding: 20, overflowY: 'auto',
        }}
        onClick={e => e.stopPropagation()}
      >
        <button
          type="button"
          onClick={onClose}
          style={{
            position: 'sticky', top: 0, float: 'right', padding: '6px 12px',
            border: '1px solid #d1d5db', borderRadius: 4, background: '#fff', cursor: 'pointer',
            fontSize: 13,
          }}
        >
          Close ×
        </button>
        <QCReviewPane
          study={study}
          isAdmin={isAdmin}
          viewerSlot={<DWVEmbed studyUid={study.study_instance_uid} />}
          onCompleted={onClose}
        />
      </div>
    </div>
  )
}

function DWVEmbed({ studyUid }: { studyUid: string }) {
  // Reuses the dashboard's existing DWV iframe pattern. The base URL is
  // baked at build time via VITE_DWV_BASE_URL (see CLAUDE.md).
  const base = (import.meta as unknown as { env: { VITE_DWV_BASE_URL?: string } })
    .env.VITE_DWV_BASE_URL ?? ''
  if (!base) {
    return (
      <div style={{ padding: 24, color: '#9ca3af' }}>
        DWV viewer URL not configured (VITE_DWV_BASE_URL). Showing study UID:{' '}
        <code>{studyUid}</code>
      </div>
    )
  }
  const url = `${base}?studyUID=${encodeURIComponent(studyUid)}&store=clean`
  return (
    <iframe
      title="DWV viewer"
      src={url}
      style={{ width: '100%', height: 520, border: 0 }}
    />
  )
}
