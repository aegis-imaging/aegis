import { useEffect, useState } from 'react'

interface InviteRedeemPageProps {
  token: string
}

interface InviteInfo {
  email: string // masked, e.g. c*****@x.com
  name: string
  project_name: string
  institution_name?: string
  expires_at: string
}

type LoadState =
  | { kind: 'loading' }
  | { kind: 'ready'; invite: InviteInfo }
  | { kind: 'notfound' }
  | { kind: 'gone'; message: string }
  | { kind: 'error' }

const PASSWORD_MIN = 8
const PASSWORD_MAX = 72

/**
 * InviteRedeemPage — served at /invite/{token} (and the legacy
 * /upload/invite/{token} path used by older invitation emails).
 *
 * Shows who the invite is for and which project it grants, then lets the
 * recipient choose a password. A successful redeem sets the session cookie
 * server-side, so we hard-navigate to '/' and land signed in.
 */
export function InviteRedeemPage({ token }: InviteRedeemPageProps) {
  const [state, setState] = useState<LoadState>({ kind: 'loading' })
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [passwordTouched, setPasswordTouched] = useState(false)
  const [confirmTouched, setConfirmTouched] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const res = await fetch(`/api/uploader-invites/${encodeURIComponent(token)}`, {
          credentials: 'same-origin',
        })
        if (cancelled) return
        if (res.ok) {
          const invite = (await res.json()) as InviteInfo
          setState({ kind: 'ready', invite })
          setName(invite.name ?? '')
          return
        }
        if (res.status === 404) {
          setState({ kind: 'notfound' })
          return
        }
        if (res.status === 410) {
          let message = 'This invitation has already been used or expired.'
          try {
            const body = (await res.json()) as { error?: string }
            if (body.error) message = body.error
          } catch { /* keep default */ }
          setState({ kind: 'gone', message })
          return
        }
        setState({ kind: 'error' })
      } catch {
        if (!cancelled) setState({ kind: 'error' })
      }
    }
    void load()
    return () => { cancelled = true }
  }, [token])

  const tooShort = password.length > 0 && password.length < PASSWORD_MIN
  const tooLong = password.length > PASSWORD_MAX
  const mismatch = confirm.length > 0 && confirm !== password
  const passwordValid = password.length >= PASSWORD_MIN && password.length <= PASSWORD_MAX
  const canSubmit = passwordValid && confirm === password && !submitting

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    setSubmitting(true)
    setSubmitError(null)
    try {
      const res = await fetch(`/api/uploader-invites/${encodeURIComponent(token)}/redeem`, {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password, name: name.trim() }),
      })
      if (res.ok) {
        // Session cookie is set — land in the app signed in.
        window.location.href = '/'
        return
      }
      let message = `Could not complete setup (${res.status}). Try again.`
      try {
        const body = (await res.json()) as { error?: string }
        if (body.error) message = body.error
      } catch { /* keep default */ }
      if (res.status === 410) {
        setState({ kind: 'gone', message })
        return
      }
      setSubmitError(message)
    } catch {
      setSubmitError('Could not reach the server. Check your connection and try again.')
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
        {state.kind === 'loading' && (
          <div style={{ display: 'flex', justifyContent: 'center', padding: '24px 0' }}>
            <div className="auth-spinner" aria-label="Loading invitation" />
          </div>
        )}

        {state.kind === 'notfound' && (
          <>
            <h2>Invitation not found</h2>
            <p className="auth-subtitle">
              This invitation link isn't recognized. Double-check that you opened
              the full link from your email, or ask your study coordinator to
              send a new invitation.
            </p>
          </>
        )}

        {state.kind === 'gone' && (
          <>
            <h2>Invitation link can't be used</h2>
            <div className="auth-error" role="alert">{state.message}</div>
            <p className="auth-subtitle">
              This invitation has already been used or expired — ask your study
              coordinator for a new one. If you already set a password, you can{' '}
              <a href="/" style={{ color: '#0f766e', fontWeight: 600 }}>sign in here</a>.
            </p>
          </>
        )}

        {state.kind === 'error' && (
          <>
            <h2>Something went wrong</h2>
            <p className="auth-subtitle">
              We couldn't load your invitation. Refresh the page to try again,
              or contact your study coordinator if the problem persists.
            </p>
          </>
        )}

        {state.kind === 'ready' && (
          <>
            <h2>Set up your uploader account</h2>
            <p className="auth-subtitle">
              You've been invited to upload imaging studies to{' '}
              <strong>{state.invite.project_name}</strong>
              {state.invite.institution_name ? <> ({state.invite.institution_name})</> : null}.
            </p>

            <div className="auth-info">
              Account email: <strong>{state.invite.email}</strong>
            </div>

            {submitError && <div className="auth-error" role="alert">{submitError}</div>}

            <form onSubmit={handleSubmit}>
              <div className="auth-field">
                <label htmlFor="redeem-name">Your name <span style={{ color: '#6b7280', fontWeight: 400 }}>(optional)</span></label>
                <input
                  id="redeem-name"
                  className="auth-input"
                  type="text"
                  autoComplete="name"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="Jane Coordinator"
                />
              </div>

              <div className="auth-field">
                <label htmlFor="redeem-password">Password</label>
                <input
                  id="redeem-password"
                  className="auth-input"
                  type="password"
                  autoComplete="new-password"
                  value={password}
                  aria-invalid={passwordTouched && (tooShort || tooLong)}
                  onChange={e => setPassword(e.target.value)}
                  onBlur={() => setPasswordTouched(true)}
                />
                {passwordTouched && tooShort ? (
                  <p className="auth-field-error">Password must be at least {PASSWORD_MIN} characters.</p>
                ) : tooLong ? (
                  <p className="auth-field-error">Password must be at most {PASSWORD_MAX} characters.</p>
                ) : (
                  <p className="auth-hint">{PASSWORD_MIN}–{PASSWORD_MAX} characters.</p>
                )}
              </div>

              <div className="auth-field">
                <label htmlFor="redeem-confirm">Confirm password</label>
                <input
                  id="redeem-confirm"
                  className="auth-input"
                  type="password"
                  autoComplete="new-password"
                  value={confirm}
                  aria-invalid={confirmTouched && mismatch}
                  onChange={e => setConfirm(e.target.value)}
                  onBlur={() => setConfirmTouched(true)}
                />
                {confirmTouched && mismatch && (
                  <p className="auth-field-error">Passwords don't match.</p>
                )}
              </div>

              <button className="auth-btn" type="submit" disabled={!canSubmit}>
                {submitting ? 'Creating account…' : 'Create account & start uploading'}
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  )
}
