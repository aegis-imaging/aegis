import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { Breadcrumbs } from './Breadcrumbs'
import { TopBarSearch } from './TopBarSearch'
import '../styles/nav.css'

// DashboardLayout is the chrome around every routed page in the XNAT-style
// hierarchy: a top bar with the global search and an "Admin" escape hatch,
// and a breadcrumb strip below it. The <Outlet/> renders the active page.
export function DashboardLayout() {
  const location = useLocation()
  const inAdmin = location.pathname.startsWith('/admin')

  return (
    <div className="xn-shell">
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
      <Breadcrumbs />
      <main className="xn-main">
        <Outlet />
      </main>
    </div>
  )
}
