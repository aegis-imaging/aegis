import { Link, NavLink, useLocation } from 'react-router-dom'
import { TopBarSearch } from './TopBarSearch'

// TopBar is the shared global header. Rendered by DashboardLayout (the
// XNAT-style researcher routes) and by AdminApp (under /admin/*), so the
// brand + Home/Admin nav + global search appear the same everywhere.
export function TopBar() {
  const location = useLocation()
  const inAdmin = location.pathname.startsWith('/admin')

  return (
    <header className="xn-topbar">
      <Link to="/" className="xn-brand" aria-label="AEGIS home">
        AEGIS
      </Link>
      <nav className="xn-topnav" aria-label="Primary">
        <NavLink to="/" end className={({ isActive }) => (isActive ? 'xn-active' : '')}>
          Home
        </NavLink>
        <NavLink to="/admin/studies" className={() => (inAdmin ? 'xn-active' : '')}>
          Admin
        </NavLink>
      </nav>
      <div className="xn-search-slot">
        <TopBarSearch />
      </div>
    </header>
  )
}
