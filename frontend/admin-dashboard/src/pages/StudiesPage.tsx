import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'

// StudiesPage is the researcher-facing studies list at /studies.
//
// Distinct from the admin pipeline-triage view at /admin/studies — that
// page has bulk actions, pipeline stage funnel, modality breakdown, and
// per-stage filters that an operator needs but a researcher mostly
// doesn't. This page just shows recent studies the caller has access to
// with the columns most useful for browsing (subject, modality, date,
// status), and each row links into the researcher study detail at
// /projects/:p/subjects/:s/studies/:study so the rest of the flow
// stays inside the researcher chrome.
//
// Backend uses the same /api/studies endpoint as admin; the caller's
// IAP identity scopes which studies the response includes.
type StudyRow = {
  id: string
  project_id: string
  subject_id?: string
  study_instance_uid: string
  modality: string
  body_part: string
  study_description: string
  study_date?: string
  status: string
  source: string
  instance_count: number
  created_at: string
}

const PAGE_SIZE = 50

export function StudiesPage() {
  const [rows, setRows] = useState<StudyRow[] | null>(null)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')

  useEffect(() => {
    let cancelled = false
    setRows(null)
    setError(null)
    const params = new URLSearchParams({
      limit: String(PAGE_SIZE),
      offset: String(page * PAGE_SIZE),
    })
    if (statusFilter) params.set('status', statusFilter)
    fetch(`/api/studies?${params}`)
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`)))
      .then((d: { studies: StudyRow[]; total: number }) => {
        if (cancelled) return
        setRows(d.studies ?? [])
        setTotal(d.total ?? 0)
      })
      .catch(e => !cancelled && setError(e instanceof Error ? e.message : String(e)))
    return () => { cancelled = true }
  }, [page, statusFilter])

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>Studies</h1>
            <p className="aegis-page-description">
              Imaging studies you have access to, across every project. Click a row to open its details.
              For the global pipeline triage view, see the admin Studies tab.
            </p>
          </div>
        </div>
      </header>

      <section className="aegis-section">
        <div className="aegis-chip-row">
          <span className="aegis-chip-row-label">Status:</span>
          <button
            type="button"
            className={`aegis-chip${statusFilter === '' ? ' aegis-chip-active' : ''}`}
            onClick={() => { setStatusFilter(''); setPage(0) }}
          >
            All
          </button>
          {['received', 'defacing', 'clean', 'defaced', 'approved', 'rejected', 'expired'].map(s => (
            <button
              key={s}
              type="button"
              className={`aegis-chip${statusFilter === s ? ' aegis-chip-active' : ''}`}
              onClick={() => { setStatusFilter(s === statusFilter ? '' : s); setPage(0) }}
            >
              {s}
            </button>
          ))}
        </div>

        {error && <div className="aegis-error">{error}</div>}
        {rows === null && !error && <div className="aegis-muted">Loading studies…</div>}
        {rows && rows.length === 0 && !error && (
          <div className="aegis-muted">No studies match the current filter.</div>
        )}

        {rows && rows.length > 0 && (
          <>
            <div className="aegis-section-bar" style={{ marginBottom: 8 }}>
              <span className="aegis-muted">{total} {total === 1 ? 'study' : 'studies'}</span>
              <div className="aegis-section-controls">
                <button type="button" className="aegis-btn-secondary" disabled={page === 0} onClick={() => setPage(p => p - 1)}>
                  ← Prev
                </button>
                <span className="aegis-muted">Page {page + 1} of {totalPages}</span>
                <button type="button" className="aegis-btn-secondary" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>
                  Next →
                </button>
              </div>
            </div>

            <div className="aegis-table-wrap">
              <table className="aegis-table">
                <thead>
                  <tr>
                    <th>Subject / UID</th>
                    <th>Modality</th>
                    <th>Body part</th>
                    <th>Study date</th>
                    <th>Status</th>
                    <th>Series</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map(s => {
                    const target = s.subject_id
                      ? `/projects/${s.project_id}/subjects/${s.subject_id}/studies/${s.id}`
                      : null
                    const cell = (content: React.ReactNode) =>
                      target
                        ? <Link to={target} style={{ color: 'inherit', textDecoration: 'none' }}>{content}</Link>
                        : content
                    return (
                      <tr key={s.id}>
                        <td>
                          {cell(
                            <>
                              {s.subject_id && <div style={{ fontWeight: 500 }}>{s.subject_id}</div>}
                              <code className="aegis-code" style={{ fontSize: 11 }}>
                                {s.study_instance_uid.length > 24
                                  ? s.study_instance_uid.slice(0, 18) + '…' + s.study_instance_uid.slice(-6)
                                  : s.study_instance_uid}
                              </code>
                            </>,
                          )}
                        </td>
                        <td>{cell(s.modality || '—')}</td>
                        <td>{cell(s.body_part || '—')}</td>
                        <td>{cell(s.study_date ? new Date(s.study_date).toLocaleDateString() : '—')}</td>
                        <td>{cell(<span className="aegis-pill">{s.status}</span>)}</td>
                        <td>{cell(s.instance_count.toLocaleString())}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </>
        )}
      </section>
    </>
  )
}
