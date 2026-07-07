import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './App.css'
import { App } from './App'
import { LoginPage } from './components/LoginPage'
import { InviteRedeemPage } from './components/InviteRedeemPage'
import { useUploaderAuth } from './hooks/useUploaderAuth'

// Invite redemption route — no router dependency, just pathname parsing.
// Supports both /invite/{token} (current emails) and /upload/invite/{token}
// (the legacy path older invitation emails linked to via the landing page).
const INVITE_PATH_RE = /^\/(?:upload\/)?invite\/([^/]+)\/?$/
const inviteMatch = INVITE_PATH_RE.exec(window.location.pathname)

/** Full-viewport spinner shown while the session probe is in flight. */
function AuthLoading() {
  return (
    <div className="auth-page">
      <div className="auth-spinner" aria-label="Loading" />
    </div>
  )
}

/**
 * Root — account-auth gate for the portal.
 *   loading → spinner
 *   anon    → login page
 *   authed  → App scoped to the user's projects
 *   open    → App with the public project list (local dev, AUTH_ENABLED=false)
 */
function Root() {
  const { status, user, projects, login, logout } = useUploaderAuth()

  if (status === 'loading') return <AuthLoading />
  if (status === 'anon') return <LoginPage onLogin={login} />

  return (
    <App
      user={user}
      projects={projects}
      openMode={status === 'open'}
      onLogout={logout}
    />
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    {inviteMatch
      ? <InviteRedeemPage token={decodeURIComponent(inviteMatch[1])} />
      : <Root />}
  </StrictMode>,
)
