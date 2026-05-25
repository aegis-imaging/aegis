import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { apiGetMe, type CurrentUser } from '../api/subjects'

// ResearcherSidebar renders on every routed page (Home, /projects/*,
// /studies, /shares, /agent, /profile/*, AND /admin/* after PR #550).
//
// All items use NavLink/SPA navigation. The plain-anchor workaround
// shipped in PR #551 for /admin/* links has been reverted: the actual
// root cause of "click admin link → URL changes but page stays blank"
// was the `key={location.pathname}` in AdminApp that forced App to
// remount on every URL change and wiped currentUser. Removing that
// key (same PR as this revert) lets SPA nav re-render the conditional
// tab block in App.tsx without losing auth state.
//
// Items are visible to everyone authed; the admin-only groups
// (Operations / Configure / Admin) only render when apiGetMe()
// returns role === 'admin'.
//
// Dedupe note: Studies / Shares / Agent live in Workspace only — the
// admin-side duplicates were dropped in #551.
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
    >
      <span className="aegis-sidenav-icon">{icon}</span>
      <span className="aegis-sidenav-label">{label}</span>
    </NavLink>
  )

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
