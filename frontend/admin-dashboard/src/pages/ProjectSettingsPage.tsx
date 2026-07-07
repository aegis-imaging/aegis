import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

// ProjectSettingsPage is the per-project settings home at
// /projects/:projectId/settings. Surfaces the project-level controls
// that used to live (or still live) under the admin tabs:
//
//   - General: description, retention_days, stuck_threshold_minutes
//   - Anonymization: the project's default profile + a list of its
//     custom profiles (with deep-link into the admin editor for changes)
//   - Protocol templates: the project's templates (likewise)
//
// This is part of the institutions/projects hierarchy reorg:
// `/admin/profiles` and `/admin/protocol_templates` remain as global
// browse views, but the natural per-project entry point is here so
// project owners don't have to filter the global tabs.
//
// Backend uses the per-project endpoints that already exist:
//   GET    /api/projects/:id
//   PUT    /api/projects/:id
//   GET    /api/projects/:id/anon-profiles
//   GET    /api/projects/:id/protocol-templates
//   GET    /api/projects/:id/uploaders            (403 for non-owners)
//   POST   /api/projects/:id/uploaders
//   DELETE /api/projects/:id/uploaders/:userId

type Project = {
  id: string
  name: string
  slug: string
  description: string
  default_anon_profile_id?: string | null
  retention_days?: number | null
  stuck_threshold_minutes?: number | null
  storage_quota_bytes?: number | null
  archived?: boolean
}

type AnonProfile = {
  id: string
  project_id: string
  name: string
  description: string
  is_default: boolean
}

type ProtocolTemplate = {
  id: string
  project_id: string
  name: string
  modality: string
  body_part: string
}

// Active project member with role=uploader (subset of the API's ProjectMember).
type ProjectUploader = {
  id: string
  admin_user_id: string
  user_email?: string
  user_name?: string
  institution_name?: string
  created_at: string
}

// Pending (un-redeemed, un-expired) invite. invite_token is blanked server-side.
type UploaderInvite = {
  id: string
  email: string
  name?: string
  invited_by?: string
  institution_name?: string
  expires_at: string
  created_at: string
}

// One registered desktop installer build (subset of the API's DesktopInstaller).
type DesktopInstaller = {
  id: string
  product: string
  platform: string
  version: string
  is_current: boolean
}

// Readable platform labels — mirrors allowedInstallerPlatforms in the API.
const INSTALLER_PLATFORM_LABELS: Record<string, string> = {
  'macos-arm64': 'macOS (Apple Silicon)',
  'macos-x64': 'macOS (Intel)',
  'windows-x64': 'Windows (64-bit)',
  'linux-deb': 'Linux (.deb)',
  'linux-appimage': 'Linux (AppImage)',
  'linux-rpm': 'Linux (.rpm)',
}

// pickUploaderInstallers returns one uploader-product installer per platform:
// the is_current build when marked, otherwise the newest (the API lists rows
// newest-first per product/platform, so the first row seen wins).
function pickUploaderInstallers(installers: DesktopInstaller[]): DesktopInstaller[] {
  const byPlatform = new Map<string, DesktopInstaller>()
  for (const inst of installers) {
    if (inst.product !== 'uploader') continue
    const cur = byPlatform.get(inst.platform)
    if (!cur || (inst.is_current && !cur.is_current)) byPlatform.set(inst.platform, inst)
  }
  return [...byPlatform.values()]
}

function fmtDate(iso: string): string {
  const d = new Date(iso)
  return isNaN(d.getTime()) ? '—' : d.toLocaleDateString()
}

