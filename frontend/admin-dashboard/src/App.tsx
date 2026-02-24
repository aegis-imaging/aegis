import { useState, useEffect, useCallback, useRef } from 'react'
import './App.css'
import { AgentPanel } from './components/AgentPanel'
import { ViewerPanel } from './components/ViewerPanel'
import { TCIAPanel } from './components/TCIAPanel'
import { SynthPanel } from './components/SynthPanel'
import { SystemHealthPanel } from './components/SystemHealthPanel'
import { ComplianceReportPanel } from './components/ComplianceReportPanel'
import { useStudyEvents } from './hooks/useStudyEvents'

// ── Types ────────────────────────────────────────────────────────────────────

type AppTab = 'studies' | 'agent' | 'audit' | 'shares' | 'routing' | 'dimse_ops' | 'institutions' | 'profiles' | 'protocol_templates' | 'notifications' | 'projects' | 'federation' | 'tcia_import' | 'users' | 'api_keys' | 'invite_codes' | 'system'

type APIKey = {
  id: string
  name: string
  key_prefix: string
  created_by: string
  enabled: boolean
  last_used_at: string | null
  expires_at: string | null
  created_at: string
  updated_at: string
}

type StudyLabel = {
  id: string
  study_id: string
  label: string
  created_by: string
  created_at: string
}

type SeriesRow = {
  id: string
  study_id: string
  series_instance_uid: string
  series_description: string
  modality: string
  body_part: string
  instance_count: number
  created_at: string
}

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
  study_size_bytes: number
  priority_flag: boolean
  deface_qa_score?: number
  subject_id?: string
  rejection_reason?: string
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
  max_downloads?: number
  download_count?: number
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
  retention_days?: number | null
  stuck_threshold_minutes?: number | null
  archived?: boolean
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

type WebhookSubscription = {
  id: string
  project_id: string | null
  url: string
  events: string[]
  secret: string
  enabled: boolean
  created_at: string
  updated_at: string
}

type WebhookDelivery = {
  id: string
  subscription_id: string
  event: string
  url: string
  attempt: number
  status_code?: number | null
  success: boolean
  error_message?: string | null
  delivered_at: string
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

type DimseRetrySnapshot = {
  pending: number
  dead_letter: number
  queued_total: number
  deduped_total: number
  retried_total: number
  retried_ok_total: number
  dead_letter_total: number
  dead_letter_deduped_total: number
  replayed_total: number
  cleared_dead_letter_total: number
  cleared_pending_total: number
  pending_oldest_age_seconds: number
  dead_letter_oldest_age_seconds: number
  pending_next_attempt_at: number
  pending_next_attempt_in_seconds: number
}

type DimseRetrySummary = {
  snapshot: DimseRetrySnapshot
  now: number
  pending_due_now: number
  dead_letter_present: boolean
  queue_max: number
  queue_utilization_percent: number
}

type DimseRetryPendingItem = {
  study_instance_uid: string
  attempts: number
  next_attempt_at: number
  seconds_until_next_attempt: number
  queued_at: number
  age_seconds: number
  file_count: number
  series_count: number
  last_error: string
}

type DimseRetryDeadLetterItem = {
  study_instance_uid: string
  attempts: number
  queued_at: number
  dead_lettered_at: number
  dead_letter_age_seconds: number
  file_count: number
  series_count: number
  last_error: string
}

type DimseRetryDetails = {
  snapshot: DimseRetrySnapshot
  now: number
  limit: number
  study_instance_uid: string
  sort: string
  pending_total: number
  dead_letter_total: number
  pending_returned: number
  dead_letter_returned: number
  pending_truncated: boolean
  dead_letter_truncated: boolean
  pending_items: DimseRetryPendingItem[]
  dead_letter_items: DimseRetryDeadLetterItem[]
}

type DimseOperatorAction = {
  action: string
  created_at: string
  detail?: Record<string, unknown>
}

type DimseRetryAlert = {
  timestamp: number
  condition: string
  message: string
  snapshot?: Record<string, number>
}

type StudyDiagnosticsSummary = {
  terminal: boolean
  stuck: boolean
  blockers: string[]
  recommended_actions: string[]
  last_audit_action?: string
  last_audit_at?: string
}

type StudyDiagnosticsResponse = {
  study: Study
  summary: StudyDiagnosticsSummary
  recent_audit: AuditEntry[]
  routing_log: RoutingLogEntry[]
  dimse_retry: {
    available: boolean
    pending_total: number
    dead_letter_total: number
    pending_items?: Record<string, unknown>[]
    dead_letter_items?: Record<string, unknown>[]
    error?: string
  }
}

type DisplayTimezoneMode = 'utc' | 'local' | 'custom'

// ── Helpers ──────────────────────────────────────────────────────────────────

const GLOBAL_DISPLAY_TZ_MODE_KEY = 'aegis.ui.display_timezone_mode'
const GLOBAL_DISPLAY_TZ_CUSTOM_KEY = 'aegis.ui.display_timezone_custom'
const LEGACY_DISPLAY_TZ_MODE_KEYS = [
  'aegis.display_timezone_mode',
  'aegis.export.display_timezone_mode',
  'aegis.upload.display_timezone_mode',
]
const LEGACY_DISPLAY_TZ_CUSTOM_KEYS = [
  'aegis.display_timezone_custom',
  'aegis.export.display_timezone_custom',
  'aegis.upload.display_timezone_custom',
]
let displayTimezoneModeForFormat: DisplayTimezoneMode = 'utc'
let displayTimezoneCustomForFormat = ''

function pad2(n: number) {
  return String(n).padStart(2, '0')
}

function browserTimeZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Local'
  } catch {
    return 'Local'
  }
}

function normalizeIanaTimeZone(value: string): string | null {
  const candidate = value.trim()
  if (!candidate) return null
  try {
    return new Intl.DateTimeFormat('en-US', { timeZone: candidate }).resolvedOptions().timeZone
  } catch {
    return null
  }
}

function readFromLocalStorage(keys: string[]): string | null {
  if (typeof window === 'undefined') return null
  for (const key of keys) {
    const value = window.localStorage.getItem(key)
    if (value !== null) return value
  }
  return null
}

function writeToLocalStorage(keys: string[], value: string) {
  if (typeof window === 'undefined') return
  for (const key of keys) {
    window.localStorage.setItem(key, value)
  }
}

function readDisplayTimezone(): { mode: DisplayTimezoneMode; customTimeZone: string } {
  if (typeof window === 'undefined') {
    return { mode: 'utc', customTimeZone: '' }
  }

  const storedMode = readFromLocalStorage([GLOBAL_DISPLAY_TZ_MODE_KEY, ...LEGACY_DISPLAY_TZ_MODE_KEYS])
  const mode: DisplayTimezoneMode =
    storedMode === 'local' || storedMode === 'custom' ? storedMode : 'utc'
  const customTimeZone = readFromLocalStorage([GLOBAL_DISPLAY_TZ_CUSTOM_KEY, ...LEGACY_DISPLAY_TZ_CUSTOM_KEYS]) ?? ''
  return { mode, customTimeZone }
}

function writeDisplayTimezone(mode: DisplayTimezoneMode, customTimeZone: string) {
  if (typeof window === 'undefined') return
  writeToLocalStorage([GLOBAL_DISPLAY_TZ_MODE_KEY, ...LEGACY_DISPLAY_TZ_MODE_KEYS], mode)
  writeToLocalStorage([GLOBAL_DISPLAY_TZ_CUSTOM_KEY, ...LEGACY_DISPLAY_TZ_CUSTOM_KEYS], customTimeZone)
}

function setDisplayTimezoneForFormatting(mode: DisplayTimezoneMode, customTimeZone: string) {
  displayTimezoneModeForFormat = mode
  displayTimezoneCustomForFormat = customTimeZone
}

function fmtDateWithIntl(dt: Date, timeZone?: string) {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
    hourCycle: 'h23',
    timeZoneName: 'short',
    ...(timeZone ? { timeZone } : {}),
  })
  const parts = formatter.formatToParts(dt)
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? ''
  const y = get('year')
  const m = get('month')
  const d = get('day')
  const hh = get('hour')
  const mm = get('minute')
  const ss = get('second')
  const tz = get('timeZoneName') || (timeZone ?? browserTimeZone())
  return `${y}-${m}-${d} ${hh}:${mm}:${ss} ${tz}`
}

function fmtDateUtc(dt: Date) {
  const y = dt.getUTCFullYear()
  const m = pad2(dt.getUTCMonth() + 1)
  const d = pad2(dt.getUTCDate())
  const hh = pad2(dt.getUTCHours())
  const mm = pad2(dt.getUTCMinutes())
  const ss = pad2(dt.getUTCSeconds())
  return `${y}-${m}-${d} ${hh}:${mm}:${ss} UTC`
}

