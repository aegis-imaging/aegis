import { Link, useLocation } from 'react-router-dom'
import { TopBarSearch } from './TopBarSearch'
import { useDarkMode } from '../hooks/useDarkMode'

// TopBar is the shared global header rendered on every page. All admin tabs
// now live at root URLs (no /admin/ prefix), so the cross-subtree NavLink
// hazard that used to force <a href> here is gone — every link is a clean
// SPA navigation via <Link>.
//
// "Admin" routes to /audit since that's the most operationally distinct
// admin landing page; admins who want the unified Studies view get it from
// either the topbar Home or the sidebar's Workspace > Studies entry.
export function TopBar() {
  const location = useLocation()
  const isHome = location.pathname === '/'
  const inProfile = location.pathname.startsWith('/profile')
  // "Admin"-ish landing pages still want the Admin nav highlight even though
  // the URLs no longer carry an /admin prefix. Match the operator-only tabs.
  const adminLandingPaths = ['/audit','/routing','/dimse_ops','/users','/api_keys',
    '/invite_codes','/notifications','/downloads','/system','/profiles',
    '/protocol_templates','/institutions','/federation','/satellites']
  const inAdmin = adminLandingPaths.some(p => location.pathname === p || location.pathname.startsWith(p + '/'))
  const [darkMode, toggleDarkMode] = useDarkMode()

  return (
    <header className="aegis-topbar">
      <Link to="/" className="aegis-brand" aria-label="AEGIS home">
        AEGIS
      </Link>
      <nav className="aegis-topnav" aria-label="Primary">
        <Link to="/" className={isHome ? 'aegis-active' : ''}>
          Home
        </Link>
        <Link to="/audit" className={inAdmin ? 'aegis-active' : ''}>
          Admin
        </Link>
        <Link to="/profile" className={inProfile ? 'aegis-active' : ''}>
          Profile
        </Link>
      </nav>
      <div className="aegis-search-slot">
        <TopBarSearch />
      </div>
      <button
        type="button"
        className="aegis-icon-btn"
        onClick={toggleDarkMode}
        title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
        aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
      >
        {darkMode ? '☀' : '☾'}
      </button>
    </header>
  )
}
