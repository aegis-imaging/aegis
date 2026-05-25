import { useEffect, useState } from 'react'
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
//   GET  /api/projects/:id
//   PUT  /api/projects/:id
//   GET  /api/projects/:id/anon-profiles
//   GET  /api/projects/:id/protocol-templates

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

  async function save() {
    if (!project) return
    setSaving(true)
    setSaveError(null)
    try {
      const body = {
        ...project,
        description,
        retention_days: retentionDays === '' ? null : retentionDays,
        stuck_threshold_minutes: stuckMinutes === '' ? null : stuckMinutes,
        default_anon_profile_id: defaultProfileId || null,
      }
      const res = await fetch(`/api/projects/${project.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const t = await res.text()
        throw new Error(t || `HTTP ${res.status}`)
      }
      setProject(await res.json())
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
          <Link to="/admin/profiles" className="aegis-btn-secondary" style={{ textDecoration: 'none' }}>
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
          <Link to="/admin/protocol_templates" className="aegis-btn-secondary" style={{ textDecoration: 'none' }}>
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
    </>
  )
}
