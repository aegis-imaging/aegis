import { useEffect, useState } from 'react'
import { apiGetMe, type CurrentUser } from '../api/subjects'

// ProfileNotifications is the per-user self-service notification preferences
// page at /profile/notifications. Lets the signed-in user pick their digest
// email frequency and which events trigger a notification, without having
// to ask an admin to edit it on their behalf via the admin UsersPanel.
//
// Backend: hits the existing /api/admin-users/{id}/preferences endpoint
// (which admins already use to edit anyone's prefs). For Matt — the only
// user today and an admin — this works directly. When non-admin researchers
// arrive, the Go middleware will need a new "you may read/write your own
// row" rule. See followup_audit_split_notifications_move.md.

type Preferences = {
  digest_frequency: 'none' | 'daily' | 'weekly' | 'monthly'
  notify_events: string[]
}

const NOTIFY_EVENT_OPTIONS: { value: string; label: string }[] = [
  { value: 'study.stuck',       label: 'Study stuck (idle beyond SLA)' },
  { value: 'pipeline.failed',   label: 'Pipeline step failed' },
  { value: 'study.phi_flagged', label: 'PHI flagged in scan' },
  { value: 'study.approved',    label: 'Study approved' },
  { value: 'study.rejected',    label: 'Study rejected' },
]

export function ProfileNotifications() {
  const [me, setMe]       = useState<CurrentUser | null>(null)
  const [prefs, setPrefs] = useState<Preferences | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving]   = useState(false)
  const [error, setError]     = useState<string | null>(null)
  const [savedAt, setSavedAt] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false
    async function load() {
      setLoading(true)
      setError(null)
      try {
        const user = await apiGetMe()
        if (cancelled) return
        setMe(user)
        const res = await fetch(`/api/admin-users/${user.id}/preferences`)
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        const body = await res.json()
        if (cancelled) return
        setPrefs({
          digest_frequency: body.digest_frequency ?? 'weekly',
          notify_events:    body.notify_events ?? [],
        })
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load preferences')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => { cancelled = true }
  }, [])

  async function save() {
    if (!me || !prefs) return
    setSaving(true)
    setError(null)
    try {
      const res = await fetch(`/api/admin-users/${me.id}/preferences`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(prefs),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error ?? `HTTP ${res.status}`)
      }
      setSavedAt(Date.now())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="aegis-muted">Loading preferences…</div>
  if (error)   return <div className="aegis-error">{error}</div>
  if (!me || !prefs) return null

  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>Notification preferences</h1>
            <p className="aegis-page-description">
              How AEGIS reaches you. Changes apply to <code className="aegis-code">{me.email}</code>.
            </p>
          </div>
        </div>
      </header>

      <section className="aegis-section" style={{ maxWidth: 640 }}>
        <div className="aegis-form-row">
          <label htmlFor="digest-freq">Digest email frequency</label>
          <select
            id="digest-freq"
            value={prefs.digest_frequency}
            onChange={e => setPrefs({ ...prefs, digest_frequency: e.target.value as Preferences['digest_frequency'] })}
          >
            <option value="none">None — no digest emails</option>
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
          </select>
          <span className="aegis-form-hint">Sent at the start of the chosen period.</span>
        </div>

        <div className="aegis-form-row">
          <label>Notify me on events</label>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {NOTIFY_EVENT_OPTIONS.map(opt => {
              const checked = prefs.notify_events.includes(opt.value)
              return (
                <label key={opt.value} style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontWeight: 'normal' }}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => {
                      setPrefs({
                        ...prefs,
                        notify_events: checked
                          ? prefs.notify_events.filter(e => e !== opt.value)
                          : [...prefs.notify_events, opt.value],
                      })
                    }}
                  />
                  <span>{opt.label}</span>
                  <code className="aegis-code" style={{ fontSize: 11 }}>{opt.value}</code>
                </label>
              )
            })}
          </div>
        </div>

        <div className="aegis-form-actions">
          <button type="button" className="aegis-btn-primary" onClick={save} disabled={saving}>
            {saving ? 'Saving…' : 'Save preferences'}
          </button>
          {savedAt && Date.now() - savedAt < 5000 && (
            <span className="aegis-muted">Saved.</span>
          )}
        </div>
      </section>
    </>
  )
}
