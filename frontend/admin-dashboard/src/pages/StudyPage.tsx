import { useEffect } from 'react'
import { useParams } from 'react-router-dom'

// StudyPage is a v1 placeholder that deep-links into the existing admin
// dashboard's study detail panel. The richer StudyDetailPanel (~1500 lines,
// lives inside App.tsx) is reused untouched in this PR to keep the diff
// small; extracting it into a routed component is a follow-up.
export function StudyPage() {
  const { studyId } = useParams<{ studyId: string }>()

  useEffect(() => {
    if (!studyId) return
    window.location.replace(`/admin/studies?study_id=${encodeURIComponent(studyId)}`)
  }, [studyId])

  if (!studyId) {
    return <div className="aegis-error">Missing study ID.</div>
  }

  const target = `/admin/studies?study_id=${encodeURIComponent(studyId)}`
  return (
    <div className="aegis-study">
      <p>
        Opening study detail… If you aren't redirected,{' '}
        <a href={target}>click here</a>.
      </p>
    </div>
  )
}
