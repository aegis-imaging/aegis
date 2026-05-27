import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import { Breadcrumbs } from './Breadcrumbs'
import { TopBar } from './TopBar'
import { ResearcherSidebar } from './ResearcherSidebar'
import '../styles/nav.css'

// DashboardLayout is the chrome around every routed page in the XNAT-style
// hierarchy: shared top bar + breadcrumb strip + sidebar + the active page.
//
// Mobile drawer: on narrow viewports (<= 768px) the sidebar becomes a
// slide-in drawer with a hamburger toggle. Closing the drawer is wired
// SYNCHRONOUSLY on each sidebar tap via onItemClick — earlier versions
// used a useEffect on location.pathname to auto-close, but iOS Safari
// dropped every NavLink tap after the first one. The exact race
// wasn't easy to nail (stale focus + the auto-close re-render in the
// same tick as the navigate() call seems to be involved), but
// synchronous close on tap removes the race entirely.
export function DashboardLayout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const closeMobileNav = () => setMobileNavOpen(false)

  return (
    <div className="aegis-shell">
      <TopBar />
      <Breadcrumbs />
      <div className={`aegis-admin-body${mobileNavOpen ? ' aegis-admin-body--mobile-nav-open' : ''}`}>
        <button
          type="button"
          className="aegis-mobile-nav-toggle"
          onClick={() => setMobileNavOpen(o => !o)}
          aria-label={mobileNavOpen ? 'Close navigation' : 'Open navigation'}
          aria-expanded={mobileNavOpen}
        >
          {mobileNavOpen ? '✕' : '☰'}
          <span className="aegis-mobile-nav-toggle-label">
            {mobileNavOpen ? 'Close' : 'Menu'}
          </span>
        </button>
        <ResearcherSidebar onItemClick={closeMobileNav} />
        {mobileNavOpen && (
          <div
            className="aegis-sidenav-overlay"
            onClick={closeMobileNav}
            aria-hidden="true"
          />
        )}
        <main className="aegis-admin-content">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
