import { useState, useCallback, useEffect, useRef } from 'react'
import './App.css'
import {
  FileDropZone,
  StudySummary,
  TagDiffTable,
  HeavyModeToggle,
  PixelScrubProgress,
  BulkStudyTable,
  type HeavyModeSettings,
  type PixelScrubProgressState,
} from '@aegis/upload-shared'
import { parseDicomFile, buildStudySummary, isDicomFile, groupByStudy, studyLooksLikeHeadScan } from '@aegis/client'
import { deidentify } from '@aegis/client'
import { bulkUploadStudies } from '@aegis/client'
import type { ParsedDicomFile, StudySummary as StudySummaryType, DicomTag, UploadResult, FaceDeidResult, BulkProgress, PixelScrubResult } from '@aegis/client'

type Stage = 'select' | 'parsing' | 'preview' | 'uploading' | 'ready'

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

interface StudyGroup {
  uid: string
  files: ParsedDicomFile[]
  summary: StudySummaryType
}

type ProjectMemberRole = 'owner' | 'coordinator' | 'reviewer' | 'site_coordinator' | 'site_viewer'

interface ProjectMember {
  admin_user_id: string
  user_email: string
  role: ProjectMemberRole
  institution_id: string | null
  institution_name?: string
}

type DisplayTimezoneMode = 'utc' | 'local' | 'custom'

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
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

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

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
  if (typeof window === 'undefined') {
    return { mode: 'utc', customTimeZone: '' }
  }
  const storedMode = readFromLocalStorage([GLOBAL_DISPLAY_TZ_MODE_KEY, ...LEGACY_DISPLAY_TZ_MODE_KEYS])
  const mode: DisplayTimezoneMode =
    storedMode === 'local' || storedMode === 'custom' ? storedMode : 'utc'
  const customTimeZone = readFromLocalStorage([GLOBAL_DISPLAY_TZ_CUSTOM_KEY, ...LEGACY_DISPLAY_TZ_CUSTOM_KEYS]) ?? ''
  return { mode, customTimeZone }
}

function writeDisplayTimezone(mode: DisplayTimezoneMode, customTimeZone: string) {
  if (typeof window === 'undefined') return
  writeToLocalStorage([GLOBAL_DISPLAY_TZ_MODE_KEY, ...LEGACY_DISPLAY_TZ_MODE_KEYS], mode)
  writeToLocalStorage([GLOBAL_DISPLAY_TZ_CUSTOM_KEY, ...LEGACY_DISPLAY_TZ_CUSTOM_KEYS], customTimeZone)
}

