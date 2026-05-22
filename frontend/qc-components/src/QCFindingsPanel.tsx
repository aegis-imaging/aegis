// Per-study findings panel — list of open + resolved QC findings, with an
// inline form to raise a new finding and resolve/reopen/delete actions on
// existing rows.

import { useEffect, useState } from 'react'
import * as api from './api'
import type { QCCategory, QCFinding, QCSeverity } from './types'
import { CATEGORY_LABELS, severityStyle } from './types'

export interface QCFindingsPanelProps {
  studyId: string
  apiOptions?: api.APIClientOptions
  isAdmin?: boolean
  onChange?: (findings: QCFinding[]) => void
}

export function QCFindingsPanel({ studyId, apiOptions, isAdmin = true, onChange }: QCFindingsPanelProps) {
  const [findings, setFindings] = useState<QCFinding[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const [refreshTick, setRefreshTick] = useState(0)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api.listFindings(studyId, false, apiOptions)
      .then(rows => {
        if (cancelled) return
        setFindings(rows)
        onChange?.(rows)
      })
      .catch(e => { if (!cancelled) setError(String(e)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [studyId, apiOptions, refreshTick])

  async function resolve(id: string, note: string) {
    await api.resolveFinding(id, note, apiOptions)
    setRefreshTick(x => x + 1)
  }
  async function reopen(id: string) {
    await api.reopenFinding(id, apiOptions)
    setRefreshTick(x => x + 1)
  }
  async function del(id: string) {
    if (!confirm('Delete this finding? This cannot be undone.')) return
    await api.deleteFinding(id, apiOptions)
    setRefreshTick(x => x + 1)
  }

  const open = findings.filter(f => !f.resolved_at)
  const resolved = findings.filter(f => f.resolved_at)

  return (
    <div className="qc-findings-panel">
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <h3 style={{ margin: 0, fontSize: 16 }}>QC findings</h3>
        <span style={{ fontSize: 12, color: '#6b7280' }}>
          {open.length} open · {resolved.length} resolved
        </span>
        {isAdmin && (
          <button type="button" style={btnPrimary()} onClick={() => setShowAdd(v => !v)}>
            {showAdd ? 'Cancel' : '+ Raise finding'}
          </button>
        )}
      </div>

      {showAdd && isAdmin && (
        <AddFindingForm
          studyId={studyId}
          apiOptions={apiOptions}
          onCreated={() => { setShowAdd(false); setRefreshTick(x => x + 1) }}
        />
      )}

      {error && (
        <div style={{ padding: '8px 12px', background: '#fee2e2', border: '1px solid #fecaca', color: '#b91c1c', borderRadius: 6, marginBottom: 8 }}>
          {error}
        </div>
      )}
      {loading && <div style={{ padding: 16, color: '#6b7280' }}>Loading findings…</div>}

      {open.length > 0 && (
        <section style={{ marginBottom: 16 }}>
          <h4 style={subhead()}>Open</h4>
          {open.map(f => (
            <FindingRow key={f.id} f={f} isAdmin={isAdmin} onResolve={resolve} onDelete={del} />
          ))}
        </section>
      )}
      {resolved.length > 0 && (
        <section>
          <h4 style={{ ...subhead(), color: '#6b7280' }}>Resolved</h4>
          {resolved.map(f => (
            <FindingRow key={f.id} f={f} isAdmin={isAdmin} onReopen={reopen} onDelete={del} resolved />
          ))}
        </section>
      )}
      {!loading && findings.length === 0 && (
        <div style={{ padding: 16, color: '#6b7280', fontSize: 13 }}>
          No findings yet. Raise one above when you spot an issue during review.
        </div>
      )}
    </div>
  )
}

function FindingRow({
  f, isAdmin, resolved, onResolve, onReopen, onDelete,
}: {
  f: QCFinding
  isAdmin: boolean
  resolved?: boolean
  onResolve?: (id: string, note: string) => Promise<void>
  onReopen?: (id: string) => Promise<void>
  onDelete?: (id: string) => Promise<void>
}) {
  const [showResolve, setShowResolve] = useState(false)
  const [note, setNote] = useState('')
  const sev = severityStyle(f.severity)
  return (
    <div style={{
      border: '1px solid #e5e7eb', borderRadius: 6, padding: 10, marginBottom: 6,
      opacity: resolved ? 0.65 : 1, background: resolved ? '#fafafa' : '#fff',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <span style={{
          padding: '2px 8px', borderRadius: 999,
          background: sev.background, color: sev.color,
          fontSize: 11, fontWeight: 600,
        }}>{sev.label}</span>
        <span style={{ fontSize: 12, color: '#374151' }}>{CATEGORY_LABELS[f.category] ?? f.category}</span>
        <span style={{ fontSize: 12, color: '#9ca3af', marginLeft: 'auto' }}>
          {f.analyst_email || f.analyst_name || 'unknown'} · {relativeTime(f.created_at)}
        </span>
      </div>
      <div style={{ fontSize: 13, color: '#1f2937', whiteSpace: 'pre-wrap' }}>{f.body}</div>
      {f.series_uid && (
        <div style={{ fontSize: 11, color: '#6b7280', marginTop: 4 }}>
          series <code>{f.series_uid}</code>{f.instance_index != null ? ` · instance ${f.instance_index}` : ''}
        </div>
      )}
      {resolved && f.resolution_note && (
        <div style={{ fontSize: 12, color: '#6b7280', marginTop: 6, padding: 6, background: '#f3f4f6', borderRadius: 4 }}>
          <strong>Resolved by {f.resolved_by}:</strong> {f.resolution_note}
        </div>
      )}
      {isAdmin && (
        <div style={{ marginTop: 8, display: 'flex', gap: 6 }}>
          {!resolved && onResolve && !showResolve && (
            <button type="button" style={btn()} onClick={() => setShowResolve(true)}>Resolve…</button>
          )}
          {!resolved && onResolve && showResolve && (
            <>
              <input
                value={note}
                onChange={e => setNote(e.target.value)}
                placeholder="Resolution note (optional)"
                style={{ flex: 1, padding: '4px 8px', border: '1px solid #d1d5db', borderRadius: 4, fontSize: 13 }}
              />
              <button type="button" style={btnPrimary()} onClick={async () => {
                await onResolve(f.id, note); setShowResolve(false); setNote('')
              }}>Resolve</button>
              <button type="button" style={btn()} onClick={() => { setShowResolve(false); setNote('') }}>Cancel</button>
            </>
          )}
          {resolved && onReopen && (
            <button type="button" style={btn()} onClick={() => onReopen(f.id)}>Reopen</button>
          )}
          {onDelete && (
            <button type="button" style={btnDanger()} onClick={() => onDelete(f.id)}>Delete</button>
          )}
        </div>
      )}
    </div>
  )
}

function AddFindingForm({ studyId, apiOptions, onCreated }: {
  studyId: string
  apiOptions?: api.APIClientOptions
  onCreated: () => void
}) {
  const [category, setCategory] = useState<QCCategory>('other')
  const [severity, setSeverity] = useState<QCSeverity>('minor')
  const [body, setBody] = useState('')
  const [seriesUid, setSeriesUid] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  async function submit() {
    if (!body.trim()) { setError('Body is required.'); return }
    setSubmitting(true)
    setError('')
    try {
      await api.createFinding(studyId, { category, severity, body: body.trim(), seriesUid: seriesUid.trim() }, apiOptions)
      onCreated()
    } catch (e) {
      setError(String(e))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div style={{ border: '1px solid #d1d5db', borderRadius: 6, padding: 10, marginBottom: 10, background: '#f9fafb' }}>
      <div style={{ display: 'flex', gap: 8, marginBottom: 6 }}>
        <label style={{ fontSize: 13 }}>Category:&nbsp;
          <select value={category} onChange={e => setCategory(e.target.value as QCCategory)}>
            {Object.entries(CATEGORY_LABELS).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </select>
        </label>
        <label style={{ fontSize: 13 }}>Severity:&nbsp;
          <select value={severity} onChange={e => setSeverity(e.target.value as QCSeverity)}>
            <option value="info">info</option>
            <option value="minor">minor</option>
            <option value="major">major</option>
            <option value="critical">critical</option>
          </select>
        </label>
      </div>
      <textarea
        value={body}
        onChange={e => setBody(e.target.value)}
        rows={3}
        placeholder="Describe the issue — what was wrong, where (slice/series), how it was discovered…"
        style={{ width: '100%', boxSizing: 'border-box', padding: '6px 8px', border: '1px solid #d1d5db', borderRadius: 4, fontSize: 13, fontFamily: 'inherit' }}
      />
      <input
        value={seriesUid}
        onChange={e => setSeriesUid(e.target.value)}
        placeholder="SeriesInstanceUID (optional)"
        style={{ width: '100%', marginTop: 6, padding: '6px 8px', border: '1px solid #d1d5db', borderRadius: 4, fontSize: 12 }}
      />
      {error && <div style={{ color: '#b91c1c', fontSize: 12, marginTop: 4 }}>{error}</div>}
      <div style={{ marginTop: 8 }}>
        <button type="button" style={btnPrimary()} disabled={submitting} onClick={submit}>
          {submitting ? 'Saving…' : 'Save finding'}
        </button>
      </div>
    </div>
  )
}

// ── styles ────────────────────────────────────────────────────────────────

function subhead(): React.CSSProperties {
  return { margin: '0 0 6px', fontSize: 12, fontWeight: 600, textTransform: 'uppercase', color: '#374151' }
}
function btn(): React.CSSProperties {
  return { padding: '4px 10px', borderRadius: 4, border: '1px solid #d1d5db', background: '#fff', cursor: 'pointer', fontSize: 12 }
}
function btnPrimary(): React.CSSProperties {
  return { ...btn(), background: '#2563eb', color: '#fff', borderColor: '#2563eb' }
}
function btnDanger(): React.CSSProperties {
  return { ...btn(), background: '#fff', color: '#b91c1c', borderColor: '#fecaca' }
}

function relativeTime(ts: string): string {
  const t = new Date(ts).getTime()
  const ageSec = (Date.now() - t) / 1000
  if (ageSec < 60) return 'just now'
  if (ageSec < 3600) return `${Math.floor(ageSec / 60)}m`
  if (ageSec < 86400) return `${Math.floor(ageSec / 3600)}h`
  return `${Math.floor(ageSec / 86400)}d`
}
