import { useEffect, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { apiGetMe, type CurrentUser } from '../api/subjects'
import { useIsMobile } from '../hooks/useIsMobile'

// ResearcherSidebar renders on every routed page.
//
// Two render modes per item, decided by the ACTUAL viewport (useIsMobile,
// matchMedia against nav.css's 768px breakpoint):
//
//   Desktop (>768px): NavLink/SPA nav. Fast, no full reload.
//
//   Mobile (≤768px): plain <a href>. Three iterations of
//   SPA-nav-in-mobile-drawer (#571 / #572 / #573) each broke differently
//   on iOS Safari:
//     - useEffect close on location change → first tap works, rest die.
//     - Synchronous onClick close → tap navigates but iOS cancels the
//       synthesized click because the drawer started animating during
//       dispatch (target moved → click discarded).
//     - requestAnimationFrame-deferred close → same "first tap works,
//       rest die" pattern as the useEffect approach.
//   Plain anchors sidestep the entire React-event surface — iOS Safari
//   does a full page load and the drawer state resets on next page boot.
//   A few hundred ms slower than SPA but reliable. Do NOT re-litigate
//   this on mobile.
//
//   NOTE: the mode decision was previously `!!onItemClick`, but
//   DashboardLayout passes onItemClick unconditionally, which silently
//   forced anchor-mode (full page reloads) on DESKTOP too. The viewport
//   query is the correct signal; onItemClick is only the drawer-close
//   callback.
//
// Items are grouped by audience for visual organisation:
//   - Workspace / You: visible to everyone authed.
//   - Operations / Configure / Admin: only when apiGetMe() returns
//     role === 'admin'.
export function ResearcherSidebar({ onItemClick }: { onItemClick?: () => void } = {}) {
  const [me, setMe] = useState<CurrentUser | null>(null)
  const location = useLocation()
  const isMobile = useIsMobile()

  useEffect(() => {
    apiGetMe().then(setMe).catch(() => setMe(null))
  }, [])

  const isAdmin = me?.role === 'admin'

  const item = (label: string, to: string, icon: string) => {
    if (isMobile) {
      // Active-class logic mirrors NavLink's, since the plain anchor
      // loses NavLink's built-in isActive prop.
      const isActive =
        to === '/'
          ? location.pathname === '/'
          : location.pathname === to || location.pathname.startsWith(to + '/')
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
        end={to === '/'}
        className={({ isActive }) =>
          `aegis-sidenav-item${isActive ? ' aegis-sidenav-item--active' : ''}`
        }
        onClick={onItemClick}
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
        {item('Home',         '/',            '\u{1F3E0}')}
        {item('Studies',      '/studies',     '\u{1F4CB}')}
        {item('Shares',       '/shares',      '\u{1F517}')}
        {item('Agent',        '/agent',       '\u{1F916}')}
        {item('TCIA import',  '/tcia_import', '\u{1F4E5}')}
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
            {item('Audit log', '/audit',     '\u{1F4DC}')}
            {item('Routing',   '/routing',   '\u{1F6E4}')}
            {item('DIMSE Ops', '/dimse_ops', '\u{1F4E1}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Configure</div>
            {item('Projects',           '/projects',           '\u{1F4C1}')}
            {item('Institutions',       '/institutions',       '\u{1F3E5}')}
            {item('Anon profiles',      '/profiles',           '\u{1F6E1}')}
            {item('Protocol templates', '/protocol_templates', '\u{1F4CF}')}
          </div>

          <div className="aegis-sidenav-group">
            <div className="aegis-sidenav-group-label">Admin</div>
            {item('Users',         '/users',         '\u{1F464}')}
            {item('API keys',      '/api_keys',      '\u{1F511}')}
            {item('Invite codes',  '/invite_codes',  '\u{1F3AB}')}
            {item('Federation',    '/federation',    '\u{1F310}')}
            {item('Notifications', '/notifications', '\u{1F514}')}
            {item('Downloads',     '/downloads',     '\u{1F4E5}')}
            {item('System',        '/system',        '\u{2699}')}
          </div>
        </>
      )}
    </aside>
  )
}