export function App() {
  const [displayTimezoneMode, setDisplayTimezoneMode] = useState<DisplayTimezoneMode>(() => readDisplayTimezone().mode)
  const [displayTimezoneCustom, setDisplayTimezoneCustom] = useState(() => readDisplayTimezone().customTimeZone)
  const [stage, setStage] = useState<Stage>('select')
  const [files, setFiles] = useState<ParsedDicomFile[]>([])
  const [studyGroups, setStudyGroups] = useState<StudyGroup[]>([])
  const [tagChanges, setTagChanges] = useState<DicomTag[]>([])
  const [privateTagsRemoved, setPrivateTagsRemoved] = useState(0)
  const [parseProgress, setParseProgress] = useState({ current: 0, total: 0 })
  const [uploadProgress, setUploadProgress] = useState({ current: 0, total: 0 })
  const [uploadResults, setUploadResults] = useState<UploadResult[]>([])
  const [uploaderEmail, setUploaderEmail] = useState('')
  const [emailTouched, setEmailTouched] = useState(false)
  const [currentFile, setCurrentFile] = useState('')
  const [currentStudyIndex, setCurrentStudyIndex] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [totalSize, setTotalSize] = useState(0)

  // Heavy-mode (in-browser pixel/face de-id) state.
  const [heavyMode, setHeavyMode] = useState<HeavyModeSettings>({ pixelScrub: false, faceDeid: false })
  const [scrubProgress, setScrubProgress] = useState<PixelScrubProgressState>({
    currentFileIndex: 0, totalFiles: 0, currentFileName: '', results: [],
  })
  const [faceDeidResults, setFaceDeidResults] = useState<{ studyUid: string; result: FaceDeidResult }[]>([])
  const [bulkProgress, setBulkProgress] = useState<BulkProgress | null>(null)
  const [bulkConcurrency, setBulkConcurrency] = useState(1)

  // Auth state — fire-and-forget; non-blocking (auth is handled at infra level)
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null)

  // Project selector state
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedProject, setSelectedProject] = useState('default')
  const [projectsLoading, setProjectsLoading] = useState(true)
  const [attributionRole, setAttributionRole] = useState<ProjectMemberRole | null>(null)
  const [attributionInstitutionId, setAttributionInstitutionId] = useState<string | null>(null)
  const [attributionInstitutionName, setAttributionInstitutionName] = useState('')
  const [attributionLoading, setAttributionLoading] = useState(false)
  const validCustomTimeZone = normalizeIanaTimeZone(displayTimezoneCustom) ?? ''
  const localTimeZone = browserTimeZone()

  // Cancel refs
  const cancelParseRef = useRef(false)
  const uploadAbortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    writeDisplayTimezone(displayTimezoneMode, displayTimezoneCustom)
  }, [displayTimezoneMode, displayTimezoneCustom])

  // Fetch current user (non-blocking — auth handled at infra level by IAP/Easy Auth/ALB)
  useEffect(() => {
    fetch('/api/auth/me')
      .then(res => res.ok ? res.json() as Promise<AuthUser> : null)
      .then(user => { if (user) setCurrentUser(user) })
      .catch(() => { /* not authenticated or auth disabled — continue as public */ })
  }, [])

  // Fetch projects on mount — API filters by membership for researcher role,
  // returns only non-restricted projects for unauthenticated callers.
  useEffect(() => {
    fetch('/api/projects')
      .then(res => res.ok ? res.json() as Promise<Project[]> : [])
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

  const attributionRequired = attributionRole === 'site_coordinator' || attributionRole === 'site_viewer'

  const emailValid = uploaderEmail === '' || EMAIL_RE.test(uploaderEmail)

  const handleFilesSelected = useCallback(async (selectedFiles: File[]) => {
    setError(null)
    setStage('parsing')
    cancelParseRef.current = false

    try {
      // Filter to DICOM files only
      const dicomChecks = await Promise.all(
        selectedFiles.map(async f => ({ file: f, isDicom: await isDicomFile(f) }))
      )
      const dicomFiles = dicomChecks.filter(c => c.isDicom).map(c => c.file)

      if (dicomFiles.length === 0) {
        setError('No valid DICOM files found. Files must be DICOM Part 10 format.')
        setStage('select')
        return
      }

      // Compute total size
      const size = dicomFiles.reduce((acc, f) => acc + f.size, 0)
      setTotalSize(size)
      setParseProgress({ current: 0, total: dicomFiles.length })

      // Parse files sequentially to avoid memory pressure
      const parsed: ParsedDicomFile[] = []
      for (let i = 0; i < dicomFiles.length; i++) {
        if (cancelParseRef.current) {
          setStage('select')
          return
        }
        const arrayBuffer = await dicomFiles[i].arrayBuffer()
        try {
          const { parsed: p } = parseDicomFile(arrayBuffer, dicomFiles[i].name)
          parsed.push(p)
        } catch (parseErr) {
          console.warn(`Skipping ${dicomFiles[i].name}: ${parseErr}`)
        }
        setParseProgress({ current: i + 1, total: dicomFiles.length })
      }

      if (cancelParseRef.current) {
        setStage('select')
        return
      }

      if (parsed.length === 0) {
        setError('Could not parse any of the selected files.')
        setStage('select')
        return
      }

      setFiles(parsed)

      // Group by study
      const groups = groupByStudy(parsed)
      const studyList: StudyGroup[] = Array.from(groups.entries()).map(([uid, studyFiles]) => ({
        uid,
        files: studyFiles,
        summary: buildStudySummary(studyFiles),
      }))
      setStudyGroups(studyList)

      // Run de-identification preview on the first file to show tag diff
      const firstBuffer = parsed[0].arrayBuffer
      const { dataset } = parseDicomFile(firstBuffer, parsed[0].filename)
      const deidResult = await deidentify(dataset, { salt: 'aegis-preview' })
      setTagChanges(deidResult.tagChanges)
      setPrivateTagsRemoved(deidResult.privateTagsRemoved)

      setStage('preview')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to parse DICOM files')
      setStage('select')
    }
  }, [])

  const handleCancelParse = useCallback(() => {
    cancelParseRef.current = true
  }, [])

  const handleCancelUpload = useCallback(() => {
    uploadAbortRef.current?.abort()
  }, [])

  const handleReset = useCallback(() => {
    setStage('select')
    setFiles([])
    setStudyGroups([])
    setTagChanges([])
    setPrivateTagsRemoved(0)
    setUploadProgress({ current: 0, total: 0 })
    setUploadResults([])
    setUploaderEmail('')
    setEmailTouched(false)
    setCurrentFile('')
    setCurrentStudyIndex(0)
    setError(null)
    setTotalSize(0)
    cancelParseRef.current = false
    uploadAbortRef.current = null
  }, [])

  const handleUpload = useCallback(async () => {
    if (studyGroups.length === 0) {
      setError('No files available for upload.')
      return
    }

    if (attributionRequired && !attributionInstitutionId) {
      setError('Institution attribution is required for your site-scoped role. Ask an admin to set your project membership institution before uploading.')
      return
    }

    setError(null)
    setStage('uploading')
    const totalFiles = files.length
    setUploadProgress({ current: 0, total: totalFiles })

    const abortController = new AbortController()
    uploadAbortRef.current = abortController

    try {
      // Fetch the project's active anonymization profile
      let retainedTags: string[] | undefined
      let keepPrivateTags = false
      try {
        const profileRes = await fetch(`/api/projects/${selectedProject}/active-anon-profile`)
        if (profileRes.ok) {
          const profile = await profileRes.json() as { retained_tags: string[]; keep_private_tags?: boolean }
          if (Array.isArray(profile.retained_tags) && profile.retained_tags.length > 0) {
            retainedTags = profile.retained_tags
          }
          if (profile.keep_private_tags) {
            keepPrivateTags = true
          }
        }
      } catch {
        // Non-fatal: if profile fetch fails, fall back to full strip
      }

      // Reset heavy-mode progress with the running total across all groups.
      if (heavyMode.pixelScrub) {
        setScrubProgress({
          currentFileIndex: 0,
          totalFiles,
          currentFileName: '',
          results: [],
        })
      }
      setBulkProgress(null)

      const bulkResult = await bulkUploadStudies(
        studyGroups.map(g => ({
          studyInstanceUid: g.uid,
          files: g.files,
          summary: g.summary,
        })),
        {
          projectSlug: selectedProject,
          concurrency: bulkConcurrency,
          signal: abortController.signal,
          onProgress: (p) => {
            setBulkProgress({ ...p, studies: p.studies.map(s => ({ ...s })) })
            const completed = p.studies.reduce((n, s) => n + (s.status === 'completed' || s.status === 'duplicate' ? s.fileCount : 0), 0)
            setUploadProgress({ current: completed, total: totalFiles })
          },
          uploadOptions: {
            uploaderEmail: uploaderEmail.trim() || undefined,
            institutionId: attributionInstitutionId ?? undefined,
            deid: (retainedTags || keepPrivateTags) ? { retainedTags, keepPrivateTags } : undefined,
            pixelScrub: heavyMode.pixelScrub
              ? {
                  enabled: true,
                  onResult: (filename, _i, r) => {
                    setScrubProgress(prev => ({
                      ...prev,
                      currentFileName: filename,
                      currentFileIndex: prev.results.length + 1,
                      results: [...prev.results, { filename, result: r }],
                    }))
                  },
                }
              : undefined,
            faceDeid: heavyMode.faceDeid
              ? {
                  enabled: true,
                  onResult: (r) => {
                    setFaceDeidResults(prev => [...prev, { studyUid: r.studyInstanceUid, result: r }])
                  },
                }
              : undefined,
          },
        }
      )

      const results: UploadResult[] = bulkResult.studies
        .filter(s => s.uploadResult)
        .map(s => s.uploadResult!)
      setUploadResults(results)
      setStage('ready')
    } catch (err) {
      if (abortController.signal.aborted) {
        setError('Upload cancelled.')
      } else {
        setError(err instanceof Error ? err.message : 'Upload failed')
      }
      setStage('preview')
    } finally {
      uploadAbortRef.current = null
    }
  }, [studyGroups, files, selectedProject, uploaderEmail, attributionRequired, attributionInstitutionId, heavyMode])

  const totalFileCount = files.length

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

      {/* Error display */}
      {error && (
        <div style={{
          padding: '12px 16px',
          backgroundColor: '#fff7ed',
          border: '1px solid #fed7aa',
          borderRadius: '8px',
          color: '#ea580c',
          marginBottom: '24px',
          fontSize: '14px',
        }}>
          {error}
        </div>
      )}

      {/* Step 1: File selection */}
      {stage === 'select' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
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

          {/* Project selector */}
          {!projectsLoading && projects.length > 1 && (
            <div style={{
              padding: '16px',
              backgroundColor: '#f9fafb',
              border: '1px solid #e5e7eb',
              borderRadius: '8px',
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

          {currentUser?.role === 'researcher' && (
            <div style={{
              padding: '10px 14px',
              backgroundColor: attributionRequired ? '#fefce8' : '#f8fafc',
              border: `1px solid ${attributionRequired ? '#fde68a' : '#e2e8f0'}`,
              borderRadius: '8px',
              fontSize: '13px',
              color: attributionRequired ? '#92400e' : '#475569',
            }}>
              {attributionLoading
                ? 'Institution attribution: loading…'
                : attributionInstitutionId
                  ? `Institution attribution: ${attributionInstitutionName || attributionInstitutionId}`
                  : attributionRequired
                    ? 'Institution attribution required for your site-scoped role (not configured).'
                    : 'Institution attribution: automatic (project context/IP fallback).'}
            </div>
          )}

          <FileDropZone onFilesSelected={handleFilesSelected} />
        </div>
      )}

      {/* Parsing progress */}
      {stage === 'parsing' && (
        <div style={{ textAlign: 'center', padding: '48px 24px' }}>
          <p style={{ fontSize: '16px', marginBottom: '8px' }}>
            Parsing DICOM files... {parseProgress.current} / {parseProgress.total}
          </p>
          {totalSize > 0 && (
            <p style={{ fontSize: '13px', color: '#6b7280', marginBottom: '16px' }}>
              {formatBytes(totalSize)} total
            </p>
          )}
          <div style={{
            height: '8px',
            backgroundColor: '#e5e7eb',
            borderRadius: '4px',
            overflow: 'hidden',
            maxWidth: '400px',
            margin: '0 auto',
          }}>
            <div style={{
              height: '100%',
              width: `${parseProgress.total ? (parseProgress.current / parseProgress.total) * 100 : 0}%`,
              backgroundColor: '#2563eb',
              borderRadius: '4px',
              transition: 'width 0.2s ease',
            }} />
          </div>
          <button
            onClick={handleCancelParse}
            style={{
              marginTop: '20px',
              padding: '8px 20px',
              borderRadius: '6px',
              border: '1px solid #d1d5db',
              backgroundColor: '#fff',
              cursor: 'pointer',
              fontSize: '13px',
              color: '#6b7280',
            }}
          >
            Cancel
          </button>
        </div>
      )}

      {/* Step 2: Preview */}
      {stage === 'preview' && studyGroups.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
          {/* File stats bar */}
          <div style={{
            padding: '12px 16px',
            backgroundColor: '#eff6ff',
            border: '1px solid #bfdbfe',
            borderRadius: '8px',
            fontSize: '14px',
            color: '#1e40af',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}>
            <span>
              {studyGroups.length > 1
                ? `${studyGroups.length} studies detected \u00b7 ${totalFileCount} files \u00b7 ${formatBytes(totalSize)}`
                : `${totalFileCount} files \u00b7 ${formatBytes(totalSize)}`
              }
            </span>
            <span style={{ fontSize: '12px', color: '#3b82f6' }}>
              Project: {projects.find(p => p.slug === selectedProject)?.name ?? selectedProject}
            </span>
          </div>

          {currentUser?.role === 'researcher' && (
            <div style={{
              padding: '10px 14px',
              backgroundColor: attributionRequired ? '#fefce8' : '#f8fafc',
              border: `1px solid ${attributionRequired ? '#fde68a' : '#e2e8f0'}`,
              borderRadius: '8px',
              fontSize: '13px',
              color: attributionRequired ? '#92400e' : '#475569',
            }}>
              {attributionInstitutionId
                ? `Institution attribution for upload: ${attributionInstitutionName || attributionInstitutionId}`
                : attributionRequired
                  ? 'Institution attribution required for this upload is missing. Upload is blocked until membership is configured.'
                  : 'Institution attribution for upload: automatic (project context/IP fallback).'}
            </div>
          )}

          {/* Multi-study notice */}
          {studyGroups.length > 1 && (
            <div style={{
              padding: '12px 16px',
              backgroundColor: '#fefce8',
              border: '1px solid #fde68a',
              borderRadius: '8px',
              fontSize: '13px',
              color: '#92400e',
            }}>
              Multiple studies detected. Each will be uploaded as a separate session.
            </div>
          )}

          {/* Study summaries */}
          {studyGroups.map((group, idx) => (
            <div key={group.uid}>
              {studyGroups.length > 1 && (
                <h3 style={{ margin: '0 0 8px', fontSize: '14px', fontWeight: 600, color: '#374151' }}>
                  Study {idx + 1} of {studyGroups.length} ({group.files.length} files)
                </h3>
              )}
              <StudySummary
                summary={group.summary}
                displayTimezoneMode={displayTimezoneMode}
                displayTimezoneCustom={validCustomTimeZone}
              />
            </div>
          ))}

          <TagDiffTable tags={tagChanges} privateTagsRemoved={privateTagsRemoved} />

          {/* Bulk-upload preview table (only when more than one study) */}
          {studyGroups.length > 1 && (
            <div>
              <h3 style={{ margin: '0 0 8px', fontSize: '15px', fontWeight: 600, color: '#374151' }}>
                Bulk upload — {studyGroups.length} studies detected
              </h3>
              <BulkStudyTable progress={{
                phase: 'uploading',
                totalStudies: studyGroups.length,
                studiesCompleted: 0,
                studiesFailed: 0,
                active: [],
                studies: studyGroups.map(g => ({
                  studyInstanceUid: g.uid,
                  studyDescription: g.summary.studyDescription || '(no description)',
                  patientId: g.summary.patientId,
                  fileCount: g.files.length,
                  status: 'pending',
                })),
              }} />
              <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8, fontSize: 13 }}>
                <span>Concurrency:</span>
                <select
                  value={String(bulkConcurrency)}
                  onChange={e => setBulkConcurrency(Number(e.target.value))}
                >
                  <option value="1">1 (sequential, gentle on network)</option>
                  <option value="2">2</option>
                  <option value="3">3</option>
                  <option value="4">4 (fastest, may saturate uplink)</option>
                </select>
              </div>
            </div>
          )}

          {/* Heavy mode: in-browser pixel/face de-id (opt-in) */}
          <HeavyModeToggle
            value={heavyMode}
            onChange={setHeavyMode}
            studyCount={studyGroups.length}
            fileCount={totalFileCount}
            studyLooksLikeHead={studyGroups.some(g => studyLooksLikeHeadScan(g.files))}
          />

          {/* Optional email for upload confirmation */}
          <div style={{
            padding: '16px',
            backgroundColor: '#f9fafb',
            border: '1px solid #e5e7eb',
            borderRadius: '8px',
          }}>
            <label
              htmlFor="uploader-email"
              style={{ display: 'block', fontSize: '14px', fontWeight: 500, marginBottom: '6px' }}
            >
              Your email <span style={{ color: '#6b7280', fontWeight: 400 }}>(optional)</span>
            </label>
            <input
              id="uploader-email"
              type="email"
              value={uploaderEmail}
              onChange={e => setUploaderEmail(e.target.value)}
              onBlur={() => setEmailTouched(true)}
              placeholder="you@institution.edu"
              style={{
                width: '100%',
                padding: '8px 12px',
                border: `1px solid ${emailTouched && !emailValid ? '#fca5a5' : '#d1d5db'}`,
                borderRadius: '6px',
                fontSize: '14px',
                boxSizing: 'border-box',
              }}
            />
            {emailTouched && !emailValid ? (
              <p style={{ margin: '6px 0 0', fontSize: '12px', color: '#ea580c' }}>
                Please enter a valid email address.
              </p>
            ) : (
              <p style={{ margin: '6px 0 0', fontSize: '12px', color: '#6b7280' }}>
                You'll receive an email when your study is approved or rejected by the reviewer.
              </p>
            )}
          </div>

          {/* Action buttons */}
          <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
            <button
              onClick={handleReset}
              style={{
                padding: '10px 24px',
                borderRadius: '8px',
                border: '1px solid #d1d5db',
                backgroundColor: '#fff',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: 500,
              }}
            >
              Start over
            </button>
            <button
              onClick={handleUpload}
              disabled={!emailValid || (attributionRequired && !attributionInstitutionId)}
              style={{
                padding: '10px 24px',
                borderRadius: '8px',
                border: 'none',
                backgroundColor: (emailValid && (!attributionRequired || !!attributionInstitutionId)) ? '#2563eb' : '#93c5fd',
                color: '#fff',
                cursor: (emailValid && (!attributionRequired || !!attributionInstitutionId)) ? 'pointer' : 'not-allowed',
                fontSize: '14px',
                fontWeight: 600,
              }}
            >
              {studyGroups.length > 1
                ? `Confirm & upload ${studyGroups.length} studies (${totalFileCount} files, ${formatBytes(totalSize)})`
                : `Confirm anonymization & upload (${totalFileCount} files, ${formatBytes(totalSize)})`
              }
            </button>
          </div>
        </div>
      )}

      {stage === 'uploading' && (
        <div style={{ textAlign: 'center', padding: '48px 24px' }}>
          <p style={{ fontSize: '16px', marginBottom: '8px' }}>
            Anonymizing & uploading... {uploadProgress.current} / {uploadProgress.total}
          </p>
          {studyGroups.length > 1 && (
            <p style={{ fontSize: '13px', color: '#6b7280', marginBottom: '8px' }}>
              Study {currentStudyIndex + 1} of {studyGroups.length}
            </p>
          )}
          <div style={{
            height: '8px',
            backgroundColor: '#e5e7eb',
            borderRadius: '4px',
            overflow: 'hidden',
            maxWidth: '400px',
            margin: '0 auto',
          }}>
            <div style={{
              height: '100%',
              width: `${uploadProgress.total ? (uploadProgress.current / uploadProgress.total) * 100 : 0}%`,
              backgroundColor: '#2563eb',
              borderRadius: '4px',
              transition: 'width 0.2s ease',
            }} />
          </div>
          {currentFile && (
            <p className="upload-current-file">
              {currentFile.length > 48 ? '\u2026' + currentFile.slice(-46) : currentFile}
            </p>
          )}
          {heavyMode.pixelScrub && (
            <div style={{ maxWidth: '500px', margin: '16px auto 0' }}>
              <PixelScrubProgress state={scrubProgress} />
            </div>
          )}
          {bulkProgress && bulkProgress.studies.length > 1 && (
            <div style={{ maxWidth: '700px', margin: '16px auto 0', textAlign: 'left' }}>
              <BulkStudyTable progress={bulkProgress} />
            </div>
          )}
          <button
            onClick={handleCancelUpload}
            style={{
              marginTop: '16px',
              padding: '8px 20px',
              borderRadius: '6px',
              border: '1px solid #d1d5db',
              backgroundColor: '#fff',
              cursor: 'pointer',
              fontSize: '13px',
              color: '#6b7280',
            }}
          >
            Cancel
          </button>
        </div>
      )}

      {/* Step 3: Complete */}
      {stage === 'ready' && (
        <div style={{
          textAlign: 'center',
          padding: '48px 24px',
          border: '1px solid #bbf7d0',
          borderRadius: '12px',
          backgroundColor: '#f0fdf4',
        }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>{'\u2705'}</div>
          <h2 style={{ margin: '0 0 8px', fontSize: '20px' }}>
            Upload complete
          </h2>
          <p style={{ color: '#6b7280', marginBottom: '24px' }}>
            {uploadResults.length > 1
              ? `${uploadResults.length} studies (${totalFileCount} files) anonymized and uploaded successfully.`
              : `${totalFileCount} files anonymized and uploaded successfully.`
            }
          </p>
          {uploadResults.map((result, idx) => (
            <p key={result.sessionId} style={{ color: '#4b5563', marginBottom: '8px', fontSize: '14px' }}>
              {uploadResults.length > 1 && `Study ${idx + 1}: `}
              Session: {result.sessionId}
              {result.study?.studyInstanceUid ? ` \u00b7 UID: ${result.study.studyInstanceUid}` : ''}
            </p>
          ))}
          <button
            onClick={handleReset}
            style={{
              marginTop: '16px',
              padding: '10px 24px',
              borderRadius: '8px',
              border: '1px solid #d1d5db',
              backgroundColor: '#fff',
              cursor: 'pointer',
              fontSize: '14px',
            }}
          >
            Upload more files
          </button>
        </div>
      )}
    </div>
  )
}
