import { Link, useLocation, useParams } from 'react-router-dom'

// Breadcrumbs renders the Project > Subject > Study chain when the active
// route has the corresponding params. Each segment links upward so users can
// pop back without using the browser back button.
//
// XNAT itself does not consistently expose a breadcrumb strip — this is one
// place we explicitly diverge to improve on the source pattern.
export function Breadcrumbs() {
  const params = useParams()
  const location = useLocation()

  const crumbs: Array<{ label: string; to?: string }> = []

  if (location.pathname === '/') {
    return null
  }

  crumbs.push({ label: 'Home', to: '/' })

  if (params.projectId) {
    crumbs.push({
      label: shortenId(params.projectId, 'Project'),
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

function shortenId(id: string, prefix: string): string {
  if (id.length <= 12) return `${prefix} ${id}`
  return `${prefix} ${id.slice(0, 8)}…`
}
