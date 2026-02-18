import { useState, useEffect, useCallback } from 'react'
import './App.css'
import { ViewerPanel } from './components/ViewerPanel'

// ── Types ────────────────────────────────────────────────────────────────────

type AppTab = 'studies' | 'audit' | 'routing' | 'institutions' | 'profiles' | 'notifications' | 'users'

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

type Destination = {
  id: string
  name: string
  slug: string
  description: string
  type: 'dicomweb' | 'dimse'
  dicomweb_url: string
  ae_title: string
  host: string
  port: number
  enabled: boolean
  created_at: string
}

type RoutingRule = {
  id: string
  name: string
  description: string
  priority: number
  enabled: boolean
  project_id: string | null
  modality: string | null
  body_part: string | null
  source: string | null
  action: string
  destination_id: string | null
  created_at: string
}

type Institution = {
  id: string
  name: string
  slug: string
  description: string
  institution_type: 'sender' | 'receiver' | 'both'
  contact_name: string
  contact_email: string
  ip_ranges: string
  ae_title: string
  enabled: boolean
  created_at: string
}

type InstitutionProject = {
  institution_id: string
  project_id: string
  role: 'sender' | 'receiver' | 'admin'
  institution_name?: string
  project_name?: string
  created_at: string
}

type Project = {
  id: string
  name: string
  slug: string
  description: string
  default_anon_profile_id?: string | null
  created_at: string
}

type AnonProfile = {
  id: string
  project_id: string
  name: string
  description: string
  retained_tags: string[]
  enabled: boolean
  created_at: string
}

type DigestSubscription = {
  id: string
  email: string
  project_id: string
  project_name: string
  frequency: 'weekly' | 'monthly'
  enabled: boolean
  last_sent_at?: string | null
  created_at: string
}

