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
// slide-in drawer with a hamburger toggle. On mobile, sidebar items are
// plain <a href> anchors (see ResearcherSidebar) so tapping one does a
// full page load — the drawer state resets on the next page boot and
// onItemClick never fires for them. onItemClick is wired to the DESKTOP
// NavLink branch (harmless no-op while the drawer is closed; closes it
// if the viewport was resized across the breakpoint with the drawer
// open) and to the backdrop overlay below.
//
// The requestAnimationFrame deferral matters for the overlay path on
// iOS Safari: closing synchronously starts the drawer's CSS transform
// while the synthesized click is still dispatching, and iOS cancels
// clicks whose target moves between touchstart and click. Deferring one
// frame lets the click complete before the animation starts.
export function DashboardLayout() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const closeMobileNav = () => {
    if (typeof requestAnimationFrame === 'function') {
      requestAnimationFrame(() => setMobileNavOpen(false))
    } else {
      setMobileNavOpen(false)
    }
  }

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
