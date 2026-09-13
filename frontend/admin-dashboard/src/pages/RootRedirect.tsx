import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { apiGetMe } from '../api/subjects'
import { HomePage } from './HomePage'

type AuthState = 'checking' | 'authed' | 'unauthed'

// RootRedirect is the index route under DashboardLayout. It probes
// /api/auth/me once on mount:
//   - 200 OK → render the XNAT-style HomePage (authenticated experience).
//   - 401 / network error → <Navigate to="/about" />.
//
// This is the single piece of plumbing that makes apex behave correctly
// for both anonymous visitors (sent to the public About page) and
// authenticated researchers (sent to their projects list). No hostname
// inspection is needed — there is only one hostname after the
// domain-reorg.
export function RootRedirect() {
  const [state, setState] = useState<AuthState>('checking')

  useEffect(() => {
    let cancelled = false
    apiGetMe()
      .then(() => !cancelled && setState('authed'))
      .catch(() => !cancelled && setState('unauthed'))
    return () => {
      cancelled = true
    }
  }, [])

  if (state === 'checking') {
    // Brief spinner-like neutral state. Avoids flashing the About page
    // for the millisecond it takes /api/auth/me to resolve.
    return <div className="aegis-muted" style={{ padding: 24 }}>Loading…</div>
  }

  if (state === 'unauthed') {
    return <Navigate to="/about" replace />
  }

  return <HomePage />
}
