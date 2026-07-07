import { useState } from 'react'

interface LoginPageProps {
  /** Performs the login; throws Error with a user-facing message on failure. */
  onLogin: (email: string, password: string) => Promise<void>
}

/**
 * LoginPage — email + password sign-in for uploader accounts.
 *
 * Accounts are provisioned by invitation only (a study coordinator invites an
 * uploader to a project; the invite email links to /invite/{token} where the
 * recipient sets a password). There is no self-signup.
 */
export function LoginPage({ onLogin }: LoginPageProps) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const canSubmit = email.trim() !== '' && password !== '' && !submitting

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    setSubmitting(true)
    setError(null)
    try {
      await onLogin(email.trim(), password)
      // On success the auth hook flips status to 'authed' and the app renders.
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed. Try again.')
      setPassword('')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-brand">
        <h1>AEGIS Upload Portal</h1>
        <p>Anonymization &amp; Exchange Gateway for Imaging Studies</p>
      </div>

      <div className="auth-card">
        <h2>Sign in</h2>
        <p className="auth-subtitle">
          Use the uploader account you created from your invitation email.
        </p>

        {error && <div className="auth-error" role="alert">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="auth-field">
            <label htmlFor="login-email">Email</label>
            <input
              id="login-email"
              className="auth-input"
              type="email"
              autoComplete="email"
              autoFocus
              value={email}
              onChange={e => { setEmail(e.target.value); setError(null) }}
              placeholder="you@institution.edu"
            />
          </div>

          <div className="auth-field">
            <label htmlFor="login-password">Password</label>
            <input
              id="login-password"
              className="auth-input"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={e => { setPassword(e.target.value); setError(null) }}
            />
          </div>

          <button className="auth-btn" type="submit" disabled={!canSubmit}>
            {submitting ? 'Signing in…' : 'Sign in'}
          </button>
        </form>

        <p className="auth-footnote">
          Access is by invitation only. If you were invited, follow the link in
          your invitation email to set a password. Need an account? Contact your
          study coordinator.
        </p>
      </div>
    </div>
  )
}
