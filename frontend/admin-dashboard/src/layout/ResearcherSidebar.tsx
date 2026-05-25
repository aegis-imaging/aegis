import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
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
      // No onClick override here — NavLink's native click intercept
      // calls navigate() internally, and if React Router fails to
      // intercept the browser's <a href> fallback does a full-page
      // navigation. The previous defensive preventDefault + navigate()
      // pattern made the click a single point of failure: when
      // navigate() silently no-op'd (the symptom users hit on
      // /admin/* routes — click updates URL but content stayed stale
      // until a manual page refresh), preventDefault had already
      // killed the fallback so the click looked completely dead.
    >
      <span className="aegis-sidenav-icon">{icon}</span>
      <span className="aegis-sidenav-label">{label}</span>
    </NavLink>
  )

  return (
    <aside className="aegis-sidenav">
      {/* Workspace group: researcher-facing routes only. Studies + Shares
          point at top-level /studies and /shares views (not /admin/*)
          so non-admin users get a simpler browse experience that doesn't
          bounce them into the admin triage panels. Admins still get the
          richer admin views via the Operations group below. */}
      <div className="aegis-sidenav-group">
        <div className="aegis-sidenav-group-label">Workspace</div>
        {item('Home',    '/',        '\u{1F3E0}')}
        {item('Studies', '/studies', '\u{1F4CB}')}
        {item('Shares',  '/shares',  '\u{1F517}')}
        {item('Agent',   '/agent',   '\u{1F916}')}
      </div>

      <div className="aegis-sidenav-group">
        <div className="aegis-sidenav-group-label">You</div>
        {item('Your activity', '/profile/activity',      '\u{1F4DC}')}
        {item('Notifications', '/profile/notifications', '\u{1F514}')}
      </div>

      {isAdmin && (
        <>
          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Operations</div>
            {item('Studies',   '/admin/studies',   '\u{1F4CB}')}
            {item('Audit log', '/admin/audit',     '\u{1F4DC}')}
            {item('Shares',    '/admin/shares',    '\u{1F517}')}
            {item('Routing',   '/admin/routing',   '\u{1F6E4}')}
            {item('DIMSE Ops', '/admin/dimse_ops', '\u{1F4E1}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Configure</div>
            {item('Projects',     '/admin/projects',     '\u{1F4C1}')}
            {item('Institutions', '/admin/institutions', '\u{1F3E5}')}
            {/* Satellites managed per-institution at /admin/institutions/:id —
                no top-level entry here. /admin/satellites still resolves
                for direct/bookmark navigation. */}
            {item('Anon profiles',       '/admin/profiles',           '\u{1F6E1}')}
            {item('Protocol templates',  '/admin/protocol_templates', '\u{1F4CF}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Admin</div>
            {item('Users',         '/admin/users',         '\u{1F464}')}
            {item('API keys',      '/admin/api_keys',      '\u{1F511}')}
            {item('Invite codes',  '/admin/invite_codes',  '\u{1F3AB}')}
            {item('Notifications', '/admin/notifications', '\u{1F514}')}
            {item('System',        '/admin/system',        '\u{2699}')}
          </div>
        </>
      )}
    </aside>
  )
}
