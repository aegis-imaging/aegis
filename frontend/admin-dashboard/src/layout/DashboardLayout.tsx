import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Breadcrumbs } from './Breadcrumbs'
import { TopBar } from './TopBar'
import { ResearcherSidebar } from './ResearcherSidebar'
import '../styles/nav.css'

// DashboardLayout is the chrome around every routed page in the XNAT-style
// hierarchy: shared top bar + breadcrumb strip + sidebar + the active page.
//
// Mobile drawer: on narrow viewports (<= 768px) the sidebar becomes a slide-in
// drawer because there isn't horizontal room for a permanent 240px rail. A
// hamburger button in this layout (NOT inside TopBar so it can be sized and
// positioned without competing with the topbar's search slot) toggles it. The
// drawer auto-closes on route change so a sidebar tap doesn't leave the
// drawer open over the destination page.
//
// Prior to this layout the sidebar was always `position: fixed` covering the
// whole mobile viewport, which made the rest of the app effectively
// unreachable on iPhone Safari — taps appeared to "do nothing" because the
// page underneath was never visible.
export function DashboardLayout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const location = useLocation()

  // Auto-close on route change so the destination page is visible.
  useEffect(() => {
    setMobileNavOpen(false)
  }, [location.pathname])

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
        <ResearcherSidebar />
        {mobileNavOpen && (
          <div
            className="aegis-sidenav-overlay"
            onClick={() => setMobileNavOpen(false)}
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
