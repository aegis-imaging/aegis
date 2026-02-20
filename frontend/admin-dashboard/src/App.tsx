import { useState, useEffect, useCallback } from 'react'
import './App.css'
import { ViewerPanel } from './components/ViewerPanel'

// ── Types ────────────────────────────────────────────────────────────────────

type AppTab = 'studies' | 'audit' | 'routing' | 'institutions' | 'profiles' | 'protocol_templates' | 'notifications' | 'projects' | 'users'

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
  project_id: string
  study_instance_uid: string
  modality: string
  body_part: string
  study_description: string
  series_count: number
  source: string
  status: string
  defacing_required: boolean
  phi_scan_required: boolean
  phi_scan_status: string
  qc_required: boolean
  qc_status: string
  bids_required: boolean
  bids_status: string
  classification_required: boolean
  classification_status: string
  protocol_required: boolean
  protocol_status: string
  export_required: boolean
  export_status: string
  dicom_store: string
  instance_count: number
  created_at: string
  updated_at: string
}

type RoutingLogEntry = {
  id: string
  study_id: string
  rule_id: string
  action: string
  destination_id?: string
  outcome: string
  detail?: Record<string, unknown>
  created_at: string
}

type Share = {
  id: string
  study_id: string
  recipient_email: string
  note: string
  expires_at: string
  expires_in_seconds?: number
  // Client-only epoch anchor used for drift-resistant countdowns from server remaining seconds.
  expires_anchor_epoch_ms?: number
  revoked_at?: string
  status?: 'active' | 'expired' | 'revoked'
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

type ProtocolTemplate = {
  id: string
  project_id: string
  name: string
  description: string
  manufacturer: string
  model: string
  software_version: string
  sequence_type: string
  rules: Record<string, unknown>
  enabled: boolean
  created_at: string
}

type AuthIdentity = {
  id: string
  email: string
  name: string
  role: string
}

// ── Helpers ──────────────────────────────────────────────────────────────────

function pad2(n: number) {
  return String(n).padStart(2, '0')
}

function fmtDate(iso: string) {
  if (!iso) return ''
  const dt = new Date(iso)
  if (Number.isNaN(dt.getTime())) return iso
  const y = dt.getUTCFullYear()
  const m = pad2(dt.getUTCMonth() + 1)
  const d = pad2(dt.getUTCDate())
  const hh = pad2(dt.getUTCHours())
  const mm = pad2(dt.getUTCMinutes())
  const ss = pad2(dt.getUTCSeconds())
  return `${y}-${m}-${d} ${hh}:${mm}:${ss} UTC`
}

function fmtRemaining(seconds: number) {
  const total = Math.max(0, Math.floor(seconds))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const mins = Math.floor((total % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

function parseExpiresEpochMs(iso: string): number | null {
  const ms = Date.parse(iso)
  return Number.isFinite(ms) ? ms : null
}

function withShareExpiryAnchor<T extends Share>(share: T, nowMs = Date.now()): T {
  const anchoredFromRemaining =
    typeof share.expires_in_seconds === 'number'
      ? nowMs + (Math.max(0, Math.floor(share.expires_in_seconds)) * 1000)
      : null
  if (anchoredFromRemaining !== null) {
    return {
      ...share,
      expires_anchor_epoch_ms: anchoredFromRemaining,
    }
  }
  const parsedExpiresAt = parseExpiresEpochMs(share.expires_at)
  if (parsedExpiresAt !== null) {
    return {
      ...share,
      expires_anchor_epoch_ms: parsedExpiresAt,
    }
  }
  return share
}

function shareRemainingSeconds(share: Share, nowMs: number): number | null {
  if (typeof share.expires_anchor_epoch_ms === 'number') {
    return Math.max(0, Math.floor((share.expires_anchor_epoch_ms - nowMs) / 1000))
  }
  const expiresAtMs = parseExpiresEpochMs(share.expires_at)
  if (expiresAtMs !== null) {
    return Math.max(0, Math.floor((expiresAtMs - nowMs) / 1000))
  }
  if (typeof share.expires_in_seconds === 'number') {
    return Math.max(0, Math.floor(share.expires_in_seconds))
  }
  return null
}

function shareStatusLabel(share: Share, nowMs = Date.now()): 'active' | 'expired' | 'revoked' {
  if (share.status === 'revoked' || share.revoked_at) return 'revoked'
  if (share.status === 'expired') return 'expired'
  const remaining = shareRemainingSeconds(share, nowMs)
  if (remaining !== null && remaining <= 0) return 'expired'
  return 'active'
}

function uidShort(uid: string) {
  return uid.length > 20 ? '…' + uid.slice(-18) : uid
}

function Badge({ label, prefix }: { label: string; prefix: 'status' | 'source' | 'phi' | 'qc' | 'bids' | 'classify' | 'protocol' | 'export' }) {
  const modifier =
    (prefix === 'phi' && label === 'clean') ? 'phi-clean' :
    (prefix === 'qc' && label === 'pass') ? 'qc-pass' :
    (prefix === 'bids' && label === 'complete') ? 'bids-complete' :
    (prefix === 'classify' && label === 'classified') ? 'classify-classified' :
    (prefix === 'protocol' && label === 'compliant') ? 'protocol-compliant' :
    (prefix === 'protocol' && label === 'minor_deviations') ? 'protocol-minor_deviations' :
    (prefix === 'protocol' && label === 'non_compliant') ? 'protocol-non_compliant' :
    (prefix === 'export') ? `export-${label}` :
    label
  const cls = `badge badge--${modifier}`
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
  const [nowMs, setNowMs] = useState(() => Date.now())

  const fetchShares = useCallback(async () => {
    try {
      const res = await fetch(`/api/studies/${study.id}/shares`)
      const now = Date.now()
      const data = (await res.json()) as Share[]
      setShares(data.map(s => withShareExpiryAnchor(s, now)))
      setNowMs(now)
    } catch {
      // non-fatal
    } finally {
      setLoading(false)
    }
  }, [study.id])

  useEffect(() => { fetchShares() }, [fetchShares])

  const hasLiveCountdown = shares.some(
    s => shareStatusLabel(s, nowMs) === 'active' && shareRemainingSeconds(s, nowMs) !== null,
  )
    || (newShare !== null
      && shareStatusLabel(newShare, nowMs) === 'active'
      && shareRemainingSeconds(newShare, nowMs) !== null)
  useEffect(() => {
    if (!hasLiveCountdown) return
    const id = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)
    return () => window.clearInterval(id)
  }, [hasLiveCountdown])

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
      const now = Date.now()
      const data = await res.json() as NewShareResult
      setNewShare(withShareExpiryAnchor(data, now))
      setNowMs(now)
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

  const newShareStatus = newShare ? shareStatusLabel(newShare, nowMs) : null
  const newShareRemainingSeconds = newShare ? shareRemainingSeconds(newShare, nowMs) : null

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
            {newShareStatus === 'active' && newShareRemainingSeconds !== null && (
              <> · {fmtRemaining(newShareRemainingSeconds)} remaining</>
            )}
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
              const shareStatus = shareStatusLabel(s, nowMs)
              const rowClass = shareStatus !== 'active' ? 'share-row--inactive' : ''
              const statusClass = `share-status--${shareStatus}`
              const remainingSeconds = shareRemainingSeconds(s, nowMs)
              const remainingLabel =
                shareStatus === 'active' && remainingSeconds !== null
                  ? fmtRemaining(remainingSeconds)
                  : ''
              return (
                <tr key={s.id} className={rowClass}>
                  <td>{s.recipient_email}</td>
                  <td className="td-muted">
                    {fmtDate(s.expires_at)}
                    {remainingLabel && <div className="td-subtle">({remainingLabel} remaining)</div>}
                  </td>
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

// ── Study Detail Panel ────────────────────────────────────────────────────────

type PipelineStage = {
  label: string
  required: boolean
  status: string
}

function pipelineColorClass(status: string): string {
  if (['clean', 'pass', 'complete', 'classified', 'compliant', 'exported', 'approved', 'defaced'].includes(status)) return 'pipeline-dot--success'
  if (['warn', 'minor_deviations', 'flagged'].includes(status)) return 'pipeline-dot--warn'
  if (['fail', 'non_compliant', 'failed', 'rejected'].includes(status)) return 'pipeline-dot--error'
  if (['scanning', 'checking', 'converting', 'classifying', 'defacing', 'exporting'].includes(status)) return 'pipeline-dot--active'
  return 'pipeline-dot--pending'
}

function PipelineNode({ stage }: { stage: PipelineStage }) {
  if (!stage.required) return (
    <div className="pipeline-node pipeline-node--skip">
      <div className="pipeline-dot pipeline-dot--skip" />
      <span className="pipeline-label">{stage.label}</span>
      <span className="pipeline-status">n/a</span>
    </div>
  )
  return (
    <div className="pipeline-node">
      <div className={`pipeline-dot ${pipelineColorClass(stage.status)}`} />
      <span className="pipeline-label">{stage.label}</span>
      <span className="pipeline-status">{stage.status || 'pending'}</span>
    </div>
  )
}

function StudyDetailPanel({ studyId, onBack, onAction, isAdmin }: {
  studyId: string
  onBack: () => void
  onAction: () => void
  isAdmin: boolean
}) {
  const [study, setStudy] = useState<Study | null>(null)
  const [audit, setAudit] = useState<AuditEntry[]>([])
  const [routingLog, setRoutingLog] = useState<RoutingLogEntry[]>([])
  const [shares, setShares] = useState<Share[]>([])
  const [loading, setLoading] = useState(true)
  const [detailTab, setDetailTab] = useState<'audit' | 'routing' | 'shares'>('audit')

  // Viewer / review state
  const [viewOpen, setViewOpen] = useState(false)
  const [defaceOpen, setDefaceOpen] = useState(false)

  // Share form state
  const [shareEmail, setShareEmail] = useState('')
  const [shareNote, setShareNote] = useState('')
  const [shareDays, setShareDays] = useState(7)
  const [shareResult, setShareResult] = useState<NewShareResult | null>(null)
  const [nowMs, setNowMs] = useState(() => Date.now())

  const loadData = useCallback(() => {
    setLoading(true)
    Promise.all([
      fetch(`/api/studies/${studyId}`).then(r => r.ok ? r.json() : null),
      fetch(`/api/studies/${studyId}/audit`).then(r => r.ok ? r.json() : []),
      fetch(`/api/studies/${studyId}/routing-log`).then(r => r.ok ? r.json() : []),
      fetch(`/api/studies/${studyId}/shares`).then(r => r.ok ? r.json() : []),
    ]).then(([s, a, rl, sh]) => {
      const now = Date.now()
      setStudy(s)
      setAudit(a ?? [])
      setRoutingLog(rl ?? [])
      const shareRows = (sh ?? []) as Share[]
      setShares(shareRows.map(row => withShareExpiryAnchor(row, now)))
      setNowMs(now)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [studyId])

  useEffect(() => { loadData() }, [loadData])

  const hasLiveShareCountdown = shares.some(
    s => shareStatusLabel(s, nowMs) === 'active' && shareRemainingSeconds(s, nowMs) !== null,
  )
  useEffect(() => {
    if (!hasLiveShareCountdown) return
    const id = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)
    return () => window.clearInterval(id)
  }, [hasLiveShareCountdown])

  if (loading) return <div className="state-loading">Loading study details…</div>
  if (!study) return <div className="state-error">Study not found. <button type="button" className="btn btn--secondary" onClick={onBack}>Back</button></div>

  const stages: PipelineStage[] = [
    { label: 'Classification', required: study.classification_required, status: study.classification_status },
    { label: 'PHI Scan', required: study.phi_scan_required, status: study.phi_scan_status },
    { label: 'Protocol', required: study.protocol_required, status: study.protocol_status },
    { label: 'Defacing', required: study.defacing_required, status: study.status === 'defaced' ? 'defaced' : study.status === 'defacing' ? 'defacing' : study.defacing_required ? 'pending' : '' },
    { label: 'QC', required: study.qc_required, status: study.qc_status },
    { label: 'BIDS', required: study.bids_required, status: study.bids_status },
    { label: 'Export', required: study.export_required, status: study.export_status },
  ]

  const canApprove = !['approved', 'rejected'].includes(study.status)
  const canReject = study.status !== 'rejected'
  const canShare = study.status === 'approved'
  const canReviewDeface = study.defacing_required && ['defaced', 'approved'].includes(study.status)
  const canPhiScan = study.phi_scan_required && study.phi_scan_status === 'pending'
  const canQcCheck = study.qc_required && study.qc_status === 'pending'
  const canBidsConvert = study.bids_required && study.bids_status === 'pending'
  const canBidsDownload = study.bids_status === 'complete'
  const canClassify = study.classification_required && study.classification_status === 'pending'
  const canProtocolCheck = study.protocol_required && study.protocol_status === 'pending'

  const doAction = async (url: string) => {
    await fetch(url, { method: 'POST' })
    loadData()
    onAction()
  }

  const handleShare = async () => {
    const expiryHours = Math.max(1, shareDays * 24)
    const resp = await fetch(`/api/studies/${study.id}/share`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ recipient_email: shareEmail, note: shareNote, expiry_hours: expiryHours }),
    })
    if (resp.ok) {
      const result = await resp.json()
      setShareResult(result)
      setShareEmail('')
      setShareNote('')
      loadData()
    }
  }

  return (
    <div className="study-detail">
      <div className="study-detail__header">
        <button type="button" className="btn btn--secondary study-detail__back" onClick={onBack}>← Back to studies</button>
        <div className="study-detail__title-row">
          <h2 className="study-detail__title">{study.study_instance_uid}</h2>
          <Badge label={study.status} prefix="status" />
          <Badge label={study.source} prefix="source" />
        </div>
        {study.study_description && <p className="study-detail__description">{study.study_description}</p>}
      </div>

      {/* Meta row */}
      <div className="study-detail__meta">
        <div className="study-detail__meta-item"><strong>Modality</strong> {study.modality || '—'}</div>
        <div className="study-detail__meta-item"><strong>Body Part</strong> {study.body_part || '—'}</div>
        <div className="study-detail__meta-item"><strong>Files</strong> {study.instance_count}</div>
        <div className="study-detail__meta-item"><strong>Series</strong> {study.series_count}</div>
        <div className="study-detail__meta-item"><strong>Store</strong> {study.dicom_store || 'raw'}</div>
        <div className="study-detail__meta-item"><strong>Received</strong> {fmtDate(study.created_at)}</div>
        <div className="study-detail__meta-item"><strong>Updated</strong> {fmtDate(study.updated_at)}</div>
      </div>

      {/* Pipeline visualization */}
      <div className="study-detail__section">
        <h3 className="study-detail__section-title">Processing Pipeline</h3>
        <div className="pipeline-row">
          {stages.map((stage, i) => (
            <div key={stage.label} className="pipeline-step">
              <PipelineNode stage={stage} />
              {i < stages.length - 1 && <div className="pipeline-arrow">→</div>}
            </div>
          ))}
        </div>
      </div>

      {/* Action buttons */}
      <div className="study-detail__section">
        <h3 className="study-detail__section-title">Actions</h3>
        <div className="study-detail__actions">
          {isAdmin && canApprove && <button type="button" className="btn btn--approve" onClick={() => doAction(`/api/studies/${study.id}/approve`)}>Approve</button>}
          {isAdmin && canReject && <button type="button" className="btn btn--reject" onClick={() => { if (confirm('Reject this study?')) doAction(`/api/studies/${study.id}/reject`) }}>Reject</button>}
          {isAdmin && canClassify && <button type="button" className="btn btn--classify" onClick={() => doAction(`/api/studies/${study.study_instance_uid}/classify`)}>Classify</button>}
          {isAdmin && canPhiScan && <button type="button" className="btn btn--phi-scan" onClick={() => doAction(`/api/studies/${study.study_instance_uid}/phi-scan`)}>Scan for PHI</button>}
          {isAdmin && canProtocolCheck && <button type="button" className="btn btn--protocol-check" onClick={() => doAction(`/api/studies/${study.study_instance_uid}/protocol-check`)}>Check Protocol</button>}
          {isAdmin && canQcCheck && <button type="button" className="btn btn--qc-check" onClick={() => doAction(`/api/studies/${study.study_instance_uid}/qc-check`)}>Run QC</button>}
          {isAdmin && canBidsConvert && <button type="button" className="btn btn--bids-convert" onClick={() => doAction(`/api/studies/${study.study_instance_uid}/bids-convert`)}>Convert to BIDS</button>}
          {canBidsDownload && <a href={`/api/studies/${study.study_instance_uid}/bids-download`} className="btn btn--bids-download" download>Download BIDS</a>}
          {study.status === 'approved' && <a href={`/api/studies/${study.study_instance_uid}/dicom-download`} className="btn btn--dicom-download" download>Download DICOM</a>}
          {isAdmin && study.export_required && (study.export_status === 'pending' || study.export_status === 'failed') && study.status === 'approved' && (
            <button type="button" className="btn btn--export" onClick={() => doAction(`/api/studies/${study.study_instance_uid}/trigger-export`)}>Export</button>
          )}
          {canReviewDeface && <button type="button" className="btn btn--deface" onClick={() => setDefaceOpen(o => !o)}>{defaceOpen ? 'Close review' : 'Review defacing'}</button>}
          <button type="button" className="btn btn--view" onClick={() => setViewOpen(o => !o)}>{viewOpen ? 'Close viewer' : 'View in OHIF'}</button>
        </div>
      </div>

      {/* Inline viewer */}
      {viewOpen && <ViewerPanel studyUID={study.study_instance_uid} onClose={() => setViewOpen(false)} />}
      {defaceOpen && <DefacingReviewPanel study={study} onClose={() => setDefaceOpen(false)} />}

      {/* Share form (admin only, approved studies) */}
      {isAdmin && canShare && (
        <div className="study-detail__section">
          <h3 className="study-detail__section-title">Create Share Link</h3>
          <div className="share-inline">
            <input type="email" placeholder="Recipient email" value={shareEmail} onChange={e => setShareEmail(e.target.value)} className="share-input" />
            <input type="text" placeholder="Note (optional)" value={shareNote} onChange={e => setShareNote(e.target.value)} className="share-input" />
            <select title="Expiry period" value={shareDays} onChange={e => setShareDays(Number(e.target.value))} className="share-select">
              <option value={7}>7 days</option>
              <option value={14}>14 days</option>
              <option value={30}>30 days</option>
              <option value={90}>90 days</option>
            </select>
            <button type="button" className="btn btn--approve" disabled={!shareEmail} onClick={handleShare}>Send</button>
          </div>
          {shareResult && (
            <div className="share-result">
              Share created! Link: <code>{shareResult.export_url}</code>
            </div>
          )}
        </div>
      )}

      {/* Detail tabs: Audit / Routing / Shares */}
      <div className="study-detail__section">
        <div className="study-detail__tab-nav">
          <button type="button" className={`tab-btn${detailTab === 'audit' ? ' tab-btn--active' : ''}`} onClick={() => setDetailTab('audit')}>
            Audit Trail ({audit.length})
          </button>
          <button type="button" className={`tab-btn${detailTab === 'routing' ? ' tab-btn--active' : ''}`} onClick={() => setDetailTab('routing')}>
            Routing Log ({routingLog.length})
          </button>
          <button type="button" className={`tab-btn${detailTab === 'shares' ? ' tab-btn--active' : ''}`} onClick={() => setDetailTab('shares')}>
            Shares ({shares.length})
          </button>
        </div>

        {detailTab === 'audit' && (
          <table className="detail-table">
            <thead>
              <tr><th>Time</th><th>Action</th><th>Actor</th><th>Detail</th></tr>
            </thead>
            <tbody>
              {audit.length === 0 && <tr><td colSpan={4}>No audit entries.</td></tr>}
              {audit.map(e => (
                <tr key={e.id}>
                  <td className="td-date">{fmtDate(e.created_at)}</td>
                  <td><code>{e.action}</code></td>
                  <td>{e.actor}</td>
                  <td className="td-detail">{e.detail ? <pre className="detail-json">{JSON.stringify(e.detail, null, 2)}</pre> : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {detailTab === 'routing' && (
          <table className="detail-table">
            <thead>
              <tr><th>Time</th><th>Action</th><th>Outcome</th><th>Detail</th></tr>
            </thead>
            <tbody>
              {routingLog.length === 0 && <tr><td colSpan={4}>No routing log entries.</td></tr>}
              {routingLog.map(e => (
                <tr key={e.id}>
                  <td className="td-date">{fmtDate(e.created_at)}</td>
                  <td><code>{e.action}</code></td>
                  <td><Badge label={e.outcome} prefix="status" /></td>
                  <td className="td-detail">{e.detail ? <pre className="detail-json">{JSON.stringify(e.detail, null, 2)}</pre> : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {detailTab === 'shares' && (
          <table className="detail-table">
            <thead>
              <tr><th>Recipient</th><th>Created</th><th>Expires</th><th>Status</th><th>Note</th></tr>
            </thead>
            <tbody>
              {shares.length === 0 && <tr><td colSpan={5}>No shares.</td></tr>}
              {shares.map(s => {
                const shareStatus = shareStatusLabel(s, nowMs)
                const statusClass = `share-status--${shareStatus}`
                const remainingSeconds = shareRemainingSeconds(s, nowMs)
                const remainingLabel =
                  shareStatus === 'active' && remainingSeconds !== null
                    ? fmtRemaining(remainingSeconds)
                    : ''
                return (
                  <tr key={s.id}>
                    <td>{s.recipient_email}</td>
                    <td className="td-date">{fmtDate(s.created_at)}</td>
                    <td className="td-date">
                      {fmtDate(s.expires_at)}
                      {remainingLabel && <div className="td-subtle">({remainingLabel} remaining)</div>}
                    </td>
                    <td><span className={statusClass}>{shareStatus}</span></td>
                    <td>{s.note || '—'}</td>
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

// ── Study Row ─────────────────────────────────────────────────────────────────

function StudyRow({ study, onAction, onSelect, isAdmin }: { study: Study; onAction: () => void; onSelect: () => void; isAdmin: boolean }) {
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
  const canPhiScan = study.phi_scan_required && study.phi_scan_status === 'pending'
  const canQcCheck = study.qc_required && study.qc_status === 'pending'
  const canBidsConvert = study.bids_required && study.bids_status === 'pending'
  const canBidsDownload = study.bids_status === 'complete'
  const canClassify = study.classification_required && study.classification_status === 'pending'
  const canProtocolCheck = study.protocol_required && study.protocol_status === 'pending'

  const handlePhiScan = async () => {
    await fetch(`/api/studies/${study.study_instance_uid}/phi-scan`, { method: 'POST' })
    onAction()
  }

  const handleQcCheck = async () => {
    await fetch(`/api/studies/${study.study_instance_uid}/qc-check`, { method: 'POST' })
    onAction()
  }

  const handleBidsConvert = async () => {
    await fetch(`/api/studies/${study.study_instance_uid}/bids-convert`, { method: 'POST' })
    onAction()
  }

  const handleClassify = async () => {
    await fetch(`/api/studies/${study.study_instance_uid}/classify`, { method: 'POST' })
    onAction()
  }

  const handleProtocolCheck = async () => {
    await fetch(`/api/studies/${study.study_instance_uid}/protocol-check`, { method: 'POST' })
    onAction()
  }
  const handleTriggerExport = async () => {
    await fetch(`/api/studies/${study.study_instance_uid}/trigger-export`, { method: 'POST' })
    onAction()
  }

  return (
    <>
      <tr>
        <td className="td-uid"><button type="button" className="btn-link" onClick={onSelect} title={study.study_instance_uid}>{uidShort(study.study_instance_uid)}</button></td>
        <td>{study.modality || '—'}</td>
        <td>{study.body_part || '—'}</td>
        <td><Badge label={study.source} prefix="source" /></td>
        <td><Badge label={study.status} prefix="status" /></td>
        <td>{study.phi_scan_required ? <Badge label={study.phi_scan_status || 'n/a'} prefix="phi" /> : '—'}</td>
        <td>{study.qc_required ? <Badge label={study.qc_status || 'n/a'} prefix="qc" /> : '—'}</td>
        <td>{study.bids_required ? <Badge label={study.bids_status || 'n/a'} prefix="bids" /> : '—'}</td>
        <td>{study.classification_required ? <Badge label={study.classification_status || 'n/a'} prefix="classify" /> : '—'}</td>
        <td>{study.protocol_required ? <Badge label={study.protocol_status || 'n/a'} prefix="protocol" /> : '—'}</td>
        <td>{study.export_required ? <Badge label={study.export_status || 'n/a'} prefix="export" /> : '—'}</td>
        <td className="td-num">{study.instance_count}</td>
        <td className="td-date">{fmtDate(study.created_at)}</td>
        <td>
          <div className="actions-cell">
            {isAdmin && canApprove && (
              <button type="button" className="btn btn--approve" onClick={handleApprove}>Approve</button>
            )}
            {isAdmin && canReject && (
              <button type="button" className="btn btn--reject" onClick={handleReject}>Reject</button>
            )}
            {isAdmin && canShare && (
              <button type="button" className="btn btn--share" onClick={() => setShareOpen(o => !o)}>
                {shareOpen ? 'Close' : 'Share'}
              </button>
            )}
            {isAdmin && canPhiScan && (
              <button type="button" className="btn btn--phi-scan" onClick={handlePhiScan}>Scan for PHI</button>
            )}
            {isAdmin && canQcCheck && (
              <button type="button" className="btn btn--qc-check" onClick={handleQcCheck}>Run QC</button>
            )}
            {isAdmin && canBidsConvert && (
              <button type="button" className="btn btn--bids-convert" onClick={handleBidsConvert}>Convert to BIDS</button>
            )}
            {canBidsDownload && (
              <a href={`/api/studies/${study.study_instance_uid}/bids-download`} className="btn btn--bids-download" download>Download BIDS</a>
            )}
            {study.status === 'approved' && (
              <a href={`/api/studies/${study.study_instance_uid}/dicom-download`} className="btn btn--dicom-download" download>Download DICOM</a>
            )}
            {isAdmin && canClassify && (
              <button type="button" className="btn btn--classify" onClick={handleClassify}>Classify</button>
            )}
            {isAdmin && canProtocolCheck && (
              <button type="button" className="btn btn--protocol-check" onClick={handleProtocolCheck}>Check Protocol</button>
            )}
            {isAdmin && study.export_required && (study.export_status === 'pending' || study.export_status === 'failed') && study.status === 'approved' && (
              <button type="button" className="btn btn--export" onClick={handleTriggerExport}>Export</button>
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
          <td colSpan={14}>
            <SharePanel study={study} onClose={() => setShareOpen(false)} />
          </td>
        </tr>
      )}
      {defaceOpen && (
        <tr>
          <td colSpan={14}>
            <DefacingReviewPanel study={study} onClose={() => setDefaceOpen(false)} />
          </td>
        </tr>
      )}
      {viewOpen && (
        <tr>
          <td colSpan={14}>
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

function RoutingPanel({ isAdmin }: { isAdmin: boolean }) {
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
          {isAdmin && <button type="button" className="btn-primary" onClick={openNewDest}>+ Add destination</button>}
        </div>
        <p className="routing-hint">External DICOM endpoints that studies can be forwarded to via <code>route_to</code> rules.</p>

        {isAdmin && showDestForm && (
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
                <option value="dimse">DIMSE (C-STORE)</option>
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
                    {isAdmin && (
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEditDest(d)}>Edit</button>
                        <button type="button" className="btn btn--revoke" onClick={() => deleteDest(d.id, d.name)}>Delete</button>
                      </div>
                    )}
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
          {isAdmin && <button type="button" className="btn-primary" onClick={openNewRule}>+ Add rule</button>}
        </div>
        <p className="routing-hint">
          Rules are evaluated in <strong>priority order</strong> (lower = first) on every study ingest.
          All matching rules fire — not just the first.
        </p>

        {isAdmin && showRuleForm && (
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
                <option value="require_phi_scan">require_phi_scan — scan pixels for burned-in PHI</option>
                <option value="require_qc_check">require_qc_check — automated image quality checks</option>
                <option value="require_bids_conversion">require_bids_conversion — convert to NIfTI/BIDS</option>
                <option value="require_classification">require_classification — classify modality/body part</option>
                <option value="require_protocol_check">require_protocol_check — check protocol compliance</option>
                <option value="require_export">require_export — auto-forward to destinations on approval</option>
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
                      {isAdmin && (
                        <div className="actions-cell">
                          <button type="button" className="btn btn--edit" onClick={() => openEditRule(r)}>Edit</button>
                          <button type="button" className="btn btn--secondary" onClick={() => toggleRule(r)}>
                            {r.enabled ? 'Disable' : 'Enable'}
                          </button>
                          <button type="button" className="btn btn--revoke" onClick={() => deleteRule(r.id, r.name)}>Delete</button>
                        </div>
                      )}
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

function ProfilesPanel({ isAdmin }: { isAdmin: boolean }) {
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
          {isAdmin && <button type="button" className="btn-primary" onClick={openCreate}>+ New profile</button>}
        </div>

        {isAdmin && showForm && (
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
                      {isAdmin && (
                        <div className="actions-cell">
                          <button type="button" className="btn btn--edit" onClick={() => openEdit(p)}>Edit</button>
                          <button type="button" className="btn btn--secondary"
                            onClick={() => setDefault(p.project_id, p.id, p.name)}>
                            {isDefault ? 'Clear default' : 'Set default'}
                          </button>
                          <button type="button" className="btn btn--revoke" onClick={() => del(p.id, p.name)}>Delete</button>
                        </div>
                      )}
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

// ── Protocol Templates Panel ─────────────────────────────────────────────────

type ProtocolTemplateForm = {
  name: string
  description: string
  manufacturer: string
  model: string
  software_version: string
  sequence_type: string
  rules: string
  enabled: boolean
}

const EMPTY_TEMPLATE_FORM: ProtocolTemplateForm = {
  name: '', description: '', manufacturer: '', model: '',
  software_version: '', sequence_type: '', rules: '{}', enabled: true,
}

function ProtocolTemplatesPanel({ isAdmin }: { isAdmin: boolean }) {
  const [projects, setProjects]     = useState<Project[]>([])
  const [templates, setTemplates]   = useState<ProtocolTemplate[]>([])
  const [loading, setLoading]       = useState(true)
  const [error, setError]           = useState<string | null>(null)

  const [form, setForm]             = useState<ProtocolTemplateForm>(EMPTY_TEMPLATE_FORM)
  const [formProject, setFormProject] = useState('')
  const [editingId, setEditingId]   = useState<string | null>(null)
  const [showForm, setShowForm]     = useState(false)
  const [saving, setSaving]         = useState(false)
  const [formError, setFormError]   = useState<string | null>(null)

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const projRes = await fetch('/api/projects')
      if (!projRes.ok) throw new Error('Failed to load projects')
      const projs: Project[] = await projRes.json()
      setProjects(projs ?? [])

      // Fetch templates for all projects in parallel
      const allTemplates: ProtocolTemplate[] = []
      await Promise.all((projs ?? []).map(async p => {
        const res = await fetch(`/api/projects/${p.id}/protocol-templates`)
        if (res.ok) {
          const list: ProtocolTemplate[] = await res.json()
          allTemplates.push(...list)
        }
      }))
      setTemplates(allTemplates)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  function openCreate() {
    setForm(EMPTY_TEMPLATE_FORM)
    setFormProject(projects[0]?.id ?? '')
    setEditingId(null)
    setFormError(null)
    setShowForm(true)
  }

  function openEdit(t: ProtocolTemplate) {
    setForm({
      name: t.name, description: t.description, manufacturer: t.manufacturer,
      model: t.model, software_version: t.software_version, sequence_type: t.sequence_type,
      rules: JSON.stringify(t.rules ?? {}, null, 2), enabled: t.enabled,
    })
    setFormProject(t.project_id)
    setEditingId(t.id)
    setFormError(null)
    setShowForm(true)
  }

  async function save() {
    if (!form.name) { setFormError('Name is required'); return }
    if (!formProject) { setFormError('Project is required'); return }
    let parsedRules: Record<string, unknown>
    try {
      parsedRules = JSON.parse(form.rules)
    } catch {
      setFormError('Rules must be valid JSON'); return
    }
    setSaving(true)
    setFormError(null)
    try {
      const url = editingId ? `/api/protocol-templates/${editingId}` : `/api/projects/${formProject}/protocol-templates`
      const method = editingId ? 'PUT' : 'POST'
      const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: form.name, description: form.description, manufacturer: form.manufacturer,
          model: form.model, software_version: form.software_version, sequence_type: form.sequence_type,
          rules: parsedRules, enabled: form.enabled,
        }),
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
    if (!confirm(`Delete protocol template "${name}"?`)) return
    await fetch(`/api/protocol-templates/${id}`, { method: 'DELETE' })
    fetchAll()
  }

  const projectName = (id: string) => projects.find(p => p.id === id)?.name ?? id

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Protocol Templates</div>
            <div className="routing-section-sub">
              Define expected acquisition parameters per manufacturer/model/sequence.
              Studies are checked against matching templates when protocol compliance is required.
            </div>
          </div>
          {isAdmin && <button type="button" className="btn-primary" onClick={openCreate}>+ New template</button>}
        </div>

        {isAdmin && showForm && (
          <div className="routing-form">
            <h3>{editingId ? 'Edit template' : 'New template'}</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" placeholder="Template name *"
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
            </div>
            <div className="routing-form-section-label">Scanner Match Criteria</div>
            <div className="form-grid">
              <input className="form-input" placeholder="Manufacturer (e.g. Siemens)"
                value={form.manufacturer}
                onChange={e => setForm(f => ({ ...f, manufacturer: e.target.value }))} />
              <input className="form-input" placeholder="Model (e.g. Prisma)"
                value={form.model}
                onChange={e => setForm(f => ({ ...f, model: e.target.value }))} />
              <input className="form-input" placeholder="Software version (e.g. syngo MR E11)"
                value={form.software_version}
                onChange={e => setForm(f => ({ ...f, software_version: e.target.value }))} />
              <input className="form-input" placeholder="Sequence type (e.g. T1w, FLAIR, DWI)"
                value={form.sequence_type}
                onChange={e => setForm(f => ({ ...f, sequence_type: e.target.value }))} />
            </div>
            <div className="routing-form-section-label">Compliance Rules (JSON)</div>
            <div className="form-grid">
              <textarea className="form-input form-input--wide" rows={6}
                placeholder='{"SliceThickness":{"min":0.5,"max":1.5},"RepetitionTime":{"min":1900,"max":2200}}'
                value={form.rules}
                onChange={e => setForm(f => ({ ...f, rules: e.target.value }))} />
            </div>
            <label className="form-checkbox">
              <input type="checkbox" checked={form.enabled}
                onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))} />
              {' '}Enabled
            </label>
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
        {!loading && !error && templates.length === 0 && (
          <div className="state-empty">No protocol templates yet. Create one to define expected acquisition parameters.</div>
        )}
        {!loading && !error && templates.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Project</th>
                <th>Manufacturer</th>
                <th>Model</th>
                <th>Software</th>
                <th>Sequence</th>
                <th>Enabled</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {templates.map(t => (
                <tr key={t.id} className={t.enabled ? '' : 'routing-row--disabled'}>
                  <td>
                    <div className="routing-name">{t.name}</div>
                    {t.description && <div className="routing-desc">{t.description}</div>}
                  </td>
                  <td>{projectName(t.project_id)}</td>
                  <td>{t.manufacturer || '—'}</td>
                  <td>{t.model || '—'}</td>
                  <td>{t.software_version || '—'}</td>
                  <td>{t.sequence_type || '—'}</td>
                  <td>{t.enabled ? 'Yes' : 'No'}</td>
                  <td>
                    {isAdmin && (
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEdit(t)}>Edit</button>
                        <button type="button" className="btn btn--revoke" onClick={() => del(t.id, t.name)}>Delete</button>
                      </div>
                    )}
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

// ── Notifications Panel ───────────────────────────────────────────────────────

function NotificationsPanel({ isAdmin }: { isAdmin: boolean }) {
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
          {isAdmin && <button type="button" className="btn-primary" onClick={openCreate}>+ New subscription</button>}
        </div>

        {isAdmin && showForm && (
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
                    {isAdmin && (
                      <div className="actions-cell">
                        <button type="button" className="btn btn--revoke"
                          onClick={() => del(sub.id, sub.email)}>Remove</button>
                      </div>
                    )}
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

function InstitutionsPanel({ isAdmin }: { isAdmin: boolean }) {
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
          {isAdmin && <button type="button" className="btn-primary" onClick={openNew}>+ Add institution</button>}
        </div>
      </div>
      <p className="routing-hint">
        Organisations that send or receive studies. Link each institution to one or more projects with a role.
      </p>

      {/* Create / Edit form */}
      {isAdmin && showForm && (
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
                    {isAdmin && <button type="button" className="btn btn--edit" onClick={() => openEdit(inst)}>Edit</button>}
                    {isAdmin && (
                      <button type="button" className="btn btn--secondary" onClick={() => toggleInst(inst)}>
                        {inst.enabled ? 'Disable' : 'Enable'}
                      </button>
                    )}
                    {isAdmin && <button type="button" className="btn btn--revoke" onClick={() => deleteInst(inst.id, inst.name)}>Delete</button>}
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
          {isAdmin && (
            <>
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
            </>
          )}

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
                      {isAdmin && (
                        <button type="button" className="btn btn--revoke" onClick={() => unlinkProject(ip.project_id)}>
                          Unlink
                        </button>
                      )}
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

// ── Projects Panel ────────────────────────────────────────────────────────────

const EMPTY_PROJECT: Omit<Project, 'id' | 'default_anon_profile_id' | 'created_at'> = {
  name: '', slug: '', description: '',
}

function ProjectsPanel({ isAdmin }: { isAdmin: boolean }) {
  const [projects, setProjects]   = useState<Project[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)

  const [form, setForm]           = useState<Omit<Project, 'id' | 'default_anon_profile_id' | 'created_at'>>(EMPTY_PROJECT)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [showForm, setShowForm]   = useState(false)
  const [saving, setSaving]       = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fetchProjects = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/projects')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setProjects(await res.json())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchProjects() }, [fetchProjects])

  function openNew() {
    setForm(EMPTY_PROJECT)
    setEditingId(null)
    setFormError(null)
    setShowForm(true)
  }

  function openEdit(p: Project) {
    setForm({ name: p.name, slug: p.slug, description: p.description })
    setEditingId(p.id)
    setFormError(null)
    setShowForm(true)
  }

  async function save() {
    if (!form.name) { setFormError('Name is required'); return }
    setSaving(true)
    setFormError(null)
    try {
      const url = editingId ? `/api/projects/${editingId}` : '/api/projects'
      const method = editingId ? 'PUT' : 'POST'
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(form) })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowForm(false)
      setEditingId(null)
      fetchProjects()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="state-loading">Loading projects…</div>
  if (error)   return <div className="state-error">{error}</div>

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Projects</div>
            <div className="routing-section-sub">
              Projects group studies and control anonymization profiles. Each upload is associated with one project.
            </div>
          </div>
          <div className="actions-cell">
            <button type="button" className="btn-refresh" onClick={fetchProjects}>Refresh</button>
            {isAdmin && <button type="button" className="btn-primary" onClick={openNew}>+ New project</button>}
          </div>
        </div>

        {isAdmin && showForm && (
          <div className="routing-form">
            <h3>{editingId ? 'Edit project' : 'New project'}</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" placeholder="Name *"
                value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              <input className="form-input" placeholder="Slug (auto-generated if blank)"
                value={form.slug} onChange={e => setForm(f => ({ ...f, slug: e.target.value }))} />
              <input className="form-input form-input--wide" placeholder="Description"
                value={form.description} onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
            </div>
            {editingId && (
              <div className="routing-hint">Note: changing the slug will break existing upload portal URLs for this project.</div>
            )}
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={save} disabled={saving}>
                {saving ? 'Saving…' : editingId ? 'Save changes' : 'Create'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {projects.length === 0 && !showForm ? (
          <div className="state-empty">No projects yet.</div>
        ) : projects.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Project</th>
                <th>Slug</th>
                <th>Default profile</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {projects.map(p => (
                <tr key={p.id}>
                  <td>
                    <div className="routing-name">{p.name}</div>
                    {p.description && <div className="routing-desc">{p.description}</div>}
                  </td>
                  <td><code className="inst-slug">{p.slug}</code></td>
                  <td>
                    {p.default_anon_profile_id
                      ? <span className="badge badge--enabled">profile set</span>
                      : <span className="routing-desc">none</span>}
                  </td>
                  <td className="td-date">{fmtDate(p.created_at)}</td>
                  <td>
                    {isAdmin && (
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEdit(p)}>Edit</button>
                      </div>
                    )}
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

const PAGE_SIZE = 50

export function App() {
  const [tab, setTab] = useState<AppTab>('studies')
  const [state, setState] = useState<StudiesState>('loading')
  const [studies, setStudies] = useState<Study[]>([])
  const [studiesTotal, setStudiesTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [selectedStudyId, setSelectedStudyId] = useState<string | null>(null)

  // Auth state
  const [currentUser, setCurrentUser] = useState<AuthIdentity | null>(null)
  const [authError, setAuthError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/api/auth/me')
      .then(r => {
        if (r.status === 401 || r.status === 403) {
          return r.json().then(body => { setAuthError(body.error || 'Unauthorized') })
        }
        if (r.ok) return r.json().then(setCurrentUser)
      })
      .catch(() => { /* non-fatal — dev mode may not have auth */ })
  }, [])

  const isAdmin = currentUser?.role === 'admin'

  // Filters
  const [filterStatus,   setFilterStatus]   = useState('')
  const [filterModality, setFilterModality] = useState('')
  const [filterSource,   setFilterSource]   = useState('')
  const [filterProject,  setFilterProject]  = useState('')
  const [filterSearch,   setFilterSearch]   = useState('')
  const [page, setPage] = useState(0)
  const [refreshTick, setRefreshTick] = useState(0)

  // Projects for filter dropdown
  const [projects, setProjects] = useState<Project[]>([])
  useEffect(() => {
    fetch('/api/projects').then(r => r.json()).then(setProjects).catch(() => {})
  }, [])

  // Fetch studies whenever filters, page, or refresh tick change
  useEffect(() => {
    let cancelled = false
    setState('loading')
    const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(page * PAGE_SIZE) })
    if (filterStatus)   params.set('status',     filterStatus)
    if (filterModality) params.set('modality',   filterModality)
    if (filterSource)   params.set('source',     filterSource)
    if (filterProject)  params.set('project_id', filterProject)
    if (filterSearch)   params.set('search',     filterSearch)

    fetch(`/api/studies?${params}`)
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json() })
      .then(data => {
        if (cancelled) return
        setStudies(data.studies ?? [])
        setStudiesTotal(data.total ?? 0)
        setState('loaded')
      })
      .catch(err => {
        if (cancelled) return
        setError(err instanceof Error ? err.message : 'Failed to load studies')
        setState('error')
      })
    return () => { cancelled = true }
  }, [page, filterStatus, filterModality, filterSource, filterProject, filterSearch, refreshTick])

  // Filter change helpers — also reset page to 0
  function setStatusF(v: string)   { setFilterStatus(v);   setPage(0) }
  function setModalityF(v: string) { setFilterModality(v); setPage(0) }
  function setSourceF(v: string)   { setFilterSource(v);   setPage(0) }
  function setProjectF(v: string)  { setFilterProject(v);  setPage(0) }
  function setSearchF(v: string)   { setFilterSearch(v);   setPage(0) }

  const hasFilters = !!(filterStatus || filterModality || filterSource || filterProject || filterSearch)

  function clearFilters() {
    setFilterStatus(''); setFilterModality(''); setFilterSource('')
    setFilterProject(''); setFilterSearch(''); setPage(0)
  }

  const totalPages = Math.max(1, Math.ceil(studiesTotal / PAGE_SIZE))
  const pageStart  = studiesTotal === 0 ? 0 : page * PAGE_SIZE + 1
  const pageEnd    = Math.min((page + 1) * PAGE_SIZE, studiesTotal)

  return (
    <div className="admin-root">
      {authError && (
        <div className="auth-error-banner">
          Access denied: {authError}
        </div>
      )}

      <header className="header">
        <div>
          <h1>AEGIS Admin Dashboard</h1>
          <p>Study review, QC, and export management</p>
        </div>
        <div className="header-actions">
          {currentUser && (
            <span className="auth-user-badge">
              {currentUser.name || currentUser.email} ({currentUser.role})
            </span>
          )}
          {tab === 'studies' && (
            <button type="button" className="btn-refresh" onClick={() => setRefreshTick(t => t + 1)}>Refresh</button>
          )}
        </div>
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
          className={`tab-btn${tab === 'protocol_templates' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('protocol_templates')}
        >
          Protocol Templates
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
          className={`tab-btn${tab === 'projects' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('projects')}
        >
          Projects
        </button>
        {isAdmin && (
          <button
            type="button"
            className={`tab-btn${tab === 'users' ? ' tab-btn--active' : ''}`}
            onClick={() => setTab('users')}
          >
            Users
          </button>
        )}
      </nav>

      {/* Studies tab */}
      {tab === 'studies' && selectedStudyId && (
        <StudyDetailPanel
          studyId={selectedStudyId}
          onBack={() => setSelectedStudyId(null)}
          onAction={() => setRefreshTick(t => t + 1)}
          isAdmin={isAdmin}
        />
      )}
      {tab === 'studies' && !selectedStudyId && (
        <>
          {/* Filter bar */}
          <div className="filter-bar">
            <input
              className="filter-input filter-input--search"
              type="search"
              placeholder="Search UID or description…"
              value={filterSearch}
              onChange={e => setSearchF(e.target.value)}
            />
            <select className="filter-select" title="Filter by status" value={filterStatus} onChange={e => setStatusF(e.target.value)}>
              <option value="">All statuses</option>
              <option value="received">Received</option>
              <option value="defacing">Defacing</option>
              <option value="clean">Clean</option>
              <option value="defaced">Defaced</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
            </select>
            <select className="filter-select" title="Filter by modality" value={filterModality} onChange={e => setModalityF(e.target.value)}>
              <option value="">All modalities</option>
              <option value="MRI">MRI</option>
              <option value="CT">CT</option>
              <option value="PET">PET</option>
              <option value="US">US</option>
              <option value="CR">CR</option>
              <option value="DX">DX</option>
              <option value="NM">NM</option>
              <option value="PT">PT</option>
            </select>
            <select className="filter-select" title="Filter by source" value={filterSource} onChange={e => setSourceF(e.target.value)}>
              <option value="">All sources</option>
              <option value="external">External</option>
              <option value="internal">Internal</option>
            </select>
            <select className="filter-select" title="Filter by project" value={filterProject} onChange={e => setProjectF(e.target.value)}>
              <option value="">All projects</option>
              {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
            </select>
            {hasFilters && (
              <button type="button" className="btn btn--secondary" onClick={clearFilters}>Clear</button>
            )}
            {state === 'loaded' && (
              <span className="filter-count">
                {studiesTotal === 0
                  ? 'No results'
                  : hasFilters
                    ? `${studiesTotal} matching`
                    : `${studiesTotal} total`}
              </span>
            )}
          </div>

          {state === 'loading' && <div className="state-loading">Loading studies…</div>}
          {state === 'error'   && <div className="state-error">{error}</div>}
          {state === 'loaded' && studiesTotal === 0 && (
            <div className="state-empty">
              {hasFilters
                ? 'No studies match your filters.'
                : 'No studies yet. Upload DICOM files via the Upload Portal.'}
            </div>
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
                    <th>PHI Scan</th>
                    <th>QC</th>
                    <th>BIDS</th>
                    <th>Class.</th>
                    <th>Protocol</th>
                    <th>Export</th>
                    <th className="align-right">Files</th>
                    <th>Received</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {studies.map(study => (
                    <StudyRow key={study.id} study={study} onAction={() => setRefreshTick(t => t + 1)} onSelect={() => setSelectedStudyId(study.id)} isAdmin={isAdmin} />
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Pagination */}
          {state === 'loaded' && studiesTotal > PAGE_SIZE && (
            <div className="pagination">
              <button
                type="button"
                className="btn btn--secondary"
                disabled={page === 0}
                onClick={() => setPage(p => p - 1)}
              >
                ← Previous
              </button>
              <span className="pagination-info">
                {pageStart}–{pageEnd} of {studiesTotal}
              </span>
              <button
                type="button"
                className="btn btn--secondary"
                disabled={page >= totalPages - 1}
                onClick={() => setPage(p => p + 1)}
              >
                Next →
              </button>
            </div>
          )}
        </>
      )}

      {/* Audit log tab */}
      {tab === 'audit' && <AuditLog />}

      {/* Routing tab */}
      {tab === 'routing' && <RoutingPanel isAdmin={isAdmin} />}

      {/* Institutions tab */}
      {tab === 'institutions' && <InstitutionsPanel isAdmin={isAdmin} />}

      {/* Profiles tab */}
      {tab === 'profiles' && <ProfilesPanel isAdmin={isAdmin} />}

      {/* Protocol Templates tab */}
      {tab === 'protocol_templates' && <ProtocolTemplatesPanel isAdmin={isAdmin} />}

      {/* Notifications tab */}
      {tab === 'notifications' && <NotificationsPanel isAdmin={isAdmin} />}

      {/* Projects tab */}
      {tab === 'projects' && <ProjectsPanel isAdmin={isAdmin} />}

      {/* Users tab — admin only */}
      {tab === 'users' && isAdmin && <UsersPanel />}
    </div>
  )
}
