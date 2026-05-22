// Triage queue — sortable, filterable list of studies awaiting analyst review.
//
// Designed so the admin dashboard and the desktop app can both drop it in
// with the same props; differences (column visibility, theme) are pure CSS.

import { useEffect, useMemo, useState } from 'react'
import * as api from './api'
import type { QCTriageItem } from './types'
import { severityStyle } from './types'

export interface QCTriageQueueProps {
  /** API connection options — set baseUrl when running in the desktop app. */
  apiOptions?: api.APIClientOptions
  /** Initial filters. */
  initialFilters?: api.TriageFilters
  /** Called when an analyst clicks a study row. */
  onSelectStudy?: (item: QCTriageItem) => void
  /** When omitted, the component fetches every 30s. */
  refreshIntervalMs?: number
  /** Hide the filter bar (useful when the host already has filters). */
  hideFilters?: boolean
}

export function QCTriageQueue({
  apiOptions,
  initialFilters,
  onSelectStudy,
  refreshIntervalMs = 30000,
  hideFilters = false,
}: QCTriageQueueProps) {
  const [filters, setFilters] = useState<api.TriageFilters>({
    assigned: 'me', openOnly: true, limit: 200, ...initialFilters,
  })
  const [items, setItems] = useState<QCTriageItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [refreshTick, setRefreshTick] = useState(0)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    api.listTriage(filters, apiOptions)
      .then(rows => { if (!cancelled) setItems(rows) })
      .catch(e => { if (!cancelled) setError(String(e)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [filters, apiOptions, refreshTick])

  useEffect(() => {
    if (refreshIntervalMs <= 0) return
    const t = setInterval(() => setRefreshTick(x => x + 1), refreshIntervalMs)
    return () => clearInterval(t)
  }, [refreshIntervalMs])

  const queueStats = useMemo(() => {
    const total = items.length
    let critical = 0, major = 0, minor = 0, openFindings = 0
    for (const it of items) {
      openFindings += it.open_finding_count
      if (it.highest_severity_open === 'critical') critical++
      else if (it.highest_severity_open === 'major') major++
      else if (it.highest_severity_open === 'minor') minor++
    }
    return { total, critical, major, minor, openFindings }
  }, [items])

  return (
    <div className="qc-triage-queue">
      <div className="qc-triage-header" style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 8 }}>
        <h3 style={{ margin: 0, fontSize: 16 }}>QC triage queue</h3>
        <span style={pill('#374151')}>{queueStats.total} studies</span>
        {queueStats.critical > 0 && <span style={pill('#b91c1c')}>{queueStats.critical} critical</span>}
        {queueStats.major > 0 && <span style={pill('#9a3412')}>{queueStats.major} major</span>}
        {queueStats.minor > 0 && <span style={pill('#854d0e')}>{queueStats.minor} minor</span>}
        <span style={{ marginLeft: 'auto', fontSize: 12, color: '#6b7280' }}>
          {queueStats.openFindings} open finding{queueStats.openFindings === 1 ? '' : 's'}
        </span>
        <button type="button" style={btn()} onClick={() => setRefreshTick(x => x + 1)}>↻</button>
      </div>

      {!hideFilters && <FilterBar filters={filters} onChange={setFilters} />}

      {error && <div style={errorBox()}>{error}</div>}
      {loading && items.length === 0 && <div style={{ padding: 24, textAlign: 'center', color: '#6b7280' }}>Loading…</div>}
      {!loading && items.length === 0 && !error && (
        <div style={{ padding: 24, textAlign: 'center', color: '#6b7280' }}>
          No studies match these filters.
        </div>
      )}

      {items.length > 0 && (
        <div style={{ border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead style={{ background: '#f9fafb' }}>
              <tr>
                <th style={th()}>Severity</th>
                <th style={th()}>Modality</th>
                <th style={th()}>Description</th>
                <th style={th()}>Project</th>
                <th style={th()}>Assigned</th>
                <th style={{ ...th(), textAlign: 'right' }}>Open</th>
                <th style={{ ...th(), textAlign: 'right' }}>Age</th>
              </tr>
            </thead>
            <tbody>
              {items.map(it => (
                <Row key={it.study_id} item={it} onClick={() => onSelectStudy?.(it)} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function Row({ item, onClick }: { item: QCTriageItem; onClick: () => void }) {
  const sev = severityStyle(item.highest_severity_open)
  const age = item.created_at ? relativeTime(item.created_at) : ''
  return (
    <tr style={{ borderTop: '1px solid #f3f4f6', cursor: 'pointer' }} onClick={onClick}>
      <td style={td()}>
        <span style={{ ...pill(sev.color, sev.background), fontSize: 11, fontWeight: 600 }}>
          {sev.label}
        </span>
      </td>
      <td style={td()}>{item.modality}{item.body_part ? ` / ${item.body_part}` : ''}</td>
      <td style={td()}>
        <div>{item.study_description || <span style={{ color: '#9ca3af' }}>(no description)</span>}</div>
        <code style={{ fontSize: 11, color: '#9ca3af' }}>{shortenUid(item.study_instance_uid)}</code>
      </td>
      <td style={td()}>{item.project_name}</td>
      <td style={td()}>{item.qc_assigned_to_email || <span style={{ color: '#9ca3af' }}>unassigned</span>}</td>
      <td style={{ ...td(), textAlign: 'right', fontWeight: item.open_finding_count > 0 ? 600 : 400 }}>
        {item.open_finding_count}
      </td>
      <td style={{ ...td(), textAlign: 'right', color: '#6b7280' }}>{age}</td>
    </tr>
  )
}

function FilterBar({ filters, onChange }: {
  filters: api.TriageFilters
  onChange: (next: api.TriageFilters) => void
}) {
  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12, fontSize: 13 }}>
      <label>Show:&nbsp;
        <select
          value={filters.assigned ?? 'all'}
          onChange={e => onChange({ ...filters, assigned: e.target.value as 'me' | 'all' })}
        >
          <option value="me">Assigned to me</option>
          <option value="all">All</option>
        </select>
      </label>
      <label style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <input
          type="checkbox"
          checked={filters.openOnly ?? true}
          onChange={e => onChange({ ...filters, openOnly: e.target.checked })}
        />
        Open only
      </label>
    </div>
  )
}

// ── styles ────────────────────────────────────────────────────────────────

function th(): React.CSSProperties {
  return { padding: '6px 10px', textAlign: 'left', fontWeight: 600, fontSize: 12, color: '#374151', borderBottom: '1px solid #e5e7eb' }
}
function td(): React.CSSProperties {
  return { padding: '8px 10px', verticalAlign: 'top' }
}
function btn(): React.CSSProperties {
  return { padding: '4px 8px', borderRadius: 4, border: '1px solid #d1d5db', background: '#fff', cursor: 'pointer', fontSize: 12 }
}
function pill(color: string, background = '#f3f4f6'): React.CSSProperties {
  return {
    display: 'inline-block', padding: '2px 8px', borderRadius: 999,
    background, color, fontSize: 12, fontWeight: 500, whiteSpace: 'nowrap',
  }
}
function errorBox(): React.CSSProperties {
  return {
    padding: '8px 12px', background: '#fee2e2', border: '1px solid #fecaca',
    color: '#b91c1c', borderRadius: 6, marginBottom: 8, fontSize: 13,
  }
}

function shortenUid(uid: string): string {
  if (uid.length <= 30) return uid
  return `…${uid.slice(-28)}`
}

function relativeTime(ts: string): string {
  const t = new Date(ts).getTime()
  const ageSec = (Date.now() - t) / 1000
  if (ageSec < 60) return 'just now'
  if (ageSec < 3600) return `${Math.floor(ageSec / 60)}m`
  if (ageSec < 86400) return `${Math.floor(ageSec / 3600)}h`
  return `${Math.floor(ageSec / 86400)}d`
}
