import { App } from '../App'

// AdminApp renders the 18-tab admin experience inside DashboardLayout's
// Outlet. App is the monolithic dispatcher: it owns ~50 useState hooks
// and renders the active tab via `{tab === 'X' && <Panel />}` conditionals
// driven by parseAdminTab(useLocation().pathname) inside App itself.
//
// Earlier versions wrapped this in `<App key={location.pathname} />` to
// force a full unmount + remount on every admin URL change, working
// around a class of stale-render bugs (#515 #526 #529 #539 #540 #548
// #549). That remount turned out to BE the bug for admin sidebar
// clicks: it reset App's `currentUser` to null, which left every
// `{tab === 'X' && isAdmin && <Panel />}` falsy until the async
// /api/auth/me refetch finished — exactly the "URL changes but page
// stays blank/stale until I refresh" symptom users reported. The
// original stale-render issue (sidebar in App.tsx not re-rendering
// on click) was already fixed by moving the sidebar into
// DashboardLayout, so there's no reason to keep the remount.
export function AdminApp() {
  return <App />
}
