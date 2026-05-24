import { useLocation } from 'react-router-dom'
import { TopBarSearch } from './TopBarSearch'

// TopBar is the shared global header. Rendered by DashboardLayout (the
// XNAT-style researcher routes) and by AdminApp (under /admin/*), so the
// brand + Home/Admin nav + global search appear the same everywhere.
//
// Implementation note: Home/Admin use plain <a href=""> rather than
// react-router Link/NavLink. From /admin/*, NavLink-driven navigation to
// "/" silently no-ops because AdminApp lives outside the DashboardLayout
// route subtree — the router doesn't unmount AdminApp and the location
// transition gets swallowed. A full-page navigation sidesteps that
// entirely; this matches the approach the Sign-in button used in #508.
export function TopBar() {
  const location = useLocation()
  const isHome = location.pathname === '/'
  const inAdmin = location.pathname.startsWith('/admin')

  return (
    <header className="xn-topbar">
      <a href="/" className="xn-brand" aria-label="AEGIS home">
        AEGIS
      </a>
      <nav className="xn-topnav" aria-label="Primary">
        <a href="/" className={isHome ? 'xn-active' : ''}>
          Home
        </a>
        <a href="/admin/studies" className={inAdmin ? 'xn-active' : ''}>
          Admin
        </a>
      </nav>
      <div className="xn-search-slot">
        <TopBarSearch />
      </div>
    </header>
  )
}
