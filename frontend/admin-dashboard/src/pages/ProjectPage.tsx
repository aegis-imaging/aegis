import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  apiGetProject,
  apiListProjectSubjects,
  type Project,
  type ProjectSubjectsResponse,
  type SubjectAggregate,
} from '../api/subjects'

// ProjectPage shows the Subjects table for a project. XNAT's project report
// page has multiple tabs (Subjects / Sessions / Resources / Pipelines); v1
// of AEGIS lands on Subjects since that's the primary drill-down for image
// analysts. Other tabs can be added incrementally without re-architecting.
export function ProjectPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [project, setProject] = useState<Project | null>(null)
  const [subjects, setSubjects] = useState<SubjectAggregate[] | null>(null)
  const [filter, setFilter] = useState('')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!projectId) return
    let cancelled = false
    setSubjects(null)
    setError(null)
    Promise.all([apiGetProject(projectId), apiListProjectSubjects(projectId)])
      .then(([p, s]: [Project, ProjectSubjectsResponse]) => {
        if (cancelled) return
        setProject(p)
        setSubjects(s.subjects)
      })
      .catch((e) => !cancelled && setError(String(e)))
    return () => {
      cancelled = true
    }
  }, [projectId])

  const filtered = subjects
    ? subjects.filter((s) =>
        filter.trim() === ''
          ? true
          : s.subject_id.toLowerCase().includes(filter.toLowerCase()),
      )
    : null

  return (
    <div className="xn-project">
      <header className="xn-page-header">
        <h1>{project?.name ?? 'Project'}</h1>
        {project?.description && <p className="xn-muted">{project.description}</p>}
      </header>

      {error && <div className="xn-error">{error}</div>}

      <section className="xn-section">
        <div className="xn-section-bar">
          <h2>Subjects</h2>
          <input
            className="xn-filter"
            type="search"
            placeholder="Filter by subject ID…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            aria-label="Filter subjects"
          />
        </div>

        {!subjects && !error && <div className="xn-muted">Loading…</div>}
        {subjects && subjects.length === 0 && (
          <div className="xn-muted">No subjects in this project yet.</div>
        )}
        {filtered && filtered.length > 0 && (
          <table className="xn-table">
            <thead>
              <tr>
                <th scope="col">Subject ID</th>
                <th scope="col">Studies</th>
                <th scope="col">Latest study</th>
                <th scope="col">Modalities</th>
                <th scope="col">Sex</th>
                <th scope="col">Age at scan</th>
                <th scope="col">Diagnosis</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((s) => (
                <tr key={s.subject_id}>
                  <td>
                    <Link to={`/projects/${projectId}/subjects/${s.subject_id}`}>
                      {s.subject_id}
                    </Link>
                  </td>
                  <td>{s.study_count}</td>
                  <td>{formatStudyDate(s.latest_study_date)}</td>
                  <td>{(s.modalities ?? []).join(', ')}</td>
                  <td>{s.demographics?.sex ?? ''}</td>
                  <td>{s.demographics?.age_at_scan ?? ''}</td>
                  <td>{s.demographics?.diagnosis ?? ''}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}

// formatStudyDate turns DICOM-style YYYYMMDD into YYYY-MM-DD for display.
// Falls back to the raw value when the format doesn't match (defensive
// against legacy rows or unexpected inputs).
function formatStudyDate(d?: string): string {
  if (!d) return ''
  if (/^\d{8}$/.test(d)) {
    return `${d.slice(0, 4)}-${d.slice(4, 6)}-${d.slice(6, 8)}`
  }
  return d
}
