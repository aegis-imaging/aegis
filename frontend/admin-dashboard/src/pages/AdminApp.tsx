import { App } from '../App'

// AdminApp wraps the existing 18-tab admin dashboard for use under the
// /admin/* route. The App component is unchanged — preserving its
// tab/state behavior — so all current operator workflows continue to work
// while the XNAT-style hierarchy at /, /projects/*, etc. is built up
// alongside.
//
// Deep-linking specific admin tabs from URLs (e.g. /admin/routing → routing
// tab) is intentionally a follow-up; the current sidebar navigation inside
// App handles tab switching.
export function AdminApp() {
  return <App />
}