function fmtDate(iso: string) {
  if (!iso) return ''
  const dt = new Date(iso)
  if (Number.isNaN(dt.getTime())) return iso
  if (displayTimezoneModeForFormat === 'local') {
    return fmtDateWithIntl(dt)
  }
  if (displayTimezoneModeForFormat === 'custom' && displayTimezoneCustomForFormat) {
    return fmtDateWithIntl(dt, displayTimezoneCustomForFormat)
  }
  return fmtDateUtc(dt)
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

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
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

const AUDIT_PAGE_SIZE = 100
const AUDIT_CATEGORIES = ['study', 'admin_user', 'pipeline', 'phi_scan', 'qc_check', 'bids', 'classification', 'protocol_check', 'export', 'routing', 'institution', 'project', 'digest', 'destination']

function AuditLog({ projectId = '' }: { projectId?: string }) {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionFilter, setActionFilter] = useState('')
  const [actorFilter, setActorFilter] = useState('')
  const [actorInput, setActorInput] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [searchFilter, setSearchFilter] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [page, setPage] = useState(0)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  type ActorSummary = { actor: string; action_count: number; last_seen_at: string; last_action: string }
  const [actors, setActors] = useState<ActorSummary[] | null>(null)
  const [showActors, setShowActors] = useState(false)
  const loadActors = async () => {
    if (actors) { setShowActors(v => !v); return }
    const res = await fetch('/api/audit/actors')
    if (res.ok) { const d = await res.json(); setActors(d.actors ?? []); setShowActors(true) }
  }

  const fetchAudit = useCallback(async (actionF: string, actorF: string, searchF: string, dateFromF: string, dateToF: string, pg: number) => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({ limit: String(AUDIT_PAGE_SIZE), offset: String(pg * AUDIT_PAGE_SIZE) })
      if (actionF) params.set('action', actionF)
      if (actorF) params.set('actor', actorF)
      if (searchF) params.set('search', searchF)
      if (dateFromF) params.set('date_from', new Date(dateFromF).toISOString())
      if (dateToF) params.set('date_to', new Date(dateToF + 'T23:59:59Z').toISOString())
      const res = await fetch(`/api/audit?${params}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setEntries(data.entries ?? [])
      setTotal(data.total ?? 0)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load audit log')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAudit(actionFilter, actorFilter, searchFilter, dateFrom, dateTo, page) }, [fetchAudit, actionFilter, actorFilter, searchFilter, dateFrom, dateTo, page])

  const setCategory = (cat: string) => {
    setActionFilter(cat)
    setPage(0)
    setExpandedId(null)
  }

  const applyActorFilter = () => {
    setActorFilter(actorInput.trim())
    setPage(0)
    setExpandedId(null)
  }

  const clearActorFilter = () => {
    setActorInput('')
    setActorFilter('')
    setPage(0)
  }

  const applySearch = () => {
    setSearchFilter(searchInput.trim())
    setPage(0)
    setExpandedId(null)
  }

  const clearSearch = () => {
    setSearchInput('')
    setSearchFilter('')
    setDateFrom('')
    setDateTo('')
    setPage(0)
  }

  const totalPages = Math.max(1, Math.ceil(total / AUDIT_PAGE_SIZE))

  const auditCsvUrl = (() => {
    const params = new URLSearchParams()
    if (actionFilter) params.set('action', actionFilter)
    if (actorFilter)  params.set('actor',  actorFilter)
    if (searchFilter) params.set('search', searchFilter)
    if (dateFrom) params.set('date_from', new Date(dateFrom).toISOString())
    if (dateTo)   params.set('date_to',   new Date(dateTo + 'T23:59:59Z').toISOString())
    const qs = params.toString()
    return `/api/audit.csv${qs ? '?' + qs : ''}`
  })()

  return (
    <div>
      <div className="audit-toolbar">
        <div className="audit-filters">
          <span className="audit-filter-label">Category:</span>
          <button
            type="button"
            className={`audit-filter-btn${actionFilter === '' ? ' audit-filter-btn--active' : ''}`}
            onClick={() => setCategory('')}
          >
            All
          </button>
          {AUDIT_CATEGORIES.map(cat => (
            <button
              key={cat}
              type="button"
              className={`audit-filter-btn${actionFilter === cat ? ' audit-filter-btn--active' : ''}`}
              onClick={() => setCategory(cat === actionFilter ? '' : cat)}
            >
              {cat}
            </button>
          ))}
        </div>
        <div className="audit-actor-filter">
          <input
            type="text"
            placeholder="Filter by actor (email)…"
            value={actorInput}
            onChange={e => setActorInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && applyActorFilter()}
            className="audit-actor-input"
          />
          <button type="button" className="btn-secondary" onClick={applyActorFilter}>Apply</button>
          {actorFilter && <button type="button" className="btn-secondary" onClick={clearActorFilter}>Clear</button>}
        </div>
        <div className="audit-actor-filter">
          <input
            type="text"
            placeholder="Search across actor, action, resource…"
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && applySearch()}
            className="audit-actor-input"
            style={{minWidth:'220px'}}
          />
          <button type="button" className="btn-secondary" onClick={applySearch}>Search</button>
          {(searchFilter || dateFrom || dateTo) && <button type="button" className="btn-secondary" onClick={clearSearch}>Clear</button>}
        </div>
        <div className="audit-actor-filter" style={{gap:'6px'}}>
          <label style={{fontSize:'0.8rem',color:'var(--text-muted)'}}>From</label>
          <input type="date" value={dateFrom} onChange={e => { setDateFrom(e.target.value); setPage(0) }} className="audit-actor-input" style={{width:'130px'}} />
          <label style={{fontSize:'0.8rem',color:'var(--text-muted)'}}>To</label>
          <input type="date" value={dateTo} onChange={e => { setDateTo(e.target.value); setPage(0) }} className="audit-actor-input" style={{width:'130px'}} />
        </div>
        <button type="button" className="btn-refresh" onClick={() => fetchAudit(actionFilter, actorFilter, searchFilter, dateFrom, dateTo, page)}>Refresh</button>
        <a href={auditCsvUrl} download="audit.csv" className="btn btn--secondary btn--csv-export">Export CSV</a>
      </div>

      {/* Recent actors summary */}
      <div style={{marginBottom:'8px'}}>
        <button type="button" className="btn-secondary" onClick={loadActors} style={{fontSize:'0.8rem'}}>
          {showActors ? '▲ Hide activity summary' : '▼ Recent admin activity'}
        </button>
        {showActors && actors && (
          <div style={{marginTop:'6px',overflowX:'auto'}}>
            <table className="audit-table" style={{fontSize:'0.8rem',maxWidth:'700px'}}>
              <thead><tr><th>Actor</th><th>Actions (30d)</th><th>Last action</th><th>Last seen</th></tr></thead>
              <tbody>
                {actors.length === 0
                  ? <tr><td colSpan={4} className="td-muted">No activity in last 30 days.</td></tr>
                  : actors.map(a => (
                    <tr key={a.actor}>
                      <td style={{fontFamily:'monospace',fontSize:'0.8rem'}}>{a.actor}</td>
                      <td>{a.action_count}</td>
                      <td style={{fontFamily:'monospace',fontSize:'0.8rem'}}>{a.last_action}</td>
                      <td className="td-date">{fmtDate(a.last_seen_at)}</td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {loading && <div className="state-loading">Loading audit log…</div>}
      {error   && <div className="state-error">{error}</div>}

      {!loading && !error && entries.length === 0 && (
        <div className="state-empty">No audit entries yet.</div>
      )}

      {!loading && !error && entries.length > 0 && (
        <div className="audit-table-wrap">
          <div className="audit-pagination-bar">
            <span className="audit-total">{total} entries</span>
            <button type="button" className="btn-secondary" disabled={page === 0} onClick={() => setPage(p => p - 1)}>← Prev</button>
            <span className="audit-page-label">Page {page + 1} of {totalPages}</span>
            <button type="button" className="btn-secondary" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
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
              {entries.map(e => {
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
          <div className="audit-pagination-bar audit-pagination-bar--bottom">
            <button type="button" className="btn-secondary" disabled={page === 0} onClick={() => setPage(p => p - 1)}>← Prev</button>
            <span className="audit-page-label">Page {page + 1} of {totalPages}</span>
            <button type="button" className="btn-secondary" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Global Shares Panel ───────────────────────────────────────────────────────

const SHARES_PAGE_SIZE = 50

type ShareRecord = {
  id: string
  study_id: string
  recipient_email: string
  note: string
  expires_at: string
  created_by: string
  revoked_at: string | null
  created_at: string
  status: string
  expires_in_seconds: number
  download_count: number
}

type DownloadRecord = {
  id: string
  share_id: string
  client_ip: string
  accessed_at: string
}

type DownloadAnalytics = {
  total_downloads: number
  last_30_days: { date: string; count: number }[]
  top_shares: { share_id: string; recipient_email: string; study_id: string; download_count: number }[]
}

function GlobalSharesPanel({ isAdmin, projectId = '' }: { isAdmin: boolean; projectId?: string }) {
  const [shares, setShares] = useState<ShareRecord[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [emailSearch, setEmailSearch] = useState('')
  const [page, setPage] = useState(0)
  const [revoking, setRevoking] = useState<string | null>(null)
  const [extending, setExtending] = useState<string | null>(null)
  const [expandedShare, setExpandedShare] = useState<string | null>(null)
  const [downloads, setDownloads] = useState<Record<string, DownloadRecord[]>>({})
  const [nowMs, setNowMs] = useState(() => Date.now())
  const [analytics, setAnalytics] = useState<DownloadAnalytics | null>(null)
  const [revokeModalId, setRevokeModalId] = useState<string | null>(null)
  const [revokeModalReason, setRevokeModalReason] = useState('')

  useEffect(() => {
    fetch('/api/export-analytics')
      .then(r => r.ok ? r.json() : null)
      .then(d => { if (d) setAnalytics(d) })
      .catch(() => {})
  }, [])

  // Reset page when projectId changes
  useEffect(() => { setPage(0) }, [projectId])

  const fetchShares = useCallback(async (sf: string, pg: number) => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({ limit: String(SHARES_PAGE_SIZE), offset: String(pg * SHARES_PAGE_SIZE) })
      if (sf) params.set('status', sf)
      if (projectId) params.set('project_id', projectId)
      const res = await fetch(`/api/shares?${params}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      const now = Date.now()
      setNowMs(now)
      setShares((data.shares ?? []).map((s: ShareRecord) => withShareExpiryAnchor(s as unknown as Share, now) as unknown as ShareRecord))
      setTotal(data.total ?? 0)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load shares')
    } finally {
      setLoading(false)
    }
  }, [projectId])

  useEffect(() => { fetchShares(statusFilter, page) }, [fetchShares, statusFilter, page, projectId])

  const hasLiveCountdown = shares.some(s => s.status === 'active')
  useEffect(() => {
    if (!hasLiveCountdown) return
    const id = window.setInterval(() => setNowMs(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [hasLiveCountdown])

  const visibleShares = emailSearch.trim()
    ? shares.filter(s => s.recipient_email.toLowerCase().includes(emailSearch.toLowerCase()))
    : shares

  const setStatusF = (v: string) => { setStatusFilter(v); setPage(0) }

  const revokeShare = (id: string) => {
    setRevokeModalReason('')
    setRevokeModalId(id)
  }

  const confirmRevoke = async () => {
    const id = revokeModalId
    if (!id) return
    setRevokeModalId(null)
    setRevoking(id)
    try {
      const res = await fetch(`/api/shares/${id}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: revokeModalReason.trim() }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      fetchShares(statusFilter, page)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to revoke share')
    } finally {
      setRevoking(null)
    }
  }

  const extendShare = async (id: string) => {
    const input = window.prompt('Extend by how many hours?', '48')
    if (!input) return
    const hours = parseInt(input, 10)
    if (!hours || hours <= 0) { alert('Enter a positive number of hours.'); return }
    setExtending(id)
    try {
      const res = await fetch(`/api/shares/${id}/extend`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ extend_hours: hours }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      fetchShares(statusFilter, page)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to extend share')
    } finally {
      setExtending(null)
    }
  }

  const toggleDownloads = async (shareId: string) => {
    if (expandedShare === shareId) {
      setExpandedShare(null)
      return
    }
    setExpandedShare(shareId)
    if (downloads[shareId]) return
    try {
      const res = await fetch(`/api/shares/${shareId}/downloads`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setDownloads(prev => ({ ...prev, [shareId]: data.downloads ?? [] }))
    } catch {
      setDownloads(prev => ({ ...prev, [shareId]: [] }))
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / SHARES_PAGE_SIZE))

  const statusBadge = (s: ShareRecord) => {
    const cls = s.status === 'active' ? 'badge--approved' : s.status === 'revoked' ? 'badge--rejected' : 'badge--neutral'
    return <span className={`badge ${cls}`}>{s.status}</span>
  }

  return (
    <div>
      <div className="audit-toolbar">
        <div className="audit-filters">
          <span className="audit-filter-label">Status:</span>
          {(['', 'active', 'expired', 'revoked'] as const).map(s => (
            <button
              key={s || 'all'}
              type="button"
              className={`audit-filter-btn${statusFilter === s ? ' audit-filter-btn--active' : ''}`}
              onClick={() => setStatusF(s)}
            >
              {s || 'All'}
            </button>
          ))}
          <input
            type="text"
            className="filter-search"
            placeholder="Search by email…"
            value={emailSearch}
            onChange={e => setEmailSearch(e.target.value)}
          />
        </div>
        <button type="button" className="btn-refresh" onClick={() => fetchShares(statusFilter, page)}>Refresh</button>
      </div>

      {analytics && (
        <div className="pipeline-stats-bar">
          <span className="pipeline-stat">
            <strong>{analytics.total_downloads}</strong> total downloads
          </span>
          {analytics.last_30_days.length > 0 && (
            <span className="pipeline-stat">
              <strong>{analytics.last_30_days.reduce((s, d) => s + d.count, 0)}</strong> in last 30 days
            </span>
          )}
          {analytics.top_shares.length > 0 && (
            <span className="pipeline-stat">
              Top share: <strong>{analytics.top_shares[0].download_count}</strong> downloads ({analytics.top_shares[0].recipient_email})
            </span>
          )}
        </div>
      )}

      {loading && <div className="state-loading">Loading shares…</div>}
      {error && <div className="state-error">{error}</div>}
      {!loading && !error && shares.length === 0 && (
        <div className="state-empty">No export shares found.</div>
      )}

      {!loading && !error && shares.length > 0 && (
        <div className="audit-table-wrap">
          <div className="audit-pagination-bar">
            <span className="audit-total">{total} shares</span>
            <button type="button" className="btn-secondary" disabled={page === 0} onClick={() => setPage(p => p - 1)}>← Prev</button>
            <span className="audit-page-label">Page {page + 1} of {totalPages}</span>
            <button type="button" className="btn-secondary" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
          <table className="audit-table">
            <thead>
              <tr>
                <th>Status</th>
                <th>Recipient</th>
                <th>Study ID</th>
                <th>Created By</th>
                <th>Downloads</th>
                <th>Expires</th>
                <th>Note</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {visibleShares.map(s => {
                const shareAsShare = s as unknown as Share
                const remaining = s.status === 'active' ? shareRemainingSeconds(shareAsShare, nowMs) : null
                return (
                <>
                  <tr key={s.id} className="audit-row">
                    <td>{statusBadge(s)}</td>
                    <td className="audit-actor">{s.recipient_email}</td>
                    <td className="audit-resource-id">{uidShort(s.study_id)}</td>
                    <td className="audit-actor">{s.created_by || '—'}</td>
                    <td>
                      <button
                        type="button"
                        className="btn-secondary"
                        onClick={() => toggleDownloads(s.id)}
                        title="View download history"
                      >
                        {s.download_count ?? 0} {expandedShare === s.id ? '▲' : '▼'}
                      </button>
                    </td>
                    <td className="audit-time">
                      {fmtDate(s.expires_at)}
                      {remaining !== null && remaining > 0 && <div className="td-subtle">{fmtRemaining(remaining)}</div>}
                    </td>
                    <td className="audit-time td-note">{s.note || '—'}</td>
                    <td className="audit-time">{fmtDate(s.created_at)}</td>
                    <td>
                      {isAdmin && (s.status === 'active' || s.status === 'expired') ? (
                        <div className="actions-cell">
                          <button
                            type="button"
                            className="btn-secondary"
                            disabled={extending === s.id}
                            onClick={() => extendShare(s.id)}
                            title="Extend share expiry"
                          >
                            {extending === s.id ? 'Extending…' : 'Extend'}
                          </button>
                          {s.status === 'active' && (
                            <button
                              type="button"
                              className="btn-secondary btn-danger-text"
                              disabled={revoking === s.id}
                              onClick={() => revokeShare(s.id)}
                            >
                              {revoking === s.id ? 'Revoking…' : 'Revoke'}
                            </button>
                          )}
                        </div>
                      ) : (
                        <span className="audit-no-detail">—</span>
                      )}
                    </td>
                  </tr>
                  {expandedShare === s.id && (
                    <tr key={`${s.id}-downloads`} className="audit-row audit-row--sub">
                      <td colSpan={9} className="audit-sub-cell">
                        {!downloads[s.id] ? (
                          <span className="td-muted">Loading…</span>
                        ) : downloads[s.id].length === 0 ? (
                          <span className="td-muted">No downloads recorded.</span>
                        ) : (
                          <table className="audit-table audit-table--inner">
                            <thead>
                              <tr>
                                <th>Downloaded At</th>
                                <th>Client IP</th>
                              </tr>
                            </thead>
                            <tbody>
                              {downloads[s.id].map(d => (
                                <tr key={d.id} className="audit-row">
                                  <td className="audit-time">{fmtDate(d.accessed_at)}</td>
                                  <td className="audit-actor">{d.client_ip || '—'}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )}
                      </td>
                    </tr>
                  )}
                </>
                )
              })}
            </tbody>
          </table>
          <div className="audit-pagination-bar audit-pagination-bar--bottom">
            <button type="button" className="btn-secondary" disabled={page === 0} onClick={() => setPage(p => p - 1)}>← Prev</button>
            <span className="audit-page-label">Page {page + 1} of {totalPages}</span>
            <button type="button" className="btn-secondary" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
        </div>
      )}
      {revokeModalId && (
        <div style={{ marginTop: 12, background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6, padding: '12px 16px' }}>
          <p style={{ margin: '0 0 4px', fontWeight: 600, color: '#9a3412' }}>Revoke share link</p>
          <p style={{ margin: '0 0 8px', fontSize: 13, color: '#78350f' }}>Recipients will lose access immediately.</p>
          <textarea
            style={{ width: '100%', minHeight: 60, resize: 'vertical', borderRadius: 4, border: '1px solid #fdba74', padding: '6px 8px', fontFamily: 'inherit', fontSize: 13, boxSizing: 'border-box' }}
            maxLength={500}
            placeholder="Reason for revocation (optional)"
            value={revokeModalReason}
            onChange={e => setRevokeModalReason(e.target.value)}
          />
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <button type="button" className="btn btn--reject" onClick={confirmRevoke}>Confirm Revoke</button>
            <button type="button" className="btn btn--secondary" onClick={() => setRevokeModalId(null)}>Cancel</button>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Defacing Review Panel ─────────────────────────────────────────────────────

const OHIF_BASE = import.meta.env.VITE_OHIF_BASE_URL || 'http://localhost:3002'

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
  const [revokeModalId, setRevokeModalId] = useState<string | null>(null)
  const [revokeModalReason, setRevokeModalReason] = useState('')

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

  const handleRevoke = (shareId: string) => {
    setRevokeModalReason('')
    setRevokeModalId(shareId)
  }

  const confirmRevoke = async () => {
    const id = revokeModalId
    if (!id) return
    setRevokeModalId(null)
    await fetch(`/api/shares/${id}`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: revokeModalReason.trim() }),
    })
    fetchShares()
    if (newShare?.id === id) setNewShare(null)
  }

  const handleExtend = async (shareId: string) => {
    const input = window.prompt('Extend by how many hours?', '48')
    if (!input) return
    const hours = parseInt(input, 10)
    if (!hours || hours <= 0) { alert('Enter a positive number of hours.'); return }
    const res = await fetch(`/api/shares/${shareId}/extend`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ extend_hours: hours }),
    })
    if (!res.ok) { alert('Failed to extend share.'); return }
    fetchShares()
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
                    {(shareStatus === 'active' || shareStatus === 'expired') ? (
                      <div className="actions-cell">
                        <button type="button" className="btn btn--share" onClick={() => handleExtend(s.id)} title="Extend share expiry">Extend</button>
                        {shareStatus === 'active' && (
                          <button type="button" className="btn btn--revoke" onClick={() => handleRevoke(s.id)}>Revoke</button>
                        )}
                      </div>
                    ) : null}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
      {revokeModalId && (
        <div style={{ marginTop: 12, background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6, padding: '12px 16px' }}>
          <p style={{ margin: '0 0 4px', fontWeight: 600, color: '#9a3412' }}>Revoke share link</p>
          <p style={{ margin: '0 0 8px', fontSize: 13, color: '#78350f' }}>The recipient will lose access immediately.</p>
          <textarea
            style={{ width: '100%', minHeight: 60, resize: 'vertical', borderRadius: 4, border: '1px solid #fdba74', padding: '6px 8px', fontFamily: 'inherit', fontSize: 13, boxSizing: 'border-box' }}
            maxLength={500}
            placeholder="Reason for revocation (optional)"
            value={revokeModalReason}
            onChange={e => setRevokeModalReason(e.target.value)}
          />
          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <button type="button" className="btn btn--reject" onClick={confirmRevoke}>Confirm Revoke</button>
            <button type="button" className="btn btn--secondary" onClick={() => setRevokeModalId(null)}>Cancel</button>
          </div>
        </div>
      )}
    </div>
  )
}

// ── Study Detail Panel ────────────────────────────────────────────────────────

type PipelineStage = {
  label: string
  required: boolean
  status: string
  step: string  // API step name for reset-pipeline-step
}

function pipelineColorClass(status: string): string {
  if (['clean', 'pass', 'complete', 'classified', 'compliant', 'exported', 'approved', 'defaced'].includes(status)) return 'pipeline-dot--success'
  if (['warn', 'minor_deviations', 'flagged'].includes(status)) return 'pipeline-dot--warn'
  if (['fail', 'non_compliant', 'failed', 'rejected'].includes(status)) return 'pipeline-dot--error'
  if (['scanning', 'checking', 'converting', 'classifying', 'defacing', 'exporting'].includes(status)) return 'pipeline-dot--active'
  return 'pipeline-dot--pending'
}

const IN_FLIGHT_STATUSES = ['scanning', 'checking', 'converting', 'classifying', 'defacing', 'exporting']

function PipelineNode({ stage, studyId, isAdmin, onRerun }: {
  stage: PipelineStage
  studyId: string
  isAdmin: boolean
  onRerun: () => void
}) {
  const [rerunning, setRerunning] = useState(false)
  const canRerun = isAdmin && stage.required && !IN_FLIGHT_STATUSES.includes(stage.status) && stage.status !== 'pending'

  const handleRerun = async () => {
    setRerunning(true)
    await fetch(`/api/studies/${studyId}/reset-pipeline-step`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ step: stage.step }),
    })
    setRerunning(false)
    onRerun()
  }

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
      {canRerun && (
        <button type="button" className="btn btn--rerun" onClick={handleRerun} disabled={rerunning} title={`Reset ${stage.label} step to pending`}>
          {rerunning ? '…' : '↺'}
        </button>
      )}
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
  const [diagnostics, setDiagnostics] = useState<StudyDiagnosticsResponse | null>(null)
  const [labels, setLabels] = useState<StudyLabel[]>([])
  const [seriesList, setSeriesList] = useState<SeriesRow[]>([])
  const [loading, setLoading] = useState(true)
  const [detailTab, setDetailTab] = useState<'audit' | 'routing' | 'shares' | 'diagnostics' | 'labels' | 'series'>('audit')
  const [newLabel, setNewLabel] = useState('')
  const [labelSaving, setLabelSaving] = useState(false)

  // Viewer / review state
  const [viewOpen, setViewOpen] = useState(false)
  const [defaceOpen, setDefaceOpen] = useState(false)
  const [tagsOpen, setTagsOpen] = useState(false)
  const [dicomTags, setDicomTags] = useState<{ tag: string; keyword: string; vr: string; value: string }[] | null>(null)
  const [tagsLoading, setTagsLoading] = useState(false)

  const openDicomTags = async () => {
    if (tagsOpen) { setTagsOpen(false); return }
    if (!study) return
    setTagsOpen(true)
    if (dicomTags) return
    setTagsLoading(true)
    try {
      const res = await fetch(`/api/studies/${study.study_instance_uid}/dicom-tags`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setDicomTags(data.tags ?? [])
    } catch {
      setDicomTags([])
    } finally {
      setTagsLoading(false)
    }
  }

  // Share form state
  const [shareEmail, setShareEmail] = useState('')
  const [shareNote, setShareNote] = useState('')
  const [shareDays, setShareDays] = useState(7)
  const [shareMaxDownloads, setShareMaxDownloads] = useState('')
  const [shareResult, setShareResult] = useState<NewShareResult | null>(null)
  const [nowMs, setNowMs] = useState(() => Date.now())

  // Internal note state
  const [noteText, setNoteText] = useState('')
  const [noteSaving, setNoteSaving] = useState(false)
  const [noteSaved, setNoteSaved] = useState(false)

  // Subject ID editor state
  const [subjectEdit, setSubjectEdit] = useState(false)
  const [subjectDraft, setSubjectDraft] = useState('')

  // Project reassignment state
  const [reassignOpen, setReassignOpen] = useState(false)
  const [allProjects, setAllProjects] = useState<{ id: string; name: string }[]>([])
  const [reassignTarget, setReassignTarget] = useState('')

  // Reject reason modal state
  const [rejectModalOpen, setRejectModalOpen] = useState(false)
  const [rejectReasonText, setRejectReasonText] = useState('')

  const loadData = useCallback(() => {
    setLoading(true)
    Promise.all([
      fetch(`/api/studies/${studyId}`).then(r => r.ok ? r.json() : null),
      fetch(`/api/studies/${studyId}/audit`).then(r => r.ok ? r.json() : []),
      fetch(`/api/studies/${studyId}/routing-log`).then(r => r.ok ? r.json() : []),
      fetch(`/api/studies/${studyId}/shares`).then(r => r.ok ? r.json() : []),
      fetch(`/api/studies/${studyId}/diagnostics`).then(r => r.ok ? r.json() : null),
      fetch(`/api/studies/${studyId}/labels`).then(r => r.ok ? r.json() : []),
      fetch(`/api/studies/${studyId}/series`).then(r => r.ok ? r.json() : { series: [] }),
    ]).then(([s, a, rl, sh, diag, lbls, sr]) => {
      const now = Date.now()
      setStudy(s)
      setAudit(a ?? [])
      setRoutingLog(rl ?? [])
      const shareRows = (sh ?? []) as Share[]
      setShares(shareRows.map(row => withShareExpiryAnchor(row, now)))
      setDiagnostics(diag ?? null)
      setLabels(lbls ?? [])
      setSeriesList((sr?.series ?? []) as SeriesRow[])
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
    { label: 'Classification', required: study.classification_required, status: study.classification_status, step: 'classify' },
    { label: 'PHI Scan', required: study.phi_scan_required, status: study.phi_scan_status, step: 'phi_scan' },
    { label: 'Protocol', required: study.protocol_required, status: study.protocol_status, step: 'protocol' },
    { label: 'Defacing', required: study.defacing_required, status: study.status === 'defaced' ? 'defaced' : study.status === 'defacing' ? 'defacing' : study.defacing_required ? 'pending' : '', step: 'deface' },
    { label: 'QC', required: study.qc_required, status: study.qc_status, step: 'qc' },
    { label: 'BIDS', required: study.bids_required, status: study.bids_status, step: 'bids' },
    { label: 'Export', required: study.export_required, status: study.export_status, step: 'export' },
  ]

  const canApprove = !['approved', 'rejected', 'expired'].includes(study.status)
  const canReject = !['rejected', 'expired'].includes(study.status)
  const canReactivate = study.status === 'expired'
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

  const handleReject = () => {
    setRejectReasonText('')
    setRejectModalOpen(true)
  }

  const confirmReject = async () => {
    await fetch(`/api/studies/${study.id}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: rejectReasonText.trim() }),
    })
    setRejectModalOpen(false)
    loadData()
    onAction()
  }

  const openReassign = async () => {
    if (allProjects.length === 0) {
      const res = await fetch('/api/projects')
      if (res.ok) {
        const data = await res.json()
        setAllProjects(data ?? [])
        const first = (data ?? []).find((p: { id: string }) => p.id !== study.project_id)
        setReassignTarget(first?.id ?? '')
      }
    }
    setReassignOpen(true)
  }

  const handleReassign = async () => {
    if (!reassignTarget) return
    await fetch(`/api/studies/${study.id}/project`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project_id: reassignTarget }),
    })
    setReassignOpen(false)
    loadData()
    onAction()
  }

  const handleShare = async () => {
    const expiryHours = Math.max(1, shareDays * 24)
    const maxDl = shareMaxDownloads.trim() === '' ? undefined : parseInt(shareMaxDownloads, 10)
    const resp = await fetch(`/api/studies/${study.id}/share`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ recipient_email: shareEmail, note: shareNote, expiry_hours: expiryHours, max_downloads: maxDl }),
    })
    if (resp.ok) {
      const result = await resp.json()
      setShareResult(result)
      setShareEmail('')
      setShareNote('')
      setShareMaxDownloads('')
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
          {study.priority_flag && <span className="badge badge--flagged">★ Priority</span>}
        </div>
        {study.study_description && <p className="study-detail__description">{study.study_description}</p>}
      </div>

      {/* Meta row */}
      <div className="study-detail__meta">
        <div className="study-detail__meta-item"><strong>Modality</strong> {study.modality || '—'}</div>
        <div className="study-detail__meta-item"><strong>Body Part</strong> {study.body_part || '—'}</div>
        <div className="study-detail__meta-item"><strong>Files</strong> {study.instance_count}</div>
        <div className="study-detail__meta-item"><strong>Series</strong> {study.series_count}</div>
        <div className="study-detail__meta-item"><strong>Size</strong> {study.study_size_bytes > 0 ? formatBytes(study.study_size_bytes) : '—'}</div>
        <div className="study-detail__meta-item"><strong>Store</strong> {study.dicom_store || 'raw'}</div>
        <div className="study-detail__meta-item"><strong>Received</strong> {fmtDate(study.created_at)}</div>
        <div className="study-detail__meta-item"><strong>Updated</strong> {fmtDate(study.updated_at)}</div>
        {study.deface_qa_score != null && (
          <div className="study-detail__meta-item">
            <strong>Deface QA</strong>
            <span className={`deface-qa-score deface-qa-score--${study.deface_qa_score >= 0.9 ? 'good' : study.deface_qa_score >= 0.7 ? 'warn' : 'poor'}`}>
              {study.deface_qa_score.toFixed(4)}
            </span>
          </div>
        )}
        <div className="study-detail__meta-item">
          <strong>Subject</strong>
          {subjectEdit ? (
            <span style={{ display: 'inline-flex', gap: 4, alignItems: 'center' }}>
              <input
                className="form-input"
                style={{ width: 140, padding: '2px 6px', fontSize: '0.85em' }}
                value={subjectDraft}
                onChange={e => setSubjectDraft(e.target.value)}
                placeholder="subject-id"
                autoFocus
              />
              <button type="button" className="btn btn--sm" onClick={async () => {
                await fetch(`/api/studies/${study.id}/subject`, {
                  method: 'PUT',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({ subject_id: subjectDraft }),
                })
                setSubjectEdit(false)
                loadData()
              }}>Save</button>
              <button type="button" className="btn btn--sm" onClick={() => setSubjectEdit(false)}>✕</button>
            </span>
          ) : (
            <span>
              {study.subject_id ? <code style={{ fontSize: '0.85em' }}>{study.subject_id}</code> : <em style={{ color: '#9ca3af' }}>unset</em>}
              {isAdmin && (
                <button type="button" className="btn btn--sm" style={{ marginLeft: 6 }}
                  onClick={() => { setSubjectDraft(study.subject_id ?? ''); setSubjectEdit(true) }}>
                  Edit
                </button>
              )}
            </span>
          )}
        </div>
      </div>

      {study.status === 'rejected' && study.rejection_reason && (
        <div className="study-detail__meta-item">
          <strong>Rejection Reason</strong>
          <span style={{ color: '#9a3412', background: '#ffedd5', padding: '2px 6px', borderRadius: 4 }}>{study.rejection_reason}</span>
        </div>
      )}

      {/* Pipeline visualization */}
      <div className="study-detail__section">
        <h3 className="study-detail__section-title">Processing Pipeline</h3>
        <div className="pipeline-row">
          {stages.map((stage, i) => (
            <div key={stage.label} className="pipeline-step">
              <PipelineNode stage={stage} studyId={study.id} isAdmin={isAdmin} onRerun={loadData} />
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
          {isAdmin && canReject && <button type="button" className="btn btn--reject" onClick={handleReject}>Reject</button>}
          {isAdmin && canReactivate && <button type="button" className="btn btn--approve" onClick={() => { if (confirm('Reactivate this expired study?')) doAction(`/api/studies/${study.id}/reactivate`) }} title="Restore expired study to approved">Reactivate</button>}
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
          <button type="button" className="btn btn--secondary" onClick={openDicomTags}>{tagsOpen ? 'Hide DICOM tags' : 'DICOM tags'}</button>
          {isAdmin && <button type="button" className="btn btn--secondary" onClick={openReassign}>Move to Project</button>}
        </div>
        {isAdmin && reassignOpen && (
          <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <select
              title="Target project"
              value={reassignTarget}
              onChange={e => setReassignTarget(e.target.value)}
              className="share-select"
            >
              {allProjects.filter(p => p.id !== study.project_id).map(p => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
            <button type="button" className="btn btn--approve" disabled={!reassignTarget} onClick={handleReassign}>Move</button>
            <button type="button" className="btn btn--secondary" onClick={() => setReassignOpen(false)}>Cancel</button>
          </div>
        )}
        {isAdmin && rejectModalOpen && (
          <div style={{ marginTop: 12, background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6, padding: '12px 16px' }}>
            <p style={{ margin: '0 0 8px', fontWeight: 600, color: '#9a3412' }}>Reject study</p>
            <textarea
              style={{ width: '100%', minHeight: 72, resize: 'vertical', borderRadius: 4, border: '1px solid #fdba74', padding: '6px 8px', fontFamily: 'inherit', fontSize: 13 }}
              maxLength={500}
              placeholder="Rejection reason (optional — shown to uploader)"
              value={rejectReasonText}
              onChange={e => setRejectReasonText(e.target.value)}
            />
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <button type="button" className="btn btn--reject" onClick={confirmReject}>Confirm Reject</button>
              <button type="button" className="btn btn--secondary" onClick={() => setRejectModalOpen(false)}>Cancel</button>
            </div>
          </div>
        )}
      </div>

      {/* DICOM tag inspection panel */}
      {tagsOpen && (
        <div className="dicom-tags-panel">
          {tagsLoading && <div className="state-loading">Loading tags…</div>}
          {!tagsLoading && dicomTags && dicomTags.length === 0 && <div className="state-empty">No tags found.</div>}
          {!tagsLoading && dicomTags && dicomTags.length > 0 && (
            <table className="audit-table dicom-tags-table">
              <thead><tr><th>Tag</th><th>Keyword</th><th>VR</th><th>Value</th></tr></thead>
              <tbody>
                {dicomTags.map(t => (
                  <tr key={t.tag}>
                    <td><code>{t.tag}</code></td>
                    <td>{t.keyword}</td>
                    <td><code>{t.vr}</code></td>
                    <td className="dicom-tag-value">{t.value}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

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
            <input
              type="number"
              min={1}
              placeholder="Max downloads (∞)"
              value={shareMaxDownloads}
              onChange={e => setShareMaxDownloads(e.target.value)}
              className="share-input"
              style={{ width: 160 }}
              title="Leave blank for unlimited downloads"
            />
            <button type="button" className="btn btn--approve" disabled={!shareEmail} onClick={handleShare}>Send</button>
          </div>
          {shareResult && (
            <div className="share-result">
              Share created! Link: <code>{shareResult.export_url}</code>
            </div>
          )}
        </div>
      )}

      {/* Internal admin note (admin only) */}
      {isAdmin && (
        <div className="study-detail__section">
          <h3 className="study-detail__section-title">Add Internal Note</h3>
          <div className="note-inline">
            <textarea
              className="note-textarea"
              placeholder="Internal note (visible only to admins in audit trail)…"
              value={noteText}
              onChange={e => { setNoteText(e.target.value); setNoteSaved(false) }}
              rows={3}
              maxLength={2000}
            />
            <div className="note-inline__footer">
              <span className="note-char-count">{noteText.length}/2000</span>
              <button
                type="button"
                className="btn btn--secondary"
                disabled={!noteText.trim() || noteSaving}
                onClick={async () => {
                  setNoteSaving(true)
                  await fetch(`/api/studies/${study.id}/notes`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ note: noteText }),
                  })
                  setNoteText('')
                  setNoteSaved(true)
                  setNoteSaving(false)
                  loadData()
                }}
              >
                {noteSaving ? 'Saving…' : 'Save note'}
              </button>
              {noteSaved && <span className="note-saved">Saved</span>}
            </div>
          </div>
        </div>
      )}

      {/* Detail tabs: Audit / Routing / Shares / Diagnostics */}
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
          <button type="button" className={`tab-btn${detailTab === 'diagnostics' ? ' tab-btn--active' : ''}`} onClick={() => setDetailTab('diagnostics')}>
            Diagnostics
          </button>
          <button type="button" className={`tab-btn${detailTab === 'labels' ? ' tab-btn--active' : ''}`} onClick={() => setDetailTab('labels')}>
            Labels ({labels.length})
          </button>
          {seriesList.length > 0 && (
            <button type="button" className={`tab-btn${detailTab === 'series' ? ' tab-btn--active' : ''}`} onClick={() => setDetailTab('series')}>
              Series ({seriesList.length})
            </button>
          )}
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
              <tr><th>Recipient</th><th>Created</th><th>Expires</th><th>Status</th><th>Downloads</th><th>Note</th></tr>
            </thead>
            <tbody>
              {shares.length === 0 && <tr><td colSpan={6}>No shares.</td></tr>}
              {shares.map(s => {
                const shareStatus = shareStatusLabel(s, nowMs)
                const statusClass = `share-status--${shareStatus}`
                const remainingSeconds = shareRemainingSeconds(s, nowMs)
                const remainingLabel =
                  shareStatus === 'active' && remainingSeconds !== null
                    ? fmtRemaining(remainingSeconds)
                    : ''
                const dlCount = s.download_count ?? 0
                const dlLabel = s.max_downloads != null
                  ? `${dlCount} / ${s.max_downloads}`
                  : String(dlCount)
                return (
                  <tr key={s.id}>
                    <td>{s.recipient_email}</td>
                    <td className="td-date">{fmtDate(s.created_at)}</td>
                    <td className="td-date">
                      {fmtDate(s.expires_at)}
                      {remainingLabel && <div className="td-subtle">({remainingLabel} remaining)</div>}
                    </td>
                    <td><span className={statusClass}>{shareStatus}</span></td>
                    <td>{dlLabel}</td>
                    <td>{s.note || '—'}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}

        {detailTab === 'labels' && (
          <div className="labels-panel">
            {/* Label chips */}
            <div className="labels-panel__chips">
              {labels.length === 0 && <span className="routing-desc">No labels yet.</span>}
              {labels.map(lbl => (
                <span key={lbl.id} className="label-chip" title={`Added by ${lbl.created_by}`}>
                  {lbl.label}
                  {isAdmin && (
                    <button
                      type="button"
                      className="label-chip__remove"
                      aria-label={`Remove label ${lbl.label}`}
                      onClick={async () => {
                        await fetch(`/api/studies/${studyId}/labels/${lbl.id}`, { method: 'DELETE' })
                        loadData()
                      }}
                    >×</button>
                  )}
                </span>
              ))}
            </div>
            {/* Add label form (admin only) */}
            {isAdmin && (
              <form
                className="labels-panel__form"
                onSubmit={async e => {
                  e.preventDefault()
                  if (!newLabel.trim()) return
                  setLabelSaving(true)
                  await fetch(`/api/studies/${studyId}/labels`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ label: newLabel.trim() }),
                  })
                  setNewLabel('')
                  setLabelSaving(false)
                  loadData()
                }}
              >
                <input
                  className="form-input"
                  type="text"
                  placeholder="Add label…"
                  maxLength={80}
                  value={newLabel}
                  onChange={e => setNewLabel(e.target.value)}
                />
                <button type="submit" className="btn-primary" disabled={labelSaving || !newLabel.trim()}>
                  {labelSaving ? 'Adding…' : 'Add'}
                </button>
              </form>
            )}
          </div>
        )}

        {detailTab === 'series' && (
          <table className="detail-table">
            <thead>
              <tr>
                <th>#</th>
                <th>Series UID</th>
                <th>Description</th>
                <th>Modality</th>
                <th>Body Part</th>
                <th>Instances</th>
              </tr>
            </thead>
            <tbody>
              {seriesList.length === 0 && (
                <tr><td colSpan={6} style={{textAlign:'center',color:'var(--color-gray-500)'}}>No series metadata recorded.</td></tr>
              )}
              {seriesList.map((s, i) => (
                <tr key={s.id}>
                  <td style={{color:'var(--color-gray-500)'}}>{i + 1}</td>
                  <td style={{fontFamily:'monospace',fontSize:'0.78rem'}}>{s.series_instance_uid}</td>
                  <td>{s.series_description || '—'}</td>
                  <td>{s.modality || '—'}</td>
                  <td>{s.body_part || '—'}</td>
                  <td>{s.instance_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {detailTab === 'diagnostics' && (
          <div className="diagnostics-panel">
            {!diagnostics && <p className="state-empty">Diagnostics not available.</p>}
            {diagnostics && (
              <>
                <div className={`diagnostics-summary diagnostics-summary--${diagnostics.summary.stuck ? 'stuck' : diagnostics.summary.terminal ? 'terminal' : 'ok'}`}>
                  <div className="diagnostics-summary__status">
                    {diagnostics.summary.terminal && <span className="diag-badge diag-badge--terminal">Terminal</span>}
                    {diagnostics.summary.stuck && <span className="diag-badge diag-badge--stuck">Stuck</span>}
                    {!diagnostics.summary.terminal && !diagnostics.summary.stuck && <span className="diag-badge diag-badge--ok">On track</span>}
                    {diagnostics.summary.last_audit_action && (
                      <span className="diag-last-action">Last: <code>{diagnostics.summary.last_audit_action}</code>{diagnostics.summary.last_audit_at ? ` at ${fmtDate(diagnostics.summary.last_audit_at)}` : ''}</span>
                    )}
                  </div>
                  {diagnostics.summary.blockers.length > 0 && (
                    <div className="diagnostics-summary__section">
                      <strong>Blockers</strong>
                      <ul className="diag-list">
                        {diagnostics.summary.blockers.map((b, i) => <li key={i}>{b}</li>)}
                      </ul>
                    </div>
                  )}
                  {diagnostics.summary.recommended_actions.length > 0 && (
                    <div className="diagnostics-summary__section">
                      <strong>Recommended actions</strong>
                      <ul className="diag-list">
                        {diagnostics.summary.recommended_actions.map((a, i) => <li key={i}>{a}</li>)}
                      </ul>
                    </div>
                  )}
                </div>

                {diagnostics.dimse_retry.available && (
                  <div className="diagnostics-summary__section">
                    <strong>DIMSE Retry</strong>
                    <table className="detail-table">
                      <tbody>
                        <tr><td>Pending</td><td>{diagnostics.dimse_retry.pending_total}</td></tr>
                        <tr><td>Dead-letter</td><td>{diagnostics.dimse_retry.dead_letter_total}</td></tr>
                        {diagnostics.dimse_retry.error && <tr><td>Error</td><td className="diag-error">{diagnostics.dimse_retry.error}</td></tr>}
                      </tbody>
                    </table>
                  </div>
                )}

                <div className="diagnostics-summary__section">
                  <strong>Recent audit ({diagnostics.recent_audit.length})</strong>
                  <table className="detail-table">
                    <thead><tr><th>Time</th><th>Action</th><th>Actor</th></tr></thead>
                    <tbody>
                      {diagnostics.recent_audit.length === 0 && <tr><td colSpan={3}>No entries.</td></tr>}
                      {diagnostics.recent_audit.map(e => (
                        <tr key={e.id}>
                          <td className="td-date">{fmtDate(e.created_at)}</td>
                          <td><code>{e.action}</code></td>
                          <td>{e.actor}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ── Study Row ─────────────────────────────────────────────────────────────────

function StudyRow({
  study,
  onAction,
  onSelect,
  onAskAgent,
  isAdmin,
  checked,
  onToggle
}: {
  study: Study
  onAction: () => void
  onSelect: () => void
  onAskAgent: () => void
  isAdmin: boolean
  checked: boolean
  onToggle: () => void
}) {
  const [shareOpen,      setShareOpen]      = useState(false)
  const [viewOpen,       setViewOpen]       = useState(false)
  const [defaceOpen,     setDefaceOpen]     = useState(false)
  const [rejectRowOpen,  setRejectRowOpen]  = useState(false)
  const [rejectRowText,  setRejectRowText]  = useState('')

  const handleApprove = async () => {
    await fetch(`/api/studies/${study.id}/approve`, { method: 'POST' })
    onAction()
  }

  const handleReject = () => {
    setRejectRowText('')
    setRejectRowOpen(true)
  }

  const confirmRejectRow = async () => {
    await fetch(`/api/studies/${study.id}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: rejectRowText.trim() }),
    })
    setRejectRowOpen(false)
    onAction()
  }

  const handleDelete = async () => {
    if (!confirm(`Permanently delete study ${uidShort(study.study_instance_uid)}?\n\nThis removes all DICOM files and cannot be undone.`)) return
    await fetch(`/api/studies/${study.id}`, { method: 'DELETE' })
    onAction()
  }

  const canApprove    = !['approved', 'rejected', 'expired'].includes(study.status)
  const canReject     = !['rejected', 'expired'].includes(study.status)
  const canReactivate = study.status === 'expired'
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

  const handleToggleFlag = async () => {
    await fetch(`/api/studies/${study.id}/flag`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ flagged: !study.priority_flag }),
    })
    onAction()
  }

  return (
    <>
      <tr className={`${checked ? 'tr--selected' : ''}${study.priority_flag ? ' tr--flagged' : ''}`}>
        <td className="td-check"><input type="checkbox" checked={checked} onChange={onToggle} aria-label="Select study" /></td>
        <td className="td-flag">
          <button
            type="button"
            className={`btn-flag${study.priority_flag ? ' btn-flag--on' : ''}`}
            onClick={isAdmin ? handleToggleFlag : undefined}
            title={isAdmin ? (study.priority_flag ? 'Remove priority flag' : 'Mark as priority') : (study.priority_flag ? 'Priority' : '')}
            style={{ cursor: isAdmin ? 'pointer' : 'default' }}
          >
            {study.priority_flag ? '★' : '☆'}
          </button>
        </td>
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
            {isAdmin && canReactivate && (
              <button type="button" className="btn btn--approve" onClick={async () => { if (confirm('Reactivate this expired study?')) { await fetch(`/api/studies/${study.id}/reactivate`, { method: 'POST' }); onAction() } }} title="Restore expired study to approved">Reactivate</button>
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
            <button type="button" className="btn btn--agent" onClick={onAskAgent}>
              Ask agent
            </button>
            {isAdmin && (
              <button type="button" className="btn btn--revoke" onClick={handleDelete} title="Permanently delete study and all DICOM files">
                Delete
              </button>
            )}
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
      {rejectRowOpen && (
        <tr>
          <td colSpan={14}>
            <div style={{ background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6, padding: '12px 16px', margin: '4px 0' }}>
              <p style={{ margin: '0 0 8px', fontWeight: 600, color: '#9a3412' }}>Reject study</p>
              <textarea
                style={{ width: '100%', minHeight: 64, resize: 'vertical', borderRadius: 4, border: '1px solid #fdba74', padding: '6px 8px', fontFamily: 'inherit', fontSize: 13 }}
                maxLength={500}
                placeholder="Rejection reason (optional — shown to uploader)"
                value={rejectRowText}
                onChange={e => setRejectRowText(e.target.value)}
              />
              <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                <button type="button" className="btn btn--reject" onClick={confirmRejectRow}>Confirm Reject</button>
                <button type="button" className="btn btn--secondary" onClick={() => setRejectRowOpen(false)}>Cancel</button>
              </div>
            </div>
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

function RoutingPanel({ isAdmin, projectId = '' }: { isAdmin: boolean; projectId?: string }) {
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

  // Destination connectivity test
  type DestTestResult = { success: boolean; latency_ms: number; error?: string; status_code?: number }
  const [destTestResults, setDestTestResults] = useState<Record<string, DestTestResult>>({})
  const [destTesting, setDestTesting]         = useState<Record<string, boolean>>({})

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

  async function testDest(id: string) {
    setDestTesting(prev => ({ ...prev, [id]: true }))
    setDestTestResults(prev => { const next = { ...prev }; delete next[id]; return next })
    try {
      const res = await fetch(`/api/destinations/${id}/test`, { method: 'POST' })
      const data = await res.json()
      setDestTestResults(prev => ({ ...prev, [id]: data }))
    } catch {
      setDestTestResults(prev => ({ ...prev, [id]: { success: false, latency_ms: 0, error: 'Request failed' } }))
    } finally {
      setDestTesting(prev => ({ ...prev, [id]: false }))
    }
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
              {destinations.map(d => {
                const testResult = destTestResults[d.id]
                const testing = destTesting[d.id]
                return (
                  <tr key={d.id} className={d.enabled ? '' : 'routing-row--disabled'}>
                    <td>
                      <div className="routing-name">{d.name}</div>
                      {d.description && <div className="routing-desc">{d.description}</div>}
                      {testResult && (
                        <div style={{
                          marginTop: 4,
                          fontSize: 12,
                          padding: '3px 7px',
                          borderRadius: 4,
                          display: 'inline-block',
                          background: testResult.success ? '#ccfbf1' : '#ffedd5',
                          color: testResult.success ? '#0f766e' : '#9a3412',
                          border: `1px solid ${testResult.success ? '#5eead4' : '#fed7aa'}`,
                        }}>
                          {testResult.success
                            ? `✓ reachable — ${testResult.latency_ms}ms${testResult.status_code ? ` (HTTP ${testResult.status_code})` : ''}`
                            : `✗ ${testResult.error || 'unreachable'}`}
                        </div>
                      )}
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
                        <button type="button" className="btn btn--secondary" onClick={() => testDest(d.id)} disabled={testing}>
                          {testing ? 'Testing…' : 'Test'}
                        </button>
                        {isAdmin && (
                          <>
                            <button type="button" className="btn btn--edit" onClick={() => openEditDest(d)}>Edit</button>
                            <button type="button" className="btn btn--revoke" onClick={() => deleteDest(d.id, d.name)}>Delete</button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              })}
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
              {rules.filter(r => !projectId || !r.project_id || r.project_id === projectId).map(r => {
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

  const [importMsg, setImportMsg]   = useState<string | null>(null)
  const [importing, setImporting]   = useState(false)
  const importRef = useRef<HTMLInputElement>(null)

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

  async function handleImport(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const targetProjectID = formProject || projects[0]?.id
    if (!targetProjectID) return
    setImporting(true)
    setImportMsg(null)
    try {
      const text = await file.text()
      const res = await fetch(`/api/projects/${targetProjectID}/protocol-templates/import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: text,
      })
      const data = await res.json()
      if (!res.ok) {
        setImportMsg(`Import failed: ${data.error ?? res.status}`)
      } else {
        setImportMsg(`Imported ${data.imported}, skipped ${data.skipped} duplicate${data.skipped !== 1 ? 's' : ''}`)
        if (data.imported > 0) fetchAll()
      }
    } catch {
      setImportMsg('Import failed: could not read file')
    } finally {
      setImporting(false)
      if (importRef.current) importRef.current.value = ''
    }
  }

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
          <div className="actions-cell">
            {projects.length > 0 && (
              <a
                className="btn btn--secondary"
                href={`/api/projects/${formProject || projects[0]?.id}/protocol-templates/export`}
                download="protocol-templates.json"
                title="Download all templates for selected project as JSON"
              >
                Export JSON
              </a>
            )}
            {isAdmin && projects.length > 0 && (
              <>
                <input
                  ref={importRef}
                  type="file"
                  accept=".json"
                  style={{ display: 'none' }}
                  onChange={handleImport}
                />
                <button
                  type="button"
                  className="btn btn--secondary"
                  onClick={() => importRef.current?.click()}
                  disabled={importing}
                  title="Import templates from a JSON file"
                >
                  {importing ? 'Importing…' : 'Import JSON'}
                </button>
              </>
            )}
            {isAdmin && <button type="button" className="btn-primary" onClick={openCreate}>+ New template</button>}
          </div>
        </div>
        {importMsg && (
          <div style={{ padding: '6px 12px', fontSize: 13, color: importMsg.startsWith('Import failed') ? '#9a3412' : '#0f766e', background: importMsg.startsWith('Import failed') ? '#ffedd5' : '#ccfbf1', borderRadius: 4, marginTop: 4 }}>
            {importMsg}
          </div>
        )}

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

const WEBHOOK_EVENTS = [
  'study.approved',
  'study.rejected',
  'study.phi_flagged',
  'study.export_complete',
  'study.stuck',
]

function NotificationsPanel({ isAdmin, projectId }: { isAdmin: boolean; projectId?: string }) {
  const [projects, setProjects]   = useState<Project[]>([])
  const [subs, setSubs]           = useState<DigestSubscription[]>([])
  const [webhooks, setWebhooks]   = useState<WebhookSubscription[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)

  // Digest form state
  const [formEmail, setFormEmail]         = useState('')
  const [formProject, setFormProject]     = useState('')
  const [formFrequency, setFormFrequency] = useState<'weekly' | 'monthly'>('weekly')
  const [showForm, setShowForm]           = useState(false)
  const [saving, setSaving]               = useState(false)
  const [formError, setFormError]         = useState<string | null>(null)

  // Webhook form state
  const [showWebhookForm, setShowWebhookForm]     = useState(false)
  const [whURL, setWhURL]                         = useState('')
  const [whEvents, setWhEvents]                   = useState<string[]>([])
  const [whProject, setWhProject]                 = useState('')
  const [whSecret, setWhSecret]                   = useState('')
  const [whEnabled, setWhEnabled]                 = useState(true)
  const [whSaving, setWhSaving]                   = useState(false)
  const [whFormError, setWhFormError]             = useState<string | null>(null)
  const [whEditId, setWhEditId]                   = useState<string | null>(null)

  // Webhook delivery log state
  const [deliveryWhId, setDeliveryWhId]           = useState<string | null>(null)
  const [deliveries, setDeliveries]               = useState<WebhookDelivery[]>([])
  const [deliveriesLoading, setDeliveriesLoading] = useState(false)

  // Webhook stats state
  const [statsWhId, setStatsWhId] = useState<string | null>(null)
  const [statsMap, setStatsMap]   = useState<Record<string, { total_deliveries: number; successful: number; failed: number; success_rate_pct: number; last_delivery_at?: string; deliveries_by_event: Record<string, number> }>>({})

  const showDeliveries = async (id: string) => {
    if (deliveryWhId === id) { setDeliveryWhId(null); return }
    setDeliveryWhId(id)
    setDeliveriesLoading(true)
    try {
      const res = await fetch(`/api/webhook-subscriptions/${id}/deliveries`)
      const data = res.ok ? await res.json() : { deliveries: [] }
      setDeliveries(data.deliveries ?? [])
    } finally {
      setDeliveriesLoading(false)
    }
  }

  const showStats = async (id: string) => {
    if (statsWhId === id) { setStatsWhId(null); return }
    setStatsWhId(id)
    if (!statsMap[id]) {
      try {
        const res = await fetch(`/api/webhook-subscriptions/${id}/stats`)
        const data = res.ok ? await res.json() : null
        if (data) setStatsMap(prev => ({ ...prev, [id]: data }))
      } catch { /* ignore */ }
    }
  }

  const retryDelivery = async (deliveryId: string, whId: string) => {
    await fetch(`/api/webhook-deliveries/${deliveryId}/retry`, { method: 'POST' })
    // Refresh the delivery log for this webhook
    setDeliveriesLoading(true)
    try {
      const res = await fetch(`/api/webhook-subscriptions/${whId}/deliveries`)
      const data = res.ok ? await res.json() : { deliveries: [] }
      setDeliveries(data.deliveries ?? [])
      // Invalidate cached stats so they reload on next toggle
      setStatsMap(prev => { const n = { ...prev }; delete n[whId]; return n })
    } finally {
      setDeliveriesLoading(false)
    }
  }

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const digestURL = projectId
        ? `/api/projects/${projectId}/digest-subscriptions`
        : '/api/digest-subscriptions'
      const webhookURL = projectId
        ? `/api/webhook-subscriptions?project_id=${projectId}`
        : '/api/webhook-subscriptions'
      const [projRes, subRes, whRes] = await Promise.all([
        fetch('/api/projects'),
        fetch(digestURL),
        fetch(webhookURL),
      ])
      if (!projRes.ok || !subRes.ok || !whRes.ok) throw new Error('Failed to load data')
      const [projs, subList, whList] = await Promise.all([projRes.json(), subRes.json(), whRes.json()])
      setProjects(projs ?? [])
      setSubs(subList ?? [])
      setWebhooks(whList ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [projectId])

  useEffect(() => { fetchAll() }, [fetchAll])

  function openCreate() {
    setFormEmail('')
    setFormProject(projectId ?? projects[0]?.id ?? '')
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

  // Webhook helpers
  function openWebhookCreate() {
    setWhEditId(null)
    setWhURL('')
    setWhEvents([])
    setWhProject(projectId ?? '')
    setWhSecret('')
    setWhEnabled(true)
    setWhFormError(null)
    setShowWebhookForm(true)
  }

  function openWebhookEdit(wh: WebhookSubscription) {
    setWhEditId(wh.id)
    setWhURL(wh.url)
    setWhEvents(wh.events)
    setWhProject(wh.project_id ?? '')
    setWhSecret('')
    setWhEnabled(wh.enabled)
    setWhFormError(null)
    setShowWebhookForm(true)
  }

  function toggleWhEvent(ev: string) {
    setWhEvents(prev => prev.includes(ev) ? prev.filter(e => e !== ev) : [...prev, ev])
  }

  async function saveWebhook() {
    if (!whURL) { setWhFormError('URL is required'); return }
    if (whEvents.length === 0) { setWhFormError('Select at least one event'); return }
    setWhSaving(true)
    setWhFormError(null)
    const body: Record<string, unknown> = {
      url: whURL, events: whEvents, enabled: whEnabled,
    }
    if (whProject) body.project_id = whProject
    if (whSecret)  body.secret = whSecret
    try {
      const res = await fetch(
        whEditId ? `/api/webhook-subscriptions/${whEditId}` : '/api/webhook-subscriptions',
        { method: whEditId ? 'PUT' : 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }
      )
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowWebhookForm(false)
      fetchAll()
    } catch (err) {
      setWhFormError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setWhSaving(false)
    }
  }

  async function testWebhook(id: string, url: string) {
    const res = await fetch(`/api/webhook-subscriptions/${id}/test`, { method: 'POST' })
    const data = res.ok ? await res.json() : null
    if (data?.success) {
      alert(`Test delivery succeeded (HTTP ${data.status_code}) → ${url}`)
    } else {
      const errMsg = data?.error ?? `HTTP ${res.status}`
      alert(`Test delivery failed → ${url}\n\n${errMsg}`)
    }
    fetchAll()
  }

  async function deleteWebhook(id: string, url: string) {
    if (!confirm(`Remove webhook for ${url}?`)) return
    await fetch(`/api/webhook-subscriptions/${id}`, { method: 'DELETE' })
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

      {/* ── Webhook subscriptions ── */}
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Webhook Subscriptions</div>
            <div className="routing-section-sub">
              HTTP POST callbacks fired on study events, signed with HMAC-SHA256 when a secret is set.
            </div>
          </div>
          {isAdmin && <button type="button" className="btn-primary" onClick={openWebhookCreate}>+ New webhook</button>}
        </div>

        {isAdmin && showWebhookForm && (
          <div className="routing-form">
            <h3>{whEditId ? 'Edit webhook' : 'New webhook'}</h3>
            {whFormError && <div className="form-error">{whFormError}</div>}
            <div className="form-grid">
              <input className="form-input" type="url" placeholder="Endpoint URL (https://…) *"
                value={whURL} onChange={e => setWhURL(e.target.value)} />
              <select className="form-select" aria-label="Project scope"
                value={whProject} onChange={e => setWhProject(e.target.value)}>
                <option value="">All projects</option>
                {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
              </select>
              <input className="form-input" type="text" placeholder="Secret (optional, for HMAC signing)"
                value={whSecret} onChange={e => setWhSecret(e.target.value)} />
            </div>
            <div className="form-row" style={{ gap: '0.75rem', flexWrap: 'wrap', marginBottom: '0.5rem' }}>
              <span style={{ fontWeight: 500, fontSize: '0.875rem' }}>Events:</span>
              {WEBHOOK_EVENTS.map(ev => (
                <label key={ev} style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', fontSize: '0.875rem', cursor: 'pointer' }}>
                  <input type="checkbox" checked={whEvents.includes(ev)} onChange={() => toggleWhEvent(ev)} />
                  {ev}
                </label>
              ))}
            </div>
            <div className="form-row" style={{ marginBottom: '0.5rem' }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', cursor: 'pointer' }}>
                <input type="checkbox" checked={whEnabled} onChange={e => setWhEnabled(e.target.checked)} />
                Enabled
              </label>
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={saveWebhook} disabled={whSaving}>
                {whSaving ? 'Saving…' : (whEditId ? 'Update' : 'Create')}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowWebhookForm(false)}>
                Cancel
              </button>
            </div>
          </div>
        )}

        {loading && <div className="state-loading">Loading…</div>}
        {!loading && !error && webhooks.length === 0 && (
          <div className="state-empty">No webhook subscriptions yet.</div>
        )}
        {!loading && !error && webhooks.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>URL</th>
                <th>Events</th>
                <th>Project</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {webhooks.map(wh => (
                <>
                  <tr key={wh.id}>
                    <td style={{ fontFamily: 'monospace', fontSize: '0.8rem', wordBreak: 'break-all' }}>{wh.url}</td>
                    <td style={{ fontSize: '0.8rem' }}>{wh.events.join(', ')}</td>
                    <td>{wh.project_id ? (projects.find(p => p.id === wh.project_id)?.name ?? wh.project_id) : <span className="routing-desc">all</span>}</td>
                    <td>
                      <span className={`status-badge status-badge--${wh.enabled ? 'clean' : 'failed'}`}>
                        {wh.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td>
                      <div className="actions-cell">
                        <button type="button" className="btn-secondary"
                          onClick={() => showStats(wh.id)}
                          title="View delivery statistics">
                          {statsWhId === wh.id ? 'Hide Stats' : 'Stats'}
                        </button>
                        <button type="button" className="btn-secondary"
                          onClick={() => showDeliveries(wh.id)}
                          title="View delivery log">
                          {deliveryWhId === wh.id ? 'Hide Log' : 'Log'}
                        </button>
                        {isAdmin && (
                          <>
                            <button type="button" className="btn btn--action"
                              title="Send a test study.approved payload"
                              onClick={() => testWebhook(wh.id, wh.url)}>Test</button>
                            <button type="button" className="btn btn--action"
                              onClick={() => openWebhookEdit(wh)}>Edit</button>
                            <button type="button" className="btn btn--revoke"
                              onClick={() => deleteWebhook(wh.id, wh.url)}>Remove</button>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                  {statsWhId === wh.id && (
                    <tr key={`${wh.id}-stats`}>
                      <td colSpan={5} className="audit-sub-cell">
                        {!statsMap[wh.id] ? (
                          <span className="td-muted">Loading stats…</span>
                        ) : (() => {
                          const s = statsMap[wh.id]
                          const rate = s.success_rate_pct ?? 0
                          return (
                            <div style={{ display: 'flex', gap: '1.5rem', flexWrap: 'wrap', padding: '0.25rem 0' }}>
                              <div><strong>{s.total_deliveries}</strong> <span className="td-muted">total</span></div>
                              <div><strong style={{ color: '#0d9488' }}>{s.successful}</strong> <span className="td-muted">ok</span></div>
                              <div><strong style={{ color: '#ea580c' }}>{s.failed}</strong> <span className="td-muted">failed</span></div>
                              <div>
                                <strong>{rate.toFixed(1)}%</strong> <span className="td-muted">success rate</span>
                                <div style={{ width: 120, height: 6, background: '#e5e7eb', borderRadius: 3, marginTop: 3 }}>
                                  <div style={{ width: `${Math.min(rate, 100)}%`, height: '100%', background: rate >= 95 ? '#0d9488' : '#ea580c', borderRadius: 3 }} />
                                </div>
                              </div>
                              {s.last_delivery_at && <div><span className="td-muted">last:</span> {fmtDate(s.last_delivery_at)}</div>}
                              {Object.keys(s.deliveries_by_event).length > 0 && (
                                <div><span className="td-muted">by event:</span> {Object.entries(s.deliveries_by_event).map(([ev, n]) => `${ev} ×${n}`).join(' · ')}</div>
                              )}
                            </div>
                          )
                        })()}
                      </td>
                    </tr>
                  )}
                  {deliveryWhId === wh.id && (
                    <tr key={`${wh.id}-deliveries`}>
                      <td colSpan={5} className="audit-sub-cell">
                        {deliveriesLoading ? (
                          <span className="td-muted">Loading…</span>
                        ) : deliveries.length === 0 ? (
                          <span className="td-muted">No deliveries recorded yet.</span>
                        ) : (
                          <table className="audit-table audit-table--inner">
                            <thead>
                              <tr>
                                <th>Event</th>
                                <th>Attempt</th>
                                <th>Status</th>
                                <th>Result</th>
                                <th>Time</th>
                                {isAdmin && <th></th>}
                              </tr>
                            </thead>
                            <tbody>
                              {deliveries.map(d => (
                                <tr key={d.id}>
                                  <td style={{fontFamily:'monospace',fontSize:'0.8rem'}}>{d.event}</td>
                                  <td>{d.attempt}</td>
                                  <td>{d.status_code ?? '—'}</td>
                                  <td>
                                    <span className={`badge ${d.success ? 'badge--enabled' : 'badge--rejected'}`}>
                                      {d.success ? 'ok' : 'failed'}
                                    </span>
                                    {d.error_message && <span className="td-subtle"> {d.error_message}</span>}
                                  </td>
                                  <td className="td-date">{fmtDate(d.delivered_at)}</td>
                                  {isAdmin && (
                                    <td>
                                      {!d.success && (
                                        <button type="button" className="btn btn--action"
                                          title="Re-deliver this payload"
                                          onClick={() => retryDelivery(d.id, wh.id)}>Retry</button>
                                      )}
                                    </td>
                                  )}
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )}
                      </td>
                    </tr>
                  )}
                </>
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
  const [instStats, setInstStats]         = useState<{ total_studies: number; by_status: Record<string, number>; by_modality: Record<string, number>; last_study_at?: string } | null>(null)

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
      const [projRes, statsRes] = await Promise.all([
        fetch(`/api/institutions/${instId}/projects`),
        fetch(`/api/institutions/${instId}/stats`),
      ])
      if (projRes.ok) setInstProjects(await projRes.json())
      else setInstProjects([])
      if (statsRes.ok) setInstStats(await statsRes.json())
      else setInstStats(null)
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

          {/* Institution stats */}
          {instStats && (
            <div className="pipeline-stats-bar">
              <span className="pipeline-stat"><strong>{instStats.total_studies}</strong> total studies</span>
              {Object.entries(instStats.by_status).map(([k, v]) => (
                <span key={k} className="pipeline-stat"><strong>{v}</strong> {k}</span>
              ))}
              {Object.keys(instStats.by_modality).length > 0 && (
                <span className="pipeline-stat">
                  {Object.entries(instStats.by_modality).map(([k, v]) => `${k}: ${v}`).join(', ')}
                </span>
              )}
              {instStats.last_study_at && (
                <span className="pipeline-stat">Last study: {new Date(instStats.last_study_at).toLocaleDateString()}</span>
              )}
            </div>
          )}

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

type ProjectPhiConfig = {
  project_id: string
  confidence_threshold: number
  min_text_length: number
  updated_at: string
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

  // PHI config editor state
  const [phiProjectId, setPhiProjectId] = useState<string | null>(null)
  const [phiConfig, setPhiConfig]       = useState<ProjectPhiConfig | null>(null)
  const [phiSaving, setPhiSaving]       = useState(false)
  const [phiError, setPhiError]         = useState<string | null>(null)

  // Retention policy editor state
  const [retentionProjectId, setRetentionProjectId]   = useState<string | null>(null)
  const [retentionDraft, setRetentionDraft]           = useState<string>('')
  const [retentionSaving, setRetentionSaving]         = useState(false)
  const [retentionError, setRetentionError]           = useState<string | null>(null)

  // SLA threshold editor state
  const [slaProjectId, setSlaProjectId]   = useState<string | null>(null)
  const [slaDraft, setSlaDraft]           = useState<string>('')
  const [slaSaving, setSlaSaving]         = useState(false)
  const [slaError, setSlaError]           = useState<string | null>(null)

  // Archive/restore state
  const [archiving, setArchiving] = useState<string | null>(null)

  // Compliance report modal
  const [complianceProjectId, setComplianceProjectId] = useState<string | null>(null)

  async function openPhiConfig(projectId: string) {
    setPhiProjectId(projectId)
    setPhiError(null)
    const res = await fetch(`/api/projects/${projectId}/phi-config`)
    if (res.ok) setPhiConfig(await res.json())
  }

  async function savePhiConfig() {
    if (!phiConfig || !phiProjectId) return
    setPhiSaving(true)
    setPhiError(null)
    try {
      const res = await fetch(`/api/projects/${phiProjectId}/phi-config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          confidence_threshold: phiConfig.confidence_threshold,
          min_text_length: phiConfig.min_text_length,
        }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setPhiConfig(await res.json())
      setPhiProjectId(null)
    } catch (err) {
      setPhiError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setPhiSaving(false)
    }
  }

  function openRetention(p: Project) {
    setRetentionProjectId(p.id)
    setRetentionDraft(p.retention_days != null ? String(p.retention_days) : '')
    setRetentionError(null)
  }

  async function saveRetention() {
    if (!retentionProjectId) return
    const days = retentionDraft.trim() === '' ? null : parseInt(retentionDraft, 10)
    if (days !== null && (isNaN(days) || days <= 0)) {
      setRetentionError('Must be a positive integer or leave blank to disable')
      return
    }
    setRetentionSaving(true)
    setRetentionError(null)
    try {
      const res = await fetch(`/api/projects/${retentionProjectId}/retention`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ retention_days: days }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setRetentionProjectId(null)
      fetchProjects()
    } catch (err) {
      setRetentionError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setRetentionSaving(false)
    }
  }

  function openSla(p: Project) {
    setSlaProjectId(p.id)
    setSlaDraft(p.stuck_threshold_minutes != null ? String(p.stuck_threshold_minutes) : '')
    setSlaError(null)
  }

  async function saveSla() {
    if (!slaProjectId) return
    const mins = slaDraft.trim() === '' ? null : parseInt(slaDraft, 10)
    if (mins !== null && (isNaN(mins) || mins <= 0)) {
      setSlaError('Must be a positive integer or leave blank to use the global default (60 min)')
      return
    }
    setSlaSaving(true)
    setSlaError(null)
    try {
      const res = await fetch(`/api/projects/${slaProjectId}/sla-threshold`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ stuck_threshold_minutes: mins }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setSlaProjectId(null)
      fetchProjects()
    } catch (err) {
      setSlaError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSlaSaving(false)
    }
  }

  const cloneProject = async (p: Project) => {
    const name = prompt(`New project name (default: "Copy of ${p.name}"):`)
    if (name === null) return // cancelled
    const body: Record<string, string> = {}
    if (name.trim()) body.name = name.trim()
    try {
      const res = await fetch(`/api/projects/${p.id}/clone`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`)
      alert(`Project cloned successfully as "${data.name}"`)
      fetchProjects()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Clone failed')
    }
  }

  const toggleArchive = async (p: Project) => {
    const action = p.archived ? 'restore' : 'archive'
    if (!confirm(`${p.archived ? 'Restore' : 'Archive'} project "${p.name}"?`)) return
    setArchiving(p.id)
    try {
      const res = await fetch(`/api/projects/${p.id}/${action}`, { method: 'POST' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      fetchProjects()
    } catch (err) {
      alert(err instanceof Error ? err.message : `Failed to ${action} project`)
    } finally {
      setArchiving(null)
    }
  }

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

        {/* Inline PHI config editor */}
        {phiProjectId && phiConfig && (
          <div className="routing-form" style={{ marginTop: '16px' }}>
            <h3>PHI Scan Config — {projects.find(p => p.id === phiProjectId)?.name}</h3>
            <div className="routing-section-sub" style={{ marginBottom: '12px' }}>
              Override the PHI detection sensitivity thresholds for this project.
              These values are read by the PHI detection service at scan time.
            </div>
            {phiError && <div className="form-error">{phiError}</div>}
            <div className="form-grid">
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.875rem' }}>
                Confidence threshold (0–1, default 0.4)
                <input className="form-input" type="number" min="0" max="1" step="0.05"
                  value={phiConfig.confidence_threshold}
                  onChange={e => setPhiConfig(c => c ? { ...c, confidence_threshold: parseFloat(e.target.value) } : c)} />
              </label>
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.875rem' }}>
                Min text length (chars, default 3)
                <input className="form-input" type="number" min="1" step="1"
                  value={phiConfig.min_text_length}
                  onChange={e => setPhiConfig(c => c ? { ...c, min_text_length: parseInt(e.target.value, 10) } : c)} />
              </label>
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={savePhiConfig} disabled={phiSaving}>
                {phiSaving ? 'Saving…' : 'Save'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setPhiProjectId(null)}>Cancel</button>
            </div>
          </div>
        )}

        {/* Inline retention policy editor */}
        {retentionProjectId && (
          <div className="routing-form" style={{ marginTop: '16px' }}>
            <h3>Retention Policy — {projects.find(p => p.id === retentionProjectId)?.name}</h3>
            <div className="routing-section-sub" style={{ marginBottom: '12px' }}>
              Approved studies older than this threshold are automatically marked as expired.
              Leave blank to keep studies indefinitely.
            </div>
            {retentionError && <div className="form-error">{retentionError}</div>}
            <div className="form-grid">
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.875rem' }}>
                Retention period (days, blank = unlimited)
                <input className="form-input" type="number" min="1" step="1" placeholder="e.g. 90"
                  value={retentionDraft}
                  onChange={e => setRetentionDraft(e.target.value)} />
              </label>
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={saveRetention} disabled={retentionSaving}>
                {retentionSaving ? 'Saving…' : 'Save'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setRetentionProjectId(null)}>Cancel</button>
            </div>
          </div>
        )}

        {/* Inline SLA threshold editor */}
        {slaProjectId && (
          <div className="routing-form" style={{ marginTop: '16px' }}>
            <h3>SLA Threshold — {projects.find(p => p.id === slaProjectId)?.name}</h3>
            <div className="routing-section-sub" style={{ marginBottom: '12px' }}>
              Studies idle beyond this threshold are surfaced as "stuck". Leave blank to use the global default (60 min).
            </div>
            {slaError && <div className="form-error">{slaError}</div>}
            <div className="form-grid">
              <label style={{ display: 'flex', flexDirection: 'column', gap: '4px', fontSize: '0.875rem' }}>
                Stuck threshold (minutes, blank = global default)
                <input className="form-input" type="number" min="1" step="1" placeholder="e.g. 120"
                  value={slaDraft}
                  onChange={e => setSlaDraft(e.target.value)} />
              </label>
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={saveSla} disabled={slaSaving}>
                {slaSaving ? 'Saving…' : 'Save'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setSlaProjectId(null)}>Cancel</button>
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
                <th>Retention</th>
                <th>SLA</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {projects.map(p => (
                <tr key={p.id}>
                  <td>
                    <div className="routing-name">
                      {p.name}
                      {p.archived && <span className="badge badge--neutral" style={{marginLeft:'6px'}}>archived</span>}
                    </div>
                    {p.description && <div className="routing-desc">{p.description}</div>}
                  </td>
                  <td><code className="inst-slug">{p.slug}</code></td>
                  <td>
                    {p.default_anon_profile_id
                      ? <span className="badge badge--enabled">profile set</span>
                      : <span className="routing-desc">none</span>}
                  </td>
                  <td>
                    {p.retention_days != null
                      ? <span className="badge badge--status">{p.retention_days}d</span>
                      : <span className="routing-desc">unlimited</span>}
                  </td>
                  <td>
                    {p.stuck_threshold_minutes != null
                      ? <span className="badge badge--status">{p.stuck_threshold_minutes}m</span>
                      : <span className="routing-desc">60m</span>}
                  </td>
                  <td className="td-date">{fmtDate(p.created_at)}</td>
                  <td>
                    {isAdmin && (
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEdit(p)}>Edit</button>
                        <button type="button" className="btn btn--action"
                          title="Dispatch export forwarding for all approved studies in this project"
                          onClick={async () => {
                            const res = await fetch(`/api/projects/${p.id}/export-batch`, { method: 'POST' })
                            const data = await res.json()
                            alert(res.ok
                              ? `Export batch dispatched: ${data.dispatched} ${data.dispatched === 1 ? 'study' : 'studies'}`
                              : `Export batch failed: ${data.error}`)
                          }}>
                          Export Batch
                        </button>
                        <button type="button" className="btn btn--action"
                          title="Configure PHI scan sensitivity for this project"
                          onClick={() => openPhiConfig(p.id)}>
                          PHI Config
                        </button>
                        <button type="button" className="btn btn--action"
                          title="Set study retention period for this project"
                          onClick={() => openRetention(p)}>
                          Retention
                        </button>
                        <button type="button" className="btn btn--action"
                          title="Set per-project SLA threshold for stuck studies"
                          onClick={() => openSla(p)}>
                          SLA
                        </button>
                        <button type="button"
                          className={p.archived ? 'btn btn--approve' : 'btn btn--action'}
                          title={p.archived ? 'Restore project' : 'Archive project'}
                          disabled={archiving === p.id}
                          onClick={() => toggleArchive(p)}>
                          {archiving === p.id ? '…' : p.archived ? 'Restore' : 'Archive'}
                        </button>
                        <button type="button" className="btn btn--action"
                          title="Duplicate this project with all settings (routing rules, profiles, templates)"
                          onClick={() => cloneProject(p)}>
                          Clone
                        </button>
                        <button type="button" className="btn btn--action"
                          title="View compliance metrics for this project"
                          onClick={() => setComplianceProjectId(p.id)}>
                          Compliance
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {complianceProjectId && (
        <ComplianceReportPanel
          projectId={complianceProjectId}
          onClose={() => setComplianceProjectId(null)}
        />
      )}
    </div>
  )
}

// ── Federation Panel ──────────────────────────────────────────────────────────

// ── Invite Codes Panel ────────────────────────────────────────────────────────

type InviteCode = {
  id: string
  code: string
  label: string
  enabled: boolean
  created_at: string
  used_at?: string
  used_by_ip?: string
}

type InviteRequest = {
  id: string
  name: string
  email: string
  org: string
  message: string
  status: 'pending' | 'approved' | 'denied'
  ip?: string
  created_at: string
  reviewed_at?: string
  reviewed_by?: string
  invite_code_id?: string
}

function InviteCodesPanel() {
  const [subTab, setSubTab]       = useState<'codes' | 'requests'>('codes')

  // ── Codes state ──────────────────────────────────────────────────────────
  const [codes, setCodes]         = useState<InviteCode[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)
  const [showForm, setShowForm]   = useState(false)
  const [formLabel, setFormLabel] = useState('')
  const [saving, setSaving]       = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [newCode, setNewCode]     = useState<string | null>(null)
  const [copied, setCopied]       = useState<string | null>(null)

  // ── Requests state ───────────────────────────────────────────────────────
  const [requests, setRequests]     = useState<InviteRequest[]>([])
  const [reqLoading, setReqLoading] = useState(false)
  const [reqError, setReqError]     = useState<string | null>(null)
  const [reqFilter, setReqFilter]   = useState<'all' | 'pending' | 'approved' | 'denied'>('pending')
  const [reqActing, setReqActing]   = useState<Record<string, boolean>>({})

  const load = useCallback(async () => {
    setLoading(true); setError(null)
    try {
      const res = await fetch('/api/invite-codes')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setCodes(data.codes ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally { setLoading(false) }
  }, [])

  const loadRequests = useCallback(async () => {
    setReqLoading(true); setReqError(null)
    try {
      const qs = reqFilter !== 'all' ? `?status=${reqFilter}` : ''
      const res = await fetch(`/api/invite/requests${qs}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      setRequests(data.requests ?? [])
    } catch (err) {
      setReqError(err instanceof Error ? err.message : 'Failed to load')
    } finally { setReqLoading(false) }
  }, [reqFilter])

  useEffect(() => { load() }, [load])
  useEffect(() => { if (subTab === 'requests') loadRequests() }, [subTab, loadRequests])

  async function create() {
    if (!formLabel.trim()) { setFormError('Label is required'); return }
    setSaving(true); setFormError(null); setNewCode(null)
    try {
      const res = await fetch('/api/invite-codes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ label: formLabel.trim() }),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Create failed') }
      const data: InviteCode = await res.json()
      setNewCode(data.code)
      setFormLabel(''); setShowForm(false)
      load()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Create failed')
    } finally { setSaving(false) }
  }

  async function revoke(ic: InviteCode) {
    if (!confirm(`Revoke invite code "${ic.code}" (${ic.label})? The recipient will no longer be able to use it.`)) return
    await fetch(`/api/invite-codes/${ic.id}/revoke`, { method: 'POST' })
    load()
  }

  async function del(ic: InviteCode) {
    if (!confirm(`Permanently delete invite code "${ic.code}" (${ic.label})?`)) return
    await fetch(`/api/invite-codes/${ic.id}`, { method: 'DELETE' })
    setNewCode(null)
    load()
  }

  async function approveRequest(id: string) {
    setReqActing(prev => ({ ...prev, [id]: true }))
    try {
      const res = await fetch(`/api/invite/requests/${id}/approve`, { method: 'POST' })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Approve failed') }
      loadRequests()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Approve failed')
    } finally {
      setReqActing(prev => ({ ...prev, [id]: false }))
    }
  }

  async function denyRequest(id: string, email: string) {
    if (!confirm(`Deny invite request from ${email}?`)) return
    setReqActing(prev => ({ ...prev, [id]: true }))
    try {
      await fetch(`/api/invite/requests/${id}/deny`, { method: 'POST' })
      loadRequests()
    } finally {
      setReqActing(prev => ({ ...prev, [id]: false }))
    }
  }

  function copy(text: string, key: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(key)
      setTimeout(() => setCopied(null), 2000)
    })
  }

  const SITE = 'https://aegisimaging.ai'

  if (loading) return <div className="state-loading">Loading…</div>
  if (error)   return <div className="state-error">{error}</div>

  return (
    <div className="routing-panel">
      {/* Sub-tab switcher */}
      <div style={{ display: 'flex', gap: '4px', padding: '0 0 16px 0', borderBottom: '1px solid #e2e8f0', marginBottom: '16px' }}>
        {(['codes', 'requests'] as const).map(t => (
          <button
            key={t}
            type="button"
            onClick={() => setSubTab(t)}
            style={{
              padding: '6px 16px', borderRadius: '6px', border: 'none', cursor: 'pointer', fontSize: '0.85rem', fontWeight: 600,
              background: subTab === t ? '#0d9488' : 'transparent',
              color: subTab === t ? '#fff' : '#64748b',
            }}
          >
            {t === 'codes' ? 'Invite Codes' : 'Access Requests'}
          </button>
        ))}
      </div>

      {/* ── Access Requests tab ──────────────────────────────────────────── */}
      {subTab === 'requests' && (
        <div className="routing-section">
          <div className="routing-section-header">
            <div>
              <div className="routing-section-title">Access Requests</div>
              <div className="routing-section-sub">
                Users who submitted an access request form. Approve to generate and email an invite code; deny to reject.
              </div>
            </div>
            <div className="actions-cell">
              <select
                value={reqFilter}
                onChange={e => setReqFilter(e.target.value as typeof reqFilter)}
                style={{ padding: '6px 10px', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.85rem', background: '#fff' }}
              >
                <option value="pending">Pending</option>
                <option value="approved">Approved</option>
                <option value="denied">Denied</option>
                <option value="all">All</option>
              </select>
              <button type="button" className="btn-refresh" onClick={loadRequests}>Refresh</button>
            </div>
          </div>

          {reqLoading && <div className="state-loading">Loading…</div>}
          {reqError   && <div className="state-error">{reqError}</div>}
          {!reqLoading && !reqError && requests.length === 0 && (
            <div className="state-empty">No {reqFilter !== 'all' ? reqFilter : ''} requests.</div>
          )}
          {!reqLoading && !reqError && requests.length > 0 && (
            <table className="routing-table">
              <thead>
                <tr>
                  <th>Requester</th>
                  <th>Org</th>
                  <th>Message</th>
                  <th>Submitted</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {requests.map(req => (
                  <tr key={req.id}>
                    <td>
                      <div style={{ fontWeight: 600, fontSize: '0.85rem' }}>{req.name}</div>
                      <div style={{ fontSize: '0.8rem', color: '#64748b' }}>{req.email}</div>
                    </td>
                    <td style={{ fontSize: '0.85rem' }}>{req.org || <span style={{ color: '#94a3b8' }}>—</span>}</td>
                    <td style={{ fontSize: '0.8rem', maxWidth: 220, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                      {req.message || <span style={{ color: '#94a3b8' }}>—</span>}
                    </td>
                    <td style={{ fontSize: '0.8rem', color: '#94a3b8', whiteSpace: 'nowrap' }}>
                      {new Date(req.created_at).toLocaleDateString()}
                    </td>
                    <td>
                      <span style={{
                        display: 'inline-block', padding: '2px 8px', borderRadius: '12px',
                        fontSize: '0.75rem', fontWeight: 600,
                        background: req.status === 'approved' ? '#ccfbf1' : req.status === 'denied' ? '#ffedd5' : '#f1f5f9',
                        color:      req.status === 'approved' ? '#0f766e' : req.status === 'denied' ? '#9a3412' : '#475569',
                      }}>
                        {req.status}
                      </span>
                    </td>
                    <td>
                      {req.status === 'pending' ? (
                        <div className="actions-cell">
                          <button
                            type="button"
                            className="btn-sm"
                            disabled={reqActing[req.id]}
                            onClick={() => approveRequest(req.id)}
                            style={{ background: '#0d9488', color: '#fff', border: 'none' }}
                          >
                            {reqActing[req.id] ? '…' : 'Approve'}
                          </button>
                          <button
                            type="button"
                            className="btn-sm btn-warning"
                            disabled={reqActing[req.id]}
                            onClick={() => denyRequest(req.id, req.email)}
                          >
                            Deny
                          </button>
                        </div>
                      ) : (
                        <span style={{ fontSize: '0.75rem', color: '#94a3b8' }}>
                          {req.reviewed_at ? new Date(req.reviewed_at).toLocaleDateString() : '—'}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* ── Invite Codes tab ─────────────────────────────────────────────── */}
      {subTab === 'codes' && <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Invite Codes</div>
            <div className="routing-section-sub">
              Per-person codes for landing page access. Each code is unique and can be individually revoked.
              Share the direct link (<code style={{ fontSize: '0.8rem' }}>{SITE}/?invite=CODE</code>) for one-click admission.
            </div>
          </div>
          <div className="actions-cell">
            <button type="button" className="btn-refresh" onClick={load}>Refresh</button>
            <button type="button" className="btn-primary" onClick={() => { setShowForm(true); setNewCode(null) }}>
              + New code
            </button>
          </div>
        </div>

        {showForm && (
          <div className="routing-form">
            <h3>New invite code</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input
                className="form-input"
                placeholder="Label (e.g. Dr. Jane Smith) *"
                value={formLabel}
                onChange={e => setFormLabel(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && create()}
                autoFocus
              />
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={create} disabled={saving}>
                {saving ? 'Creating…' : 'Generate code'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {newCode && (
          <div className="routing-form" style={{ background: '#f0fdfa', border: '1px solid #99f6e4' }}>
            <strong style={{ color: '#0f766e' }}>New invite code — share with your recipient:</strong>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '8px' }}>
              <code style={{ background: '#ccfbf1', padding: '6px 12px', borderRadius: '6px', fontSize: '0.95rem', letterSpacing: '0.1em', flex: 1 }}>
                {newCode}
              </code>
              <button type="button" className="btn-secondary" onClick={() => copy(newCode, 'code')}>
                {copied === 'code' ? 'Copied!' : 'Copy code'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => copy(`${SITE}/?invite=${newCode}`, 'link')}>
                {copied === 'link' ? 'Copied!' : 'Copy link'}
              </button>
            </div>
          </div>
        )}

        {codes.length === 0 && !showForm ? (
          <div className="state-empty">No invite codes yet. Create one to grant landing page access.</div>
        ) : codes.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Label</th>
                <th>Status</th>
                <th>Created</th>
                <th>Used</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {codes.map(ic => (
                <tr key={ic.id}>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                      <code style={{ fontSize: '0.85rem', letterSpacing: '0.08em' }}>{ic.code}</code>
                      <button
                        type="button"
                        className="btn-sm"
                        title="Copy code"
                        onClick={() => copy(ic.code, ic.id + '-code')}
                        style={{ fontSize: '0.7rem', padding: '2px 6px' }}
                      >
                        {copied === ic.id + '-code' ? '✓' : 'Copy'}
                      </button>
                      <button
                        type="button"
                        className="btn-sm"
                        title="Copy invite link"
                        onClick={() => copy(`${SITE}/?invite=${ic.code}`, ic.id + '-link')}
                        style={{ fontSize: '0.7rem', padding: '2px 6px' }}
                      >
                        {copied === ic.id + '-link' ? '✓' : 'Link'}
                      </button>
                    </div>
                  </td>
                  <td>{ic.label || <span style={{ color: '#64748b' }}>—</span>}</td>
                  <td>
                    <span style={{
                      display: 'inline-block',
                      padding: '2px 8px',
                      borderRadius: '12px',
                      fontSize: '0.75rem',
                      fontWeight: 600,
                      background: ic.enabled ? '#ccfbf1' : '#ffedd5',
                      color: ic.enabled ? '#0f766e' : '#9a3412',
                    }}>
                      {ic.enabled ? 'Active' : 'Revoked'}
                    </span>
                  </td>
                  <td style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                    {new Date(ic.created_at).toLocaleDateString()}
                  </td>
                  <td style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                    {ic.used_at
                      ? <span title={ic.used_by_ip ?? ''}>{new Date(ic.used_at).toLocaleDateString()}{ic.used_by_ip ? ` (${ic.used_by_ip})` : ''}</span>
                      : <span style={{ color: '#64748b' }}>Unused</span>}
                  </td>
                  <td>
                    <div className="actions-cell">
                      {ic.enabled && (
                        <button type="button" className="btn-sm btn-warning" onClick={() => revoke(ic)}>
                          Revoke
                        </button>
                      )}
                      <button type="button" className="btn-sm btn-danger" onClick={() => del(ic)}>
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>}
    </div>
  )
}

type FederationPeer = {
  id: string
  name: string
  slug: string
  api_url: string
  enabled: boolean
  notes: string
  created_at: string
  updated_at: string
}

const EMPTY_PEER: Omit<FederationPeer, 'id' | 'created_at' | 'updated_at'> = {
  name: '', slug: '', api_url: '', enabled: true, notes: '',
}

function FederationPanel({ isAdmin }: { isAdmin: boolean }) {
  const [peers, setPeers]         = useState<FederationPeer[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)
  const [form, setForm]           = useState<Omit<FederationPeer, 'id' | 'created_at' | 'updated_at'>>(EMPTY_PEER)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [showForm, setShowForm]   = useState(false)
  const [saving, setSaving]       = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const fetchPeers = useCallback(async () => {
    setLoading(true); setError(null)
    try {
      const res = await fetch('/api/federation-peers')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setPeers(await res.json())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchPeers() }, [fetchPeers])

  function openNew() { setForm(EMPTY_PEER); setEditingId(null); setFormError(null); setShowForm(true) }
  function openEdit(p: FederationPeer) {
    setForm({ name: p.name, slug: p.slug, api_url: p.api_url, enabled: p.enabled, notes: p.notes })
    setEditingId(p.id); setFormError(null); setShowForm(true)
  }

  async function save() {
    if (!form.name) { setFormError('Name is required'); return }
    if (!form.api_url) { setFormError('API URL is required'); return }
    setSaving(true); setFormError(null)
    try {
      const url = editingId ? `/api/federation-peers/${editingId}` : '/api/federation-peers'
      const method = editingId ? 'PUT' : 'POST'
      const res = await fetch(url, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(form) })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Save failed') }
      setShowForm(false); setEditingId(null); fetchPeers()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Save failed')
    } finally { setSaving(false) }
  }

  async function deletePeer(p: FederationPeer) {
    if (!confirm(`Delete federation peer "${p.name}"?`)) return
    await fetch(`/api/federation-peers/${p.id}`, { method: 'DELETE' })
    fetchPeers()
  }

  if (loading) return <div className="state-loading">Loading…</div>
  if (error)   return <div className="state-error">{error}</div>

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">Federation Peers</div>
            <div className="routing-section-sub">
              Trusted remote AEGIS instances for future cross-tenant study federation.
              No data flows between peers yet — this is a configuration stub.
            </div>
          </div>
          <div className="actions-cell">
            <button type="button" className="btn-refresh" onClick={fetchPeers}>Refresh</button>
            {isAdmin && <button type="button" className="btn-primary" onClick={openNew}>+ Add peer</button>}
          </div>
        </div>

        {isAdmin && showForm && (
          <div className="routing-form">
            <h3>{editingId ? 'Edit peer' : 'New federation peer'}</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" placeholder="Name *"
                value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
              <input className="form-input" placeholder="Slug (auto-generated)"
                value={form.slug} onChange={e => setForm(f => ({ ...f, slug: e.target.value }))} />
              <input className="form-input form-input--wide" placeholder="API URL * (e.g. https://peer.example.com)"
                value={form.api_url} onChange={e => setForm(f => ({ ...f, api_url: e.target.value }))} />
              <input className="form-input form-input--wide" placeholder="Notes"
                value={form.notes} onChange={e => setForm(f => ({ ...f, notes: e.target.value }))} />
            </div>
            {editingId && (
              <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.875rem', marginTop: '8px' }}>
                <input type="checkbox" checked={form.enabled}
                  onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))} />
                Enabled
              </label>
            )}
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={save} disabled={saving}>
                {saving ? 'Saving…' : editingId ? 'Save changes' : 'Add peer'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {peers.length === 0 && !showForm ? (
          <div className="state-empty">No federation peers configured.</div>
        ) : peers.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Peer</th>
                <th>Slug</th>
                <th>API URL</th>
                <th>Status</th>
                <th>Added</th>
                {isAdmin && <th>Actions</th>}
              </tr>
            </thead>
            <tbody>
              {peers.map(p => (
                <tr key={p.id}>
                  <td>
                    <div className="routing-name">{p.name}</div>
                    {p.notes && <div className="routing-desc">{p.notes}</div>}
                  </td>
                  <td><code className="inst-slug">{p.slug}</code></td>
                  <td><a href={p.api_url} target="_blank" rel="noopener noreferrer">{p.api_url}</a></td>
                  <td>
                    {p.enabled
                      ? <span className="badge badge--enabled">enabled</span>
                      : <span className="badge badge--disabled">disabled</span>}
                  </td>
                  <td className="td-date">{fmtDate(p.created_at)}</td>
                  {isAdmin && (
                    <td>
                      <div className="actions-cell">
                        <button type="button" className="btn btn--edit" onClick={() => openEdit(p)}>Edit</button>
                        <button type="button" className="btn btn--delete" onClick={() => deletePeer(p)}>Delete</button>
                      </div>
                    </td>
                  )}
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

function DimseOpsPanel() {
  const [summary, setSummary] = useState<DimseRetrySummary | null>(null)
  const [details, setDetails] = useState<DimseRetryDetails | null>(null)
  const [alerts, setAlerts] = useState<DimseRetryAlert[]>([])
  const [actions, setActions] = useState<DimseOperatorAction[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [statusMessage, setStatusMessage] = useState('')
  const [busyAction, setBusyAction] = useState('')
  const [studyFilter, setStudyFilter] = useState('')
  const [bulkLimit, setBulkLimit] = useState(100)

  const readErrorBody = useCallback(async (res: Response) => {
    try {
      const body = await res.json()
      if (typeof body?.error === 'string') return body.error
      if (typeof body?.detail === 'string') return body.detail
      if (typeof body?.message === 'string') return body.message
    } catch {
      // ignore json parse failures
    }
    return `HTTP ${res.status}`
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({ limit: '200', sort: 'age_desc' })
      if (studyFilter.trim()) params.set('study_instance_uid', studyFilter.trim())
      const [summaryRes, detailsRes, alertsRes, actionsRes] = await Promise.all([
        fetch('/api/dimse/retry/summary'),
        fetch(`/api/dimse/retry/details?${params.toString()}`),
        fetch('/api/dimse/retry/alerts?limit=20'),
        fetch('/api/dimse/retry/actions?limit=20'),
      ])

      if (!summaryRes.ok) throw new Error(await readErrorBody(summaryRes))
      if (!detailsRes.ok) throw new Error(await readErrorBody(detailsRes))
      if (!alertsRes.ok) throw new Error(await readErrorBody(alertsRes))
      if (!actionsRes.ok) throw new Error(await readErrorBody(actionsRes))

      const summaryBody = await summaryRes.json()
      const detailsBody = await detailsRes.json()
      const alertsBody = await alertsRes.json()
      const actionsBody = await actionsRes.json()
      setSummary(summaryBody.ingest_retry as DimseRetrySummary)
      setDetails(detailsBody.ingest_retry as DimseRetryDetails)
      setAlerts((alertsBody.alerts?.items ?? []) as DimseRetryAlert[])
      setActions((actionsBody.actions?.items ?? []) as DimseOperatorAction[])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load DIMSE retry state')
    } finally {
      setLoading(false)
    }
  }, [readErrorBody, studyFilter])

  useEffect(() => { load() }, [load])

  const runAction = useCallback(async (
    label: string,
    path: string,
    confirmMessage?: string,
  ) => {
    if (confirmMessage && !confirm(confirmMessage)) return
    setBusyAction(label)
    setStatusMessage('')
    try {
      const res = await fetch(path, { method: 'POST' })
      if (!res.ok) throw new Error(await readErrorBody(res))
      setStatusMessage(`${label} completed`)
      await load()
    } catch (err) {
      setStatusMessage(`${label} failed: ${err instanceof Error ? err.message : 'unknown error'}`)
    } finally {
      setBusyAction('')
    }
  }, [load, readErrorBody])

  const disableBulk = busyAction.length > 0
  const limit = Math.max(1, bulkLimit)
  const snapshot = summary?.snapshot

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">DIMSE Retry Operations</div>
            <div className="routing-section-sub">
              Monitor and control DIMSE ingest pending/dead-letter queues.
            </div>
          </div>
          <button type="button" className="btn-refresh" onClick={load} disabled={loading || disableBulk}>
            Refresh
          </button>
        </div>

        <p className="routing-hint">
          This panel proxies <code>/ingest/retry*</code> controls from the DIMSE sidecar through the API.
        </p>

        {error && <div className="state-error">{error}</div>}
        {loading && <div className="state-loading">Loading DIMSE operations…</div>}

        {!loading && summary && (
          <>
            <div className="dimse-stats-grid">
              <div className="stat-card">
                <div className="stat-number stat-number--neutral">{snapshot?.pending ?? 0}</div>
                <div className="stat-label">Pending</div>
              </div>
              <div className="stat-card">
                <div className="stat-number stat-number--error">{snapshot?.dead_letter ?? 0}</div>
                <div className="stat-label">Dead-letter</div>
              </div>
              <div className="stat-card">
                <div className="stat-number stat-number--warning">{summary.pending_due_now}</div>
                <div className="stat-label">Due now</div>
              </div>
              <div className="stat-card">
                <div className="stat-number stat-number--success">{summary.queue_utilization_percent}%</div>
                <div className="stat-label">Queue usage</div>
              </div>
              <div className="stat-card">
                <div className="stat-number stat-number--neutral">{snapshot?.pending_oldest_age_seconds ?? 0}s</div>
                <div className="stat-label">Oldest pending age</div>
              </div>
              <div className="stat-card">
                <div className="stat-number stat-number--neutral">{snapshot?.dead_letter_oldest_age_seconds ?? 0}s</div>
                <div className="stat-label">Oldest dead-letter age</div>
              </div>
            </div>

            <div className="routing-form">
              <h3>Bulk Controls</h3>
              <div className="form-grid">
                <input
                  className="form-input"
                  type="text"
                  placeholder="Filter by StudyInstanceUID (optional)"
                  value={studyFilter}
                  onChange={(e) => setStudyFilter(e.target.value)}
                />
                <input
                  className="form-input"
                  type="number"
                  min={1}
                  max={50000}
                  value={bulkLimit}
                  onChange={(e) => setBulkLimit(Number(e.target.value) || 1)}
                />
              </div>
              <div className="form-row form-row--actions">
                <button
                  type="button"
                  className="btn btn--secondary"
                  disabled={disableBulk}
                  onClick={() => runAction('Process due', '/api/dimse/retry/process')}
                >
                  {busyAction === 'Process due' ? 'Processing…' : 'Process due'}
                </button>
                <button
                  type="button"
                  className="btn btn--secondary"
                  disabled={disableBulk}
                  onClick={() => runAction('Process all', `/api/dimse/retry/process-all?limit=${limit}`)}
                >
                  {busyAction === 'Process all' ? 'Processing…' : `Process all (${limit})`}
                </button>
                <button
                  type="button"
                  className="btn btn--secondary"
                  disabled={disableBulk}
                  onClick={() => runAction('Replay dead-letter', `/api/dimse/retry/replay?limit=${limit}`)}
                >
                  {busyAction === 'Replay dead-letter' ? 'Replaying…' : `Replay dead-letter (${limit})`}
                </button>
                <button
                  type="button"
                  className="btn btn--secondary"
                  disabled={disableBulk}
                  onClick={() => runAction(
                    'Clear pending',
                    `/api/dimse/retry/clear-pending?limit=${limit}`,
                    `Clear up to ${limit} pending retry entries?`,
                  )}
                >
                  {busyAction === 'Clear pending' ? 'Clearing…' : `Clear pending (${limit})`}
                </button>
                <button
                  type="button"
                  className="btn btn--revoke"
                  disabled={disableBulk}
                  onClick={() => runAction(
                    'Clear dead-letter',
                    `/api/dimse/retry/clear-dead-letter?limit=${limit}`,
                    `Clear up to ${limit} dead-letter entries?`,
                  )}
                >
                  {busyAction === 'Clear dead-letter' ? 'Clearing…' : `Clear dead-letter (${limit})`}
                </button>
                <button type="button" className="btn btn--secondary" onClick={load} disabled={disableBulk}>
                  Apply filter
                </button>
              </div>
              {statusMessage && <div className="dimse-status-message">{statusMessage}</div>}
            </div>
          </>
        )}
      </div>

      {!loading && details && (
        <div className="routing-section">
          <div className="routing-section-header">
            <div>
              <div className="routing-section-title">Pending Queue ({details.pending_total})</div>
            </div>
          </div>

          {details.pending_items.length === 0 ? (
            <div className="state-empty">No pending retry items.</div>
          ) : (
            <table className="routing-table">
              <thead>
                <tr>
                  <th>Study UID</th>
                  <th>Attempts</th>
                  <th>Next attempt</th>
                  <th>Age</th>
                  <th>Counts</th>
                  <th>Last error</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {details.pending_items.map(item => (
                  <tr key={`pending-${item.study_instance_uid}`}>
                    <td><code>{uidShort(item.study_instance_uid)}</code></td>
                    <td>{item.attempts}</td>
                    <td>
                      {item.next_attempt_at > 0 ? `${item.seconds_until_next_attempt}s` : 'n/a'}
                      {item.next_attempt_at > 0 && (
                        <div className="routing-desc">{fmtDate(new Date(item.next_attempt_at * 1000).toISOString())}</div>
                      )}
                    </td>
                    <td>{item.age_seconds}s</td>
                    <td>{item.file_count} files / {item.series_count} series</td>
                    <td><code>{item.last_error || '—'}</code></td>
                    <td>
                      <div className="actions-cell">
                        <button
                          type="button"
                          className="btn btn--secondary"
                          disabled={disableBulk}
                          onClick={() => runAction(
                            'Process study',
                            `/api/dimse/retry/process/${encodeURIComponent(item.study_instance_uid)}`,
                          )}
                        >
                          Process
                        </button>
                        <button
                          type="button"
                          className="btn btn--revoke"
                          disabled={disableBulk}
                          onClick={() => runAction(
                            'Clear pending study',
                            `/api/dimse/retry/clear-pending/${encodeURIComponent(item.study_instance_uid)}`,
                            `Clear pending entry for ${item.study_instance_uid}?`,
                          )}
                        >
                          Clear
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {!loading && details && (
        <div className="routing-section">
          <div className="routing-section-header">
            <div>
              <div className="routing-section-title">Dead-letter Queue ({details.dead_letter_total})</div>
            </div>
          </div>

          {details.dead_letter_items.length === 0 ? (
            <div className="state-empty">No dead-letter items.</div>
          ) : (
            <table className="routing-table">
              <thead>
                <tr>
                  <th>Study UID</th>
                  <th>Attempts</th>
                  <th>Dead-letter age</th>
                  <th>Counts</th>
                  <th>Last error</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {details.dead_letter_items.map(item => (
                  <tr key={`dead-${item.study_instance_uid}`}>
                    <td><code>{uidShort(item.study_instance_uid)}</code></td>
                    <td>{item.attempts}</td>
                    <td>{item.dead_letter_age_seconds}s</td>
                    <td>{item.file_count} files / {item.series_count} series</td>
                    <td><code>{item.last_error || '—'}</code></td>
                    <td>
                      <div className="actions-cell">
                        <button
                          type="button"
                          className="btn btn--secondary"
                          disabled={disableBulk}
                          onClick={() => runAction(
                            'Replay dead-letter study',
                            `/api/dimse/retry/replay/${encodeURIComponent(item.study_instance_uid)}`,
                          )}
                        >
                          Replay
                        </button>
                        <button
                          type="button"
                          className="btn btn--revoke"
                          disabled={disableBulk}
                          onClick={() => runAction(
                            'Clear dead-letter study',
                            `/api/dimse/retry/clear-dead-letter/${encodeURIComponent(item.study_instance_uid)}`,
                            `Clear dead-letter entry for ${item.study_instance_uid}?`,
                          )}
                        >
                          Clear
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {!loading && (
        <div className="routing-section">
          <div className="routing-section-header">
            <div className="routing-section-title">Recent Retry Alerts</div>
          </div>
          {alerts.length === 0 ? (
            <div className="state-empty">No recent retry threshold alerts.</div>
          ) : (
            <table className="routing-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Condition</th>
                  <th>Message</th>
                  <th>Snapshot</th>
                </tr>
              </thead>
              <tbody>
                {alerts.map((alert, idx) => (
                  <tr key={`${alert.condition}-${alert.timestamp}-${idx}`}>
                    <td className="td-date">{fmtDate(new Date(alert.timestamp * 1000).toISOString())}</td>
                    <td><code>{alert.condition}</code></td>
                    <td>{alert.message}</td>
                    <td>
                      <pre className="detail-json">{JSON.stringify(alert.snapshot ?? {}, null, 2)}</pre>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {!loading && (
        <div className="routing-section">
          <div className="routing-section-header">
            <div className="routing-section-title">Recent Operator Actions</div>
          </div>
          {actions.length === 0 ? (
            <div className="state-empty">No recent retry-control actions.</div>
          ) : (
            <table className="routing-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Action</th>
                  <th>Detail</th>
                </tr>
              </thead>
              <tbody>
                {actions.map((action, idx) => (
                  <tr key={`${action.action}-${action.created_at}-${idx}`}>
                    <td className="td-date">{fmtDate(action.created_at)}</td>
                    <td><code className={`routing-action routing-action--${action.action}`}>{action.action}</code></td>
                    <td>
                      <pre className="detail-json">{JSON.stringify(action.detail ?? {}, null, 2)}</pre>
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

// ── API Keys Panel ────────────────────────────────────────────────────────────

function APIKeysPanel() {
  const [keys, setKeys]       = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)
  const [showForm, setShowForm]   = useState(false)
  const [formName, setFormName]   = useState('')
  const [formExpiry, setFormExpiry] = useState('')
  const [saving, setSaving]       = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [newKeyValue, setNewKeyValue] = useState<string | null>(null)
  const [newKeyLabel, setNewKeyLabel] = useState<'created' | 'rotated'>('created')
  const [rotatingId, setRotatingId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/api-keys')
      if (!res.ok) throw new Error('Failed to load')
      setKeys((await res.json()) ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  async function create() {
    if (!formName.trim()) { setFormError('Name is required'); return }
    setSaving(true)
    setFormError(null)
    setNewKeyValue(null)
    const body: Record<string, unknown> = { name: formName.trim() }
    if (formExpiry) body.expires_at = new Date(formExpiry).toISOString()
    try {
      const res = await fetch('/api/api-keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Create failed') }
      const data = await res.json()
      setNewKeyLabel('created')
      setNewKeyValue(data.key)
      setFormName('')
      setFormExpiry('')
      setShowForm(false)
      load()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Create failed')
    } finally {
      setSaving(false)
    }
  }

  async function toggle(key: APIKey) {
    const action = key.enabled ? 'disable' : 'enable'
    await fetch(`/api/api-keys/${key.id}/${action}`, { method: 'PATCH' })
    load()
  }

  async function rotate(key: APIKey) {
    if (!confirm(`Rotate API key "${key.name}"? The current key value will stop working immediately.`)) return
    setRotatingId(key.id)
    try {
      const res = await fetch(`/api/api-keys/${key.id}/rotate`, { method: 'POST' })
      if (!res.ok) { const b = await res.json(); throw new Error(b.error ?? 'Rotate failed') }
      const data = await res.json()
      setNewKeyLabel('rotated')
      setNewKeyValue(data.key)
      load()
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Rotate failed')
    } finally {
      setRotatingId(null)
    }
  }

  async function del(key: APIKey) {
    if (!confirm(`Permanently delete API key "${key.name}"? This cannot be undone.`)) return
    await fetch(`/api/api-keys/${key.id}`, { method: 'DELETE' })
    setNewKeyValue(null)
    load()
  }

  return (
    <div className="routing-panel">
      <div className="routing-section">
        <div className="routing-section-header">
          <div>
            <div className="routing-section-title">API Keys</div>
            <div className="routing-section-sub">
              Machine-to-machine credentials for programmatic API access. The raw key is shown only once at creation.
            </div>
          </div>
          <button type="button" className="btn-primary" onClick={() => { setShowForm(true); setNewKeyValue(null) }}>
            + New API key
          </button>
        </div>

        {newKeyValue && (
          <div className="routing-form" style={{ background: '#f0fdf4', border: '1px solid #bbf7d0' }}>
            <strong style={{ color: '#166534' }}>API key {newKeyLabel} — copy it now, it will not be shown again:</strong>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginTop: '8px' }}>
              <code style={{ background: '#dcfce7', padding: '6px 12px', borderRadius: '6px', fontSize: '0.85rem', wordBreak: 'break-all', flex: 1 }}>
                {newKeyValue}
              </code>
              <button type="button" className="btn-secondary"
                onClick={() => navigator.clipboard.writeText(newKeyValue!)}>Copy</button>
            </div>
            <button type="button" className="btn-secondary" style={{ marginTop: '8px' }} onClick={() => setNewKeyValue(null)}>Dismiss</button>
          </div>
        )}

        {showForm && (
          <div className="routing-form">
            <h3>New API key</h3>
            {formError && <div className="form-error">{formError}</div>}
            <div className="form-grid">
              <input className="form-input" type="text" placeholder="Key name *"
                value={formName} onChange={e => setFormName(e.target.value)} />
              <input className="form-input" type="date" placeholder="Expiry date (optional)"
                value={formExpiry} onChange={e => setFormExpiry(e.target.value)}
                title="Expiry date (optional)" />
            </div>
            <div className="form-row form-row--actions">
              <button type="button" className="btn-primary" onClick={create} disabled={saving}>
                {saving ? 'Creating…' : 'Create'}
              </button>
              <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
            </div>
          </div>
        )}

        {loading && <div className="state-loading">Loading…</div>}
        {error   && <div className="state-error">{error}</div>}
        {!loading && !error && keys.length === 0 && (
          <div className="state-empty">No API keys yet.</div>
        )}
        {!loading && !error && keys.length > 0 && (
          <table className="routing-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Prefix</th>
                <th>Created by</th>
                <th>Last used</th>
                <th>Expires</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map(k => (
                <tr key={k.id}>
                  <td>{k.name}</td>
                  <td><code style={{ fontSize: '0.8rem' }}>{k.key_prefix}…</code></td>
                  <td>{k.created_by}</td>
                  <td>{k.last_used_at ? fmtDate(k.last_used_at) : <span className="routing-desc">never</span>}</td>
                  <td>{k.expires_at ? fmtDate(k.expires_at) : <span className="routing-desc">never</span>}</td>
                  <td>
                    <span className={`status-badge status-badge--${k.enabled ? 'clean' : 'failed'}`}>
                      {k.enabled ? 'active' : 'disabled'}
                    </span>
                  </td>
                  <td>
                    <div className="actions-cell">
                      <button type="button" className="btn btn--action" onClick={() => toggle(k)}>
                        {k.enabled ? 'Disable' : 'Enable'}
                      </button>
                      <button type="button" className="btn btn--action" onClick={() => rotate(k)}
                        disabled={rotatingId === k.id}>
                        {rotatingId === k.id ? 'Rotating…' : 'Rotate'}
                      </button>
                      <button type="button" className="btn btn--revoke" onClick={() => del(k)}>Delete</button>
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

const GLOBAL_PROJECT_KEY = 'aegis_global_project_id'

export function App() {
  const [displayTimezoneMode, setDisplayTimezoneMode] = useState<DisplayTimezoneMode>(() => readDisplayTimezone().mode)
  const [displayTimezoneCustom, setDisplayTimezoneCustom] = useState(() => readDisplayTimezone().customTimeZone)
  const [tab, setTab] = useState<AppTab>('studies')
  const [state, setState] = useState<StudiesState>('loading')
  const [studies, setStudies] = useState<Study[]>([])
  const [studiesTotal, setStudiesTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [selectedStudyId, setSelectedStudyId] = useState<string | null>(null)

  // Fire-and-forget: record that the current user viewed this study (HIPAA access audit).
  const selectStudy = (id: string | null) => {
    setSelectedStudyId(id)
    if (id) {
      fetch(`/api/studies/${id}/viewed`, { method: 'POST' }).catch(() => {/* best-effort */})
    }
  }
  const [agentPrefill, setAgentPrefill] = useState<{ studyId: string; studyUid: string } | null>(null)
  const [bulkSelected, setBulkSelected] = useState<Set<string>>(new Set())
  const [bulkWorking, setBulkWorking] = useState(false)
  const [bulkLabelInput, setBulkLabelInput] = useState('')
  const [bulkPipelineStep, setBulkPipelineStep] = useState('qc')
  const [stuckCount, setStuckCount] = useState(0)

  // Global project selector — persisted to localStorage.
  const [globalProjectId, setGlobalProjectId] = useState<string>(() => localStorage.getItem(GLOBAL_PROJECT_KEY) ?? '')

  // Auth state
  const [currentUser, setCurrentUser] = useState<AuthIdentity | null>(null)
  const [authError, setAuthError] = useState<string | null>(null)
  const validCustomTimeZone = normalizeIanaTimeZone(displayTimezoneCustom) ?? ''
  const localTimeZone = browserTimeZone()

  setDisplayTimezoneForFormatting(displayTimezoneMode, validCustomTimeZone)

  useEffect(() => {
    writeDisplayTimezone(displayTimezoneMode, displayTimezoneCustom)
  }, [displayTimezoneMode, displayTimezoneCustom])

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

  // Poll stuck studies every 5 minutes for the warning badge.
  useEffect(() => {
    const fetchStuck = () => {
      const params = new URLSearchParams({ minutes: '60' })
      if (globalProjectId) params.set('project_id', globalProjectId)
      fetch(`/api/studies/stuck?${params}`)
        .then(r => r.ok ? r.json() : null)
        .then(d => d && setStuckCount(d.total ?? 0))
        .catch(() => {})
    }
    fetchStuck()
    const id = setInterval(fetchStuck, 5 * 60 * 1000)
    return () => clearInterval(id)
  }, [globalProjectId])

  const isAdmin = currentUser?.role === 'admin'

  // Filters
  const [filterStatus,   setFilterStatus]   = useState('')
  const [filterModality, setFilterModality] = useState('')
  const [filterBodyPart, setFilterBodyPart] = useState('')
  const [filterSource,   setFilterSource]   = useState('')
  const [filterProject,  setFilterProject]  = useState('')
  const [filterSearch,   setFilterSearch]   = useState('')
  const [filterSubject,  setFilterSubject]  = useState('')
  const [filterLabel,    setFilterLabel]    = useState('')
  const [filterDateFrom, setFilterDateFrom] = useState('')
  const [filterDateTo,   setFilterDateTo]   = useState('')
  const [filterFlagged,  setFilterFlagged]  = useState(false)
  const [page, setPage] = useState(0)
  const [refreshTick, setRefreshTick] = useState(0)

  // Real-time SSE updates — bump refreshTick on any study change so the list
  // re-fetches automatically without requiring a manual refresh.
  useStudyEvents({
    projectId: globalProjectId || undefined,
    onEvent: () => setRefreshTick(t => t + 1),
  })

  // Persist global project selection to localStorage and sync to filterProject.
  useEffect(() => {
    if (globalProjectId) {
      localStorage.setItem(GLOBAL_PROJECT_KEY, globalProjectId)
    } else {
      localStorage.removeItem(GLOBAL_PROJECT_KEY)
    }
    setFilterProject(globalProjectId)
    setPage(0)
    setBulkSelected(new Set())
    setBreakdown(null)
    setShowBreakdown(false)
    setTimeline(null)
    setShowTimeline(false)
  }, [globalProjectId])

  // Projects for filter dropdown
  const [projects, setProjects] = useState<Project[]>([])
  useEffect(() => {
    fetch('/api/projects').then(r => r.json()).then(setProjects).catch(() => {})
  }, [])

  // Dashboard pipeline stats
  type PipelineStats = {
    study_counts: { received: number; defacing: number; clean: number; defaced: number; approved: number; rejected: number; expired: number; total: number }
    active_shares: number
  }
  const [pipelineStats, setPipelineStats] = useState<PipelineStats | null>(null)
  const fetchStats = useCallback(() => {
    const params = new URLSearchParams()
    if (globalProjectId) params.set('project_id', globalProjectId)
    const qs = params.toString()
    fetch(`/api/stats${qs ? '?' + qs : ''}`).then(r => r.ok ? r.json() : null).then(data => {
      if (data) setPipelineStats(data as PipelineStats)
    }).catch(() => {})
  }, [globalProjectId])
  useEffect(() => { fetchStats() }, [fetchStats, refreshTick])

  type BreakdownRow = { modality: string; body_part: string; count: number }
  const [breakdown, setBreakdown] = useState<BreakdownRow[] | null>(null)
  const [showBreakdown, setShowBreakdown] = useState(false)
  const loadBreakdown = async () => {
    const params = new URLSearchParams()
    if (globalProjectId) params.set('project_id', globalProjectId)
    const qs = params.toString()
    const res = await fetch(`/api/stats/breakdown${qs ? '?' + qs : ''}`)
    if (res.ok) { const d = await res.json(); setBreakdown(d.breakdown ?? []); setShowBreakdown(true) }
    else setShowBreakdown(v => !v)
  }

  type StorageStats = { raw_file_count: number; clean_file_count: number; total_file_count: number; total_studies: number; total_size_bytes: number }
  const [storageStats, setStorageStats] = useState<StorageStats | null>(null)
  const loadStorageStats = useCallback(async () => {
    const params = new URLSearchParams()
    if (globalProjectId) params.set('project_id', globalProjectId)
    const qs = params.toString()
    const res = await fetch(`/api/storage/stats${qs ? '?' + qs : ''}`)
    if (res.ok) setStorageStats(await res.json())
  }, [globalProjectId])
  useEffect(() => { loadStorageStats() }, [loadStorageStats])

  type TimelineDay = { date: string; received: number; approved: number }
  const [timeline, setTimeline] = useState<TimelineDay[] | null>(null)
  const [showTimeline, setShowTimeline] = useState(false)
  const loadTimeline = async () => {
    const params = new URLSearchParams({ days: '30' })
    if (globalProjectId) params.set('project_id', globalProjectId)
    const res = await fetch(`/api/stats/timeline?${params}`)
    if (res.ok) { const d = await res.json(); setTimeline(d.timeline ?? []); setShowTimeline(true) }
    else setShowTimeline(v => !v)
  }

  // ── Quick-search palette (Cmd/Ctrl+K) ──────────────────────────────────────
  type PaletteNavItem  = { kind: 'nav';   label: string; tab: AppTab; icon: string }
  type PaletteStudyItem = { kind: 'study'; label: string; sub: string; id: string }
  type PaletteItem = PaletteNavItem | PaletteStudyItem

  const NAV_ITEMS: PaletteNavItem[] = [
    { kind: 'nav', label: 'Studies',              tab: 'studies',            icon: '🗂' },
    { kind: 'nav', label: 'Audit Log',             tab: 'audit',              icon: '📋' },
    { kind: 'nav', label: 'Shares',                tab: 'shares',             icon: '🔗' },
    { kind: 'nav', label: 'Routing Rules',         tab: 'routing',            icon: '🔀' },
    { kind: 'nav', label: 'DIMSE Operations',      tab: 'dimse_ops',          icon: '📡' },
    { kind: 'nav', label: 'Institutions',          tab: 'institutions',       icon: '🏥' },
    { kind: 'nav', label: 'Anonymization Profiles',tab: 'profiles',           icon: '🔒' },
    { kind: 'nav', label: 'Protocol Templates',    tab: 'protocol_templates', icon: '📐' },
    { kind: 'nav', label: 'Notifications',         tab: 'notifications',      icon: '🔔' },
    { kind: 'nav', label: 'Projects',              tab: 'projects',           icon: '📁' },
    { kind: 'nav', label: 'Federation Peers',      tab: 'federation',         icon: '🌐' },
    { kind: 'nav', label: 'TCIA Import',           tab: 'tcia_import',        icon: '🔬' },
    { kind: 'nav', label: 'System Health',         tab: 'system',             icon: '⚙️' },
    ...(isAdmin ? [
      { kind: 'nav' as const, label: 'Users',        tab: 'users' as AppTab,         icon: '👤' },
      { kind: 'nav' as const, label: 'API Keys',     tab: 'api_keys' as AppTab,      icon: '🔑' },
      { kind: 'nav' as const, label: 'Invite Codes', tab: 'invite_codes' as AppTab,  icon: '🎟️' },
    ] : []),
  ]

  const [paletteOpen, setPaletteOpen] = useState(false)
  const [paletteQuery, setPaletteQuery] = useState('')
  const [paletteStudies, setPaletteStudies] = useState<PaletteStudyItem[]>([])
  const [paletteHighlight, setPaletteHighlight] = useState(0)
  const paletteInputRef = useRef<HTMLInputElement>(null)

  // Open with Cmd/Ctrl+K
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setPaletteOpen(v => { if (!v) { setPaletteQuery(''); setPaletteStudies([]); setPaletteHighlight(0) }; return !v })
      }
      if (e.key === 'Escape') setPaletteOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  // Focus input when palette opens
  useEffect(() => {
    if (paletteOpen) setTimeout(() => paletteInputRef.current?.focus(), 0)
  }, [paletteOpen])

  // Debounced study search
  useEffect(() => {
    if (!paletteOpen || paletteQuery.trim().length < 2) { setPaletteStudies([]); return }
    const t = setTimeout(async () => {
      const res = await fetch(`/api/studies?search=${encodeURIComponent(paletteQuery.trim())}&limit=6`)
      if (!res.ok) return
      const data = await res.json()
      setPaletteStudies((data.studies ?? []).map((s: Study) => ({
        kind: 'study' as const,
        label: uidShort(s.study_instance_uid),
        sub: [s.modality, s.body_part, s.status].filter(Boolean).join(' · '),
        id: s.id,
      })))
    }, 250)
    return () => clearTimeout(t)
  }, [paletteQuery, paletteOpen])

  const paletteNavFiltered = NAV_ITEMS.filter(n =>
    !paletteQuery.trim() || n.label.toLowerCase().includes(paletteQuery.trim().toLowerCase())
  )
  const paletteItems: PaletteItem[] = [...paletteNavFiltered, ...paletteStudies]

  const paletteSelect = (item: PaletteItem) => {
    setPaletteOpen(false)
    if (item.kind === 'nav') {
      setTab(item.tab)
    } else {
      setTab('studies')
      selectStudy(item.id)
    }
  }

  const paletteKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); setPaletteHighlight(h => Math.min(h + 1, paletteItems.length - 1)) }
    if (e.key === 'ArrowUp')   { e.preventDefault(); setPaletteHighlight(h => Math.max(h - 1, 0)) }
    if (e.key === 'Enter' && paletteItems[paletteHighlight]) paletteSelect(paletteItems[paletteHighlight])
  }

  // Reset highlight when results change
  useEffect(() => { setPaletteHighlight(0) }, [paletteItems.length])
  // ── end palette ─────────────────────────────────────────────────────────────

  // Fetch studies whenever filters, page, or refresh tick change
  useEffect(() => {
    let cancelled = false
    setState('loading')
    const params = new URLSearchParams({ limit: String(PAGE_SIZE), offset: String(page * PAGE_SIZE) })
    if (filterStatus)   params.set('status',     filterStatus)
    if (filterModality) params.set('modality',   filterModality)
    if (filterBodyPart) params.set('body_part',  filterBodyPart)
    if (filterSource)   params.set('source',     filterSource)
    if (filterProject)  params.set('project_id', filterProject)
    if (filterSearch)   params.set('search',     filterSearch)
    if (filterSubject)  params.set('subject_id', filterSubject)
    if (filterLabel)    params.set('label',      filterLabel)
    if (filterDateFrom) params.set('date_from',  new Date(filterDateFrom).toISOString())
    if (filterDateTo)   params.set('date_to',    new Date(filterDateTo + 'T23:59:59Z').toISOString())
    if (filterFlagged)  params.set('flagged',    'true')

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
  }, [page, filterStatus, filterModality, filterBodyPart, filterSource, filterProject, filterSearch, filterSubject, filterLabel, filterDateFrom, filterDateTo, filterFlagged, refreshTick])

  // Filter change helpers — also reset page to 0
  function setStatusF(v: string)   { setFilterStatus(v);   setPage(0); setBulkSelected(new Set()) }
  function setModalityF(v: string) { setFilterModality(v); setPage(0); setBulkSelected(new Set()) }
  function setBodyPartF(v: string) { setFilterBodyPart(v); setPage(0); setBulkSelected(new Set()) }
  function setSourceF(v: string)   { setFilterSource(v);   setPage(0); setBulkSelected(new Set()) }
  function setProjectF(v: string)  { setFilterProject(v);  setPage(0); setBulkSelected(new Set()) }
  function setSearchF(v: string)    { setFilterSearch(v);    setPage(0); setBulkSelected(new Set()) }
  function setSubjectF(v: string)   { setFilterSubject(v);   setPage(0); setBulkSelected(new Set()) }
  function setLabelF(v: string)     { setFilterLabel(v);     setPage(0); setBulkSelected(new Set()) }
  function setDateFromF(v: string)  { setFilterDateFrom(v);  setPage(0); setBulkSelected(new Set()) }
  function setDateToF(v: string)    { setFilterDateTo(v);    setPage(0); setBulkSelected(new Set()) }
  function setFlaggedF(v: boolean)  { setFilterFlagged(v);   setPage(0); setBulkSelected(new Set()) }

  const hasFilters = !!(filterStatus || filterModality || filterBodyPart || filterSource || filterProject || filterSearch || filterSubject || filterLabel || filterDateFrom || filterDateTo || filterFlagged)

  function clearFilters() {
    setFilterStatus(''); setFilterModality(''); setFilterBodyPart('')
    setFilterSource(''); setFilterProject(''); setFilterSearch('')
    setFilterSubject(''); setFilterLabel(''); setFilterDateFrom(''); setFilterDateTo('')
    setFilterFlagged(false); setPage(0)
    setBulkSelected(new Set())
  }

  const allPageIds = studies.map(s => s.id)
  const allPageSelected = allPageIds.length > 0 && allPageIds.every(id => bulkSelected.has(id))
  const somePageSelected = allPageIds.some(id => bulkSelected.has(id))

  function toggleSelectAll() {
    if (allPageSelected) {
      setBulkSelected(prev => {
        const next = new Set(prev)
        allPageIds.forEach(id => next.delete(id))
        return next
      })
    } else {
      setBulkSelected(prev => {
        const next = new Set(prev)
        allPageIds.forEach(id => next.add(id))
        return next
      })
    }
  }

  async function doBulkAction(action: 'approve' | 'reject') {
    const ids = Array.from(bulkSelected)
    if (ids.length === 0) return
    if (!confirm(`${action === 'approve' ? 'Approve' : 'Reject'} ${ids.length} selected ${ids.length === 1 ? 'study' : 'studies'}?`)) return
    setBulkWorking(true)
    try {
      await fetch('/api/studies/bulk', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, study_ids: ids }),
      })
      setBulkSelected(new Set())
      setRefreshTick(t => t + 1)
    } finally {
      setBulkWorking(false)
    }
  }

  async function doBulkLabel(action: 'add' | 'remove') {
    const label = bulkLabelInput.trim()
    if (!label) { alert('Enter a label first'); return }
    const ids = Array.from(bulkSelected)
    if (ids.length === 0) return
    setBulkWorking(true)
    try {
      const res = await fetch('/api/studies/bulk-label', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ study_ids: ids, label, action }),
      })
      if (res.ok) {
        const d = await res.json()
        const verb = action === 'add' ? 'Applied' : 'Removed'
        const count = action === 'add' ? d.applied : d.removed
        alert(`${verb} "${label}" on ${count} of ${ids.length} ${ids.length === 1 ? 'study' : 'studies'}`)
        setBulkLabelInput('')
      }
    } finally {
      setBulkWorking(false)
    }
  }

  async function doBulkPipelineTrigger() {
    const ids = Array.from(bulkSelected)
    if (ids.length === 0) return
    if (!confirm(`Trigger "${bulkPipelineStep}" step for ${ids.length} selected ${ids.length === 1 ? 'study' : 'studies'}?`)) return
    setBulkWorking(true)
    try {
      const res = await fetch('/api/studies/bulk-pipeline-trigger', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ study_ids: ids, step: bulkPipelineStep }),
      })
      if (res.ok) {
        const d = await res.json()
        alert(`Triggered ${d.triggered} / Skipped ${d.skipped}${d.errors?.length ? ` / ${d.errors.length} error(s)` : ''}`)
        setRefreshTick(t => t + 1)
      }
    } finally {
      setBulkWorking(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(studiesTotal / PAGE_SIZE))
  const pageStart  = studiesTotal === 0 ? 0 : page * PAGE_SIZE + 1

  const csvUrl = (() => {
    const params = new URLSearchParams()
    if (filterStatus)   params.set('status',     filterStatus)
    if (filterModality) params.set('modality',   filterModality)
    if (filterBodyPart) params.set('body_part',  filterBodyPart)
    if (filterSource)   params.set('source',     filterSource)
    if (filterProject)  params.set('project_id', filterProject)
    if (filterSearch)   params.set('search',     filterSearch)
    if (filterLabel)    params.set('label',      filterLabel)
    if (filterDateFrom) params.set('date_from',  new Date(filterDateFrom).toISOString())
    if (filterDateTo)   params.set('date_to',    new Date(filterDateTo + 'T23:59:59Z').toISOString())
    if (filterFlagged)  params.set('flagged',    'true')
    const qs = params.toString()
    return `/api/studies.csv${qs ? '?' + qs : ''}`
  })()
  const pageEnd    = Math.min((page + 1) * PAGE_SIZE, studiesTotal)

  return (
    <div className="admin-root">
      {/* Cmd/Ctrl+K quick-search palette */}
      {paletteOpen && (
        <div className="palette-overlay" onClick={() => setPaletteOpen(false)}>
          <div className="palette-modal" onClick={e => e.stopPropagation()}>
            <div className="palette-search-row">
              <span className="palette-search-icon">⌕</span>
              <input
                ref={paletteInputRef}
                className="palette-input"
                placeholder="Search studies, navigate…"
                value={paletteQuery}
                onChange={e => setPaletteQuery(e.target.value)}
                onKeyDown={paletteKeyDown}
                autoComplete="off"
                spellCheck={false}
              />
              <kbd className="palette-esc-hint">Esc</kbd>
            </div>
            {paletteItems.length > 0 ? (
              <ul className="palette-results">
                {paletteNavFiltered.length > 0 && (
                  <li className="palette-group-label">Navigate</li>
                )}
                {paletteNavFiltered.map((item, i) => (
                  <li
                    key={item.tab}
                    className={`palette-result${paletteHighlight === i ? ' palette-result--active' : ''}`}
                    onMouseEnter={() => setPaletteHighlight(i)}
                    onClick={() => paletteSelect(item)}
                  >
                    <span className="palette-result__icon">{item.icon}</span>
                    <span className="palette-result__label">{item.label}</span>
                  </li>
                ))}
                {paletteStudies.length > 0 && (
                  <li className="palette-group-label">Studies</li>
                )}
                {paletteStudies.map((item, j) => {
                  const idx = paletteNavFiltered.length + j
                  return (
                    <li
                      key={item.id}
                      className={`palette-result${paletteHighlight === idx ? ' palette-result--active' : ''}`}
                      onMouseEnter={() => setPaletteHighlight(idx)}
                      onClick={() => paletteSelect(item)}
                    >
                      <span className="palette-result__icon">🔬</span>
                      <span className="palette-result__label">{item.label}</span>
                      <span className="palette-result__sub">{item.sub}</span>
                    </li>
                  )
                })}
              </ul>
            ) : paletteQuery.trim().length >= 2 ? (
              <div className="palette-empty">No results</div>
            ) : null}
            <div className="palette-footer">
              <span><kbd>↑↓</kbd> navigate</span>
              <span><kbd>↵</kbd> select</span>
              <span><kbd>Esc</kbd> close</span>
            </div>
          </div>
        </div>
      )}

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
          {/* Global project selector */}
          {projects.length > 1 && (
            <div className="tz-control">
              <label className="tz-label" htmlFor="global-project-select">Project</label>
              <select
                id="global-project-select"
                className="tz-select"
                value={globalProjectId}
                onChange={e => setGlobalProjectId(e.target.value)}
              >
                <option value="">All projects</option>
                {projects.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
              </select>
              {globalProjectId && (
                <button type="button" className="btn-secondary" style={{ padding: '0.2rem 0.5rem', fontSize: '0.75rem' }}
                  onClick={() => setGlobalProjectId('')}>Clear</button>
              )}
            </div>
          )}
          <div className="tz-control">
            <label className="tz-label" htmlFor="display-timezone-mode">Time Zone</label>
            <select
              id="display-timezone-mode"
              className="tz-select"
              value={displayTimezoneMode}
              onChange={(e) => setDisplayTimezoneMode(e.target.value as DisplayTimezoneMode)}
            >
              <option value="utc">UTC</option>
              <option value="local">Local ({localTimeZone})</option>
              <option value="custom">Custom</option>
            </select>
            {displayTimezoneMode === 'custom' && (
              <>
                <input
                  className="tz-input"
                  type="text"
                  placeholder="America/Chicago"
                  value={displayTimezoneCustom}
                  onChange={(e) => setDisplayTimezoneCustom(e.target.value)}
                />
                {!validCustomTimeZone && displayTimezoneCustom.trim() && (
                  <span className="tz-warning">Invalid IANA time zone</span>
                )}
              </>
            )}
          </div>
          {currentUser && (
            <span className="auth-user-badge">
              {currentUser.name || currentUser.email} ({currentUser.role})
            </span>
          )}
          <button
            type="button"
            className="btn-palette-trigger"
            onClick={() => { setPaletteOpen(true); setPaletteQuery(''); setPaletteStudies([]); setPaletteHighlight(0) }}
            title="Quick search (⌘K)"
          >
            <span>Search…</span>
            <kbd>⌘K</kbd>
          </button>
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
          {stuckCount > 0 && <span className="tab-stuck-badge">{stuckCount} stuck</span>}
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
          className={`tab-btn${tab === 'agent' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('agent')}
        >
          Agent
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'shares' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('shares')}
        >
          Shares
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'routing' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('routing')}
        >
          Routing
        </button>
        {isAdmin && (
          <button
            type="button"
            className={`tab-btn${tab === 'dimse_ops' ? ' tab-btn--active' : ''}`}
            onClick={() => setTab('dimse_ops')}
          >
            DIMSE Ops
          </button>
        )}
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
        <button
          type="button"
          className={`tab-btn${tab === 'federation' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('federation')}
        >
          Federation
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'tcia_import' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('tcia_import')}
        >
          TCIA Import
        </button>
        <button
          type="button"
          className={`tab-btn${tab === 'system' ? ' tab-btn--active' : ''}`}
          onClick={() => setTab('system')}
        >
          System
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
        {isAdmin && (
          <button
            type="button"
            className={`tab-btn${tab === 'api_keys' ? ' tab-btn--active' : ''}`}
            onClick={() => setTab('api_keys')}
          >
            API Keys
          </button>
        )}
        {isAdmin && (
          <button
            type="button"
            className={`tab-btn${tab === 'invite_codes' ? ' tab-btn--active' : ''}`}
            onClick={() => setTab('invite_codes')}
          >
            Invite Codes
          </button>
        )}
      </nav>

      {/* Studies tab */}
      {tab === 'studies' && selectedStudyId && (
        <StudyDetailPanel
          studyId={selectedStudyId}
          onBack={() => selectStudy(null)}
          onAction={() => setRefreshTick(t => t + 1)}
          isAdmin={isAdmin}
        />
      )}
      {tab === 'agent' && (
        <AgentPanel
          prefillStudyId={agentPrefill?.studyId}
          prefillStudyUid={agentPrefill?.studyUid}
        />
      )}
      {tab === 'studies' && !selectedStudyId && (
        <>
          {/* Pipeline stats banner */}
          {pipelineStats && (
            <div className="stats-banner">
              {(
                [
                  ['received', 'Received',  pipelineStats.study_counts.received],
                  ['defacing', 'Defacing',  pipelineStats.study_counts.defacing],
                  ['clean',    'Clean',     pipelineStats.study_counts.clean],
                  ['defaced',  'Defaced',   pipelineStats.study_counts.defaced],
                  ['approved', 'Approved',  pipelineStats.study_counts.approved],
                  ['rejected', 'Rejected',  pipelineStats.study_counts.rejected],
                  ['expired',  'Expired',   pipelineStats.study_counts.expired ?? 0],
                ] as [string, string, number][]
              ).map(([status, label, count]) => (
                <button
                  key={status}
                  type="button"
                  className={`stats-pill stats-pill--${status}${filterStatus === status ? ' stats-pill--active' : ''}`}
                  onClick={() => setStatusF(filterStatus === status ? '' : status)}
                  title={`Filter by ${label.toLowerCase()}`}
                >
                  <span className="stats-pill__count">{count}</span>
                  <span className="stats-pill__label">{label}</span>
                </button>
              ))}
              <span className="stats-banner__sep" />
              <span className="stats-banner__shares" title="Active export shares">
                {pipelineStats.active_shares} active {pipelineStats.active_shares === 1 ? 'share' : 'shares'}
              </span>
              {storageStats && (
                <>
                  <span className="stats-banner__sep" />
                  <span className="stats-banner__shares" title="DICOM file counts and total storage size">
                    {storageStats.total_file_count} files ({storageStats.raw_file_count} raw, {storageStats.clean_file_count} clean)
                    {storageStats.total_size_bytes > 0 && ` · ${formatBytes(storageStats.total_size_bytes)}`}
                  </span>
                </>
              )}
            </div>
          )}

          {/* Breakdown stats toggle */}
          <div style={{marginBottom:'8px'}}>
            <button type="button" className="btn-secondary" onClick={loadBreakdown} style={{fontSize:'0.8rem'}}>
              {showBreakdown ? '▲ Hide breakdown' : '▼ Modality / body part breakdown'}
            </button>
            {showBreakdown && breakdown && (
              <div style={{marginTop:'6px',overflowX:'auto'}}>
                <table className="audit-table" style={{fontSize:'0.8rem',maxWidth:'600px'}}>
                  <thead><tr><th>Modality</th><th>Body part</th><th>Count</th></tr></thead>
                  <tbody>
                    {breakdown.length === 0
                      ? <tr><td colSpan={3} className="td-muted">No studies yet.</td></tr>
                      : breakdown.map((r, i) => (
                        <tr key={i}>
                          <td>{r.modality || <span className="td-muted">—</span>}</td>
                          <td>{r.body_part || <span className="td-muted">—</span>}</td>
                          <td>{r.count}</td>
                        </tr>
                      ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Timeline (daily ingestion) toggle */}
          <div style={{marginBottom:'8px'}}>
            <button type="button" className="btn-secondary" onClick={loadTimeline} style={{fontSize:'0.8rem'}}>
              {showTimeline ? '▲ Hide timeline' : '▼ Daily ingestion (last 30 days)'}
            </button>
            {showTimeline && timeline && (
              <div style={{marginTop:'6px',overflowX:'auto'}}>
                {timeline.length === 0
                  ? <span className="td-muted" style={{fontSize:'0.8rem'}}>No studies in the last 30 days.</span>
                  : (
                    <table className="audit-table" style={{fontSize:'0.8rem',maxWidth:'420px'}}>
                      <thead><tr><th>Date</th><th>Received</th><th>Approved</th></tr></thead>
                      <tbody>
                        {timeline.map(d => (
                          <tr key={d.date}>
                            <td>{d.date}</td>
                            <td>{d.received}</td>
                            <td>{d.approved}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
              </div>
            )}
          </div>

          {/* Synthetic MRI generator */}
          <SynthPanel isAdmin={isAdmin} onStudyGenerated={() => setRefreshTick(t => t + 1)} />

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
              <option value="expired">Expired</option>
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
            <select className="filter-select" title="Filter by body part" value={filterBodyPart} onChange={e => setBodyPartF(e.target.value)}>
              <option value="">All body parts</option>
              <option value="HEAD">Head</option>
              <option value="BRAIN">Brain</option>
              <option value="CHEST">Chest</option>
              <option value="ABDOMEN">Abdomen</option>
              <option value="SPINE">Spine</option>
              <option value="EXTREMITY">Extremity</option>
              <option value="NECK">Neck</option>
              <option value="PELVIS">Pelvis</option>
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
            <input
              className="filter-input filter-input--subject"
              type="search"
              placeholder="Subject ID…"
              value={filterSubject}
              onChange={e => setSubjectF(e.target.value)}
            />
            <input
              className="filter-input filter-input--label"
              type="search"
              placeholder="Label…"
              value={filterLabel}
              onChange={e => setLabelF(e.target.value)}
            />
            <label className="filter-flagged-label" title="Show flagged studies only">
              <input
                type="checkbox"
                checked={filterFlagged}
                onChange={e => setFlaggedF(e.target.checked)}
              />
              {' '}Priority only
            </label>
            <input
              type="date"
              className="filter-date"
              title="Created on or after"
              value={filterDateFrom}
              onChange={e => setDateFromF(e.target.value)}
            />
            <span className="filter-date-sep">–</span>
            <input
              type="date"
              className="filter-date"
              title="Created on or before"
              value={filterDateTo}
              onChange={e => setDateToF(e.target.value)}
            />
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
            {state === 'loaded' && studiesTotal > 0 && (
              <a href={csvUrl} download="studies.csv" className="btn btn--secondary btn--csv-export">Export CSV</a>
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

          {state === 'loaded' && bulkSelected.size > 0 && isAdmin && (
            <div className="bulk-action-bar">
              <span className="bulk-action-bar__count">{bulkSelected.size} selected</span>
              <button type="button" className="btn btn--approve" disabled={bulkWorking} onClick={() => doBulkAction('approve')}>Approve selected</button>
              <button type="button" className="btn btn--reject" disabled={bulkWorking} onClick={() => doBulkAction('reject')}>Reject selected</button>
              <span className="bulk-action-bar__sep" style={{margin:'0 4px',color:'var(--text-muted)'}}>|</span>
              <input
                type="text"
                className="audit-actor-input"
                placeholder="Label name…"
                value={bulkLabelInput}
                onChange={e => setBulkLabelInput(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && doBulkLabel('add')}
                style={{width:'130px'}}
                disabled={bulkWorking}
              />
              <button type="button" className="btn btn--action" disabled={bulkWorking || !bulkLabelInput.trim()} onClick={() => doBulkLabel('add')} title="Apply label to selected studies">+ Label</button>
              <button type="button" className="btn btn--secondary" disabled={bulkWorking || !bulkLabelInput.trim()} onClick={() => doBulkLabel('remove')} title="Remove label from selected studies">− Label</button>
              <span className="bulk-action-bar__sep" style={{margin:'0 4px',color:'var(--text-muted)'}}>|</span>
              <select
                className="audit-actor-input"
                value={bulkPipelineStep}
                onChange={e => setBulkPipelineStep(e.target.value)}
                disabled={bulkWorking}
                style={{height:'28px'}}
              >
                <option value="classify">Classify</option>
                <option value="phi_scan">PHI Scan</option>
                <option value="protocol">Protocol Check</option>
                <option value="deface">Deface</option>
                <option value="qc">QC Check</option>
                <option value="bids">BIDS Convert</option>
                <option value="export">Export</option>
              </select>
              <button type="button" className="btn btn--action" disabled={bulkWorking} onClick={doBulkPipelineTrigger} title="Trigger pipeline step for selected studies">Trigger Step →</button>
              <button type="button" className="btn btn--secondary" disabled={bulkWorking} onClick={() => setBulkSelected(new Set())}>Clear selection</button>
            </div>
          )}

          {state === 'loaded' && studies.length > 0 && (
            <div className="studies-table-wrap">
              <table className="studies-table">
                <thead>
                  <tr>
                    <th className="th-check">
                      <input
                        type="checkbox"
                        checked={allPageSelected}
                        ref={el => { if (el) el.indeterminate = somePageSelected && !allPageSelected }}
                        onChange={toggleSelectAll}
                        aria-label="Select all on page"
                      />
                    </th>
                    <th className="th-flag" title="Priority flag">★</th>
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
                    <StudyRow
                      key={study.id}
                      study={study}
                      onAction={() => setRefreshTick(t => t + 1)}
                      onSelect={() => selectStudy(study.id)}
                      onAskAgent={() => {
                        setAgentPrefill({ studyId: study.id, studyUid: study.study_instance_uid })
                        setTab('agent')
                      }}
                      isAdmin={isAdmin}
                      checked={bulkSelected.has(study.id)}
                      onToggle={() => setBulkSelected(prev => {
                        const next = new Set(prev)
                        if (next.has(study.id)) next.delete(study.id)
                        else next.add(study.id)
                        return next
                      })}
                    />
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
      {tab === 'audit' && <AuditLog projectId={globalProjectId} />}

      {/* Global shares tab */}
      {tab === 'shares' && <GlobalSharesPanel isAdmin={isAdmin} projectId={globalProjectId} />}

      {/* Routing tab */}
      {tab === 'routing' && <RoutingPanel isAdmin={isAdmin} projectId={globalProjectId} />}

      {/* DIMSE operations tab — admin only */}
      {tab === 'dimse_ops' && isAdmin && <DimseOpsPanel />}

      {/* Institutions tab */}
      {tab === 'institutions' && <InstitutionsPanel isAdmin={isAdmin} />}

      {/* Profiles tab */}
      {tab === 'profiles' && <ProfilesPanel isAdmin={isAdmin} />}

      {/* Protocol Templates tab */}
      {tab === 'protocol_templates' && <ProtocolTemplatesPanel isAdmin={isAdmin} />}

      {/* Notifications tab */}
      {tab === 'notifications' && <NotificationsPanel isAdmin={isAdmin} projectId={globalProjectId} />}

      {/* Projects tab */}
      {tab === 'projects' && <ProjectsPanel isAdmin={isAdmin} />}

      {/* Federation tab */}
      {tab === 'federation' && <FederationPanel isAdmin={isAdmin} />}

      {/* TCIA Import tab */}
      {tab === 'tcia_import' && <TCIAPanel isAdmin={isAdmin} />}

      {/* Users tab — admin only */}
      {tab === 'users' && isAdmin && <UsersPanel />}

      {/* API Keys tab — admin only */}
      {tab === 'api_keys' && isAdmin && <APIKeysPanel />}

      {/* Invite Codes tab — admin only */}
      {tab === 'invite_codes' && isAdmin && <InviteCodesPanel />}

      {/* System Health tab */}
      {tab === 'system' && <SystemHealthPanel />}
    </div>
  )
}
