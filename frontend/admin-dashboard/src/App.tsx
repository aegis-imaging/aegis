import { useState, useEffect, useCallback } from 'react'
import './App.css'
import { ViewerPanel } from './components/ViewerPanel'

// ── Types ────────────────────────────────────────────────────────────────────

type AppTab = 'studies' | 'audit'

type AuditEntry = {
  id: string
  action: string
  actor: string
  resource_type: string
  resource_id: string
  detail?: Record<string, unknown>
  ip_address: string
  created_at: string
}

type Study = {
  id: string
  study_instance_uid: string
  modality: string
  body_part: string
  source: string
  status: string
  defacing_required: boolean
  instance_count: number
  created_at: string
}

type Share = {
  id: string
  study_id: string
  recipient_email: string
  note: string
  expires_at: string
  revoked_at?: string
  created_at: string
}

type NewShareResult = Share & {
  token: string
  export_url: string
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function fmtDate(iso: string) {
  return new Date(iso).toLocaleString()
}

function uidShort(uid: string) {
  return uid.length > 20 ? '…' + uid.slice(-18) : uid
}

function Badge({ label, prefix }: { label: string; prefix: 'status' | 'source' }) {
  const cls = `badge badge--${label}`
  return <span className={cls}>{label}</span>
}

// ── Audit Log ─────────────────────────────────────────────────────────────────

const ACTION_GROUPS: Record<string, string> = {
  'upload.init':      'upload',
  'upload.complete':  'upload',
  'ingest.internal':  'upload',
  'study.approved':   'study-ok',
  'study.rejected':   'study-err',
  'share.created':    'share',
  'share.revoked':    'share-revoked',
  'export.redeemed':  'share',
  'deface.triggered': 'deface',
  'deface.complete':  'deface',
  'deface.failed':    'deface-err',
}

function AuditLog() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const fetchAudit = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/audit?limit=200')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setEntries(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load audit log')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAudit() }, [fetchAudit])

  const filtered = filter
    ? entries.filter(e => e.action.startsWith(filter))
    : entries

  const uniqueActions = [...new Set(entries.map(e => e.action.split('.')[0]))].sort()

  return (
    <div>
      <div className="audit-toolbar">
        <div className="audit-filters">
          <span className="audit-filter-label">Filter by category:</span>
          <button
            type="button"
            className={`audit-filter-btn${filter === '' ? ' audit-filter-btn--active' : ''}`}
            onClick={() => setFilter('')}
          >
            All ({entries.length})
          </button>
          {uniqueActions.map(prefix => {
            const count = entries.filter(e => e.action.startsWith(prefix)).length
            return (
              <button
                key={prefix}
                type="button"
                className={`audit-filter-btn${filter === prefix ? ' audit-filter-btn--active' : ''}`}
                onClick={() => setFilter(prefix === filter.split('.')[0] && filter === prefix ? '' : prefix)}
              >
                {prefix} ({count})
              </button>
            )
          })}
        </div>
        <button type="button" className="btn-refresh" onClick={fetchAudit}>Refresh</button>
      </div>

      {loading && <div className="state-loading">Loading audit log…</div>}
      {error   && <div className="state-error">{error}</div>}

      {!loading && !error && filtered.length === 0 && (
        <div className="state-empty">No audit entries yet.</div>
      )}

      {!loading && !error && filtered.length > 0 && (
        <div className="audit-table-wrap">
          <table className="audit-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Action</th>
                <th>Actor</th>
                <th>Resource</th>
                <th>IP</th>
                <th>Detail</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(e => {
                const group = ACTION_GROUPS[e.action] ?? 'neutral'
                const hasDetail = e.detail && Object.keys(e.detail).length > 0
                const isExpanded = expandedId === e.id
                return (
                  <>
                    <tr key={e.id} className="audit-row">
                      <td className="audit-time">{fmtDate(e.created_at)}</td>
                      <td><span className={`audit-action audit-action--${group}`}>{e.action}</span></td>
                      <td className="audit-actor">{e.actor || '—'}</td>
                      <td className="audit-resource">
                        <span className="audit-resource-type">{e.resource_type}</span>
                        <span className="audit-resource-id">{uidShort(e.resource_id)}</span>
                      </td>
                      <td className="audit-ip">{e.ip_address || '—'}</td>
                      <td>
                        {hasDetail ? (
                          <button
                            type="button"
                            className="audit-detail-toggle"
                            onClick={() => setExpandedId(isExpanded ? null : e.id)}
                          >
                            {isExpanded ? 'hide' : 'show'}
                          </button>
                        ) : (
                          <span className="audit-no-detail">—</span>
                        )}
                      </td>
                    </tr>
                    {isExpanded && hasDetail && (
                      <tr key={`${e.id}-detail`} className="audit-detail-row">
                        <td colSpan={6}>
                          <pre className="audit-detail-pre">{JSON.stringify(e.detail, null, 2)}</pre>
                        </td>
                      </tr>
                    )}
                  </>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ── Defacing Review Panel ─────────────────────────────────────────────────────

const OHIF_BASE = 'http://localhost:3002'

function DefacingReviewPanel({ study, onClose }: { study: Study; onClose: () => void }) {
  const beforeUrl = `${OHIF_BASE}/viewer?StudyInstanceUIDs=${study.study_instance_uid}&dataSource=dicomweb-raw`
  const afterUrl  = `${OHIF_BASE}/viewer?StudyInstanceUIDs=${study.study_instance_uid}&dataSource=dicomweb`

  return (
    <div className="deface-panel">
      <div className="deface-panel-header">
        <div>
          <span className="deface-panel-title">Defacing Review</span>
          <span className="deface-panel-uid">{uidShort(study.study_instance_uid)}</span>
        </div>
        <button type="button" className="btn-icon" onClick={onClose} aria-label="Close review panel">×</button>
      </div>
      <p className="deface-panel-hint">
        Verify that facial features have been removed. Approve only if the right panel (defaced) shows no identifiable face.
      </p>
      <div className="deface-viewers">
        <div className="deface-viewer-col">
          <div className="deface-viewer-label deface-viewer-label--before">Before (raw)</div>
          <a href={beforeUrl} target="_blank" rel="noreferrer" className="viewer-open-tab deface-open-tab">Open ↗</a>
          <iframe src={beforeUrl} className="deface-iframe" title="Pre-defacing DICOM" allow="fullscreen" />
        </div>
        <div className="deface-viewer-col">
          <div className="deface-viewer-label deface-viewer-label--after">After (defaced)</div>
          <a href={afterUrl} target="_blank" rel="noreferrer" className="viewer-open-tab deface-open-tab">Open ↗</a>
          <iframe src={afterUrl} className="deface-iframe" title="Defaced DICOM" allow="fullscreen" />
        </div>
      </div>
    </div>
  )
}

// ── Share Panel ───────────────────────────────────────────────────────────────

function SharePanel({ study, onClose }: { study: Study; onClose: () => void }) {
  const [shares, setShares] = useState<Share[]>([])
  const [loading, setLoading] = useState(true)
  const [email, setEmail] = useState('')
  const [note, setNote] = useState('')
  const [expiryHours, setExpiryHours] = useState(168)
  const [submitting, setSubmitting] = useState(false)
  const [newShare, setNewShare] = useState<NewShareResult | null>(null)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchShares = useCallback(async () => {
    try {
      const res = await fetch(`/api/studies/${study.id}/shares`)
      const data = await res.json()
      setShares(data)
    } catch {
      // non-fatal
    } finally {
      setLoading(false)
    }
  }, [study.id])

  useEffect(() => { fetchShares() }, [fetchShares])

  const handleCreate = async () => {
    if (!email) { setError('Recipient email is required'); return }
    setError(null)
    setSubmitting(true)
    try {
      const res = await fetch(`/api/studies/${study.id}/share`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ recipient_email: email, note, expiry_hours: expiryHours }),
      })
      if (!res.ok) {
        const body = await res.json()
        throw new Error(body.error ?? 'Failed to create share')
      }
      const data = await res.json() as NewShareResult
      setNewShare(data)
      setEmail('')
      setNote('')
      fetchShares()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create share')
    } finally {
      setSubmitting(false)
    }
  }

  const handleRevoke = async (shareId: string) => {
    if (!confirm('Revoke this share? The recipient will lose access immediately.')) return
    await fetch(`/api/shares/${shareId}`, { method: 'DELETE' })
    fetchShares()
    if (newShare?.id === shareId) setNewShare(null)
  }

  const handleCopy = () => {
    if (newShare?.export_url) {
      navigator.clipboard.writeText(newShare.export_url)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div className="share-panel">
      <div className="share-panel-header">
        <h3>Export Shares — {uidShort(study.study_instance_uid)}</h3>
        <button type="button" className="btn-icon" onClick={onClose} aria-label="Close share panel">×</button>
      </div>

      {newShare && (
        <div className="share-url-box">
          <div className="share-url-label">Share created — send this URL to the recipient</div>
          <div className="share-url-row">
            <code className="share-url-code">{newShare.export_url}</code>
            <button type="button" className="btn-copy" onClick={handleCopy}>
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>
          <div className="share-url-meta">
            Expires {fmtDate(newShare.expires_at)} · Recipient: {newShare.recipient_email}
          </div>
        </div>
      )}

      <div className="form-section-label">New share</div>
      {error && <div className="form-error">{error}</div>}
      <div className="form-group">
        <input
          type="email"
          className="form-input"
          placeholder="Recipient email *"
          value={email}
          onChange={e => setEmail(e.target.value)}
        />
        <input
          type="text"
          className="form-input"
          placeholder="Note (optional)"
          value={note}
          onChange={e => setNote(e.target.value)}
        />
        <div className="form-row">
          <select
            className="form-select"
            aria-label="Share expiry"
            value={expiryHours}
            onChange={e => setExpiryHours(Number(e.target.value))}
          >
            <option value={24}>Expires in 24 hours</option>
            <option value={168}>Expires in 7 days</option>
            <option value={720}>Expires in 30 days</option>
          </select>
          <button type="button" className="btn-primary" onClick={handleCreate} disabled={submitting}>
            {submitting ? 'Creating…' : 'Create share'}
          </button>
        </div>
      </div>

      <div className="form-section-label">Active shares</div>
      {loading ? (
        <div className="shares-empty">Loading…</div>
      ) : shares.length === 0 ? (
        <div className="shares-empty">No shares yet.</div>
      ) : (
        <table className="shares-table">
          <thead>
            <tr>
              <th>Recipient</th>
              <th>Expires</th>
              <th>Status</th>
              <th>Created</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {shares.map(s => {
              const revoked = !!s.revoked_at
              const expired = !revoked && new Date(s.expires_at) < new Date()
              const shareStatus = revoked ? 'revoked' : expired ? 'expired' : 'active'
              const rowClass = shareStatus !== 'active' ? 'share-row--inactive' : ''
              const statusClass = `share-status--${shareStatus}`
              return (
                <tr key={s.id} className={rowClass}>
                  <td>{s.recipient_email}</td>
                  <td className="td-muted">{fmtDate(s.expires_at)}</td>
                  <td className={statusClass}>{shareStatus}</td>
                  <td className="td-muted">{fmtDate(s.created_at)}</td>
                  <td>
                    {shareStatus === 'active' && (
                      <button type="button" className="btn btn--revoke" onClick={() => handleRevoke(s.id)}>
                        Revoke
                      </button>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}

// ── Study Row ─────────────────────────────────────────────────────────────────

function StudyRow({ study, onAction }: { study: Study; onAction: () => void }) {
  const [shareOpen,  setShareOpen]  = useState(false)
  const [viewOpen,   setViewOpen]   = useState(false)
  const [defaceOpen, setDefaceOpen] = useState(false)

  const handleApprove = async () => {
    await fetch(`/api/studies/${study.id}/approve`, { method: 'POST' })
    onAction()
  }

  const handleReject = async () => {
    if (!confirm('Reject this study? This cannot be undone.')) return
    await fetch(`/api/studies/${study.id}/reject`, { method: 'POST' })
    onAction()
  }

  const canApprove    = !['approved', 'rejected'].includes(study.status)
  const canReject     = study.status !== 'rejected'
  const canShare      = study.status === 'approved'
  // Show "Review defacing" for head studies that have been defaced (raw files preserved).
  const canReviewDeface = study.defacing_required &&
    ['defaced', 'approved'].includes(study.status)

  return (
    <>
      <tr>
        <td className="td-uid">{uidShort(study.study_instance_uid)}</td>
        <td>{study.modality || '—'}</td>
        <td>{study.body_part || '—'}</td>
        <td><Badge label={study.source} prefix="source" /></td>
        <td><Badge label={study.status} prefix="status" /></td>
        <td className="td-num">{study.instance_count}</td>
        <td className="td-date">{fmtDate(study.created_at)}</td>
        <td>
          <div className="actions-cell">
            {canApprove && (
              <button type="button" className="btn btn--approve" onClick={handleApprove}>Approve</button>
            )}
            {canReject && (
              <button type="button" className="btn btn--reject" onClick={handleReject}>Reject</button>
            )}
            {canShare && (
              <button type="button" className="btn btn--share" onClick={() => setShareOpen(o => !o)}>
                {shareOpen ? 'Close' : 'Share'}
              </button>
            )}
            {canReviewDeface && (
              <button type="button" className="btn btn--deface" onClick={() => setDefaceOpen(o => !o)}>
                {defaceOpen ? 'Close review' : 'Review defacing'}
              </button>
            )}
            <button type="button" className="btn btn--view" onClick={() => setViewOpen(o => !o)}>
              {viewOpen ? 'Close viewer' : 'View'}
            </button>
          </div>
        </td>
      </tr>
      {shareOpen && (
        <tr>
          <td colSpan={8}>
            <SharePanel study={study} onClose={() => setShareOpen(false)} />
          </td>
        </tr>
      )}
      {defaceOpen && (
        <tr>
          <td colSpan={8}>
            <DefacingReviewPanel study={study} onClose={() => setDefaceOpen(false)} />
          </td>
        </tr>
      )}
      {viewOpen && (
        <tr>
          <td colSpan={8}>
            <ViewerPanel studyUID={study.study_instance_uid} onClose={() => setViewOpen(false)} />
          </td>
        </tr>
      )}
    </>
  )
}

// ── App ───────────────────────────────────────────────────────────────────────

type StudiesState = 'loading' | 'loaded' | 'error'

export function App() {
  const [tab, setTab] = useState<AppTab>('studies')
  const [state, setState] = useState<StudiesState>('loading')
  const [studies, setStudies] = useState<Study[]>([])
  const [error, setError] = useState<string | null>(null)

  const fetchStudies = useCallback(async () => {
    setState('loading')
    try {
      const res = await fetch('/api/studies?limit=100')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setStudies(data)
      setState('loaded')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load studies')
      setState('error')
    }
  }, [])

  useEffect(() => { fetchStudies() }, [fetchStudies])

  const pending  = studies.filter(s => ['received', 'defacing', 'clean'].includes(s.status))
  const approved = studies.filter(s => s.status === 'approved')
  const rejected = studies.filter(s => s.status === 'rejected')

  const stats = [
    { label: 'Pending review', count: pending.length,  variant: 'warning'  },
    { label: 'Approved',       count: approved.length, variant: 'success'  },
    { label: 'Rejected',       count: rejected.length, variant: 'error'    },
    { label: 'Total',          count: studies.length,  variant: 'neutral'  },
  ] as const

  return (
    <div className="admin-root">
      <header className="header">
        <div>
          <h1>AEGIS Admin Dashboard</h1>
          <p>Study review, QC, and export management</p>
        </div>
        {tab === 'studies' && (
          <button type="button" className="btn-refresh" onClick={fetchStudies}>Refresh</button>
        )}
      </header>

      {/* Tab nav */}
      <nav className="tab-nav">
        <button
          type="button"
          className={`tab-btn${tab === 'studies' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('studies')}
        >
          Studies
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'audit' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('audit')}
        >
          Audit Log
        </button>
      </nav>

      {/* Studies tab */}
      {tab === 'studies' && (
        <>
          {state === 'loaded' && (
            <div className="stats-bar">
              {stats.map(({ label, count, variant }) => (
                <div key={label} className="stat-card">
                  <div className={`stat-number stat-number--${variant}`}>{count}</div>
                  <div className="stat-label">{label}</div>
                </div>
              ))}
            </div>
          )}

          {state === 'loading' && <div className="state-loading">Loading studies…</div>}
          {state === 'error'   && <div className="state-error">{error}</div>}
          {state === 'loaded' && studies.length === 0 && (
            <div className="state-empty">No studies yet. Upload DICOM files via the Upload Portal.</div>
          )}

          {state === 'loaded' && studies.length > 0 && (
            <div className="studies-table-wrap">
              <table className="studies-table">
                <thead>
                  <tr>
                    <th>Study UID</th>
                    <th>Modality</th>
                    <th>Body Part</th>
                    <th>Source</th>
                    <th>Status</th>
                    <th className="align-right">Files</th>
                    <th>Received</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {studies.map(study => (
                    <StudyRow key={study.id} study={study} onAction={fetchStudies} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Audit log tab */}
      {tab === 'audit' && <AuditLog />}
    </div>
  )
}