type AdminUser = {
  id: string
  email: string
  name: string
  role: 'admin' | 'viewer'
  enabled: boolean
  notes: string
  created_at: string
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

// ── Routing Panel ─────────────────────────────────────────────────────────────

const EMPTY_DEST: Omit<Destination, 'id' | 'created_at'> = {
  name: '', slug: '', description: '', type: 'dicomweb',
  dicomweb_url: '', ae_title: '', host: '', port: 0, enabled: true,
}

const EMPTY_RULE: Omit<RoutingRule, 'id' | 'created_at'> = {
  name: '', description: '', priority: 100, enabled: true,
  project_id: null, modality: null, body_part: null, source: null,
  action: 'require_qa', destination_id: null,
}

function RoutingPanel() {
  const [destinations, setDestinations] = useState<Destination[]>([])
  const [rules, setRules]               = useState<RoutingRule[]>([])
  const [loading, setLoading]           = useState(true)
  const [error, setError]               = useState<string | null>(null)

  // Destination form
  const [destForm, setDestForm]         = useState<Omit<Destination, 'id' | 'created_at'>>(EMPTY_DEST)
  const [editingDestId, setEditingDestId] = useState<string | null>(null)
  const [showDestForm, setShowDestForm] = useState(false)
  const [destSaving, setDestSaving]     = useState(false)
  const [destError, setDestError]       = useState<string | null>(null)

  // Rule form
  const [ruleForm, setRuleForm]         = useState<Omit<RoutingRule, 'id' | 'created_at'>>(EMPTY_RULE)
  const [editingRuleId, setEditingRuleId] = useState<string | null>(null)
  const [showRuleForm, setShowRuleForm] = useState(false)
  const [ruleSaving, setRuleSaving]     = useState(false)
  const [ruleError, setRuleError]       = useState<string | null>(null)

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [destRes, ruleRes] = await Promise.all([
        fetch('/api/destinations'),
        fetch('/api/routing-rules'),
      ])
      if (!destRes.ok || !ruleRes.ok) throw new Error('Failed to load routing data')
      const [dests, ruleList] = await Promise.all([destRes.json(), ruleRes.json()])
      setDestinations(dests)
      setRules(ruleList)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  // ── Destination CRUD ────────────────────────────────────────────────────────

  function openNewDest() {
    setDestForm(EMPTY_DEST)
    setEditingDestId(null)
    setDestError(null)
    setShowDestForm(true)
  }

  function openEditDest(d: Destination) {
    setDestForm({ name: d.name, slug: d.slug, description: d.description, type: d.type,
      dicomweb_url: d.dicomweb_url, ae_title: d.ae_title, host: d.host, port: d.port, enabled: d.enabled })
    setEditingDestId(d.id)
    setDestError(null)
    setShowDestForm(true)
  }

  async function saveDest() {
    if (!destForm.name || !destForm.type) { setDestError('Name and type are required'); return }
    setDestSaving(true)
    setDestError(null)
    try {
      const url = editingDestId ? `/api/destinations/${editingDestId}` : '/api/destinations'
      const method = editingDestId ? 'PUT' : 'POST'
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(destForm) })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowDestForm(false)
      setEditingDestId(null)
      fetchAll()
    } catch (err) {
      setDestError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setDestSaving(false)
    }
  }

  async function deleteDest(id: string, name: string) {
    if (!confirm(`Delete destination "${name}"? Rules using it will lose their target.`)) return
    await fetch(`/api/destinations/${id}`, { method: 'DELETE' })
    fetchAll()
  }

  // ── Routing Rule CRUD ───────────────────────────────────────────────────────

  function openNewRule() {
    setRuleForm(EMPTY_RULE)
    setEditingRuleId(null)
    setRuleError(null)
    setShowRuleForm(true)
  }

  function openEditRule(r: RoutingRule) {
    setRuleForm({ name: r.name, description: r.description, priority: r.priority, enabled: r.enabled,
      project_id: r.project_id, modality: r.modality, body_part: r.body_part, source: r.source,
      action: r.action, destination_id: r.destination_id })
    setEditingRuleId(r.id)
    setRuleError(null)
    setShowRuleForm(true)
  }

  async function saveRule() {
    if (!ruleForm.name) { setRuleError('Name is required'); return }
    if (ruleForm.action === 'route_to' && !ruleForm.destination_id) { setRuleError('Destination required for route_to action'); return }
    setRuleSaving(true)
    setRuleError(null)
    const body = {
      ...ruleForm,
      modality:   ruleForm.modality  || null,
      body_part:  ruleForm.body_part || null,
      source:     ruleForm.source    || null,
      destination_id: ruleForm.action === 'route_to' ? ruleForm.destination_id : null,
    }
    try {
      const url = editingRuleId ? `/api/routing-rules/${editingRuleId}` : '/api/routing-rules'
      const method = editingRuleId ? 'PUT' : 'POST'
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowRuleForm(false)
      setEditingRuleId(null)
      fetchAll()
    } catch (err) {
      setRuleError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setRuleSaving(false)
    }
  }

  async function deleteRule(id: string, name: string) {
    if (!confirm(`Delete rule "${name}"?`)) return
    await fetch(`/api/routing-rules/${id}`, { method: 'DELETE' })
    fetchAll()
  }

  async function toggleRule(r: RoutingRule) {
    await fetch(`/api/routing-rules/${r.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...r, enabled: !r.enabled }),
    })
    fetchAll()
  }

  // ── Render ──────────────────────────────────────────────────────────────────

  if (loading) return <div className="state-loading">Loading routing configuration…</div>
  if (error)   return <div className="state-error">{error}</div>

  return (
    <div className="routing-panel">

      {/* ── Destinations ── */}
      <section className="routing-section">
        <div className="routing-section-header">
          <h2>Destinations</h2>
          <button type="button" className="btn-primary" onClick={openNewDest}>+ Add destination</button>
        </div>
        <p className="routing-hint">External DICOM endpoints that studies can be forwarded to via <code>route_to</code> rules.</p>

        {showDestForm && (
          <div className="routing-form">
            <h3>{editingDestId ? 'Edit destination' : 'New destination'}</h3>
            {destError && <div className="form-error">{destError}</div>}
            <div className="form-grid">
              <input className="form-input" placeholder="Name *" value={destForm.name}
                onChange={e => setDestForm(f => ({ ...f, name: e.target.value }))} />
              <input className="form-input" placeholder="Slug (auto-generated if blank)" value={destForm.slug}
                onChange={e => setDestForm(f => ({ ...f, slug: e.target.value }))} />
              <select className="form-select" aria-label="Type" value={destForm.type}
                onChange={e => setDestForm(f => ({ ...f, type: e.target.value as 'dicomweb' | 'dimse' }))}>
                <option value="dicomweb">DICOMweb (STOW-RS)</option>
                <option value="dimse">DIMSE (future)</option>
              </select>
              <input className="form-input" placeholder="Description" value={destForm.description}
                onChange={e => setDestForm(f => ({ ...f, description: e.target.value }))} />
              {destForm.type === 'dicomweb' && (
                <input className="form-input form-input--wide" placeholder="DICOMweb base URL *" value={destForm.dicomweb_url}
                  onChange={e => setDestForm(f => ({ ...f, dicomweb_url: e.target.value }))} />
              )}
              {destForm.type === 'dimse' && (
                <>
                  <input className="form-input" placeholder="AE Title" value={destForm.ae_title}
                    onChange={e => setDestForm(f => ({ ...f, ae_title: e.target.value }))} />
                  <input className="form-input" placeholder="Host" value={destForm.host}
                    onChange={e => setDestForm(f => ({ ...f, host: e.target.value }))} />
                  <input className="form-input" placeholder="Port" type="number" value={destForm.port || ''}
                    onChange={e => setDestForm(f => ({ ...f, port: Number(e.target.value) }))} />
                </>
              )}
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={saveDest} disabled={destSaving}>
                {destSaving ? 'Saving…' : editingDestId ? 'Save changes' : 'Create'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowDestForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {destinations.length === 0 && !showDestForm ? (
          <div className="state-empty">No destinations yet.</div>
        ) : destinations.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Target</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {destinations.map(d => (
                <tr key={d.id} className={d.enabled ? '' : 'routing-row--disabled'}>
                  <td>
                    <div className="routing-name">{d.name}</div>
                    {d.description && <div className="routing-desc">{d.description}</div>}
                  </td>
                  <td><code>{d.type}</code></td>
                  <td className="routing-target">
                    {d.type === 'dicomweb' ? (d.dicomweb_url || '—') : `${d.ae_title}@${d.host}:${d.port}`}
                  </td>
                  <td>
                    <span className={`badge badge--${d.enabled ? 'enabled' : 'disabled'}`}>
                      {d.enabled ? 'enabled' : 'disabled'}
                    </span>
                  </td>
                  <td>
                    <div className="actions-cell">
                      <button type="button" className="btn btn--edit" onClick={() => openEditDest(d)}>Edit</button>
                      <button type="button" className="btn btn--revoke" onClick={() => deleteDest(d.id, d.name)}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* ── Routing Rules ── */}
      <section className="routing-section">
        <div className="routing-section-header">
          <h2>Routing Rules</h2>
          <button type="button" className="btn-primary" onClick={openNewRule}>+ Add rule</button>
        </div>
        <p className="routing-hint">
          Rules are evaluated in <strong>priority order</strong> (lower = first) on every study ingest.
          All matching rules fire — not just the first.
        </p>

        {showRuleForm && (
          <div className="routing-form">
            <h3>{editingRuleId ? 'Edit rule' : 'New rule'}</h3>
            {ruleError && <div className="form-error">{ruleError}</div>}
            <div className="form-grid">
              <input className="form-input" placeholder="Rule name *" value={ruleForm.name}
                onChange={e => setRuleForm(f => ({ ...f, name: e.target.value }))} />
              <input className="form-input" placeholder="Description" value={ruleForm.description}
                onChange={e => setRuleForm(f => ({ ...f, description: e.target.value }))} />
              <input className="form-input" placeholder="Priority (default 100)" type="number" value={ruleForm.priority}
                onChange={e => setRuleForm(f => ({ ...f, priority: Number(e.target.value) }))} />
            </div>
            <div className="routing-form-section-label">Conditions (leave blank = match any)</div>
            <div className="form-grid">
              <input className="form-input" placeholder="Modality (e.g. MRI, CT, PET)" value={ruleForm.modality ?? ''}
                onChange={e => setRuleForm(f => ({ ...f, modality: e.target.value || null }))} />
              <input className="form-input" placeholder="Body part (e.g. HEAD, CHEST)" value={ruleForm.body_part ?? ''}
                onChange={e => setRuleForm(f => ({ ...f, body_part: e.target.value || null }))} />
              <select className="form-select" aria-label="Source filter" value={ruleForm.source ?? ''}
                onChange={e => setRuleForm(f => ({ ...f, source: e.target.value || null }))}>
                <option value="">Any source</option>
                <option value="external">external</option>
                <option value="internal">internal</option>
              </select>
            </div>
            <div className="routing-form-section-label">Action</div>
            <div className="form-grid">
              <select className="form-select" aria-label="Action" value={ruleForm.action}
                onChange={e => setRuleForm(f => ({ ...f, action: e.target.value, destination_id: null }))}>
                <option value="require_qa">require_qa — hold for manual review (default)</option>
                <option value="require_defacing">require_defacing — force defacing even if not head</option>
                <option value="auto_approve">auto_approve — skip QC, approve immediately</option>
                <option value="reject">reject — auto-reject</option>
                <option value="route_to">route_to — forward to external destination</option>
              </select>
              {ruleForm.action === 'route_to' && (
                <select className="form-select" aria-label="Destination" value={ruleForm.destination_id ?? ''}
                  onChange={e => setRuleForm(f => ({ ...f, destination_id: e.target.value || null }))}>
                  <option value="">Select destination…</option>
                  {destinations.map(d => (
                    <option key={d.id} value={d.id}>{d.name} ({d.type})</option>
                  ))}
                </select>
              )}
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={saveRule} disabled={ruleSaving}>
                {ruleSaving ? 'Saving…' : editingRuleId ? 'Save changes' : 'Create'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowRuleForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {rules.length === 0 && !showRuleForm ? (
          <div className="state-empty">No routing rules yet. Studies follow the default pipeline.</div>
        ) : rules.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Priority</th>
                <th>Rule</th>
                <th>Conditions</th>
                <th>Action</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rules.map(r => {
                const destName = r.destination_id
                  ? (destinations.find(d => d.id === r.destination_id)?.name ?? r.destination_id)
                  : null
                const conditions = [
                  r.modality  ? `modality=${r.modality}`   : null,
                  r.body_part ? `body_part=${r.body_part}` : null,
                  r.source    ? `source=${r.source}`       : null,
                ].filter(Boolean)
                return (
                  <tr key={r.id} className={r.enabled ? '' : 'routing-row--disabled'}>
                    <td className="routing-priority">{r.priority}</td>
                    <td>
                      <div className="routing-name">{r.name}</div>
                      {r.description && <div className="routing-desc">{r.description}</div>}
                    </td>
                    <td className="routing-conditions">
                      {conditions.length > 0
                        ? conditions.map(c => <code key={c} className="routing-condition-tag">{c}</code>)
                        : <span className="td-muted">any</span>}
                    </td>
                    <td>
                      <code className={`routing-action routing-action--${r.action}`}>{r.action}</code>
                      {destName && <div className="routing-desc">→ {destName}</div>}
                    </td>
                    <td>
                      <span className={`badge badge--${r.enabled ? 'enabled' : 'disabled'}`}>
                        {r.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td>
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEditRule(r)}>Edit</button>
                        <button type="button" className="btn btn--secondary" onClick={() => toggleRule(r)}>
                          {r.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button type="button" className="btn btn--revoke" onClick={() => deleteRule(r.id, r.name)}>Delete</button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </section>
    </div>
  )
}

// ── Profiles Panel ────────────────────────────────────────────────────────────

const EMPTY_PROFILE: Omit<AnonProfile, 'id' | 'project_id' | 'created_at'> = {
  name: '', description: '', retained_tags: [], enabled: true,
}

function ProfilesPanel() {
  const [projects, setProjects]   = useState<Project[]>([])
  const [profiles, setProfiles]   = useState<AnonProfile[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)

  const [form, setForm]           = useState<Omit<AnonProfile, 'id' | 'project_id' | 'created_at'>>(EMPTY_PROFILE)
  const [formProject, setFormProject] = useState('')
  const [tagsInput, setTagsInput] = useState('')  // comma-separated tags string in form
  const [editingId, setEditingId] = useState<string | null>(null)
  const [showForm, setShowForm]   = useState(false)
  const [saving, setSaving]       = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const projRes = await fetch('/api/projects')
      if (!projRes.ok) throw new Error('Failed to load projects')
      const projs: Project[] = await projRes.json()
      setProjects(projs ?? [])

      // Fetch profiles for all projects in parallel
      const allProfiles: AnonProfile[] = []
      await Promise.all((projs ?? []).map(async p => {
        const res = await fetch(`/api/projects/${p.id}/anon-profiles`)
        if (res.ok) {
          const list: AnonProfile[] = await res.json()
          allProfiles.push(...list)
        }
      }))
      setProfiles(allProfiles)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  function openCreate() {
    setForm(EMPTY_PROFILE)
    setFormProject(projects[0]?.id ?? '')
    setTagsInput('')
    setEditingId(null)
    setFormError(null)
    setShowForm(true)
  }

  function openEdit(p: AnonProfile) {
    setForm({ name: p.name, description: p.description, retained_tags: p.retained_tags, enabled: p.enabled })
    setFormProject(p.project_id)
    setTagsInput((p.retained_tags ?? []).join(', '))
    setEditingId(p.id)
    setFormError(null)
    setShowForm(true)
  }

  async function save() {
    if (!form.name) { setFormError('Name is required'); return }
    if (!formProject) { setFormError('Project is required'); return }
    const tags = tagsInput.split(',').map(t => t.trim()).filter(Boolean)
    setSaving(true)
    setFormError(null)
    try {
      const url = editingId ? `/api/anon-profiles/${editingId}` : `/api/projects/${formProject}/anon-profiles`
      const method = editingId ? 'PUT' : 'POST'
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, retained_tags: tags }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowForm(false)
      setEditingId(null)
      fetchAll()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function del(id: string, name: string) {
    if (!confirm(`Delete profile "${name}"?`)) return
    await fetch(`/api/anon-profiles/${id}`, { method: 'DELETE' })
    fetchAll()
  }

  async function setDefault(projectID: string, profileID: string, profileName: string) {
    const proj = projects.find(p => p.id === projectID)
    const isAlready = proj?.default_anon_profile_id === profileID
    if (isAlready) {
      // Clear default
      if (!confirm(`Clear default profile for this project?`)) return
      await fetch(`/api/projects/${projectID}/default-anon-profile`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profile_id: '' }),
      })
    } else {
      if (!confirm(`Set "${profileName}" as the default profile for this project?`)) return
      await fetch(`/api/projects/${projectID}/default-anon-profile`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profile_id: profileID }),
      })
    }
    fetchAll()
  }

  const projectName = (id: string) => projects.find(p => p.id === id)?.name ?? id

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Anonymization Profiles</div>
            <div className="routing-section-sub">
              Named DICOM tag retention overrides applied during upload de-identification.
              The project's default profile is automatically used by the upload portal.
            </div>
          </div>
          <button type="button" className="btn-primary" onClick={openCreate}>+ New profile</button>
        </div>

        {showForm && (
          <div className="routing-form">
            <h3>{editingId ? 'Edit profile' : 'New profile'}</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" placeholder="Profile name *"
                value={form.name}
                onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              {!editingId && (
                <select className="form-select" aria-label="Project"
                  value={formProject}
                  onChange={e => setFormProject(e.target.value)}>
                  {projects.map(p => (
                    <option key={p.id} value={p.id}>{p.name}</option>
                  ))}
                </select>
              )}
              <input className="form-input form-input--wide" placeholder="Description"
                value={form.description}
                onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
              <input className="form-input form-input--wide"
                placeholder="Retained tags — comma-separated DICOM keywords (e.g. PatientAge, StudyDate)"
                value={tagsInput}
                onChange={e => setTagsInput(e.target.value)} />
              <label className="form-checkbox">
                <input type="checkbox" checked={form.enabled}
                  onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))} />
                {' '}Enabled
              </label>
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={save} disabled={saving}>
                {saving ? 'Saving…' : editingId ? 'Save changes' : 'Create'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {loading && <div className="state-loading">Loading…</div>}
        {error   && <div className="state-error">{error}</div>}
        {!loading && !error && profiles.length === 0 && (
          <div className="state-empty">No profiles yet. Create one to override tag retention per project.</div>
        )}
        {!loading && !error && profiles.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Project</th>
                <th>Tags retained</th>
                <th>Enabled</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {profiles.map(p => {
                const proj = projects.find(pr => pr.id === p.project_id)
                const isDefault = proj?.default_anon_profile_id === p.id
                return (
                  <tr key={p.id} className={p.enabled ? '' : 'routing-row--disabled'}>
                    <td>
                      <div className="routing-name">{p.name}</div>
                      {p.description && <div className="routing-desc">{p.description}</div>}
                      {isDefault && <span className="badge badge--status-approved">default</span>}
                    </td>
                    <td>{projectName(p.project_id)}</td>
                    <td>
                      {p.retained_tags?.length > 0
                        ? <span title={p.retained_tags.join(', ')}>{p.retained_tags.length} tag{p.retained_tags.length !== 1 ? 's' : ''}</span>
                        : <span className="routing-desc">none (full strip)</span>
                      }
                    </td>
                    <td>{p.enabled ? 'Yes' : 'No'}</td>
                    <td>
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEdit(p)}>Edit</button>
                        <button type="button" className="btn btn--secondary"
                          onClick={() => setDefault(p.project_id, p.id, p.name)}>
                          {isDefault ? 'Clear default' : 'Set default'}
                        </button>
                        <button type="button" className="btn btn--revoke" onClick={() => del(p.id, p.name)}>Delete</button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

// ── Notifications Panel ───────────────────────────────────────────────────────

function NotificationsPanel() {
  const [projects, setProjects]   = useState<Project[]>([])
  const [subs, setSubs]           = useState<DigestSubscription[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)

  const [formEmail, setFormEmail]         = useState('')
  const [formProject, setFormProject]     = useState('')
  const [formFrequency, setFormFrequency] = useState<'weekly' | 'monthly'>('weekly')
  const [showForm, setShowForm]           = useState(false)
  const [saving, setSaving]               = useState(false)
  const [formError, setFormError]         = useState<string | null>(null)

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [projRes, subRes] = await Promise.all([
        fetch('/api/projects'),
        fetch('/api/digest-subscriptions'),
      ])
      if (!projRes.ok || !subRes.ok) throw new Error('Failed to load data')
      const [projs, subList] = await Promise.all([projRes.json(), subRes.json()])
      setProjects(projs ?? [])
      setSubs(subList ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  function openCreate() {
    setFormEmail('')
    setFormProject(projects[0]?.id ?? '')
    setFormFrequency('weekly')
    setFormError(null)
    setShowForm(true)
  }

  async function save() {
    if (!formEmail) { setFormError('Email is required'); return }
    if (!formProject) { setFormError('Project is required'); return }
    setSaving(true)
    setFormError(null)
    try {
      const res = await fetch(`/api/projects/${formProject}/digest-subscriptions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: formEmail, frequency: formFrequency }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowForm(false)
      fetchAll()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function del(id: string, email: string) {
    if (!confirm(`Remove digest subscription for ${email}?`)) return
    await fetch(`/api/digest-subscriptions/${id}`, { method: 'DELETE' })
    fetchAll()
  }

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Email Digest Subscriptions</div>
            <div className="routing-section-sub">
              Periodic study summary emails sent per project. Weekly digests send every 7 days;
              monthly every 30 days. No PHI is included.
            </div>
          </div>
          <button type="button" className="btn-primary" onClick={openCreate}>+ New subscription</button>
        </div>

        {showForm && (
          <div className="routing-form">
            <h3>New subscription</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" type="email" placeholder="Email address *"
                value={formEmail}
                onChange={e => setFormEmail(e.target.value)} />
              <select className="form-select" aria-label="Project"
                value={formProject}
                onChange={e => setFormProject(e.target.value)}>
                {projects.map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
              <select className="form-select" aria-label="Frequency"
                value={formFrequency}
                onChange={e => setFormFrequency(e.target.value as 'weekly' | 'monthly')}>
                <option value="weekly">Weekly</option>
                <option value="monthly">Monthly</option>
              </select>
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={save} disabled={saving}>
                {saving ? 'Saving…' : 'Subscribe'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {loading && <div className="state-loading">Loading…</div>}
        {error   && <div className="state-error">{error}</div>}
        {!loading && !error && subs.length === 0 && (
          <div className="state-empty">No subscriptions yet.</div>
        )}
        {!loading && !error && subs.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Email</th>
                <th>Project</th>
                <th>Frequency</th>
                <th>Last sent</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {subs.map(sub => (
                <tr key={sub.id}>
                  <td>{sub.email}</td>
                  <td>{sub.project_name}</td>
                  <td className="text-capitalize">{sub.frequency}</td>
                  <td>{sub.last_sent_at ? fmtDate(sub.last_sent_at) : <span className="routing-desc">never</span>}</td>
                  <td>
                    <div className="actions-cell">
                      <button type="button" className="btn btn--revoke"
                        onClick={() => del(sub.id, sub.email)}>Remove</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

// ── Institutions Panel ────────────────────────────────────────────────────────

const EMPTY_INSTITUTION: Omit<Institution, 'id' | 'created_at'> = {
  name: '', slug: '', description: '', institution_type: 'sender',
  contact_name: '', contact_email: '', ip_ranges: '', ae_title: '', enabled: true,
}

function InstitutionsPanel() {
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [loading, setLoading]           = useState(true)
  const [error, setError]               = useState<string | null>(null)

  // Selected institution for project-link detail
  const [selectedInst, setSelectedInst]   = useState<Institution | null>(null)
  const [instProjects, setInstProjects]   = useState<InstitutionProject[]>([])
  const [projLoading, setProjLoading]     = useState(false)

  // Form
  const [form, setForm]               = useState<Omit<Institution, 'id' | 'created_at'>>(EMPTY_INSTITUTION)
  const [editingId, setEditingId]     = useState<string | null>(null)
  const [showForm, setShowForm]       = useState(false)
  const [saving, setSaving]           = useState(false)
  const [formError, setFormError]     = useState<string | null>(null)

  // Link-to-project form
  const [linkProjectID, setLinkProjectID] = useState('')
  const [linkRole, setLinkRole]           = useState<'sender' | 'receiver' | 'admin'>('sender')
  const [linkSaving, setLinkSaving]       = useState(false)
  const [linkError, setLinkError]         = useState<string | null>(null)

  const fetchInstitutions = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/institutions')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setInstitutions(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchInstitutions() }, [fetchInstitutions])

  const fetchInstProjects = useCallback(async (instId: string) => {
    setProjLoading(true)
    try {
      const res = await fetch(`/api/institutions/${instId}/projects`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setInstProjects(await res.json())
    } catch {
      setInstProjects([])
    } finally {
      setProjLoading(false)
    }
  }, [])

  function openNew() {
    setForm(EMPTY_INSTITUTION)
    setEditingId(null)
    setFormError(null)
    setShowForm(true)
    setSelectedInst(null)
  }

  function openEdit(inst: Institution) {
    setForm({ name: inst.name, slug: inst.slug, description: inst.description,
      institution_type: inst.institution_type, contact_name: inst.contact_name,
      contact_email: inst.contact_email, ip_ranges: inst.ip_ranges,
      ae_title: inst.ae_title, enabled: inst.enabled })
    setEditingId(inst.id)
    setFormError(null)
    setShowForm(true)
    setSelectedInst(null)
  }

  async function save() {
    if (!form.name) { setFormError('Name is required'); return }
    setSaving(true)
    setFormError(null)
    try {
      const url = editingId ? `/api/institutions/${editingId}` : '/api/institutions'
      const method = editingId ? 'PUT' : 'POST'
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(form) })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowForm(false)
      setEditingId(null)
      fetchInstitutions()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function deleteInst(id: string, name: string) {
    if (!confirm(`Delete institution "${name}"? This cannot be undone.`)) return
    await fetch(`/api/institutions/${id}`, { method: 'DELETE' })
    if (selectedInst?.id === id) setSelectedInst(null)
    fetchInstitutions()
  }

  async function toggleInst(inst: Institution) {
    await fetch(`/api/institutions/${inst.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...inst, enabled: !inst.enabled }),
    })
    fetchInstitutions()
  }

  function selectInst(inst: Institution) {
    setSelectedInst(inst)
    setShowForm(false)
    setLinkError(null)
    fetchInstProjects(inst.id)
  }

  async function linkProject() {
    if (!selectedInst || !linkProjectID) { setLinkError('Project ID is required'); return }
    setLinkSaving(true)
    setLinkError(null)
    try {
      const res = await fetch(`/api/institutions/${selectedInst.id}/projects`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project_id: linkProjectID, role: linkRole }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Link failed') }
      setLinkProjectID('')
      fetchInstProjects(selectedInst.id)
    } catch (err) {
      setLinkError(err instanceof Error ? err.message : 'Link failed')
    } finally {
      setLinkSaving(false)
    }
  }

  async function unlinkProject(projectID: string) {
    if (!selectedInst) return
    if (!confirm('Remove this project link?')) return
    await fetch(`/api/institutions/${selectedInst.id}/projects/${projectID}`, { method: 'DELETE' })
    fetchInstProjects(selectedInst.id)
  }

  if (loading) return <div className="state-loading">Loading institutions…</div>
  if (error)   return <div className="state-error">{error}</div>

  return (
    <div className="institutions-panel">
      <div className="routing-section-header">
        <h2>Institutions</h2>
        <div className="actions-cell">
          <button type="button" className="btn-refresh" onClick={fetchInstitutions}>Refresh</button>
          <button type="button" className="btn-primary" onClick={openNew}>+ Add institution</button>
        </div>
      </div>
      <p className="routing-hint">
        Organisations that send or receive studies. Link each institution to one or more projects with a role.
      </p>

      {/* Create / Edit form */}
      {showForm && (
        <div className="routing-form">
          <h3>{editingId ? 'Edit institution' : 'New institution'}</h3>
          {formError && <div className="form-error">{formError}</div>}
          <div className="form-grid">
            <input className="form-input" placeholder="Name *" value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
            <input className="form-input" placeholder="Slug (auto-generated)" value={form.slug}
              onChange={e => setForm(f => ({ ...f, slug: e.target.value }))} />
            <select className="form-select" aria-label="Type" value={form.institution_type}
              onChange={e => setForm(f => ({ ...f, institution_type: e.target.value as Institution['institution_type'] }))}>
              <option value="sender">Sender (uploads studies)</option>
              <option value="receiver">Receiver (receives studies)</option>
              <option value="both">Both</option>
            </select>
            <input className="form-input form-input--wide" placeholder="Description" value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
          </div>
          <div className="routing-form-section-label">Contact</div>
          <div className="form-grid">
            <input className="form-input" placeholder="Contact name" value={form.contact_name}
              onChange={e => setForm(f => ({ ...f, contact_name: e.target.value }))} />
            <input className="form-input" type="email" placeholder="Contact email" value={form.contact_email}
              onChange={e => setForm(f => ({ ...f, contact_email: e.target.value }))} />
          </div>
          <div className="routing-form-section-label">Network Identity</div>
          <div className="form-grid">
            <input className="form-input" placeholder="IP ranges (CIDR, comma-separated)" value={form.ip_ranges}
              onChange={e => setForm(f => ({ ...f, ip_ranges: e.target.value }))} />
            <input className="form-input" placeholder="DICOM AE title" value={form.ae_title}
              onChange={e => setForm(f => ({ ...f, ae_title: e.target.value }))} />
          </div>
          <div className="form-row form-row--actions">
            <button type="button" className="btn-primary" onClick={save} disabled={saving}>
              {saving ? 'Saving…' : editingId ? 'Save changes' : 'Create'}
            </button>
            <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
          </div>
        </div>
      )}

      {/* Institutions table */}
      {institutions.length === 0 && !showForm ? (
        <div className="state-empty">No institutions yet.</div>
      ) : institutions.length > 0 && (
        <table className="routing-table">
          <thead>
            <tr>
              <th>Institution</th>
              <th>Type</th>
              <th>Contact</th>
              <th>Network</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {institutions.map(inst => (
              <tr
                key={inst.id}
                className={[
                  inst.enabled ? '' : 'routing-row--disabled',
                  selectedInst?.id === inst.id ? 'inst-row--selected' : '',
                ].join(' ')}
              >
                <td>
                  <div className="routing-name">{inst.name}</div>
                  {inst.description && <div className="routing-desc">{inst.description}</div>}
                  <code className="inst-slug">{inst.slug}</code>
                </td>
                <td>
                  <span className={`routing-action routing-action--${inst.institution_type}`}>
                    {inst.institution_type}
                  </span>
                </td>
                <td>
                  {inst.contact_name && <div>{inst.contact_name}</div>}
                  {inst.contact_email && <div className="routing-desc">{inst.contact_email}</div>}
                </td>
                <td className="routing-desc">
                  {inst.ip_ranges && <div>{inst.ip_ranges}</div>}
                  {inst.ae_title  && <div>AE: {inst.ae_title}</div>}
                </td>
                <td>
                  <span className={`badge badge--${inst.enabled ? 'enabled' : 'disabled'}`}>
                    {inst.enabled ? 'enabled' : 'disabled'}
                  </span>
                </td>
                <td>
                  <div className="actions-cell">
                    <button type="button" className="btn btn--share"
                      onClick={() => selectedInst?.id === inst.id ? setSelectedInst(null) : selectInst(inst)}>
                      {selectedInst?.id === inst.id ? 'Close' : 'Projects'}
                    </button>
                    <button type="button" className="btn btn--edit" onClick={() => openEdit(inst)}>Edit</button>
                    <button type="button" className="btn btn--secondary" onClick={() => toggleInst(inst)}>
                      {inst.enabled ? 'Disable' : 'Enable'}
                    </button>
                    <button type="button" className="btn btn--revoke" onClick={() => deleteInst(inst.id, inst.name)}>Delete</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* Project links for selected institution */}
      {selectedInst && (
        <div className="inst-projects-panel">
          <h3>Projects — {selectedInst.name}</h3>

          {/* Link form */}
          <div className="routing-form-section-label">Link to project</div>
          {linkError && <div className="form-error">{linkError}</div>}
          <div className="form-row">
            <input className="form-input" placeholder="Project ID (UUID)"
              value={linkProjectID} onChange={e => setLinkProjectID(e.target.value)} />
            <select className="form-select" aria-label="Role" value={linkRole}
              onChange={e => setLinkRole(e.target.value as 'sender' | 'receiver' | 'admin')}>
              <option value="sender">sender</option>
              <option value="receiver">receiver</option>
              <option value="admin">admin</option>
            </select>
            <button type="button" className="btn-primary" onClick={linkProject} disabled={linkSaving}>
              {linkSaving ? 'Linking…' : 'Link'}
            </button>
          </div>

          {projLoading ? (
            <div className="state-loading">Loading…</div>
          ) : instProjects.length === 0 ? (
            <div className="state-empty">No projects linked yet.</div>
          ) : (
            <table className="routing-table">
              <thead>
                <tr>
                  <th>Project</th>
                  <th>Role</th>
                  <th>Linked</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {instProjects.map(ip => (
                  <tr key={ip.project_id}>
                    <td>
                      <div className="routing-name">{ip.project_name || ip.project_id}</div>
                    </td>
                    <td><code className={`routing-action routing-action--${ip.role}`}>{ip.role}</code></td>
                    <td className="td-date">{fmtDate(ip.created_at)}</td>
                    <td>
                      <button type="button" className="btn btn--revoke" onClick={() => unlinkProject(ip.project_id)}>
                        Unlink
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  )
}

// ── Users Panel ───────────────────────────────────────────────────────────────

const EMPTY_USER: Omit<AdminUser, 'id' | 'created_at'> = {
  email: '', name: '', role: 'admin', enabled: true, notes: '',
}

function UsersPanel() {
  const [users, setUsers]         = useState<AdminUser[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)

  const [form, setForm]           = useState<Omit<AdminUser, 'id' | 'created_at'>>(EMPTY_USER)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [showForm, setShowForm]   = useState(false)
  const [saving, setSaving]       = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/admin-users')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setUsers(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchUsers() }, [fetchUsers])

  function openNew() {
    setForm(EMPTY_USER)
    setEditingId(null)
    setFormError(null)
    setShowForm(true)
  }

  function openEdit(u: AdminUser) {
    setForm({ email: u.email, name: u.name, role: u.role, enabled: u.enabled, notes: u.notes })
    setEditingId(u.id)
    setFormError(null)
    setShowForm(true)
  }

  async function save() {
    if (!form.email) { setFormError('Email is required'); return }
    setSaving(true)
    setFormError(null)
    try {
      const url = editingId ? `/api/admin-users/${editingId}` : '/api/admin-users'
      const method = editingId ? 'PUT' : 'POST'
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(form) })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowForm(false)
      setEditingId(null)
      fetchUsers()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  async function deleteUser(id: string, email: string) {
    if (!confirm(`Delete user "${email}"? This cannot be undone.`)) return
    await fetch(`/api/admin-users/${id}`, { method: 'DELETE' })
    fetchUsers()
  }

  async function toggleUser(u: AdminUser) {
    await fetch(`/api/admin-users/${u.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...u, enabled: !u.enabled }),
    })
    fetchUsers()
  }

  if (loading) return <div className="state-loading">Loading users…</div>
  if (error)   return <div className="state-error">{error}</div>

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Admin Users</div>
            <div className="routing-section-sub">
              Authorised dashboard users and their roles. Authentication is handled by GCP IAP in production.
            </div>
          </div>
          <div className="actions-cell">
            <button type="button" className="btn-refresh" onClick={fetchUsers}>Refresh</button>
            <button type="button" className="btn-primary" onClick={openNew}>+ Add user</button>
          </div>
        </div>

        {showForm && (
          <div className="routing-form">
            <h3>{editingId ? 'Edit user' : 'New user'}</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" type="email" placeholder="Email address *"
                value={form.email} onChange={e => setForm(f => ({ ...f, email: e.target.value }))} />
              <input className="form-input" placeholder="Display name"
                value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              <select className="form-select" aria-label="Role" value={form.role}
                onChange={e => setForm(f => ({ ...f, role: e.target.value as 'admin' | 'viewer' }))}>
                <option value="admin">admin — full access</option>
                <option value="viewer">viewer — read-only</option>
              </select>
              <input className="form-input form-input--wide" placeholder="Notes (optional)"
                value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} />
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={save} disabled={saving}>
                {saving ? 'Saving…' : editingId ? 'Save changes' : 'Create'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {users.length === 0 && !showForm ? (
          <div className="state-empty">No users yet.</div>
        ) : users.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>User</th>
                <th>Role</th>
                <th>Status</th>
                <th>Notes</th>
                <th>Added</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id} className={u.enabled ? '' : 'routing-row--disabled'}>
                  <td>
                    <div className="routing-name">{u.name || u.email}</div>
                    {u.name && <div className="routing-desc">{u.email}</div>}
                  </td>
                  <td>
                    <span className={`routing-action routing-action--${u.role}`}>{u.role}</span>
                  </td>
                  <td>
                    <span className={`badge badge--${u.enabled ? 'enabled' : 'disabled'}`}>
                      {u.enabled ? 'enabled' : 'disabled'}
                    </span>
                  </td>
                  <td className="routing-desc">{u.notes || '—'}</td>
                  <td className="td-date">{fmtDate(u.created_at)}</td>
                  <td>
                    <div className="actions-cell">
                      <button type="button" className="btn btn--edit" onClick={() => openEdit(u)}>Edit</button>
                      <button type="button" className="btn btn--secondary" onClick={() => toggleUser(u)}>
                        {u.enabled ? 'Disable' : 'Enable'}
                      </button>
                      <button type="button" className="btn btn--revoke" onClick={() => deleteUser(u.id, u.email)}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
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
        <button
          type="button"
          className={`tab-btn${tab === 'routing' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('routing')}
        >
          Routing
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'institutions' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('institutions')}
        >
          Institutions
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'profiles' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('profiles')}
        >
          Profiles
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'notifications' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('notifications')}
        >
          Notifications
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'users' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('users')}
        >
          Users
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

      {/* Routing tab */}
      {tab === 'routing' && <RoutingPanel />}

      {/* Institutions tab */}
      {tab === 'institutions' && <InstitutionsPanel />}

      {/* Profiles tab */}
      {tab === 'profiles' && <ProfilesPanel />}

      {/* Notifications tab */}
      {tab === 'notifications' && <NotificationsPanel />}

      {/* Users tab */}
      {tab === 'users' && <UsersPanel />}
    </div>
  )
}
