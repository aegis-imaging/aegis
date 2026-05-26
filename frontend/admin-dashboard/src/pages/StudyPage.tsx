import { Navigate, useParams } from 'react-router-dom'

// StudyPage bridges the researcher-tree URL /projects/:p/subjects/:s/studies/:st
// to the canonical study-detail location at /studies?study_id=<id>. App.tsx
// reads the query param on mount and opens the StudyDetailPanel with every
// action surface visible (gated by role inside the panel itself).
//
// React Router's <Navigate> is used instead of window.location.replace so
// the redirect stays inside the SPA — no full-page reload, no flash.
export function StudyPage() {
  const { studyId } = useParams<{ studyId: string }>()
  if (!studyId) {
    return <div className="aegis-error">Missing study ID.</div>
  }
  return <Navigate to={`/studies?study_id=${encodeURIComponent(studyId)}`} replace />
}
