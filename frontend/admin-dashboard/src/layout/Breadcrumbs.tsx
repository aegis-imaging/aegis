import { useEffect, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { apiGetProject } from '../api/subjects'

// Breadcrumbs renders a path-aware nav trail:
//   /                                      → null (no breadcrumbs on Home)
//   /projects/:id                          → Home > Project name
//   /projects/:p/subjects/:s               → Home > Project name > Subject
//   /projects/:p/subjects/:s/studies/:st   → Home > Project > Subject > Study
//   /profile                               → Home > Profile
//   /profile/notifications                 → Home > Profile > Notifications
//   /profile/activity                      → Home > Profile > Activity
//   /<tab>                                 → Home > Tab name        (root-level admin tabs)
//   /institutions/:id                      → Home > Institutions > Institution name
//
// Each segment links upward so users can pop back without using the browser
// back button. The deepest segment is rendered as plain text with
// aria-current="page" for screen readers.
//
// All admin tabs now live at root URLs (no /admin/ prefix) — the
// "Home > Admin > X" trail collapsed to "Home > X" since the audience
// distinction (admin vs researcher) is enforced by visibility in the
// sidebar, not by URL nesting.
export function Breadcrumbs() {
  const params = useParams()
  const location = useLocation()

  // Fetch project name when a projectId param is present.
  const projectName = useProjectName(params.projectId)

  // Fetch institution name when on /institutions/:id (or legacy /admin/institutions/:id).
  const adminInstitutionId = useInstitutionIdFromPath(location.pathname)
  const institutionName = useInstitutionName(adminInstitutionId)

  const crumbs: Array<{ label: string; to?: string }> = []

  if (location.pathname === '/') {
    return null
  }

  crumbs.push({ label: 'Home', to: '/' })

  // Researcher-tree crumbs.
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
  // /projects/:id/settings — sub-route under the project crumb.
  if (params.projectId && location.pathname.endsWith('/settings') && !params.subjectId) {
    crumbs.push({ label: 'Settings' })
  }

  // Profile / Agent top-level crumbs.
  if (location.pathname.startsWith('/profile')) {
    crumbs.push({ label: 'Profile', to: '/profile' })
    if (location.pathname === '/profile/notifications') {
      crumbs.push({ label: 'Notifications' })
    } else if (location.pathname === '/profile/activity') {
      crumbs.push({ label: 'Activity' })
    }
  }
  // Root-level admin tabs (/audit, /routing, /institutions, etc.). Only
  // fires when we haven't already pushed researcher-tree crumbs above —
  // /projects/:id renders ProjectPage and never reaches App / admin tabs.
  if (!params.projectId && !location.pathname.startsWith('/profile')) {
    const rootTabMatch = location.pathname.match(/^\/([^/]+)/)
    const tabSlug = rootTabMatch?.[1]
    if (tabSlug && ADMIN_TAB_LABELS[tabSlug]) {
      const tabPath = `/${tabSlug}`
      const isLeaf = location.pathname === tabPath
      crumbs.push(isLeaf ? { label: ADMIN_TAB_LABELS[tabSlug] } : { label: ADMIN_TAB_LABELS[tabSlug], to: tabPath })

      // Per-tab leaf crumbs. Today only institutions has a detail URL.
      if (tabSlug === 'institutions' && adminInstitutionId) {
        crumbs.push({
          label: institutionName ?? shortenId(adminInstitutionId, 'Institution'),
        })
      }
    }
  }

  return (
    <nav className="aegis-breadcrumbs" aria-label="Breadcrumb">
      {crumbs.map((c, i) => {
        const isLast = i === crumbs.length - 1
        return (
          <span key={i} className="aegis-crumb">
            {!isLast && c.to ? (
              <Link to={c.to}>{c.label}</Link>
            ) : (
              <span aria-current={isLast ? 'page' : undefined}>{c.label}</span>
            )}
            {!isLast && <span className="aegis-crumb-sep" aria-hidden>›</span>}
          </span>
        )
      })}
    </nav>
  )
}

// Maps the URL slug for each admin tab to the display label used in
// breadcrumbs. Mirrors TAB_META in App.tsx (kept here so the layout
// chrome doesn't pull the whole 12k-line App module into its dependency
// graph).
const ADMIN_TAB_LABELS: Record<string, string> = {
  studies: 'Studies',
  audit: 'Audit Log',
  agent: 'AI Agent',
  shares: 'Shares',
  routing: 'Routing',
  dimse_ops: 'DIMSE Ops',
  projects: 'Projects',
  institutions: 'Institutions',
  satellites: 'Satellites',
  profiles: 'Anon Profiles',
  protocol_templates: 'Protocol Templates',
  notifications: 'Notifications',
  federation: 'Federation',
  system: 'System',
  users: 'Users',
  api_keys: 'API Keys',
  invite_codes: 'Invite Codes',
  downloads: 'Downloads',
  tcia_import: 'TCIA Import',
}

// useProjectName fetches the project by ID and returns its display name.
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

// useInstitutionIdFromPath extracts the institution UUID from the pathname
// when on /institutions/:id (or legacy /admin/institutions/:id), returns
// null otherwise.
function useInstitutionIdFromPath(pathname: string): string | null {
  const m = pathname.match(/^\/(?:admin\/)?institutions\/([^/]+)/)
  return m?.[1] ?? null
}

// useInstitutionName fetches the named institution and returns its
// display name. Uses the existing /api/institutions list (no per-id
// endpoint) since institution lists are small.
function useInstitutionName(institutionId: string | null): string | null {
  const [name, setName] = useState<string | null>(null)
  useEffect(() => {
    if (!institutionId) {
      setName(null)
      return
    }
    let cancelled = false
    fetch('/api/institutions')
      .then((r) => r.ok ? r.json() : Promise.reject())
      .then((rows: Array<{ id: string; name: string }>) => {
        if (cancelled) return
        const match = rows.find((r) => r.id === institutionId)
        setName(match?.name ?? null)
      })
      .catch(() => !cancelled && setName(null))
    return () => {
      cancelled = true
    }
  }, [institutionId])
  return name
}

function shortenId(id: string, prefix: string): string {
  if (id.length <= 12) return `${prefix} ${id}`
  return `${prefix} ${id.slice(0, 8)}…`
}
