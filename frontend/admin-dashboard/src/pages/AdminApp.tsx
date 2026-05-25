import { useLocation } from 'react-router-dom'
import { App } from '../App'

// AdminApp wraps the existing 18-tab admin dashboard for use under the
// /admin/* route. The App component is the original monolithic dispatch:
// it owns ~50 useState hooks and renders the active tab via
// `{tab === 'X' && <Panel />}` conditionals.
//
// Routing quirk this works around — and why this wrapper is more than a
// pass-through:
//
//   The researcher tree (Home, /projects, /agent, /studies, /shares,
//   /profile/*) lives under DashboardLayout. Each researcher click
//   swaps the Outlet's child component, so React naturally tears the
//   old route down and mounts the new one. Re-render is unambiguous.
//
//   Inside /admin/*, every URL maps to the *same* route element
//   (AdminApp → App). Clicking from /admin/studies to /admin/audit
//   doesn't change the rendered React element — it just updates
//   location.pathname inside the still-mounted App. Several layers of
//   defensive code (#526 NavLink, #529 preventDefault+navigate,
//   #539 isActive from location, #540 key on the inner content section,
//   #548 drop preventDefault) tried to make the conditional renders
//   below pick up the new tab, and each fix worked for some renders but
//   not others — the symptom users repeatedly reported was "URL
//   changes but the page content stays on the previous tab until I
//   hit refresh."
//
//   The decisive fix: key the entire App on location.pathname. Every
//   in-admin click now causes React to fully unmount the previous
//   App instance and mount a fresh one with the new URL. No surviving
//   state, no stale closures, no missed re-renders. It's heavy-handed
//   (data fetches restart, scroll position resets), but the previous
//   tab's data was about to be re-fetched on the new tab anyway, and
//   the user explicitly asked for "copy the nav from home" — this
//   matches the route-swap semantics DashboardLayout's Outlet gives
//   the researcher tree for free.
export function AdminApp() {
  const location = useLocation()
  return <App key={location.pathname} />
}
