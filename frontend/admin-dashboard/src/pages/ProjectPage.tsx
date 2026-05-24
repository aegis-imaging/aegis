import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  apiGetProject,
  apiListProjectSubjects,
  type Project,
  type ProjectSubjectsResponse,
  type SubjectAggregate,
} from '../api/subjects'

type SortKey = 'subject_id' | 'study_count' | 'latest_study_date'

// ProjectPage shows the Subjects table for a project. v1 lands on
// Subjects since that's the primary drill-down for image analysts. The
// filter bar (modality chips, sort selector, free-text filter) is
// modeled on the rich Studies-tab filters in App.tsx so analysts coming
// from the admin view see familiar controls — though scoped to the
// fields that actually vary at the subject level (modalities, study
// count, latest date), not the per-study pipeline fields.
export function ProjectPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [project, setProject] = useState<Project | null>(null)
  const [subjects, setSubjects] = useState<SubjectAggregate[] | null>(null)
  const [textFilter, setTextFilter] = useState('')
  const [selectedModalities, setSelectedModalities] = useState<Set<string>>(new Set())
  const [sortKey, setSortKey] = useState<SortKey>('subject_id')
  const [sortDesc, setSortDesc] = useState(false)
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

  // Distinct modalities across the loaded subjects — used to render the
  // chip filter row only with values actually present in this project.
  const availableModalities = useMemo(() => {
    if (!subjects) return []
    const set = new Set<string>()
    for (const s of subjects) for (const m of s.modalities ?? []) set.add(m)
    return Array.from(set).sort()
  }, [subjects])

  const filteredSorted = useMemo(() => {
    if (!subjects) return null
    const lc = textFilter.trim().toLowerCase()
    const modFilter = selectedModalities
    const filtered = subjects.filter((s) => {
      if (lc && !s.subject_id.toLowerCase().includes(lc)) return false
      if (modFilter.size > 0) {
        const mods = s.modalities ?? []
        const overlap = mods.some((m) => modFilter.has(m))
        if (!overlap) return false
      }
      return true
    })
    const sorted = [...filtered].sort((a, b) => {
      let cmp = 0
      if (sortKey === 'subject_id') {
        cmp = a.subject_id.localeCompare(b.subject_id)
      } else if (sortKey === 'study_count') {
        cmp = a.study_count - b.study_count
      } else if (sortKey === 'latest_study_date') {
        cmp = (a.latest_study_date ?? '').localeCompare(b.latest_study_date ?? '')
      }
      return sortDesc ? -cmp : cmp
    })
    return sorted
  }, [subjects, textFilter, selectedModalities, sortKey, sortDesc])

  function toggleModality(m: string) {
    setSelectedModalities((prev) => {
      const next = new Set(prev)
      if (next.has(m)) next.delete(m)
      else next.add(m)
      return next
    })
  }

  return (
    <div className="aegis-project">
      <header className="aegis-page-header">
        <h1>{project?.name ?? 'Project'}</h1>
        {project?.description && <p className="aegis-muted">{project.description}</p>}
      </header>

      {error && <div className="aegis-error">{error}</div>}

      <section className="aegis-section">
        <div className="aegis-section-bar">
          <h2>Subjects</h2>
          <div className="aegis-section-controls">
            <label className="aegis-control">
              <span className="aegis-control-label">Sort by</span>
              <select
                value={sortKey}
                onChange={(e) => setSortKey(e.target.value as SortKey)}
              >
                <option value="subject_id">Subject ID</option>
                <option value="study_count">Study count</option>
                <option value="latest_study_date">Latest study</option>
              </select>
            </label>
            <button
              type="button"
              className="aegis-icon-btn"
              onClick={() => setSortDesc((d) => !d)}
              aria-label={sortDesc ? 'Sort ascending' : 'Sort descending'}
              title={sortDesc ? 'Switch to ascending' : 'Switch to descending'}
            >
              {sortDesc ? '↓' : '↑'}
            </button>
            <input
              className="aegis-filter"
              type="search"
              placeholder="Filter by subject ID…"
              value={textFilter}
              onChange={(e) => setTextFilter(e.target.value)}
              aria-label="Filter subjects"
            />
          </div>
        </div>

        {availableModalities.length > 0 && (
          <div className="aegis-chip-row" role="group" aria-label="Modality filter">
            <span className="aegis-chip-row-label">Modality:</span>
            {availableModalities.map((m) => {
              const active = selectedModalities.has(m)
              return (
                <button
                  key={m}
                  type="button"
                  className={`aegis-chip ${active ? 'aegis-chip-active' : ''}`}
                  onClick={() => toggleModality(m)}
                  aria-pressed={active}
                >
                  {m}
                </button>
              )
            })}
            {selectedModalities.size > 0 && (
              <button
                type="button"
                className="aegis-chip-clear"
                onClick={() => setSelectedModalities(new Set())}
              >
                Clear
              </button>
            )}
          </div>
        )}

        {!subjects && !error && <div className="aegis-muted">Loading…</div>}
        {subjects && subjects.length === 0 && (
          <div className="aegis-muted">No subjects in this project yet.</div>
        )}
        {filteredSorted && subjects && subjects.length > 0 && filteredSorted.length === 0 && (
          <div className="aegis-muted">No subjects match the current filters.</div>
        )}
        {filteredSorted && filteredSorted.length > 0 && (
          <table className="aegis-table">
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
              {filteredSorted.map((s) => (
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
