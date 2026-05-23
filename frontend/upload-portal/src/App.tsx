import { useEffect, useState } from 'react'
import './App.css'
import { UploadFlow, type DisplayTimezoneMode } from '@aegis/upload-shared'

interface Project {
  id: string
  name: string
  slug: string
  description: string
}

interface AuthUser {
  id: string
  email: string
  name: string
  role: 'admin' | 'viewer' | 'researcher'
}

type ProjectMemberRole = 'owner' | 'coordinator' | 'reviewer' | 'site_coordinator' | 'site_viewer'

interface ProjectMember {
  admin_user_id: string
  user_email: string
  role: ProjectMemberRole
  institution_id: string | null
  institution_name?: string
}

const GLOBAL_DISPLAY_TZ_MODE_KEY = 'aegis.ui.display_timezone_mode'
const GLOBAL_DISPLAY_TZ_CUSTOM_KEY = 'aegis.ui.display_timezone_custom'
const LEGACY_DISPLAY_TZ_MODE_KEYS = [
  'aegis.display_timezone_mode',
  'aegis.export.display_timezone_mode',
  'aegis.upload.display_timezone_mode',
]
const LEGACY_DISPLAY_TZ_CUSTOM_KEYS = [
  'aegis.display_timezone_custom',
  'aegis.export.display_timezone_custom',
  'aegis.upload.display_timezone_custom',
]

function browserTimeZone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Local'
  } catch {
    return 'Local'
  }
}

function normalizeIanaTimeZone(value: string): string | null {
  const candidate = value.trim()
  if (!candidate) return null
  try {
    return new Intl.DateTimeFormat('en-US', { timeZone: candidate }).resolvedOptions().timeZone
  } catch {
    return null
  }
}

function readFromLocalStorage(keys: string[]): string | null {
  if (typeof window === 'undefined') return null
  for (const key of keys) {
    const value = window.localStorage.getItem(key)
    if (value !== null) return value
  }
  return null
}

function writeToLocalStorage(keys: string[], value: string) {
  if (typeof window === 'undefined') return
  for (const key of keys) {
    window.localStorage.setItem(key, value)
  }
}

function readDisplayTimezone(): { mode: DisplayTimezoneMode; customTimeZone: string } {
  const mode = readFromLocalStorage([GLOBAL_DISPLAY_TZ_MODE_KEY, ...LEGACY_DISPLAY_TZ_MODE_KEYS])
  const customTimeZone = readFromLocalStorage([GLOBAL_DISPLAY_TZ_CUSTOM_KEY, ...LEGACY_DISPLAY_TZ_CUSTOM_KEYS]) ?? ''
  const parsedMode: DisplayTimezoneMode =
    mode === 'local' || mode === 'custom' ? mode : 'utc'
  return { mode: parsedMode, customTimeZone }
}

function writeDisplayTimezone(mode: DisplayTimezoneMode, customTimeZone: string) {
  writeToLocalStorage([GLOBAL_DISPLAY_TZ_MODE_KEY, ...LEGACY_DISPLAY_TZ_MODE_KEYS], mode)
  writeToLocalStorage([GLOBAL_DISPLAY_TZ_CUSTOM_KEY, ...LEGACY_DISPLAY_TZ_CUSTOM_KEYS], customTimeZone)
}

/**
 * Upload portal host shell.
 *
 * Owns the *host* concerns: auth user fetch, project list + selector,
 * researcher attribution lookup, timezone preference UI, the page chrome
 * (header, error banner). Everything inside the stage machine
 * (select → parsing → preview → uploading → ready) is now delegated to
 * the shared `<UploadFlow>` component from `@aegis/upload-shared`, so the
 * upload UI stays identical to the desktop app and any future host.
 */
