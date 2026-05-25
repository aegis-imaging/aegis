import { useLocation } from 'react-router-dom'
import { App } from '../App'

// AdminApp renders the 18-tab admin experience inside DashboardLayout's
// Outlet. App is the monolithic dispatcher: it owns ~50 useState hooks
// and renders the active tab via `{tab === 'X' && <Panel />}` conditionals
// driven by parseAdminTab(location.pathname).
//
// Why the key={location.pathname}:
//
//   Every URL under /admin/* maps to this same route element, so React
//   Router would normally keep App mounted across admin tab clicks. The
//   conditional renders ARE supposed to re-evaluate when location.pathname
//   changes, but a recurring class of bugs (PRs #515 #526 #529 #539 #540
//   #548 #549) showed they don't always — the URL would update, the
//   breadcrumb would update, but the inner panel kept showing the
//   previous tab's content until a manual page refresh. The proximate
//   causes shifted (NavLink stale closures, batched state, useEffect
//   ordering) but the underlying fragility came from cramming 18 tabs
//   into one long-lived component.
//
//   Keying App on location.pathname forces a full unmount + remount on
//   every URL change inside /admin/*. Trade-off: data fetches restart
//   and scroll position resets per tab click. We accept that because
//   (a) most tab switches were going to re-fetch their data anyway,
//   (b) the DashboardLayout sidebar (ResearcherSidebar) stays mounted
//   so the nav itself doesn't flicker, and
//   (c) it gives the same clean route-swap semantics the researcher
//   tree (Home, /studies, /shares, /projects/*) gets for free via the
//   Outlet swapping its child component on each route match.
//
// The admin sidebar used to live inside App; with the move under
// DashboardLayout, ResearcherSidebar covers the Operations / Configure /
// Admin groups when the user is admin. App now only renders the tab
// content area (no topbar, no breadcrumbs, no sidebar of its own).
export function AdminApp() {
  const location = useLocation()
  return <App key={location.pathname} />
}
