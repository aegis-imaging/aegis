import { useLocation } from 'react-router-dom'
import { TopBarSearch } from './TopBarSearch'
import { useDarkMode } from '../hooks/useDarkMode'

// TopBar is the shared global header. Rendered on every page (Home,
// per-project, per-subject, per-study, every admin tab) so the brand,
// primary nav, search, and theme toggle look identical everywhere.
//
// Home/Admin use plain <a href=""> rather than react-router Link/NavLink:
// navigating between AdminApp and DashboardLayout crosses route subtree
// boundaries, and NavLink-driven navigation through that boundary silently
// no-ops because React Router doesn't unmount the previous element. A
// full-page nav sidesteps that — same approach Sign-in used in #508.
export function TopBar() {
  const location = useLocation()
  const isHome = location.pathname === '/'
  const inAdmin = location.pathname.startsWith('/admin')
  const inProfile = location.pathname.startsWith('/profile')
  const [darkMode, toggleDarkMode] = useDarkMode()

  return (
    <header className="aegis-topbar">
      <a href="/" className="aegis-brand" aria-label="AEGIS home">
        AEGIS
      </a>
      <nav className="aegis-topnav" aria-label="Primary">
        <a href="/" className={isHome ? 'aegis-active' : ''}>
          Home
        </a>
        <a href="/admin/studies" className={inAdmin ? 'aegis-active' : ''}>
          Admin
        </a>
        {/* Profile is reachable from any page (admin or researcher) so users
            can always find their own notification prefs + activity feed
            without going through the sidebar. Full-page nav so transitions
            between AdminApp and DashboardLayout subtrees work reliably. */}
        <a href="/profile" className={inProfile ? 'aegis-active' : ''}>
          Profile
        </a>
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
