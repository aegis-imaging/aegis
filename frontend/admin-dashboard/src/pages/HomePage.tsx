import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  apiCreateProject,
  apiGetMe,
  apiListProjects,
  type CurrentUser,
  type Project,
} from '../api/subjects'

// HomePage shows the projects the caller has access to, with a primary
// "+ New project" action. Modeled on XNAT's post-login home (projects
// list + create-project affordance), built fresh against AEGIS
// components.
export function HomePage() {
  const [projects, setProjects] = useState<Project[] | null>(null)
  const [me, setMe] = useState<CurrentUser | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    let cancelled = false
    Promise.all([apiListProjects(), apiGetMe().catch(() => null)])
      .then(([ps, user]) => {
        if (cancelled) return
        setProjects(ps.filter((p) => !p.archived))
        setMe(user)
      })
      .catch((e) => !cancelled && setError(String(e)))
    return () => {
      cancelled = true
    }
  }, [])

  function handleCreated(p: Project) {
    setProjects((current) => (current ? [...current, p] : [p]))
    setCreating(false)
  }

  const isAdmin = me?.role === 'admin'

  return (
    <div className="xn-home">
      <header className="xn-home-greeting">
        <h1>Welcome{me?.name ? `, ${me.name}` : ''}</h1>
        <p className="xn-muted">
          Your projects appear below. Open a project to view its subjects and imaging sessions.
        </p>
      </header>

      {error && <div className="xn-error">Couldn't load projects: {error}</div>}

      <section className="xn-home-projects">
        <div className="xn-section-bar">
          <h2>Projects</h2>
          {isAdmin && !creating && (
            <button
              type="button"
              className="xn-btn-primary"
              onClick={() => setCreating(true)}
            >
              + New project
            </button>
          )}
        </div>

        {creating && (
          <NewProjectForm
            onCancel={() => setCreating(false)}
            onCreated={handleCreated}
          />
        )}

        {!projects && !error && <div className="xn-muted">Loading…</div>}
        {projects && projects.length === 0 && !creating && (
          <div className="xn-muted">
            {isAdmin
              ? 'No projects yet. Click "+ New project" above to create one.'
              : "You don't have access to any projects yet. Ask a project owner to invite you."}
          </div>
        )}
        {projects && projects.length > 0 && (
          <ul className="xn-card-grid">
            {projects.map((p) => (
              <li key={p.id} className="xn-card">
                <Link to={`/projects/${p.id}`} className="xn-card-link">
                  <div className="xn-card-title">{p.name}</div>
                  <div className="xn-card-meta">
                    <span className="xn-pill">{p.slug}</span>
                    {p.member_count != null && (
                      <span className="xn-muted">{p.member_count} member{p.member_count === 1 ? '' : 's'}</span>
                    )}
                  </div>
                  {p.description && <p className="xn-card-desc">{p.description}</p>}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

// NewProjectForm is a small inline form for creating a new project. Lives
// here rather than as a modal because the action is non-destructive and
// the form is short enough that a modal would be overkill.
function NewProjectForm({
  onCancel,
  onCreated,
}: {
  onCancel: () => void
  onCreated: (p: Project) => void
}) {
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [description, setDescription] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  // Auto-derive slug from name if the user hasn't typed a custom slug.
  const [slugTouched, setSlugTouched] = useState(false)
  const derivedSlug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  const effectiveSlug = slugTouched ? slug : derivedSlug

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!name.trim()) {
      setError('Name is required')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const p = await apiCreateProject({
        name: name.trim(),
        slug: effectiveSlug || undefined,
        description: description.trim(),
      })
      onCreated(p)
      // Drop the user straight into their new project so they can start
      // configuring it (subjects, routing, etc.).
      navigate(`/projects/${p.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form className="xn-new-project" onSubmit={handleSubmit}>
      <div className="xn-form-row">
        <label htmlFor="np-name">Name</label>
        <input
          id="np-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. ADNI Brain MRI"
          autoFocus
          required
        />
      </div>
      <div className="xn-form-row">
        <label htmlFor="np-slug">Slug</label>
        <input
          id="np-slug"
          type="text"
          value={effectiveSlug}
          onChange={(e) => {
            setSlugTouched(true)
            setSlug(e.target.value)
          }}
          placeholder={derivedSlug || 'auto-generated from name'}
          pattern="[a-z0-9\-]*"
          title="Lowercase letters, digits, and dashes only"
        />
        <span className="xn-form-hint">Used in URLs &amp; routing rules. Auto-generated from name if left blank.</span>
      </div>
      <div className="xn-form-row">
        <label htmlFor="np-description">Description</label>
        <textarea
          id="np-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          placeholder="Optional. What's this project for?"
        />
      </div>
      {error && <div className="xn-error">{error}</div>}
      <div className="xn-form-actions">
        <button type="submit" className="xn-btn-primary" disabled={submitting}>
          {submitting ? 'Creating…' : 'Create project'}
        </button>
        <button type="button" className="xn-btn-secondary" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
      </div>
    </form>
  )
}
