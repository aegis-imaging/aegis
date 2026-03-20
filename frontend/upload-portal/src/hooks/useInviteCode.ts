import { useState, useEffect } from 'react'

const STORAGE_KEY = 'aegis_invited'

// Gate is enabled when VITE_INVITE_GATE_ENABLED=true is baked in at build time.
// When disabled (default for dev), the gate is transparent.
const GATE_ENABLED = import.meta.env.VITE_INVITE_GATE_ENABLED === 'true'

/**
 * Controls upload portal access via server-side invite codes.
 *
 * All API calls are relative (/api/invite/validate) — nginx proxies them to
 * the Go API regardless of which cloud the service is deployed on.
 *
 * Access control logic:
 * 1. If VITE_INVITE_GATE_ENABLED != 'true', everyone is admitted (dev / open mode).
 * 2. On page load, check localStorage for a prior admission token.
 * 3. On code submit, call POST /api/invite/validate — the server checks the code
 *    against the invite_codes table (no secret token baked into the bundle).
 * 4. On success, save 'admitted' to localStorage so the user isn't re-prompted.
 *
 * URL param ?invite=<code> auto-submits on page load for direct share links.
 */
export function useInviteCode() {
  const gatingEnabled = GATE_ENABLED

  function isStoredAdmitted(): boolean {
    try {
      return localStorage.getItem(STORAGE_KEY) === 'admitted'
    } catch {
      return false
    }
  }

  const [admitted, setAdmitted] = useState<boolean>(
    gatingEnabled ? isStoredAdmitted() : true
  )
  const [autoSubmitting, setAutoSubmitting] = useState(false)

  // On mount: auto-submit ?invite=<code> from URL if present and not already admitted.
  useEffect(() => {
    if (!gatingEnabled || admitted) return
    const params = new URLSearchParams(window.location.search)
    const urlCode = params.get('invite')
    if (!urlCode) return

    setAutoSubmitting(true)
    submitCode(urlCode).finally(() => setAutoSubmitting(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function submitCode(code: string): Promise<boolean> {
    if (!gatingEnabled) return true
    try {
      const res = await fetch('/api/invite/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: code.trim() }),
      })
      if (!res.ok) return false
      const data = (await res.json()) as { valid: boolean }
      if (data.valid) {
        try { localStorage.setItem(STORAGE_KEY, 'admitted') } catch { /* ignore */ }
        setAdmitted(true)
        return true
      }
    } catch {
      /* network error — treat as invalid */
    }
    return false
  }

  function revoke() {
    try { localStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
    setAdmitted(false)
  }

  return { admitted, gatingEnabled, autoSubmitting, submitCode, revoke }
}
