import { Outlet } from 'react-router-dom'
import { Breadcrumbs } from './Breadcrumbs'
import { TopBar } from './TopBar'
import { ResearcherSidebar } from './ResearcherSidebar'
import '../styles/nav.css'

// DashboardLayout is the chrome around every routed page in the XNAT-style
// hierarchy: shared top bar + breadcrumb strip + sidebar + the active page.
// AdminApp renders its own (heavier) sidebar under /admin/*, but the
// top-level shell is the same so theme + brand + nav are consistent.
export function DashboardLayout() {
  return (
    <div className="aegis-shell">
      <TopBar />
      <Breadcrumbs />
      <div className="aegis-admin-body">
        <ResearcherSidebar />
        <main className="aegis-admin-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
