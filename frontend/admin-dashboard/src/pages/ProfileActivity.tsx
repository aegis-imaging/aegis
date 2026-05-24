import { useEffect, useState } from 'react'
import { apiGetMe, type CurrentUser } from '../api/subjects'

// ProfileActivity is the per-user activity feed at /profile/activity. Shows
// the signed-in user's own audit entries — what they uploaded, approved,
// shared, etc. Backed by the existing /api/audit?actor= query parameter,
// which already supports filtering by email. No new backend endpoint
// needed for the admin case; non-admin researchers will need a relaxed
// auth rule on /api/audit when self-filtered (see followup memory).

type AuditEntry = {
  id: string
  actor: string
  action: string
  resource_type: string
  resource_id: string
  ip_address: string
  created_at: string
  detail: Record<string, unknown> | null
}

export function ProfileActivity() {
  const [me, setMe] = useState<CurrentUser | null>(null)
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setError(null)
      try {
        const user = await apiGetMe()
        if (cancelled) return
        setMe(user)
        const params = new URLSearchParams({ actor: user.email, limit: '100' })
        const res = await fetch(`/api/audit?${params}`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const body = await res.json()
        if (cancelled) return
        setEntries(body.entries ?? [])
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load activity')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => { cancelled = true }
  }, [])

  if (loading) return <div className="aegis-muted">Loading activity…</div>
  if (error)   return <div className="aegis-error">{error}</div>

  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>Your activity</h1>
            <p className="aegis-page-description">
              Recent actions performed by {me?.email ? <code className="aegis-code">{me.email}</code> : 'you'}.
              Showing up to the last 100 entries.
            </p>
          </div>
        </div>
      </header>

      <section className="aegis-section">
        {entries.length === 0 ? (
          <div className="aegis-muted">No activity recorded yet.</div>
        ) : (
          <div className="aegis-table-wrap">
            <table className="aegis-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Action</th>
                  <th>Resource</th>
                  <th>Detail</th>
                </tr>
              </thead>
              <tbody>
                {entries.map(e => {
                  const hasDetail = e.detail && Object.keys(e.detail).length > 0
                  const isExpanded = expandedId === e.id
                  return (
                    <tr key={e.id}>
                      <td className="aegis-muted">{new Date(e.created_at).toLocaleString()}</td>
                      <td><span className="aegis-pill">{e.action}</span></td>
                      <td>
                        <span className="aegis-muted" style={{ marginRight: 6 }}>{e.resource_type}</span>
                        <code className="aegis-code">{e.resource_id.slice(0, 8)}</code>
                      </td>
                      <td>
                        {hasDetail ? (
                          <button
                            type="button"
                            className="aegis-btn-secondary"
                            style={{ padding: '2px 8px', fontSize: 12 }}
                            onClick={() => setExpandedId(isExpanded ? null : e.id)}
                          >
                            {isExpanded ? 'hide' : 'show'}
                          </button>
                        ) : (
                          <span className="aegis-muted">—</span>
                        )}
                        {isExpanded && hasDetail && (
                          <pre style={{
                            whiteSpace: 'pre-wrap',
                            margin: '6px 0 0',
                            padding: 8,
                            background: 'var(--aegis-bg)',
                            borderRadius: 4,
                            fontFamily: 'ui-monospace, "SF Mono", Menlo, monospace',
                            fontSize: 11,
                          }}>
                            {JSON.stringify(e.detail, null, 2)}
                          </pre>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  )
}
