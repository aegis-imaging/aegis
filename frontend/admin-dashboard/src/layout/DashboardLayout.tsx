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
// slide-in drawer with a hamburger toggle.
//
// Closing the drawer on sidebar tap requires deferring the close one
// animation frame past the synthesized click event. Two iOS Safari
// quirks pile up otherwise:
//
//   1. The useEffect-on-location approach (drawer closes after
//      navigate finishes reconciling) leaves the first tap working
//      and every subsequent one dead — focus rewind on overlay
//      unmount during the effect-after-paint window swallows the
//      next click target.
//   2. The synchronous-onItemClick approach starts the drawer's
//      CSS transform animation while the synthesized click is still
//      pending. iOS Safari cancels the click whenever its target
//      element MOVES between touchstart and click — the NavLink is
//      mid-translateX(-100%) so navigate() never fires. The drawer
//      closes (because setState already ran) but the URL doesn't
//      change.
//
// requestAnimationFrame defers the state update to after the click
// event has fully dispatched, so navigate() runs first and the
// element-moved cancellation can't trigger.
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
