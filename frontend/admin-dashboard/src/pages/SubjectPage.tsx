import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { apiGetProjectSubject, type ProjectSubjectResponse } from '../api/subjects'

// SubjectPage shows demographics for a subject and the list of imaging
// sessions (studies) belonging to them, sorted newest-first. XNAT's subject
// report has additional sections (custom variable groups, assessors) that we
// can layer on as features are added — the current shape covers the primary
// drill-down that image analysts use day-to-day.
export function SubjectPage() {
  const { projectId, subjectId } = useParams<{ projectId: string; subjectId: string }>()
  const [data, setData] = useState<ProjectSubjectResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!projectId || !subjectId) return
    let cancelled = false
    setData(null)
    setError(null)
    apiGetProjectSubject(projectId, subjectId)
      .then((r) => !cancelled && setData(r))
      .catch((e) => !cancelled && setError(String(e)))
    return () => {
      cancelled = true
    }
  }, [projectId, subjectId])

  if (error) return <div className="xn-error">{error}</div>
  if (!data) return <div className="xn-muted">Loading…</div>

  const d = data.demographics
  return (
    <div className="xn-subject">
      <header className="xn-page-header">
        <h1>{data.subject_id}</h1>
        <p className="xn-muted">
          {data.study_count} study{data.study_count === 1 ? '' : 's'}
          {(data.modalities ?? []).length > 0 && ` · ${(data.modalities ?? []).join(', ')}`}
        </p>
      </header>

      <section className="xn-section">
        <h2>Demographics</h2>
        {d ? (
          <dl className="xn-defs">
            <dt>Sex</dt><dd>{d.sex || '—'}</dd>
            <dt>Age at scan</dt><dd>{d.age_at_scan ?? '—'}</dd>
            <dt>Diagnosis</dt><dd>{d.diagnosis || '—'}</dd>
            <dt>Education (years)</dt><dd>{d.education_years ?? '—'}</dd>
            <dt>MMSE</dt><dd>{d.mmse_score ?? '—'}</dd>
            <dt>MoCA</dt><dd>{d.moca_score ?? '—'}</dd>
            <dt>CDR global</dt><dd>{d.cdr_global ?? '—'}</dd>
            <dt>APOE genotype</dt><dd>{d.apoe_genotype || '—'}</dd>
            {d.notes && (
              <>
                <dt>Notes</dt>
                <dd>{d.notes}</dd>
              </>
            )}
          </dl>
        ) : (
          <div className="xn-muted">No demographic record. Add one from the admin tools.</div>
        )}
      </section>

      <section className="xn-section">
        <h2>Imaging sessions</h2>
        {data.studies.length === 0 && <div className="xn-muted">No studies for this subject.</div>}
        {data.studies.length > 0 && (
          <table className="xn-table">
            <thead>
              <tr>
                <th>Study date</th>
                <th>Modality</th>
                <th>Body part</th>
                <th>Description</th>
                <th>Status</th>
                <th>Instances</th>
              </tr>
            </thead>
            <tbody>
              {data.studies.map((s) => (
                <tr key={s.id}>
                  <td>
                    <Link to={`/projects/${projectId}/subjects/${subjectId}/studies/${s.id}`}>
                      {formatStudyDate(s.study_date)}
                    </Link>
                  </td>
                  <td>{s.modality ?? ''}</td>
                  <td>{s.body_part ?? ''}</td>
                  <td>{s.study_description ?? ''}</td>
                  <td>{s.status}</td>
                  <td>{s.instance_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}

function formatStudyDate(d?: string | null): string {
  if (!d) return '(unknown date)'
  if (/^\d{8}$/.test(d)) {
    return `${d.slice(0, 4)}-${d.slice(4, 6)}-${d.slice(6, 8)}`
  }
  return d
}
