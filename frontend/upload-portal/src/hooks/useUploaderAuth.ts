import { useCallback, useEffect, useState } from 'react'

export interface UploaderProject {
  id: string
  name: string
  slug: string
  description?: string
}

export interface UploaderUser {
  id: string
  email: string
  name: string
}

/**
 * - 'loading' — initial /api/auth/uploader-me probe in flight
 * - 'authed'  — valid uploader session cookie; user + scoped projects set
 * - 'open'    — no session, but the API served the public project list
 *               (local dev with AUTH_ENABLED=false); app renders without login
 * - 'anon'    — no session and no open access; render the login page
 */
export type UploaderAuthStatus = 'loading' | 'authed' | 'open' | 'anon'

interface MeResponse {
  id: string
  email: string
  name: string
  projects: UploaderProject[]
}

/**
 * Account auth for the upload portal.
 *
 * On mount, GET /api/auth/uploader-me decides between the login page and the
 * app. All calls are same-origin (nginx proxies /api to the Go API) — we pass
 * credentials:'same-origin' explicitly so the session cookie always flows.
 *
 * Open-mode fallback: when uploader-me 401s we probe GET /api/auth/me and
 * GET /api/projects. With AUTH_ENABLED=false (docker-compose local dev) there
 * are no uploader accounts at all, so a hard login wall would brick the
 * portal; /api/auth/me answering 200 (the API auto-authenticates every
 * request in dev, and infra-level auth like IAP also lands here) plus a
 * non-empty project list means the app can render in "open mode". Anonymous
 * visitors on a real deployment get 401 from BOTH probes (only /api/projects
 * would succeed, since non-restricted projects are public), so they see the
 * login page — and /api/upload/* would reject them anyway.
 */
export function useUploaderAuth() {
  const [status, setStatus] = useState<UploaderAuthStatus>('loading')
  const [user, setUser] = useState<UploaderUser | null>(null)
  const [projects, setProjects] = useState<UploaderProject[]>([])

  const refresh = useCallback(async () => {
    try {
      const res = await fetch('/api/auth/uploader-me', { credentials: 'same-origin' })
      if (res.ok) {
        const me = (await res.json()) as MeResponse
        setUser({ id: me.id, email: me.email, name: me.name })
        setProjects(Array.isArray(me.projects) ? me.projects : [])
        setStatus('authed')
        return
      }
    } catch {
      /* network error — fall through to the open-mode probe */
    }

    // No uploader session. Detect open (dev / infra-authed) mode: the general
    // auth identity endpoint must answer 200 (it 401s for anonymous callers
    // when AUTH_ENABLED=true) AND the project list must be non-empty.
    try {
      const meRes = await fetch('/api/auth/me', { credentials: 'same-origin' })
      if (meRes.ok) {
        const res = await fetch('/api/projects', { credentials: 'same-origin' })
        if (res.ok) {
          const data = (await res.json()) as UploaderProject[]
          if (Array.isArray(data) && data.length > 0) {
            setUser(null)
            setProjects(data)
            setStatus('open')
            return
          }
        }
      }
    } catch {
      /* ignore — treated as anon */
    }

    setUser(null)
    setProjects([])
    setStatus('anon')
  }, [])

  useEffect(() => {
    void refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  /** Returns on success; throws Error with a user-facing message on failure. */
  const login = useCallback(async (email: string, password: string) => {
    let res: Response
    try {
      res = await fetch('/api/auth/uploader-login', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })
    } catch {
      throw new Error('Could not reach the server. Check your connection and try again.')
    }
    if (res.status === 401) {
      throw new Error('Invalid email or password')
    }
    if (res.status === 429) {
      throw new Error('Too many attempts — wait a moment and try again.')
    }
    if (!res.ok) {
      throw new Error(`Sign-in failed (${res.status}). Try again in a moment.`)
    }
    // Session cookie is now set; reload identity + project list.
    await refresh()
  }, [refresh])

  /**
   * Ends the session and returns to the login page. Deliberately does NOT
   * re-run the open-mode probe: an explicit sign-out should always land on
   * the login screen, even if public projects are visible anonymously.
   */
  const logout = useCallback(async () => {
    try {
      await fetch('/api/auth/uploader-logout', { method: 'POST', credentials: 'same-origin' })
    } catch {
      /* clear local state regardless */
    }
    setUser(null)
    setProjects([])
    setStatus('anon')
  }, [])

  return { status, user, projects, login, logout, refresh }
}
