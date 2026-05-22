// Spokes management — list enrolled / pending AEGIS Routers, mint new
// enrollment tokens, revoke certs, view recent activity. Mirrors the data
// model that powers /api/spokes and the enrollment-token endpoints.

import { useEffect, useMemo, useState } from 'react'

type Spoke = {
  institution_id: string
  institution_name: string
  institution_slug: string
  enabled: boolean
  cert_thumbprint: string
  cert_subject_dn: string
  cert_enrolled_at?: string | null
  study_count: number
  last_study_at?: string | null
  active_token_count: number
}

type Institution = {
  id: string
  name: string
  slug: string
  institution_type: string
  enabled: boolean
}

type EnrollmentToken = {
  id: string
  label: string
  created_by: string
  created_at: string
  expires_at: string
  used_at?: string | null
  revoked_at?: string | null
  status: 'active' | 'used' | 'expired' | 'revoked'
}

type MintedTokenResponse = {
  id: string
  institution_id: string
  institution_slug: string
  label: string
  expires_at: string
  token: string
  install_command: string
}

export function SpokesPanel({ isAdmin }: { isAdmin: boolean }) {
  const [spokes, setSpokes] = useState<Spoke[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [showAdd, setShowAdd] = useState(false)
  const [drawerSpoke, setDrawerSpoke] = useState<Spoke | null>(null)
  const [refreshTick, setRefreshTick] = useState(0)

  useEffect(() => {
    setLoading(true)
    setError('')
    fetch('/api/spokes')
      .then(r => (r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`))))
      .then(d => setSpokes(d.spokes || []))
      .catch(e => setError(String(e)))
      .finally(() => setLoading(false))
  }, [refreshTick])

  const enrolledCount = useMemo(() => spokes.filter(s => s.cert_thumbprint).length, [spokes])
  const pendingCount = useMemo(() => spokes.filter(s => !s.cert_thumbprint && s.active_token_count > 0).length, [spokes])

  return (
    <div className="spokes-panel" style={{ padding: '16px 24px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 16 }}>
        <h2 style={{ margin: 0, flex: 1 }}>Spoke Routers</h2>
        <span className="badge" title="Spokes with a signed mTLS client cert">
          {enrolledCount} enrolled
        </span>
        {pendingCount > 0 && (
          <span className="badge badge-amber" title="Spokes with an active enrollment token but no cert yet">
            {pendingCount} pending
          </span>
        )}
        <button type="button" className="btn-refresh" onClick={() => setRefreshTick(t => t + 1)}>
          Refresh
        </button>
        {isAdmin && (
          <button type="button" className="btn-primary" onClick={() => setShowAdd(true)}>
            + Add spoke
          </button>
        )}
      </div>

      <p style={{ color: '#666', fontSize: 13, marginTop: 0 }}>
        On-prem AEGIS Routers that authenticate to this cloud via mTLS client cert. Add
        a new spoke by minting an enrollment token below and handing it to the site IT
        admin — they run <code>./bin/aegis-router-init --site-token=&lt;token&gt;</code> and
        the router auto-enrolls.
      </p>

      {loading && <div className="state-loading">Loading spokes…</div>}
      {error && <div className="state-error">{error}</div>}

      {!loading && !error && spokes.length === 0 && (
        <div className="state-empty" style={{ padding: '32px 16px', textAlign: 'center', color: '#666' }}>
          No spokes enrolled yet.
          {isAdmin && (
            <>
              <br />
              <button type="button" className="btn-primary" style={{ marginTop: 12 }} onClick={() => setShowAdd(true)}>
                + Add your first spoke
              </button>
            </>
          )}
        </div>
      )}

      {!loading && !error && spokes.length > 0 && (
        <table className="data-table" style={{ width: '100%' }}>
          <thead>
            <tr>
              <th>Site</th>
              <th>Status</th>
              <th>Studies received</th>
              <th>Last activity</th>
              <th>Enrolled</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {spokes.map(s => {
              const status = spokeStatus(s)
              return (
                <tr key={s.institution_id}>
                  <td>
                    <div style={{ fontWeight: 600 }}>{s.institution_name}</div>
                    <div style={{ color: '#888', fontSize: 12 }}>{s.institution_slug}</div>
                  </td>
                  <td><StatusBadge status={status} /></td>
                  <td>{s.study_count.toLocaleString()}</td>
                  <td>{relTime(s.last_study_at)}</td>
                  <td>{relTime(s.cert_enrolled_at)}</td>
                  <td>
                    <button type="button" className="btn-link" onClick={() => setDrawerSpoke(s)}>
                      Manage →
                    </button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}

      {showAdd && isAdmin && (
        <AddSpokeModal
          existingSpokes={spokes}
          onClose={() => setShowAdd(false)}
          onMinted={() => { setShowAdd(false); setRefreshTick(t => t + 1) }}
        />
      )}

      {drawerSpoke && (
        <SpokeDrawer
          spoke={drawerSpoke}
          isAdmin={isAdmin}
          onClose={() => setDrawerSpoke(null)}
          onChanged={() => { setRefreshTick(t => t + 1) }}
        />
      )}
    </div>
  )
}

// ── Status helpers ────────────────────────────────────────────────────────

type SpokeUIStatus = 'active' | 'pending' | 'idle' | 'disabled'

function spokeStatus(s: Spoke): SpokeUIStatus {
  if (!s.enabled) return 'disabled'
  if (!s.cert_thumbprint) return 'pending'
  if (s.last_study_at) {
    const ageMs = Date.now() - new Date(s.last_study_at).getTime()
    if (ageMs < 30 * 24 * 60 * 60 * 1000) return 'active'
  }
  return 'idle'
}

function StatusBadge({ status }: { status: SpokeUIStatus }) {
  const colors: Record<SpokeUIStatus, string> = {
    active: '#10b981',
    pending: '#f59e0b',
    idle: '#9ca3af',
    disabled: '#ef4444',
  }
  const labels: Record<SpokeUIStatus, string> = {
    active: '● active',
    pending: '◐ pending',
    idle: '○ idle',
    disabled: '⊘ disabled',
  }
  return (
    <span style={{
      color: colors[status],
      fontWeight: 500,
      fontSize: 13,
      whiteSpace: 'nowrap',
    }}>{labels[status]}</span>
  )
}

function relTime(ts: string | null | undefined): string {
  if (!ts) return '—'
  const d = new Date(ts)
  const ageSec = (Date.now() - d.getTime()) / 1000
  if (ageSec < 60) return 'just now'
  if (ageSec < 3600) return `${Math.floor(ageSec / 60)}m ago`
  if (ageSec < 86400) return `${Math.floor(ageSec / 3600)}h ago`
  if (ageSec < 30 * 86400) return `${Math.floor(ageSec / 86400)}d ago`
  return d.toLocaleDateString()
}

// ── Add spoke modal ───────────────────────────────────────────────────────

function AddSpokeModal({
  existingSpokes,
  onClose,
  onMinted,
}: {
  existingSpokes: Spoke[]
  onClose: () => void
  onMinted: () => void
}) {
  const [institutions, setInstitutions] = useState<Institution[]>([])
  const [selectedID, setSelectedID] = useState('')
  const [label, setLabel] = useState('')
  const [ttlHours, setTTLHours] = useState(72)
  const [minted, setMinted] = useState<MintedTokenResponse | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    fetch('/api/institutions')
      .then(r => r.json())
      .then((d: Institution[]) => setInstitutions(d || []))
      .catch(() => setInstitutions([]))
  }, [])

  const existingByID = useMemo(() => {
    const s = new Set<string>()
    for (const sp of existingSpokes) {
      if (sp.cert_thumbprint) s.add(sp.institution_id)
    }
    return s
  }, [existingSpokes])

  const eligible = useMemo(
    () => institutions.filter(i =>
      i.enabled
      && (i.institution_type === 'sender' || i.institution_type === 'both')
      && !existingByID.has(i.id)
    ),
    [institutions, existingByID]
  )

  async function mint() {
    if (!selectedID) { setError('Pick an institution.'); return }
    setSubmitting(true)
    setError('')
    try {
      const r = await fetch(`/api/institutions/${selectedID}/enrollment-tokens`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ label, ttl_hours: ttlHours }),
      })
      if (!r.ok) {
        const body = await r.text()
        throw new Error(`HTTP ${r.status}: ${body}`)
      }
      const data: MintedTokenResponse = await r.json()
      setMinted(data)
    } catch (e) {
      setError(String(e))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 640 }}>
        <h3 style={{ marginTop: 0 }}>Add a spoke router</h3>

        {!minted && (
          <>
            <p style={{ color: '#666', fontSize: 13 }}>
              Mint a one-time enrollment token for an institution. The spoke admin runs
              the install command on their router host; the cert auto-enrolls.
            </p>

            <label>
              Institution
              <select value={selectedID} onChange={e => setSelectedID(e.target.value)}>
                <option value="">— select —</option>
                {eligible.map(i => (
                  <option key={i.id} value={i.id}>{i.name} ({i.slug})</option>
                ))}
              </select>
            </label>

            {eligible.length === 0 && (
              <div className="state-empty" style={{ fontSize: 12, color: '#888' }}>
                All sender/both institutions already have an enrolled spoke. Create a new
                institution in the Institutions tab to add another.
              </div>
            )}

            <label>
              Label (optional, e.g. "umn-lab-laptop-test")
              <input value={label} onChange={e => setLabel(e.target.value)} />
            </label>

            <label>
              Valid for
              <select value={String(ttlHours)} onChange={e => setTTLHours(Number(e.target.value))}>
                <option value="1">1 hour</option>
                <option value="24">24 hours</option>
                <option value="72">3 days</option>
                <option value="168">7 days</option>
                <option value="720">30 days (max)</option>
              </select>
            </label>

            {error && <div className="state-error" style={{ marginTop: 8 }}>{error}</div>}

            <div className="modal-actions">
              <button type="button" className="btn" onClick={onClose}>Cancel</button>
              <button type="button" className="btn-primary" disabled={submitting || !selectedID} onClick={mint}>
                {submitting ? 'Minting…' : 'Mint token'}
              </button>
            </div>
          </>
        )}

        {minted && (
          <>
            <div className="state-success" style={{ marginBottom: 12 }}>
              ✓ Token minted. This is the only time it will be shown — copy it now.
            </div>

            <label>
              Raw token (one-time)
              <textarea
                readOnly
                value={minted.token}
                rows={2}
                style={{ fontFamily: 'monospace', fontSize: 13 }}
                onFocus={e => e.currentTarget.select()}
              />
            </label>

            <label>
              Install command — paste this on the spoke host
              <textarea
                readOnly
                value={minted.install_command}
                rows={3}
                style={{ fontFamily: 'monospace', fontSize: 13 }}
                onFocus={e => e.currentTarget.select()}
              />
            </label>

            <div style={{ fontSize: 12, color: '#666' }}>
              Token expires {new Date(minted.expires_at).toLocaleString()}.
              When the spoke completes the bootstrap, it appears in the Spokes list
              with a signed cert and status <em>active</em>.
            </div>

            <div className="modal-actions">
              <button type="button" className="btn-primary" onClick={onMinted}>Done</button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

// ── Spoke management drawer ──────────────────────────────────────────────

function SpokeDrawer({
  spoke,
  isAdmin,
  onClose,
  onChanged,
}: {
  spoke: Spoke
  isAdmin: boolean
  onClose: () => void
  onChanged: () => void
}) {
  const [tokens, setTokens] = useState<EnrollmentToken[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    fetch(`/api/institutions/${spoke.institution_id}/enrollment-tokens`)
      .then(r => r.json())
      .then(d => setTokens(d.tokens || []))
      .finally(() => setLoading(false))
  }, [spoke.institution_id])

  async function revokeCert() {
    if (!confirm(`Revoke the mTLS cert for ${spoke.institution_name}? The spoke will stop being able to push studies until re-enrolled.`)) return
    const r = await fetch(`/api/institutions/${spoke.institution_id}/client-cert`, { method: 'DELETE' })
    if (r.ok) { onChanged(); onClose() }
    else alert(`Revoke failed: HTTP ${r.status}`)
  }

  async function revokeToken(tokenID: string) {
    const r = await fetch(`/api/institutions/${spoke.institution_id}/enrollment-tokens/${tokenID}`, { method: 'DELETE' })
    if (r.ok) {
      setTokens(tokens.map(t => t.id === tokenID ? { ...t, status: 'revoked' as const } : t))
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()} style={{ maxWidth: 720 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <h3 style={{ margin: 0, flex: 1 }}>{spoke.institution_name}</h3>
          <button type="button" className="btn" onClick={onClose}>Close</button>
        </div>

        <section style={{ marginTop: 16 }}>
          <h4 style={{ margin: '0 0 8px 0' }}>Identity</h4>
          <table className="kv-table">
            <tbody>
              <tr><th>Slug</th><td>{spoke.institution_slug}</td></tr>
              <tr><th>Status</th><td><StatusBadge status={spokeStatus(spoke)} /></td></tr>
              <tr><th>Studies received (spoke source)</th><td>{spoke.study_count.toLocaleString()}</td></tr>
              <tr><th>Last activity</th><td>{relTime(spoke.last_study_at)}</td></tr>
              <tr><th>Cert thumbprint</th><td><code style={{ fontSize: 11 }}>{spoke.cert_thumbprint || '—'}</code></td></tr>
              <tr><th>Cert subject DN</th><td><code style={{ fontSize: 11 }}>{spoke.cert_subject_dn || '—'}</code></td></tr>
              <tr><th>Cert enrolled at</th><td>{spoke.cert_enrolled_at ? new Date(spoke.cert_enrolled_at).toLocaleString() : '—'}</td></tr>
            </tbody>
          </table>
        </section>

        <section style={{ marginTop: 16 }}>
          <h4 style={{ margin: '0 0 8px 0' }}>Enrollment tokens</h4>
          {loading ? (
            <div className="state-loading">Loading…</div>
          ) : tokens.length === 0 ? (
            <div className="state-empty" style={{ fontSize: 13, color: '#888' }}>No tokens ever minted.</div>
          ) : (
            <table className="data-table" style={{ width: '100%', fontSize: 13 }}>
              <thead>
                <tr><th>Label</th><th>Status</th><th>Created</th><th>Expires</th><th /></tr>
              </thead>
              <tbody>
                {tokens.map(t => (
                  <tr key={t.id}>
                    <td>{t.label || <span style={{ color: '#888' }}>(none)</span>}</td>
                    <td>{t.status}</td>
                    <td>{new Date(t.created_at).toLocaleString()}</td>
                    <td>{new Date(t.expires_at).toLocaleString()}</td>
                    <td>
                      {isAdmin && t.status === 'active' && (
                        <button type="button" className="btn-link" onClick={() => revokeToken(t.id)}>Revoke</button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

        {isAdmin && spoke.cert_thumbprint && (
          <section style={{ marginTop: 16 }}>
            <h4 style={{ margin: '0 0 8px 0', color: '#b91c1c' }}>Danger zone</h4>
            <button type="button" className="btn-danger" onClick={revokeCert}>
              Revoke mTLS cert
            </button>
            <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>
              Permanently removes this spoke's cert thumbprint. To restore access, mint a
              new enrollment token and re-run the bootstrap.
            </div>
          </section>
        )}
      </div>
    </div>
  )
}
