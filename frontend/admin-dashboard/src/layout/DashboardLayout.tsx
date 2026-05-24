import { Outlet } from 'react-router-dom'
import { Breadcrumbs } from './Breadcrumbs'
import { TopBar } from './TopBar'
import '../styles/nav.css'

// DashboardLayout is the chrome around every routed page in the XNAT-style
// hierarchy: shared top bar + breadcrumb strip + the active page. AdminApp
// renders the same TopBar/Breadcrumbs under /admin/*, so the global header
// stays unified across both surfaces.
export function DashboardLayout() {
  return (
    <div className="aegis-shell">
      <TopBar />
      <Breadcrumbs />
      <main className="aegis-main">
        <Outlet />
      </main>
    </div>
  )
}
