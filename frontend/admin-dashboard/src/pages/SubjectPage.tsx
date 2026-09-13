import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  apiGetProjectSubject,
  type PipelineStatus,
  type ProjectSubjectResponse,
  type StudyStub,
} from '../api/subjects'

type StudySortKey = 'study_date' | 'modality' | 'status' | 'instance_count'

// SubjectPage shows demographics for a subject and the list of imaging
// sessions (studies) belonging to them. Per-pipeline-stage status icons
// (PHI / QC / BIDS / Classification / Protocol / Analytics) mirror the
// admin Studies tab so analysts can see at a glance which sessions still
// need review without leaving the subject view.
export function SubjectPage() {
  const { projectId, subjectId } = useParams<{ projectId: string; subjectId: string }>()
  const [data, setData] = useState<ProjectSubjectResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [sortKey, setSortKey] = useState<StudySortKey>('study_date')
  const [sortDesc, setSortDesc] = useState(true)

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

  const sortedStudies = useMemo(() => {
    if (!data) return []
    return [...data.studies].sort((a, b) => {
      let cmp = 0
      switch (sortKey) {
        case 'study_date':
          cmp = (a.study_date ?? '').localeCompare(b.study_date ?? '')
          break
        case 'modality':
          cmp = (a.modality ?? '').localeCompare(b.modality ?? '')
          break
        case 'status':
          cmp = a.status.localeCompare(b.status)
          break
        case 'instance_count':
          cmp = a.instance_count - b.instance_count
          break
      }
      return sortDesc ? -cmp : cmp
    })
  }, [data, sortKey, sortDesc])

  if (error) return <div className="aegis-error">{error}</div>
  if (!data) return <div className="aegis-muted">Loading…</div>

  const d = data.demographics

  return (
    <div className="aegis-subject">
      <header className="aegis-page-header">
        <h1>{data.subject_id}</h1>
        <p className="aegis-muted">
          {data.study_count} study{data.study_count === 1 ? '' : 's'}
          {(data.modalities ?? []).length > 0 && ` · ${(data.modalities ?? []).join(', ')}`}
        </p>
      </header>

      <section className="aegis-section">
        <h2>Demographics</h2>
        {d ? (
          <dl className="aegis-defs">
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
          <div className="aegis-muted">No demographic record. Add one from the admin tools.</div>
        )}
      </section>

      <section className="aegis-section">
        <div className="aegis-section-bar">
          <h2>Imaging sessions</h2>
          <div className="aegis-section-controls">
            <label className="aegis-control">
              <span className="aegis-control-label">Sort by</span>
              <select
                value={sortKey}
                onChange={(e) => setSortKey(e.target.value as StudySortKey)}
              >
                <option value="study_date">Study date</option>
                <option value="modality">Modality</option>
                <option value="status">Status</option>
                <option value="instance_count">Instances</option>
              </select>
            </label>
            <button
              type="button"
              className="aegis-icon-btn"
              onClick={() => setSortDesc((s) => !s)}
              aria-label={sortDesc ? 'Sort ascending' : 'Sort descending'}
              title={sortDesc ? 'Switch to ascending' : 'Switch to descending'}
            >
              {sortDesc ? '↓' : '↑'}
            </button>
          </div>
        </div>
        {data.studies.length === 0 && <div className="aegis-muted">No studies for this subject.</div>}
        {data.studies.length > 0 && (
          <div className="aegis-table-wrap">
            <table className="aegis-table">
              <thead>
                <tr>
                  <th>Study date</th>
                  <th>Modality</th>
                  <th>Body part</th>
                  <th>Description</th>
                  <th>Status</th>
                  <th title="PHI Scan">PHI</th>
                  <th title="Quality control">QC</th>
                  <th title="BIDS conversion">BIDS</th>
                  <th title="Classification">Class</th>
                  <th title="Protocol check">Proto</th>
                  <th title="Analytics">Anlx</th>
                  <th>Instances</th>
                </tr>
              </thead>
              <tbody>
                {sortedStudies.map((s) => (
                  <StudyRow
                    key={s.id}
                    study={s}
                    projectId={projectId!}
                    subjectId={subjectId!}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}

function StudyRow({
  study,
  projectId,
  subjectId,
}: {
  study: StudyStub
  projectId: string
  subjectId: string
}) {
  return (
    <tr>
      <td>
        <Link to={`/projects/${projectId}/subjects/${subjectId}/studies/${study.id}`}>
          {formatStudyDate(study.study_date)}
        </Link>
      </td>
      <td>{study.modality ?? ''}</td>
      <td>{study.body_part ?? ''}</td>
      <td className="aegis-cell-truncate" title={study.study_description ?? ''}>
        {study.study_description ?? ''}
      </td>
      <td>
        <StatusPill status={study.status} />
      </td>
      <td><StagePill required={study.phi_scan_required} status={study.phi_scan_status} /></td>
      <td><StagePill required={study.qc_required} status={study.qc_status} /></td>
      <td><StagePill required={study.bids_required} status={study.bids_status} /></td>
      <td><StagePill required={study.classification_required} status={study.classification_status} /></td>
      <td><StagePill required={study.protocol_required} status={study.protocol_status} /></td>
      <td><StagePill required={study.analytics_required} status={study.analytics_status} /></td>
      <td>{study.instance_count}</td>
    </tr>
  )
}

// StagePill renders a one-glance indicator of a single pipeline stage's
// state for a study. Empty pill when the stage isn't required for this
// study (routing rules decided not to run it). Color is keyed off the
// project's colorblind-friendly palette (teal for positive, orange for
// negative; never red/green).
function StagePill({
  required,
  status,
}: {
  required?: boolean
  status?: PipelineStatus
}) {
  if (!required) return <span className="aegis-stage aegis-stage-na" aria-label="not required">—</span>
  const cls = stageClass(status)
  const label = status || 'pending'
  return (
    <span className={`aegis-stage ${cls}`} title={label} aria-label={`Status: ${label}`}>
      {stageGlyph(status)}
    </span>
  )
}

function StatusPill({ status }: { status: string }) {
  const cls = studyStatusClass(status)
  return <span className={`aegis-status ${cls}`}>{status}</span>
}

function stageClass(status?: PipelineStatus): string {
  switch (status) {
    case 'complete':
      return 'aegis-stage-complete'
    case 'partial':
      return 'aegis-stage-partial'
    case 'failed':
      return 'aegis-stage-failed'
    case 'running':
    case 'scanning':
    case 'analyzing':
    case 'converting':
      return 'aegis-stage-running'
    case 'pending':
    case '':
    case undefined:
      return 'aegis-stage-pending'
    default:
      return 'aegis-stage-pending'
  }
}

function stageGlyph(status?: PipelineStatus): string {
  switch (status) {
    case 'complete':
      return '●'
    case 'partial':
      return '◐'
    case 'failed':
      return '✕'
    case 'running':
    case 'scanning':
    case 'analyzing':
    case 'converting':
      return '◌'
    default:
      return '○'
  }
}

function studyStatusClass(status: string): string {
  switch (status) {
    case 'approved':
      return 'aegis-status-approved'
    case 'rejected':
    case 'expired':
      return 'aegis-status-rejected'
    case 'defacing':
    case 'defaced':
      return 'aegis-status-active'
    default:
      return 'aegis-status-default'
  }
}

function formatStudyDate(d?: string | null): string {
  if (!d) return '(unknown date)'
  if (/^\d{8}$/.test(d)) {
    return `${d.slice(0, 4)}-${d.slice(4, 6)}-${d.slice(6, 8)}`
  }
  return d
}
