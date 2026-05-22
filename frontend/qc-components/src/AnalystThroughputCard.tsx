// Simple stats card showing every active analyst's QC throughput over the
// configured lookback window.

import { useEffect, useState } from 'react'
import * as api from './api'
import type { AnalystThroughput } from './types'

export interface AnalystThroughputCardProps {
  apiOptions?: api.APIClientOptions
  /** Lookback window in days. Default 7. */
  days?: number
  /** Hide the days dropdown — useful when host already provides it. */
  hideControls?: boolean
}

export function AnalystThroughputCard({
  apiOptions, days: initialDays = 7, hideControls,
}: AnalystThroughputCardProps) {
  const [days, setDays] = useState(initialDays)
  const [rows, setRows] = useState<AnalystThroughput[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    api.getAnalystThroughput(days, apiOptions)
      .then(r => { if (!cancelled) setRows(r) })
      .catch(e => { if (!cancelled) setError(String(e)) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [days, apiOptions])

  return (
    <div className="qc-analyst-throughput" style={{ border: '1px solid #e5e7eb', borderRadius: 8, padding: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 10 }}>
        <h3 style={{ margin: 0, fontSize: 16 }}>Analyst throughput</h3>
        {!hideControls && (
          <label style={{ fontSize: 12, color: '#6b7280' }}>
            Last&nbsp;
            <select value={days} onChange={e => setDays(Number(e.target.value))}>
              <option value={1}>day</option>
              <option value={7}>7 days</option>
              <option value={30}>30 days</option>
              <option value={90}>90 days</option>
            </select>
          </label>
        )}
      </div>

      {error && (
        <div style={{ padding: 8, background: '#fee2e2', color: '#b91c1c', borderRadius: 4, fontSize: 12 }}>{error}</div>
      )}
      {loading && rows.length === 0 && <div style={{ color: '#6b7280', fontSize: 13 }}>Loading…</div>}

      {rows.length > 0 && (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
          <thead>
            <tr>
              <th style={th()}>Analyst</th>
              <th style={{ ...th(), textAlign: 'right' }}>Completed</th>
              <th style={{ ...th(), textAlign: 'right' }}>Open</th>
              <th style={{ ...th(), textAlign: 'right' }}>Findings raised</th>
              <th style={{ ...th(), textAlign: 'right' }}>Findings resolved</th>
              <th style={{ ...th(), textAlign: 'right' }}>Avg min/study</th>
              <th style={{ ...th(), textAlign: 'right' }}>Studies/hr</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(r => (
              <tr key={r.analyst_id} style={{ borderTop: '1px solid #f3f4f6' }}>
                <td style={td()}>
                  <div style={{ fontWeight: 500 }}>{r.analyst_name || r.analyst_email}</div>
                  <div style={{ fontSize: 11, color: '#9ca3af' }}>{r.analyst_email}</div>
                </td>
                <td style={{ ...td(), textAlign: 'right' }}>{r.studies_completed}</td>
                <td style={{ ...td(), textAlign: 'right' }}>{r.open_assignments}</td>
                <td style={{ ...td(), textAlign: 'right' }}>{r.findings_raised}</td>
                <td style={{ ...td(), textAlign: 'right' }}>{r.findings_resolved}</td>
                <td style={{ ...td(), textAlign: 'right' }}>
                  {r.avg_review_minutes > 0 ? r.avg_review_minutes.toFixed(1) : '—'}
                </td>
                <td style={{ ...td(), textAlign: 'right', fontWeight: 500 }}>
                  {r.studies_per_hour > 0 ? r.studies_per_hour.toFixed(1) : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {!loading && rows.length === 0 && !error && (
        <div style={{ color: '#6b7280', fontSize: 13, padding: 8 }}>
          No analyst activity in the selected window.
        </div>
      )}
    </div>
  )
}

function th(): React.CSSProperties {
  return { padding: '6px 10px', textAlign: 'left', fontWeight: 600, fontSize: 12, color: '#374151', borderBottom: '1px solid #e5e7eb' }
}
function td(): React.CSSProperties {
  return { padding: '6px 10px', verticalAlign: 'top' }
}
