import { useState, useEffect } from 'react'

const STORAGE_KEY = 'aegis_invite_token'
const REQUIRED_TOKEN = import.meta.env.VITE_INVITE_TOKEN as string | undefined

/**
 * Returns whether the current visitor has a valid invite code.
 *
 * Access control logic:
 * 1. If VITE_INVITE_TOKEN is not set (empty/undefined), everyone is admitted (dev/open mode).
 * 2. If set, check URL ?invite=<token> first → save to localStorage on match.
 * 3. Fall back to checking localStorage.aegis_invite_token.
 */
export function useInviteCode() {
  // If no token is configured, everyone is admitted.
  const gatingEnabled = Boolean(REQUIRED_TOKEN)

  function checkAccess(): boolean {
    if (!gatingEnabled) return true

    // Check URL param first.
    const params = new URLSearchParams(window.location.search)
    const urlToken = params.get('invite')
    if (urlToken && urlToken === REQUIRED_TOKEN) {
      try {
        localStorage.setItem(STORAGE_KEY, urlToken)
      } catch { /* localStorage may be unavailable */ }
      return true
    }

    // Check stored token.
    try {
      return localStorage.getItem(STORAGE_KEY) === REQUIRED_TOKEN
    } catch {
      return false
    }
  }

  const [admitted, setAdmitted] = useState(checkAccess)

  // Re-check if URL params change (e.g., SPA navigation).
  useEffect(() => {
    setAdmitted(checkAccess())
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function submitCode(code: string): boolean {
    if (!gatingEnabled) return true
    if (code.trim() === REQUIRED_TOKEN) {
      try {
        localStorage.setItem(STORAGE_KEY, code.trim())
      } catch { /* ignore */ }
      setAdmitted(true)
      return true
    }
    return false
  }

  function revoke() {
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch { /* ignore */ }
    setAdmitted(false)
  }

  return { admitted, gatingEnabled, submitCode, revoke }
}
