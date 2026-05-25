import { useEffect, useState } from 'react'

// SharesPage is the researcher-facing shares list at /shares.
//
// Distinct from /admin/shares — that page is the global share manager
// with bulk download analytics and per-share download history. This
// page just shows the user's outgoing shares with status + expiry, so
// researchers can answer "who did I share what with, and is the link
// still alive?" without bouncing into admin chrome.
//
// Backend uses the same /api/shares endpoint; scoping is handled by
// the caller's IAP identity. If a researcher hits this page and they
// own no shares yet, they see the empty state with a hint toward the
// per-study share button.
type ShareRow = {
  id: string
  study_id: string
  recipient_email: string
  note: string
  expires_at: string
  revoked_at?: string | null
  status?: 'active' | 'expired' | 'revoked'
  created_at: string
  download_count?: number
}

export function SharesPage() {
  const [rows, setRows] = useState<ShareRow[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'expired' | 'revoked'>('all')

  useEffect(() => {
    let cancelled = false
    setRows(null)
    setError(null)
    const params = new URLSearchParams({ limit: '100' })
    if (statusFilter !== 'all') params.set('status', statusFilter)
    fetch(`/api/shares?${params}`)
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`)))
      .then((d: { shares: ShareRow[] }) => {
        if (cancelled) return
        setRows(d.shares ?? [])
      })
      .catch(e => !cancelled && setError(e instanceof Error ? e.message : String(e)))
    return () => { cancelled = true }
  }, [statusFilter])

  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>Shares</h1>
            <p className="aegis-page-description">
              Outgoing share links you've created. To mint a new share for a specific study,
              open the study and use the Share action.
            </p>
          </div>
        </div>
      </header>

      <section className="aegis-section">
        <div className="aegis-chip-row">
          <span className="aegis-chip-row-label">Status:</span>
          {(['all', 'active', 'expired', 'revoked'] as const).map(s => (
            <button
              key={s}
              type="button"
              className={`aegis-chip${statusFilter === s ? ' aegis-chip-active' : ''}`}
              onClick={() => setStatusFilter(s)}
            >
              {s}
            </button>
          ))}
        </div>

        {error && <div className="aegis-error">{error}</div>}
        {rows === null && !error && <div className="aegis-muted">Loading shares…</div>}
        {rows && rows.length === 0 && !error && (
          <div className="aegis-muted">No shares match the current filter.</div>
        )}

        {rows && rows.length > 0 && (
          <div className="aegis-table-wrap">
            <table className="aegis-table">
              <thead>
                <tr>
                  <th>Recipient</th>
                  <th>Note</th>
                  <th>Status</th>
                  <th>Expires</th>
                  <th>Downloads</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {rows.map(s => (
                  <tr key={s.id}>
                    <td><code className="aegis-code">{s.recipient_email}</code></td>
                    <td className="aegis-muted aegis-cell-truncate">{s.note || '—'}</td>
                    <td>
                      <span className="aegis-pill">
                        {s.status ?? (s.revoked_at ? 'revoked' : 'active')}
                      </span>
                    </td>
                    <td className="aegis-muted">{new Date(s.expires_at).toLocaleDateString()}</td>
                    <td>{s.download_count ?? 0}</td>
                    <td className="aegis-muted">{new Date(s.created_at).toLocaleDateString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </>
  )
}
