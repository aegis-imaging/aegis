import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { apiListProjects, apiGetMe, type Project, type CurrentUser } from '../api/subjects'

// HomePage shows the projects the caller has access to, plus a short
// "recent" placeholder. Modeled on XNAT's post-login home (projects list +
// recent activity panel), built fresh against AEGIS components.
export function HomePage() {
  const [projects, setProjects] = useState<Project[] | null>(null)
  const [me, setMe] = useState<CurrentUser | null>(null)
  const [error, setError] = useState<string | null>(null)

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
        <h2>Projects</h2>
        {!projects && !error && <div className="xn-muted">Loading…</div>}
        {projects && projects.length === 0 && (
          <div className="xn-muted">
            You don't have access to any projects yet. Ask a project owner to invite you.
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
