import { useEffect, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { apiGetMe, type CurrentUser } from '../api/subjects'

// ResearcherSidebar renders on every routed page (Home, /projects/*,
// /studies, /shares, /agent, /tcia, /profile/*, AND /admin/*).
//
// Two modes per click:
//   - Same-subtree (researcher → researcher, admin → admin):
//     NavLink/SPA. Fast, preserves auth state.
//   - Cross-subtree (admin → researcher, or researcher → admin):
//     plain <a href>, full-page reload. Same pattern TopBar uses.
//     SPA nav across the /admin/* ↔ rest-of-app boundary silently
//     no-ops (URL doesn't update, page stays put). PR #553 fixed
//     this for in-admin clicks, but the cross-boundary case still
//     fails — full-page nav is the reliable escape hatch without
//     chasing more state in App.tsx's 12k lines.
//
// Items are visible to everyone authed; the admin-only groups
// (Operations / Configure / Admin) only render when apiGetMe()
// returns role === 'admin'.
export function ResearcherSidebar() {
  const [me, setMe] = useState<CurrentUser | null>(null)
  const location = useLocation()

  useEffect(() => {
    apiGetMe().then(setMe).catch(() => setMe(null))
  }, [])

  const isAdmin = me?.role === 'admin'

  // True when navigating to `to` would cross the /admin/* ↔ rest-of-app
  // route-subtree boundary. SPA nav across that boundary is unreliable;
  // a full-page reload is.
  const crossesAdminBoundary = (to: string): boolean => {
    const here = location.pathname.startsWith('/admin')
    const there = to.startsWith('/admin')
    return here !== there
  }

  const item = (label: string, to: string, icon: string) => {
    if (crossesAdminBoundary(to)) {
      const isActive = location.pathname === to || location.pathname.startsWith(to + '/')
      return (
        <a
          href={to}
          className={`aegis-sidenav-item${isActive ? ' aegis-sidenav-item--active' : ''}`}
        >
          <span className="aegis-sidenav-icon">{icon}</span>
          <span className="aegis-sidenav-label">{label}</span>
        </a>
      )
    }
    return (
      <NavLink
        to={to}
        className={({ isActive }) =>
          `aegis-sidenav-item${isActive ? ' aegis-sidenav-item--active' : ''}`
        }
      >
        <span className="aegis-sidenav-icon">{icon}</span>
        <span className="aegis-sidenav-label">{label}</span>
      </NavLink>
    )
  }

  return (
    <aside className="aegis-sidenav">
      <div className="aegis-sidenav-group">
        <div className="aegis-sidenav-group-label">Workspace</div>
        {item('Home',         '/',        '\u{1F3E0}')}
        {item('Studies',      '/studies', '\u{1F4CB}')}
        {item('Shares',       '/shares',  '\u{1F517}')}
        {item('Agent',        '/agent',   '\u{1F916}')}
        {item('TCIA import',  '/tcia',    '\u{1F4E5}')}
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
            {item('Audit log', '/admin/audit',     '\u{1F4DC}')}
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
