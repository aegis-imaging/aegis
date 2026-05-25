import { useEffect, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { apiGetMe, type CurrentUser } from '../api/subjects'

// ResearcherSidebar renders on every routed page (Home, /projects/*,
// /studies, /shares, /agent, /profile/*, AND /admin/* after PR #550).
// Two navigation modes:
//
//   item(...)      — NavLink + SPA navigation. Works reliably for the
//                    researcher tree (/, /studies, /shares, /agent, ...).
//                    Used inside Workspace / You groups.
//
//   adminItem(...) — plain <a href>. Forces a full-page navigation so
//                    each click is functionally identical to "click, then
//                    refresh the browser" — which is what users had to do
//                    manually before this PR for every admin sidebar
//                    click. The /admin/* routes have a recurring bug
//                    (#515 #526 #529 #539 #540 #548 #549 #550) where SPA
//                    navigation updates the URL but the React tree fails
//                    to re-render until refresh; the brute-force fix is
//                    to skip React Router entirely for those links.
//                    Used inside Operations / Configure / Admin groups.
//                    Slower than SPA nav by a few hundred ms, but
//                    reliable.
//
// Items are visible to everyone authed; the admin-only groups
// (Operations / Configure / Admin) only render when apiGetMe()
// returns role === 'admin'.
//
// Dedupe note: Studies / Shares / Agent are intentionally NOT repeated
// in Operations — /studies, /shares, /agent in Workspace already point
// at the same destinations (with /admin/studies, /admin/shares,
// /admin/agent redirecting to those in main.tsx).
export function ResearcherSidebar() {
  const [me, setMe] = useState<CurrentUser | null>(null)
  const location = useLocation()

  useEffect(() => {
    apiGetMe().then(setMe).catch(() => setMe(null))
  }, [])

  const isAdmin = me?.role === 'admin'

  // SPA nav — works inside the researcher tree, used for Workspace / You.
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

  // Full-page nav — used for /admin/* links to dodge the SPA-stale-render
  // bug. The active-class logic is hand-rolled since we lose NavLink's
  // built-in isActive.
  const adminItem = (label: string, to: string, icon: string) => {
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
    <aside className="aegis-sidenav">
      {/* Workspace group: researcher-facing routes only. Studies + Shares
          point at top-level /studies and /shares views (not /admin/*)
          so non-admin users get a simpler browse experience that doesn't
          bounce them into the admin triage panels. Admins still get the
          richer admin views via the Operations group below. */}
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
            {/* Studies + Shares intentionally omitted — they live in
                Workspace above. Adding /admin/studies and /admin/shares
                here just duplicated the Workspace entries and pointed at
                the broken admin routes. */}
            {adminItem('Audit log', '/admin/audit',     '\u{1F4DC}')}
            {adminItem('Routing',   '/admin/routing',   '\u{1F6E4}')}
            {adminItem('DIMSE Ops', '/admin/dimse_ops', '\u{1F4E1}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Configure</div>
            {adminItem('Projects',     '/admin/projects',     '\u{1F4C1}')}
            {adminItem('Institutions', '/admin/institutions', '\u{1F3E5}')}
            {/* Satellites managed per-institution at /admin/institutions/:id —
                no top-level entry here. /admin/satellites still resolves
                for direct/bookmark navigation. */}
            {adminItem('Anon profiles',       '/admin/profiles',           '\u{1F6E1}')}
            {adminItem('Protocol templates',  '/admin/protocol_templates', '\u{1F4CF}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Admin</div>
            {adminItem('Users',         '/admin/users',         '\u{1F464}')}
            {adminItem('API keys',      '/admin/api_keys',      '\u{1F511}')}
            {adminItem('Invite codes',  '/admin/invite_codes',  '\u{1F3AB}')}
            {adminItem('Notifications', '/admin/notifications', '\u{1F514}')}
            {adminItem('System',        '/admin/system',        '\u{2699}')}
          </div>
        </>
      )}
    </aside>
  )
}
