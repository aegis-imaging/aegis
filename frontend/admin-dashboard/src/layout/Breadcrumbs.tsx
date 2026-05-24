import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { apiGetProject } from '../api/subjects'

// Breadcrumbs renders the Project > Subject > Study chain when the
// active route has the corresponding params. Each segment links upward
// so users can pop back without using the browser back button.
//
// XNAT itself does not consistently expose a breadcrumb strip — this is
// one place we explicitly diverge to improve on the source pattern.
//
// The project crumb fetches the project's display name independently of
// ProjectPage so the breadcrumb shows the friendly name (e.g. "ADNI
// Brain MRI") rather than a raw UUID. Costs one extra GET on project /
// subject / study pages; cached by the browser when possible.
export function Breadcrumbs() {
  const params = useParams()
  const location = useLocation()

  // Pull the project name when the route has a projectId param. Falls
  // back to a truncated UUID if the fetch fails or hasn't completed yet,
  // so the breadcrumb is never blank.
  const projectName = useProjectName(params.projectId)

  const crumbs: Array<{ label: string; to?: string }> = []

  if (location.pathname === '/') {
    return null
  }

  crumbs.push({ label: 'Home', to: '/' })

  if (params.projectId) {
    crumbs.push({
      label: projectName ?? shortenId(params.projectId, 'Project'),
      to: `/projects/${params.projectId}`,
    })
  }

  if (params.subjectId) {
    crumbs.push({
      label: params.subjectId,
      to: `/projects/${params.projectId}/subjects/${params.subjectId}`,
    })
  }

  if (params.studyId) {
    crumbs.push({ label: shortenId(params.studyId, 'Study') })
  }

  if (location.pathname.startsWith('/admin')) {
    crumbs.push({ label: 'Admin' })
  }

  return (
    <nav className="xn-breadcrumbs" aria-label="Breadcrumb">
      {crumbs.map((c, i) => {
        const isLast = i === crumbs.length - 1
        return (
          <span key={i} className="xn-crumb">
            {!isLast && c.to ? (
              <Link to={c.to}>{c.label}</Link>
            ) : (
              <span aria-current={isLast ? 'page' : undefined}>{c.label}</span>
            )}
            {!isLast && <span className="xn-crumb-sep" aria-hidden>›</span>}
          </span>
        )
      })}
    </nav>
  )
}

// useProjectName fetches the project by ID and returns its display name.
// Returns null until the fetch resolves (so the caller can render a
// fallback label in the meantime).
function useProjectName(projectId: string | undefined): string | null {
  const [name, setName] = useState<string | null>(null)
  useEffect(() => {
    if (!projectId) {
      setName(null)
      return
    }
    let cancelled = false
    apiGetProject(projectId)
      .then((p) => !cancelled && setName(p.name))
      .catch(() => !cancelled && setName(null))
    return () => {
      cancelled = true
    }
  }, [projectId])
  return name
}

function shortenId(id: string, prefix: string): string {
  if (id.length <= 12) return `${prefix} ${id}`
  return `${prefix} ${id.slice(0, 8)}…`
}
