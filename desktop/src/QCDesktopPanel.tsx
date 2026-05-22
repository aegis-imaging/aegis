// Desktop-side host for the shared QC components. Same components as the
// admin-dashboard, just bound to a remote AEGIS via an API key rather than
// a session cookie.

import { useMemo, useState } from 'react'
import {
  AnalystThroughputCard,
  QCReviewPane,
  QCTriageQueue,
  type QCTriageItem,
} from '@aegis/qc-components'

export interface QCDesktopPanelProps {
  apiBaseUrl: string
  apiKey: string
}

export function QCDesktopPanel({ apiBaseUrl, apiKey }: QCDesktopPanelProps) {
  const [selected, setSelected] = useState<QCTriageItem | null>(null)

  const apiOptions = useMemo(() => ({
    baseUrl: apiBaseUrl,
    authorization: apiKey ? `Bearer ${apiKey}` : undefined,
  }), [apiBaseUrl, apiKey])

  if (!apiKey) {
    return (
      <div style={{
        padding: 16, border: '1px solid #fde68a', background: '#fefce8', borderRadius: 8,
        fontSize: 13, color: '#92400e',
      }}>
        Set an API key in the Connection panel to use QC review. Generate one from the
        admin-dashboard (Settings → API keys) and paste it above.
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <AnalystThroughputCard apiOptions={apiOptions} />
      <QCTriageQueue apiOptions={apiOptions} onSelectStudy={setSelected} />
      {selected && (
        <div style={{ borderTop: '2px solid #e5e7eb', paddingTop: 16 }}>
          <button
            type="button"
            onClick={() => setSelected(null)}
            style={{ float: 'right', padding: '4px 10px', border: '1px solid #d1d5db', borderRadius: 4, cursor: 'pointer' }}
          >
            Back to queue
          </button>
          <QCReviewPane
            study={selected}
            apiOptions={apiOptions}
            onCompleted={() => setSelected(null)}
            viewerSlot={
              <div style={{ padding: 40, color: '#9ca3af', textAlign: 'center' }}>
                Inline DICOM viewer not bundled in desktop yet — open in browser instead:{' '}
                <a href={`${apiBaseUrl}/study/${selected.study_id}`} target="_blank" rel="noopener">
                  view in cloud dashboard ↗
                </a>
              </div>
            }
          />
        </div>
      )}
    </div>
  )
}
