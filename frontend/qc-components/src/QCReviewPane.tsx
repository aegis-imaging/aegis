// Single-study review pane. Hosts know best where to put a DICOM viewer, so
// they pass it in via `viewerSlot`; the pane lays out viewer + findings panel
// + actions. Used by both the admin-dashboard (inline drawer) and the
// desktop app (full-screen mode).

import { useState } from 'react'
import * as api from './api'
import { QCFindingsPanel } from './QCFindingsPanel'
import type { QCTriageItem } from './types'

export interface QCReviewPaneProps {
  /** The study being reviewed. */
  study: QCTriageItem
  apiOptions?: api.APIClientOptions
  isAdmin?: boolean
  /** A React node that renders the DICOM viewer. Hosts pass DWV/OHIF here. */
  viewerSlot?: React.ReactNode
  /** Triggered when the analyst hits "Complete review". Host typically pops
   *  the pane and refreshes the queue. */
  onCompleted?: () => void
}

export function QCReviewPane({
  study, apiOptions, isAdmin = true, viewerSlot, onCompleted,
}: QCReviewPaneProps) {
  const [reviewStartedAt, setReviewStartedAt] = useState(study.qc_review_started_at)
  const [reviewEndedAt, setReviewEndedAt] = useState(study.qc_review_ended_at)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function startReview() {
    setBusy(true)
    try {
      await api.startQC(study.study_id, apiOptions)
      setReviewStartedAt(new Date().toISOString())
    } catch (e) {
      setError(String(e))
    } finally { setBusy(false) }
  }
  async function completeReview() {
    setBusy(true)
    try {
      await api.completeQC(study.study_id, apiOptions)
      setReviewEndedAt(new Date().toISOString())
      onCompleted?.()
    } catch (e) {
      setError(String(e))
    } finally { setBusy(false) }
  }
  async function selfAssign() {
    setBusy(true)
    try {
      await api.assignQC(study.study_id, '', apiOptions)
    } catch (e) {
      setError(String(e))
    } finally { setBusy(false) }
  }

  const reviewActive = !!reviewStartedAt && !reviewEndedAt
  const reviewComplete = !!reviewEndedAt

  return (
    <div className="qc-review-pane" style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <header style={{ display: 'flex', alignItems: 'baseline', gap: 12 }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>
          {study.modality}{study.body_part ? ` · ${study.body_part}` : ''}{study.study_description ? ` · ${study.study_description}` : ''}
        </h2>
        <code style={{ fontSize: 11, color: '#9ca3af' }}>{study.study_instance_uid}</code>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: 8 }}>
          {!study.qc_assigned_to_id && isAdmin && (
            <button type="button" style={btn()} disabled={busy} onClick={selfAssign}>
              Assign to me
            </button>
          )}
          {study.qc_assigned_to_id && !reviewActive && !reviewComplete && isAdmin && (
            <button type="button" style={btnPrimary()} disabled={busy} onClick={startReview}>
              Start review
            </button>
          )}
          {reviewActive && isAdmin && (
            <button type="button" style={btnPrimary()} disabled={busy} onClick={completeReview}>
              Complete review
            </button>
          )}
          {reviewComplete && (
            <span style={{
              padding: '4px 10px', borderRadius: 4, background: '#dcfce7', color: '#166534', fontSize: 12, fontWeight: 500,
            }}>
              Reviewed
            </span>
          )}
        </div>
      </header>

      {error && (
        <div style={{ padding: 8, background: '#fee2e2', color: '#b91c1c', borderRadius: 4, fontSize: 13 }}>{error}</div>
      )}

      <div style={{ fontSize: 13, color: '#6b7280', display: 'flex', gap: 16, flexWrap: 'wrap' }}>
        <span>Project: <strong>{study.project_name}</strong></span>
        {study.institution_name && <span>Institution: <strong>{study.institution_name}</strong></span>}
        <span>Status: <strong>{study.status}</strong></span>
        <span>QC status: <strong>{study.qc_status || 'pending'}</strong></span>
        <span>Assigned to: <strong>{study.qc_assigned_to_email || 'unassigned'}</strong></span>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 2fr) minmax(280px, 1fr)', gap: 16 }}>
        <div style={{ minHeight: 480, border: '1px solid #e5e7eb', borderRadius: 8, padding: 0 }}>
          {viewerSlot ?? (
            <div style={{ padding: 40, color: '#9ca3af', textAlign: 'center' }}>
              Host did not provide a DICOM viewer. Pass <code>viewerSlot</code> with your
              DWV/OHIF embed to populate this region.
            </div>
          )}
        </div>
        <div>
          <QCFindingsPanel
            studyId={study.study_id}
            apiOptions={apiOptions}
            isAdmin={isAdmin}
          />
        </div>
      </div>
    </div>
  )
}

function btn(): React.CSSProperties {
  return { padding: '6px 12px', borderRadius: 4, border: '1px solid #d1d5db', background: '#fff', cursor: 'pointer', fontSize: 13 }
}
function btnPrimary(): React.CSSProperties {
  return { ...btn(), background: '#2563eb', color: '#fff', borderColor: '#2563eb' }
}
