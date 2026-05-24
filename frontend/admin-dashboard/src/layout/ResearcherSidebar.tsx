import { useEffect, useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { apiGetMe, type CurrentUser } from '../api/subjects'

// ResearcherSidebar renders on every non-admin route (Home, /projects/*,
// /subjects/*, /studies/*). It's deliberately a thinner, link-only version
// of the big admin sidebar — admin-tab destinations are reachable from
// here for users who have permission, so a researcher who's also an admin
// doesn't need to bounce through /admin/* to reach Audit / Shares / Routing.
//
// Items are split into a "Researcher" group (visible to everyone authed)
// and an "Admin" group (only when apiGetMe() returns role === 'admin').
// Click handlers use plain NavLink — these navigate WITHIN the
// DashboardLayout subtree for the researcher items, and cross over into
// the AdminApp subtree for the admin items (so they still get a full
// in-app nav, not a full-page reload).
export function ResearcherSidebar() {
  const [me, setMe] = useState<CurrentUser | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    apiGetMe().then(setMe).catch(() => setMe(null))
  }, [])

  const isAdmin = me?.role === 'admin'

  const item = (label: string, to: string, icon: string) => (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `aegis-sidenav-item${isActive ? ' aegis-sidenav-item--active' : ''}`
      }
      onClick={(e) => {
        // Same defensive pattern as the admin sidebar — explicit navigate
        // ensures both the URL and the rendered route update reliably even
        // when crossing route subtrees.
        e.preventDefault()
        navigate(to)
      }}
    >
      <span className="aegis-sidenav-icon">{icon}</span>
      <span className="aegis-sidenav-label">{label}</span>
    </NavLink>
  )

  return (
    <aside className="aegis-sidenav">
      <div className="aegis-sidenav-group">
        <div className="aegis-sidenav-group-label">Workspace</div>
        {item('Home',     '/',              '\u{1F3E0}')}
        {item('Studies',  '/admin/studies', '\u{1F4CB}')}
        {item('Shares',   '/admin/shares',  '\u{1F517}')}
        {item('Activity', '/admin/audit',   '\u{1F4DC}')}
      </div>

      {isAdmin && (
        <>
          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Config</div>
            {item('Projects',     '/admin/projects',     '\u{1F4C1}')}
            {item('Institutions', '/admin/institutions', '\u{1F3E5}')}
            {item('Satellites',   '/admin/satellites',   '\u{1F4F6}')}
            {item('Routing',      '/admin/routing',      '\u{1F6E4}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Admin</div>
            {item('Users',         '/admin/users',         '\u{1F464}')}
            {item('Notifications', '/admin/notifications', '\u{1F514}')}
            {item('System',        '/admin/system',        '\u{2699}')}
          </div>
        </>
      )}
    </aside>
  )
}