export function App() {
  // ── Timezone preference ────────────────────────────────────────────────
  const [displayTimezoneMode, setDisplayTimezoneMode] = useState<DisplayTimezoneMode>(
    () => readDisplayTimezone().mode
  )
  const [displayTimezoneCustom, setDisplayTimezoneCustom] = useState(
    () => readDisplayTimezone().customTimeZone
  )

  useEffect(() => {
    writeDisplayTimezone(displayTimezoneMode, displayTimezoneCustom)
  }, [displayTimezoneMode, displayTimezoneCustom])

  const validCustomTimeZone = normalizeIanaTimeZone(displayTimezoneCustom) ?? ''
  const localTimeZone = browserTimeZone()

  // ── Auth ───────────────────────────────────────────────────────────────
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null)

  useEffect(() => {
    fetch('/api/auth/me')
      .then(res => (res.ok ? (res.json() as Promise<AuthUser>) : null))
      .then(user => { if (user) setCurrentUser(user) })
      .catch(() => { /* not authenticated or auth disabled — continue as public */ })
  }, [])

  // ── Projects ───────────────────────────────────────────────────────────
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedProject, setSelectedProject] = useState('default')
  const [projectsLoading, setProjectsLoading] = useState(true)

  useEffect(() => {
    fetch('/api/projects')
      .then(res => (res.ok ? (res.json() as Promise<Project[]>) : []))
      .then(data => {
        setProjects(data)
        if (data.length === 1) setSelectedProject(data[0].slug)
        else if (data.length > 0 && !data.find(p => p.slug === 'default')) {
          setSelectedProject(data[0].slug)
        }
      })
      .catch(() => setProjects([]))
      .finally(() => setProjectsLoading(false))
  }, [])

  // ── Researcher attribution (institution lookup) ────────────────────────
  const [attributionRole, setAttributionRole] = useState<ProjectMemberRole | null>(null)
  const [attributionInstitutionId, setAttributionInstitutionId] = useState<string | null>(null)
  const [attributionInstitutionName, setAttributionInstitutionName] = useState('')
  const [attributionLoading, setAttributionLoading] = useState(false)

  useEffect(() => {
    const selected = projects.find(p => p.slug === selectedProject)
    if (!currentUser || currentUser.role !== 'researcher' || !selected) {
      setAttributionRole(null)
      setAttributionInstitutionId(null)
      setAttributionInstitutionName('')
      setAttributionLoading(false)
      return
    }
    setAttributionLoading(true)
    fetch(`/api/projects/${selected.id}/members`)
      .then(res => (res.ok ? res.json() : null))
      .then(data => {
        const members = (data?.members ?? []) as ProjectMember[]
        const me = members.find(m => m.admin_user_id === currentUser.id || m.user_email === currentUser.email)
        if (!me) {
          setAttributionRole(null)
          setAttributionInstitutionId(null)
          setAttributionInstitutionName('')
          return
        }
        setAttributionRole(me.role)
        setAttributionInstitutionId(me.institution_id ?? null)
        setAttributionInstitutionName(me.institution_name ?? '')
      })
      .catch(() => {
        setAttributionRole(null)
        setAttributionInstitutionId(null)
        setAttributionInstitutionName('')
      })
      .finally(() => setAttributionLoading(false))
  }, [projects, selectedProject, currentUser])

  const attributionRequired =
    attributionRole === 'site_coordinator' || attributionRole === 'site_viewer'

  // ── Anonymization profile (per project) ────────────────────────────────
  const [retainedTags, setRetainedTags] = useState<string[] | undefined>(undefined)
  const [keepPrivateTags, setKeepPrivateTags] = useState(false)

  useEffect(() => {
    if (!selectedProject) return
    fetch(`/api/projects/${selectedProject}/active-anon-profile`)
      .then(res => (res.ok ? res.json() : null))
      .then((profile: { retained_tags?: string[]; keep_private_tags?: boolean } | null) => {
        if (!profile) {
          setRetainedTags(undefined)
          setKeepPrivateTags(false)
          return
        }
        if (Array.isArray(profile.retained_tags) && profile.retained_tags.length > 0) {
          setRetainedTags(profile.retained_tags)
        } else {
          setRetainedTags(undefined)
        }
        setKeepPrivateTags(!!profile.keep_private_tags)
      })
      .catch(() => {
        setRetainedTags(undefined)
        setKeepPrivateTags(false)
      })
  }, [selectedProject])

  // ── Render ─────────────────────────────────────────────────────────────

  const attributionHint = currentUser?.role === 'researcher'
    ? attributionLoading
      ? 'Institution attribution: loading…'
      : attributionInstitutionId
        ? `Institution attribution: ${attributionInstitutionName || attributionInstitutionId}`
        : attributionRequired
          ? 'Institution attribution required for your site-scoped role (not configured).'
          : 'Institution attribution: automatic (project context/IP fallback).'
    : undefined

  return (
    <div style={{
      maxWidth: '1000px',
      margin: '0 auto',
      padding: '32px 24px',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    }}>
      {/* Header */}
      <header style={{ marginBottom: '32px' }}>
        <h1 style={{ margin: '0 0 4px', fontSize: '28px', fontWeight: 700 }}>
          AEGIS Upload Portal
        </h1>
        <p style={{ margin: 0, color: '#6b7280', fontSize: '15px' }}>
          Anonymization & Exchange Gateway for Imaging Studies
        </p>
        <div className="tz-control">
          <label className="tz-label" htmlFor="upload-display-timezone-mode">Time Zone</label>
          <select
            id="upload-display-timezone-mode"
            className="tz-select"
            value={displayTimezoneMode}
            onChange={e => setDisplayTimezoneMode(e.target.value as DisplayTimezoneMode)}
          >
            <option value="utc">UTC</option>
            <option value="local">Local ({localTimeZone})</option>
            <option value="custom">Custom</option>
          </select>
          {displayTimezoneMode === 'custom' && (
            <>
              <input
                className="tz-input"
                type="text"
                placeholder="America/Chicago"
                value={displayTimezoneCustom}
                onChange={e => setDisplayTimezoneCustom(e.target.value)}
              />
              {!validCustomTimeZone && displayTimezoneCustom.trim() && (
                <span className="tz-warning">Invalid IANA time zone</span>
              )}
            </>
          )}
        </div>
      </header>

      {/* Auth context banner — shown when a scoped researcher is identified */}
      {currentUser?.role === 'researcher' && (
        <div style={{
          padding: '10px 14px',
          backgroundColor: '#f0fdfa',
          border: '1px solid #99f6e4',
          borderRadius: '8px',
          fontSize: '13px',
          color: '#0f766e',
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          marginBottom: '16px',
        }}>
          <span>Logged in as <strong>{currentUser.name || currentUser.email}</strong>
            {projects.length === 1
              ? ` — viewing project: ${projects[0].name}`
              : projects.length > 1
                ? ` — ${projects.length} projects available`
                : ' — no projects assigned'}
          </span>
        </div>
      )}

      {/* Project selector — shown when more than one project is available */}
      {!projectsLoading && projects.length > 1 && (
        <div style={{
          padding: '16px',
          backgroundColor: '#f9fafb',
          border: '1px solid #e5e7eb',
          borderRadius: '8px',
          marginBottom: '16px',
        }}>
          <label
            htmlFor="project-select"
            style={{ display: 'block', fontSize: '14px', fontWeight: 500, marginBottom: '6px' }}
          >
            Project
          </label>
          <select
            id="project-select"
            value={selectedProject}
            onChange={e => setSelectedProject(e.target.value)}
            style={{
              width: '100%',
              padding: '8px 12px',
              border: '1px solid #d1d5db',
              borderRadius: '6px',
              fontSize: '14px',
              backgroundColor: '#fff',
              boxSizing: 'border-box',
            }}
          >
            {projects.map(p => (
              <option key={p.slug} value={p.slug}>
                {p.name}{p.description ? ` — ${p.description}` : ''}
              </option>
            ))}
          </select>
        </div>
      )}

      {/* The whole stage machine. */}
      <UploadFlow
        projectSlug={selectedProject}
        attributionInstitutionId={attributionInstitutionId}
        attributionInstitutionName={attributionInstitutionName}
        attributionRequired={attributionRequired}
        attributionHint={attributionHint}
        retainedTags={retainedTags}
        keepPrivateTags={keepPrivateTags}
        displayTimezoneMode={displayTimezoneMode}
        displayTimezoneCustom={validCustomTimeZone}
      />
    </div>
  )
}