export function ProjectSettingsPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const [project, setProject] = useState<Project | null>(null)
  const [profiles, setProfiles] = useState<AnonProfile[]>([])
  const [templates, setTemplates] = useState<ProtocolTemplate[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Edit form state — mirrors the project fields, initialized after load.
  const [description, setDescription] = useState('')
  const [retentionDays, setRetentionDays] = useState<number | ''>('')
  const [stuckMinutes, setStuckMinutes] = useState<number | ''>('')
  const [defaultProfileId, setDefaultProfileId] = useState<string>('')
  const [saving, setSaving] = useState(false)
  const [savedAt, setSavedAt] = useState<number | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)

  // Uploaders section — loaded separately so a 403 (non-owner researcher)
  // degrades to an access note without breaking the rest of the page.
  const [uploaders, setUploaders] = useState<ProjectUploader[]>([])
  const [pendingInvites, setPendingInvites] = useState<UploaderInvite[]>([])
  const [uploadersLoading, setUploadersLoading] = useState(true)
  const [uploadersError, setUploadersError] = useState<string | null>(null)
  const [uploadersForbidden, setUploadersForbidden] = useState(false)
  const [revokeError, setRevokeError] = useState<string | null>(null)

  // Invite form state
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteName, setInviteName] = useState('')
  const [inviteExpiryDays, setInviteExpiryDays] = useState(7)
  const [inviting, setInviting] = useState(false)
  const [inviteError, setInviteError] = useState<string | null>(null)
  const [smtpUnavailable, setSmtpUnavailable] = useState(false)
  const [inviteSentTo, setInviteSentTo] = useState<string | null>(null)
  const [redeemUrl, setRedeemUrl] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  // Desktop uploader sub-block — send a project-scoped installer invite.
  // Pairing the installed app then grants upload access to this project.
  const [installers, setInstallers] = useState<DesktopInstaller[]>([])
  const [installersLoading, setInstallersLoading] = useState(true)
  const [desktopEmail, setDesktopEmail] = useState('')
  const [desktopInstallerId, setDesktopInstallerId] = useState('')
  const [desktopSending, setDesktopSending] = useState(false)
  const [desktopError, setDesktopError] = useState<string | null>(null)
  const [desktopSmtpUnavailable, setDesktopSmtpUnavailable] = useState(false)
  const [desktopSentTo, setDesktopSentTo] = useState<string | null>(null)

  useEffect(() => {
    if (!projectId) return
    let cancelled = false
    setLoading(true)
    setError(null)
    Promise.all([
      fetch(`/api/projects/${projectId}`).then(r => r.ok ? r.json() : Promise.reject(new Error(`project HTTP ${r.status}`))),
      fetch(`/api/projects/${projectId}/anon-profiles`).then(r => r.ok ? r.json() : []),
      fetch(`/api/projects/${projectId}/protocol-templates`).then(r => r.ok ? r.json() : []),
    ])
      .then(([p, pr, tpl]: [Project, AnonProfile[] | { profiles: AnonProfile[] }, ProtocolTemplate[] | { templates: ProtocolTemplate[] }]) => {
        if (cancelled) return
        setProject(p)
        setDescription(p.description ?? '')
        setRetentionDays(p.retention_days ?? '')
        setStuckMinutes(p.stuck_threshold_minutes ?? '')
        setDefaultProfileId(p.default_anon_profile_id ?? '')
        // Endpoints sometimes return bare arrays, sometimes wrap in {profiles}/{templates}.
        setProfiles(Array.isArray(pr) ? pr : (pr.profiles ?? []))
        setTemplates(Array.isArray(tpl) ? tpl : (tpl.templates ?? []))
      })
      .catch(e => !cancelled && setError(e instanceof Error ? e.message : String(e)))
      .finally(() => !cancelled && setLoading(false))
    return () => { cancelled = true }
  }, [projectId])

  const loadUploaders = useCallback(async () => {
    if (!projectId) return
    setUploadersError(null)
    try {
      const res = await fetch(`/api/projects/${projectId}/uploaders`)
      if (res.status === 403) {
        // Non-owners can see the settings page but not manage uploaders —
        // render the access note instead of the table+form.
        setUploadersForbidden(true)
        return
      }
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as {
        uploaders?: ProjectUploader[]
        pending_invites?: UploaderInvite[]
      }
      setUploadersForbidden(false)
      setUploaders(data.uploaders ?? [])
      setPendingInvites(data.pending_invites ?? [])
    } catch (e) {
      setUploadersError(e instanceof Error ? e.message : String(e))
    } finally {
      setUploadersLoading(false)
    }
  }, [projectId])

  useEffect(() => { void loadUploaders() }, [loadUploaders])

  useEffect(() => {
    let cancelled = false
    fetch('/api/desktop-installers')
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`HTTP ${r.status}`)))
      .then((data: { installers?: DesktopInstaller[] }) => {
        if (cancelled) return
        const all = data.installers ?? []
        setInstallers(all)
        const picked = pickUploaderInstallers(all)
        if (picked.length > 0) setDesktopInstallerId(picked[0].id)
      })
      .catch(() => {
        // Endpoint unavailable — the sub-block degrades to the "no builds
        // registered" note rather than breaking the settings page.
        if (!cancelled) setInstallers([])
      })
      .finally(() => !cancelled && setInstallersLoading(false))
    return () => { cancelled = true }
  }, [])

  async function inviteUploader() {
    if (!projectId) return
    setInviteError(null)
    setSmtpUnavailable(false)
    setInviteSentTo(null)
    setRedeemUrl(null)
    setCopied(false)
    const email = inviteEmail.trim().toLowerCase()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setInviteError('Enter a valid email address.')
      return
    }
    setInviting(true)
    try {
      const res = await fetch(`/api/projects/${projectId}/uploaders`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, name: inviteName.trim(), expiry_days: inviteExpiryDays }),
      })
      if (res.status === 503) {
        // The handler refuses before creating the invite when SMTP_HOST is
        // unset — nothing was created, so don't refresh or clear the form.
        setSmtpUnavailable(true)
        return
      }
      const data = (await res.json().catch(() => ({}))) as {
        error?: string
        redeem_url?: string
        email_sent?: boolean
      }
      if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`)
      if (data.email_sent) {
        setInviteSentTo(email)
      } else {
        // Invite row exists but the email bounced at send time (200 +
        // email_sent:false) — surface the redeem link for manual delivery.
        setRedeemUrl(data.redeem_url ?? null)
      }
      setInviteEmail('')
      setInviteName('')
      await loadUploaders()
    } catch (err) {
      setInviteError(err instanceof Error ? err.message : 'Invite failed')
    } finally {
      setInviting(false)
    }
  }

  async function revokeUploader(u: ProjectUploader) {
    if (!projectId) return
    const label = u.user_email || u.user_name || u.admin_user_id
    if (!window.confirm(`Revoke upload access for ${label}? Their active sessions will be ended.`)) return
    setRevokeError(null)
    try {
      const res = await fetch(`/api/projects/${projectId}/uploaders/${u.admin_user_id}`, {
        method: 'DELETE',
      })
      if (!res.ok) {
        // 409 = target isn't role=uploader; surface the server's message.
        const data = (await res.json().catch(() => ({}))) as { error?: string }
        throw new Error(data.error || `HTTP ${res.status}`)
      }
      await loadUploaders()
    } catch (err) {
      setRevokeError(err instanceof Error ? err.message : 'Revoke failed')
    }
  }

  async function sendDesktopInvite() {
    if (!projectId || !desktopInstallerId) return
    setDesktopError(null)
    setDesktopSmtpUnavailable(false)
    setDesktopSentTo(null)
    const email = desktopEmail.trim().toLowerCase()
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setDesktopError('Enter a valid email address.')
      return
    }
    setDesktopSending(true)
    try {
      const res = await fetch(`/api/desktop-installers/${desktopInstallerId}/email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ recipient_email: email, project_id: projectId }),
      })
      if (res.status === 503) {
        // Handler refuses before creating the invite when SMTP_HOST is unset.
        setDesktopSmtpUnavailable(true)
        return
      }
      if (!res.ok) {
        const data = (await res.json().catch(() => ({}))) as { error?: string }
        throw new Error(data.error || `HTTP ${res.status}`)
      }
      setDesktopSentTo(email)
      setDesktopEmail('')
    } catch (err) {
      setDesktopError(err instanceof Error ? err.message : 'Send failed')
    } finally {
      setDesktopSending(false)
    }
  }

  async function copyRedeemUrl() {
    if (!redeemUrl) return
    try {
      await navigator.clipboard.writeText(redeemUrl)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard API unavailable (e.g. non-secure context) — the readonly
      // input still lets the user select and copy manually.
    }
  }

  async function save() {
    if (!project) return
    setSaving(true)
    setSaveError(null)
    try {
      const putJSON = async (url: string, body: unknown) => {
        const res = await fetch(url, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        })
        if (!res.ok) {
          const t = await res.text()
          throw new Error(t || `HTTP ${res.status}`)
        }
        return res
      }
      // PUT /api/projects/:id only accepts name/slug/description — send just
      // those (previously we spread the whole project object, whose extra
      // fields the handler silently ignored). The remaining editable settings
      // each have a dedicated endpoint.
      const res = await putJSON(`/api/projects/${project.id}`, {
        name: project.name,
        slug: project.slug,
        description,
      })
      await putJSON(`/api/projects/${project.id}/retention`, {
        retention_days: retentionDays === '' ? null : retentionDays,
      })
      await putJSON(`/api/projects/${project.id}/sla-threshold`, {
        stuck_threshold_minutes: stuckMinutes === '' ? null : stuckMinutes,
      })
      await putJSON(`/api/projects/${project.id}/default-anon-profile`, {
        profile_id: defaultProfileId,
      })
      const updated = (await res.json()) as Project
      setProject({
        ...updated,
        retention_days: retentionDays === '' ? null : retentionDays,
        stuck_threshold_minutes: stuckMinutes === '' ? null : stuckMinutes,
        default_anon_profile_id: defaultProfileId || null,
      })
      setSavedAt(Date.now())
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="aegis-muted">Loading project settings…</div>
  if (error)   return <div className="aegis-error">{error}</div>
  if (!project) return null

  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>{project.name}</h1>
            <p className="aegis-page-description">
              Settings for <code className="aegis-code">{project.slug}</code>. Changes apply to every
              study in this project.
              {' '}
              <Link to={`/projects/${project.id}`} style={{ color: 'var(--aegis-link)' }}>
                ← Back to subjects
              </Link>
            </p>
          </div>
        </div>
      </header>

      {/* General settings — editable inline */}
      <section className="aegis-section">
        <div className="aegis-section-bar">
          <h2>General</h2>
        </div>
        {saveError && <div className="aegis-error">{saveError}</div>}
        <div className="aegis-form-row">
          <label htmlFor="proj-description">Description</label>
          <textarea
            id="proj-description"
            rows={2}
            value={description}
            onChange={e => setDescription(e.target.value)}
            placeholder="What is this project for?"
          />
        </div>
        <div className="aegis-form-row">
          <label htmlFor="proj-retention">Retention (days)</label>
          <input
            id="proj-retention"
            type="number"
            min={1}
            value={retentionDays}
            onChange={e => setRetentionDays(e.target.value === '' ? '' : parseInt(e.target.value, 10) || '')}
            placeholder="Leave blank for indefinite"
          />
          <span className="aegis-form-hint">
            Approved studies are automatically expired after this many days. Leave blank to keep forever.
          </span>
        </div>
        <div className="aegis-form-row">
          <label htmlFor="proj-stuck">Stuck threshold (minutes)</label>
          <input
            id="proj-stuck"
            type="number"
            min={1}
            value={stuckMinutes}
            onChange={e => setStuckMinutes(e.target.value === '' ? '' : parseInt(e.target.value, 10) || '')}
            placeholder="Default: 60"
          />
          <span className="aegis-form-hint">
            Studies still in the pipeline longer than this trigger stuck-study alerts on the digest.
          </span>
        </div>
        <div className="aegis-form-row">
          <label htmlFor="proj-default-profile">Default anonymization profile</label>
          <select
            id="proj-default-profile"
            value={defaultProfileId}
            onChange={e => setDefaultProfileId(e.target.value)}
          >
            <option value="">No default (use system fallback)</option>
            {profiles.map(p => (
              <option key={p.id} value={p.id}>
                {p.name}{p.is_default ? ' (current default)' : ''}
              </option>
            ))}
          </select>
          <span className="aegis-form-hint">
            Applies to new uploads. Already-anonymized studies keep their original profile.
          </span>
        </div>
        <div className="aegis-form-actions">
          <button type="button" className="aegis-btn-primary" onClick={save} disabled={saving}>
            {saving ? 'Saving…' : 'Save settings'}
          </button>
          {savedAt && Date.now() - savedAt < 5000 && (
            <span className="aegis-muted">Saved.</span>
          )}
        </div>
      </section>

      {/* Anonymization profiles attached to this project */}
      <section className="aegis-section">
        <div className="aegis-section-bar">
          <h2>Anonymization profiles ({profiles.length})</h2>
          <Link to="/profiles" className="aegis-btn-secondary" style={{ textDecoration: 'none' }}>
            Manage profiles
          </Link>
        </div>
        {profiles.length === 0 ? (
          <div className="aegis-muted">
            No anon profiles attached. The system default applies until you add one — manage them in
            the admin Profiles tab.
          </div>
        ) : (
          <div className="aegis-table-wrap">
            <table className="aegis-table">
              <thead><tr><th>Name</th><th>Default?</th><th>Description</th></tr></thead>
              <tbody>
                {profiles.map(p => (
                  <tr key={p.id}>
                    <td><code className="aegis-code">{p.name}</code></td>
                    <td>{p.is_default ? <span className="aegis-pill">default</span> : <span className="aegis-muted">—</span>}</td>
                    <td className="aegis-muted">{p.description || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Protocol templates attached to this project */}
      <section className="aegis-section">
        <div className="aegis-section-bar">
          <h2>Protocol templates ({templates.length})</h2>
          <Link to="/protocol_templates" className="aegis-btn-secondary" style={{ textDecoration: 'none' }}>
            Manage templates
          </Link>
        </div>
        {templates.length === 0 ? (
          <div className="aegis-muted">
            No protocol templates yet. Templates define the acquisition parameters expected for a
            modality/body-part combination — manage them in the admin Protocol Templates tab.
          </div>
        ) : (
          <div className="aegis-table-wrap">
            <table className="aegis-table">
              <thead><tr><th>Name</th><th>Modality</th><th>Body part</th></tr></thead>
              <tbody>
                {templates.map(t => (
                  <tr key={t.id}>
                    <td><code className="aegis-code">{t.name}</code></td>
                    <td>{t.modality || '—'}</td>
                    <td>{t.body_part || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      {/* Uploaders — outside contributors with upload-only access */}
      <section className="aegis-section">
        <div className="aegis-section-bar">
          <h2>
            Uploaders
            {!uploadersForbidden && !uploadersLoading && !uploadersError
              ? ` (${uploaders.length + pendingInvites.length})`
              : ''}
          </h2>
        </div>
        {uploadersForbidden ? (
          <div className="aegis-muted">
            Only project owners and platform admins can manage uploaders.
          </div>
        ) : uploadersLoading ? (
          <div className="aegis-muted">Loading uploaders…</div>
        ) : (
          <>
            {uploadersError && <div className="aegis-error">{uploadersError}</div>}
            {revokeError && <div className="aegis-error">{revokeError}</div>}
            {!uploadersError && (
              uploaders.length === 0 && pendingInvites.length === 0 ? (
                <div className="aegis-muted">
                  No uploaders yet. Invite an outside contributor to let them upload studies to this
                  project.
                </div>
              ) : (
                <div className="aegis-table-wrap">
                  <table className="aegis-table">
                    <thead>
                      <tr>
                        <th>Uploader</th>
                        <th>Institution</th>
                        <th>Status</th>
                        <th>Invited by</th>
                        <th>Added</th>
                        <th>Expires</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {uploaders.map(u => (
                        <tr key={`member-${u.admin_user_id}`}>
                          <td>
                            {u.user_name
                              ? <>{u.user_name} <span className="aegis-muted">({u.user_email || 'no email'})</span></>
                              : (u.user_email || <span className="aegis-muted">—</span>)}
                          </td>
                          <td>{u.institution_name || <span className="aegis-muted">—</span>}</td>
                          <td><span className="aegis-pill">Active</span></td>
                          <td className="aegis-muted">—</td>
                          <td>{fmtDate(u.created_at)}</td>
                          <td className="aegis-muted">—</td>
                          <td>
                            <button
                              type="button"
                              className="aegis-btn-secondary aegis-btn-compact"
                              onClick={() => revokeUploader(u)}
                            >
                              Revoke
                            </button>
                          </td>
                        </tr>
                      ))}
                      {pendingInvites.map(inv => (
                        <tr key={`invite-${inv.id}`}>
                          <td>
                            {inv.name
                              ? <>{inv.name} <span className="aegis-muted">({inv.email})</span></>
                              : inv.email}
                          </td>
                          <td>{inv.institution_name || <span className="aegis-muted">—</span>}</td>
                          <td><span className="aegis-pill-outline">Pending</span></td>
                          <td>{inv.invited_by || <span className="aegis-muted">—</span>}</td>
                          <td>{fmtDate(inv.created_at)}</td>
                          <td>{fmtDate(inv.expires_at)}</td>
                          <td></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )
            )}

            {/* Invite form */}
            <div className="aegis-subform">
              <h3>Invite an uploader</h3>
              {smtpUnavailable && (
                <div className="aegis-warning-banner aegis-note-block">
                  Email is not configured on this server (SMTP).
                </div>
              )}
              {inviteError && <div className="aegis-error">{inviteError}</div>}
              {inviteSentTo && (
                <div className="aegis-success-note aegis-note-block">
                  Invitation sent to {inviteSentTo}
                </div>
              )}
              {redeemUrl && (
                <div className="aegis-note-block">
                  <div className="aegis-warning-banner">
                    Email couldn&apos;t be sent — copy this link and send it manually
                  </div>
                  <div className="aegis-copy-row">
                    <div className="aegis-form-row">
                      <input
                        readOnly
                        aria-label="Invite redeem link"
                        value={redeemUrl}
                        onFocus={e => e.currentTarget.select()}
                      />
                    </div>
                    <button type="button" className="aegis-btn-secondary" onClick={copyRedeemUrl}>
                      {copied ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                </div>
              )}
              <div className="aegis-form-row">
                <label htmlFor="uploader-email">Email</label>
                <input
                  id="uploader-email"
                  type="email"
                  value={inviteEmail}
                  onChange={e => setInviteEmail(e.target.value)}
                  placeholder="colleague@university.edu"
                />
              </div>
              <div className="aegis-form-row">
                <label htmlFor="uploader-name">Name (optional)</label>
                <input
                  id="uploader-name"
                  type="text"
                  value={inviteName}
                  onChange={e => setInviteName(e.target.value)}
                  placeholder="Dr. Jane Contributor"
                />
              </div>
              <div className="aegis-form-row">
                <label htmlFor="uploader-expiry">Invitation expires after</label>
                <select
                  id="uploader-expiry"
                  value={inviteExpiryDays}
                  onChange={e => setInviteExpiryDays(parseInt(e.target.value, 10))}
                >
                  <option value={3}>3 days</option>
                  <option value={7}>7 days</option>
                  <option value={14}>14 days</option>
                  <option value={30}>30 days</option>
                </select>
                <span className="aegis-form-hint">
                  The invite link stops working after this window — you can always send a new one.
                </span>
              </div>
              <div className="aegis-form-actions">
                <button
                  type="button"
                  className="aegis-btn-primary"
                  onClick={inviteUploader}
                  disabled={inviting || !inviteEmail.trim()}
                >
                  {inviting ? 'Sending…' : 'Send invitation'}
                </button>
              </div>
            </div>

            {/* Desktop uploader — project-scoped installer invite. Pairing the
                installed app provisions upload access to this project. */}
            <div className="aegis-subform">
              <h3>Desktop uploader</h3>
              {installersLoading ? (
                <div className="aegis-muted">Loading installer builds…</div>
              ) : pickUploaderInstallers(installers).length === 0 ? (
                <div className="aegis-muted">
                  No desktop installer builds are registered yet — upload one under Admin → Downloads.
                </div>
              ) : (
                <>
                  <span className="aegis-form-hint">
                    Email a download link for the AEGIS Desktop Uploader. When the recipient installs
                    and pairs the app, it is automatically granted upload access to this project.
                  </span>
                  {desktopSmtpUnavailable && (
                    <div className="aegis-warning-banner aegis-note-block">
                      Email is not configured on this server (SMTP).
                    </div>
                  )}
                  {desktopError && <div className="aegis-error">{desktopError}</div>}
                  {desktopSentTo && (
                    <div className="aegis-success-note aegis-note-block">
                      Install link sent to {desktopSentTo} — pairing will grant upload access to this
                      project.
                    </div>
                  )}
                  <div className="aegis-form-row">
                    <label htmlFor="desktop-uploader-email">Email</label>
                    <input
                      id="desktop-uploader-email"
                      type="email"
                      value={desktopEmail}
                      onChange={e => setDesktopEmail(e.target.value)}
                      placeholder="colleague@university.edu"
                    />
                  </div>
                  <div className="aegis-form-row">
                    <label htmlFor="desktop-uploader-installer">Installer</label>
                    <select
                      id="desktop-uploader-installer"
                      value={desktopInstallerId}
                      onChange={e => setDesktopInstallerId(e.target.value)}
                    >
                      {pickUploaderInstallers(installers).map(inst => (
                        <option key={inst.id} value={inst.id}>
                          {(INSTALLER_PLATFORM_LABELS[inst.platform] ?? inst.platform)} — v{inst.version}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="aegis-form-actions">
                    <button
                      type="button"
                      className="aegis-btn-primary"
                      onClick={sendDesktopInvite}
                      disabled={desktopSending || !desktopEmail.trim() || !desktopInstallerId}
                    >
                      {desktopSending ? 'Sending…' : 'Send install link'}
                    </button>
                  </div>
                </>
              )}
            </div>
          </>
        )}
      </section>
    </>
  )
}
