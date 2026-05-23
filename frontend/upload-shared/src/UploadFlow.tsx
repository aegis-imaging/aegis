// The whole upload flow as a single drop-in component.
//
// Hosts (upload-portal, desktop, anywhere else we embed the uploader later)
// just render this; the stage machine, parsing, preview UI, heavy-mode
// orchestration, bulk upload, and progress views all live inside.
//
// Host-specific concerns — auth, project list, invite gates, timezone
// preference UI — stay in the host. The host passes their resolved values
// in as props and we run with them.
//
// Public surface (exported from @aegis/upload-shared/index.ts):
//   <UploadFlow projectSlug="default" .../>
//
// State the host can drive from outside:
//   - heavyModeDefaults — initial pixel-scrub / face-deid toggle state
//   - pickFolderOverride — Tauri can use the native folder picker here
//   - retainedTags / keepPrivateTags — from the project's anon profile
//   - attributionInstitutionId / attributionRequired — researcher rules

import { useCallback, useRef, useState } from 'react'
import {
  bulkUploadStudies,
  buildStudySummary,
  deidentify,
  groupByStudy,
  isDicomFile,
  parseDicomFile,
  studyLooksLikeHeadScan,
  type BulkProgress,
  type DicomTag,
  type FaceDeidResult,
  type ParsedDicomFile,
  type StudySummary as StudySummaryType,
  type UploadOptions,
  type UploadResult,
} from '@aegis/client'
import { FileDropZone } from './components/FileDropZone'
import { StudySummary, type DisplayTimezoneMode } from './components/StudySummary'
import { TagDiffTable } from './components/TagDiffTable'
import { HeavyModeToggle, type HeavyModeSettings } from './components/HeavyModeToggle'
import { PixelScrubProgress, type PixelScrubProgressState } from './components/PixelScrubProgress'
import { BulkStudyTable } from './components/BulkStudyTable'

export type UploadFlowStage = 'select' | 'parsing' | 'preview' | 'uploading' | 'ready'

export interface UploadFlowProps {
  /** Project slug studies will be uploaded into. Required. */
  projectSlug: string

  /** Base URL of the AEGIS API. Default '' (same origin). */
  apiBaseUrl?: string

  /** Bearer token for cross-origin auth (desktop). Web typically omits this and relies on session cookies. */
  authorization?: string

  /** Pre-fill the optional uploader email. */
  initialUploaderEmail?: string

  /** Show the email input (default true). Desktop sometimes hides it because the user is logged in. */
  showEmailInput?: boolean

  /** Institution UUID for researcher attribution. */
  attributionInstitutionId?: string | null

  /** Display name of the institution (shown in the attribution notice). */
  attributionInstitutionName?: string | null

  /** Set true when the current user *must* have an institution selected before upload. */
  attributionRequired?: boolean

  /** Optional informational message shown in the select/preview stages above the dropzone. */
  attributionHint?: string

  /** Retained DICOM tags (from the project's active anonymization profile). */
  retainedTags?: string[]
  /** Keep all private tags (from the project's active anonymization profile). */
  keepPrivateTags?: boolean

  /** Timezone display mode for the StudySummary block. */
  displayTimezoneMode?: DisplayTimezoneMode
  /** IANA tz name when displayTimezoneMode === 'custom'. */
  displayTimezoneCustom?: string

  /** Initial heavy-mode toggle state. */
  heavyModeDefaults?: HeavyModeSettings

  /**
   * Override the file picker. Default behavior uses the HTML
   * <input type=file webkitdirectory> click. Desktop hosts override this to
   * use Tauri's native folder dialog and read files via the fs plugin.
   */
  pickFolderOverride?: () => Promise<File[]>

  /** Concurrency for the bulk upload worker pool (1-8). Default 2. */
  defaultConcurrency?: number

  /** Called when the user clicks "Confirm & upload" and the upload kicks off. */
  onUploadStart?: () => void

  /** Called when every study finishes (success / dup / fail). Host typically refreshes its lists. */
  onCompleted?: (results: UploadResult[]) => void

  /** Called when the user cancels mid-upload. */
  onCancelled?: () => void
}

interface StudyGroup {
  uid: string
  files: ParsedDicomFile[]
  summary: StudySummaryType
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function UploadFlow(props: UploadFlowProps) {
  const {
    projectSlug,
    apiBaseUrl = '',
    authorization,
    initialUploaderEmail = '',
    showEmailInput = true,
    attributionInstitutionId,
    attributionInstitutionName,
    attributionRequired = false,
    attributionHint,
    retainedTags,
    keepPrivateTags,
    displayTimezoneMode = 'utc',
    displayTimezoneCustom = '',
    heavyModeDefaults,
    pickFolderOverride,
    defaultConcurrency = 2,
    onUploadStart,
    onCompleted,
    onCancelled,
  } = props

  // ── Stage state ───────────────────────────────────────────────────────
  const [stage, setStage] = useState<UploadFlowStage>('select')
  const [files, setFiles] = useState<ParsedDicomFile[]>([])
  const [studyGroups, setStudyGroups] = useState<StudyGroup[]>([])
  const [tagChanges, setTagChanges] = useState<DicomTag[]>([])
  const [privateTagsRemoved, setPrivateTagsRemoved] = useState(0)
  const [parseProgress, setParseProgress] = useState({ current: 0, total: 0 })
  const [uploadProgress, setUploadProgress] = useState({ current: 0, total: 0 })
  const [uploadResults, setUploadResults] = useState<UploadResult[]>([])
  const [uploaderEmail, setUploaderEmail] = useState(initialUploaderEmail)
  const [emailTouched, setEmailTouched] = useState(false)
  const [currentFile, setCurrentFile] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [totalSize, setTotalSize] = useState(0)

  // Heavy mode + bulk progress.
  const [heavyMode, setHeavyMode] = useState<HeavyModeSettings>(
    heavyModeDefaults ?? { pixelScrub: false, faceDeid: false }
  )
  const [scrubProgress, setScrubProgress] = useState<PixelScrubProgressState>({
    currentFileIndex: 0, totalFiles: 0, currentFileName: '', results: [],
  })
  const [, setFaceDeidResults] = useState<{ studyUid: string; result: FaceDeidResult }[]>([])
  const [bulkProgress, setBulkProgress] = useState<BulkProgress | null>(null)
  const [bulkConcurrency, setBulkConcurrency] = useState(defaultConcurrency)

  const cancelParseRef = useRef(false)
  const uploadAbortRef = useRef<AbortController | null>(null)

  const emailValid = uploaderEmail === '' || EMAIL_RE.test(uploaderEmail)

  // ── File-selected handler ────────────────────────────────────────────
  const handleFilesSelected = useCallback(async (selectedFiles: File[]) => {
    setError(null)
    setStage('parsing')
    cancelParseRef.current = false

    try {
      const dicomChecks = await Promise.all(
        selectedFiles.map(async f => ({ file: f, isDicom: await isDicomFile(f) }))
      )
      const dicomFiles = dicomChecks.filter(c => c.isDicom).map(c => c.file)

      if (dicomFiles.length === 0) {
        setError('No valid DICOM files found. Files must be DICOM Part 10 format.')
        setStage('select')
        return
      }

      const size = dicomFiles.reduce((acc, f) => acc + f.size, 0)
      setTotalSize(size)
      setParseProgress({ current: 0, total: dicomFiles.length })

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
          // Skip unparseable; the user sees the final count.
          // eslint-disable-next-line no-console
          console.warn(`Skipping ${dicomFiles[i].name}: ${parseErr}`)
        }
        setParseProgress({ current: i + 1, total: dicomFiles.length })
      }
      if (cancelParseRef.current) { setStage('select'); return }
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

      // Run de-id preview on the first file to populate the diff table.
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
    onCancelled?.()
  }, [onCancelled])

  const handleReset = useCallback(() => {
    setStage('select')
    setFiles([])
    setStudyGroups([])
    setTagChanges([])
    setPrivateTagsRemoved(0)
    setUploadProgress({ current: 0, total: 0 })
    setUploadResults([])
    setUploaderEmail(initialUploaderEmail)
    setEmailTouched(false)
    setCurrentFile('')
    setError(null)
    setTotalSize(0)
    setBulkProgress(null)
    cancelParseRef.current = false
    uploadAbortRef.current = null
  }, [initialUploaderEmail])

  // ── Upload handler ───────────────────────────────────────────────────
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
    onUploadStart?.()
    const totalFiles = files.length
    setUploadProgress({ current: 0, total: totalFiles })

    const abortController = new AbortController()
    uploadAbortRef.current = abortController

    try {
      if (heavyMode.pixelScrub) {
        setScrubProgress({ currentFileIndex: 0, totalFiles, currentFileName: '', results: [] })
      }
      setBulkProgress(null)

      const uploadOpts: UploadOptions = {
        apiBaseUrl,
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
      }
      if (authorization) {
        // Pass-through for cross-origin (desktop). uploadStudy doesn't read
        // this directly, but the desktop's apiBaseUrl-based requests need
        // the same auth header — left as a TODO since the underlying
        // client doesn't take a per-call Authorization yet. The current
        // shared library uses session cookies same-origin.
      }

      const bulkResult = await bulkUploadStudies(
        studyGroups.map(g => ({
          studyInstanceUid: g.uid,
          files: g.files,
          summary: g.summary,
        })),
        {
          projectSlug,
          concurrency: bulkConcurrency,
          signal: abortController.signal,
          onProgress: (p) => {
            setBulkProgress({ ...p, studies: p.studies.map(s => ({ ...s })) })
            const completed = p.studies.reduce(
              (n, s) => n + (s.status === 'completed' || s.status === 'duplicate' ? s.fileCount : 0),
              0,
            )
            setUploadProgress({ current: completed, total: totalFiles })
          },
          uploadOptions: uploadOpts,
        }
      )

      const results: UploadResult[] = bulkResult.studies
        .filter(s => s.uploadResult)
        .map(s => s.uploadResult!)
      setUploadResults(results)
      setStage('ready')
      onCompleted?.(results)
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
  }, [
    studyGroups, files, projectSlug, uploaderEmail, attributionRequired,
    attributionInstitutionId, heavyMode, bulkConcurrency, retainedTags,
    keepPrivateTags, apiBaseUrl, authorization, onUploadStart, onCompleted,
  ])

  const totalFileCount = files.length
  const project = projectSlug

  // ── Render ────────────────────────────────────────────────────────────

  return (
    <div className="aegis-upload-flow" style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
      {error && (
        <div style={{
          padding: '12px 16px', borderRadius: 8,
          background: '#fee2e2', border: '1px solid #fca5a5',
          color: '#b91c1c', fontSize: 14,
        }}>
          {error}
        </div>
      )}

      {stage === 'select' && (
        <>
          {attributionHint && <AttributionNotice required={attributionRequired} text={attributionHint} />}
          <FileDropZone onFilesSelected={async () => {
            // Default click → host's HTML picker; override goes through pickFolderOverride.
            if (pickFolderOverride) {
              try {
                const files = await pickFolderOverride()
                if (files.length > 0) await handleFilesSelected(files)
              } catch (e) {
                setError(e instanceof Error ? e.message : 'File picker failed')
              }
            }
          }} />
          {!pickFolderOverride && (
            // FileDropZone manages its own picker click in default mode.
            <FileDropZone onFilesSelected={handleFilesSelected} />
          )}
        </>
      )}

      {stage === 'parsing' && (
        <ParsingView
          progress={parseProgress}
          totalSize={totalSize}
          onCancel={handleCancelParse}
        />
      )}

      {stage === 'preview' && studyGroups.length > 0 && (
        <PreviewView
          studyGroups={studyGroups}
          totalFileCount={totalFileCount}
          totalSize={totalSize}
          tagChanges={tagChanges}
          privateTagsRemoved={privateTagsRemoved}
          project={project}
          attributionInstitutionId={attributionInstitutionId}
          attributionInstitutionName={attributionInstitutionName}
          attributionRequired={attributionRequired}
          attributionHint={attributionHint}
          showEmailInput={showEmailInput}
          uploaderEmail={uploaderEmail}
          setUploaderEmail={setUploaderEmail}
          emailTouched={emailTouched}
          setEmailTouched={setEmailTouched}
          emailValid={emailValid}
          heavyMode={heavyMode}
          setHeavyMode={setHeavyMode}
          bulkConcurrency={bulkConcurrency}
          setBulkConcurrency={setBulkConcurrency}
          displayTimezoneMode={displayTimezoneMode}
          displayTimezoneCustom={displayTimezoneCustom}
          onReset={handleReset}
          onUpload={handleUpload}
        />
      )}

      {stage === 'uploading' && (
        <UploadingView
          uploadProgress={uploadProgress}
          currentFile={currentFile}
          studyGroupCount={studyGroups.length}
          heavyMode={heavyMode}
          scrubProgress={scrubProgress}
          bulkProgress={bulkProgress}
          onCancel={handleCancelUpload}
        />
      )}

      {stage === 'ready' && (
        <ReadyView
          uploadResults={uploadResults}
          studyGroupCount={studyGroups.length}
          onReset={handleReset}
        />
      )}
    </div>
  )
}

// ── Sub-views ──────────────────────────────────────────────────────────────

function AttributionNotice({ required, text }: { required: boolean; text: string }) {
  return (
    <div style={{
      padding: '10px 14px', fontSize: 13,
      background: required ? '#fefce8' : '#f8fafc',
      border: `1px solid ${required ? '#fde68a' : '#e2e8f0'}`,
      color: required ? '#92400e' : '#475569',
      borderRadius: 8,
    }}>{text}</div>
  )
}

function ParsingView({ progress, totalSize, onCancel }: {
  progress: { current: number; total: number }
  totalSize: number
  onCancel: () => void
}) {
  return (
    <div style={{ textAlign: 'center', padding: '48px 24px' }}>
      <p style={{ fontSize: 16, marginBottom: 8 }}>
        Parsing DICOM files… {progress.current} / {progress.total}
      </p>
      <p style={{ fontSize: 12, color: '#9ca3af', marginBottom: 16 }}>
        Total size: {formatBytes(totalSize)}
      </p>
      <div style={{
        height: 8, background: '#e5e7eb', borderRadius: 4, overflow: 'hidden',
        maxWidth: 400, margin: '0 auto',
      }}>
        <div style={{
          height: '100%',
          width: `${progress.total ? (progress.current / progress.total) * 100 : 0}%`,
          background: '#2563eb', transition: 'width 0.2s ease',
        }} />
      </div>
      <button
        type="button"
        onClick={onCancel}
        style={{ marginTop: 16, padding: '6px 14px', borderRadius: 6, border: '1px solid #d1d5db', cursor: 'pointer' }}
      >
        Cancel
      </button>
    </div>
  )
}

function PreviewView(props: {
  studyGroups: StudyGroup[]
  totalFileCount: number
  totalSize: number
  tagChanges: DicomTag[]
  privateTagsRemoved: number
  project: string
  attributionInstitutionId?: string | null
  attributionInstitutionName?: string | null
  attributionRequired: boolean
  attributionHint?: string
  showEmailInput: boolean
  uploaderEmail: string
  setUploaderEmail: (s: string) => void
  emailTouched: boolean
  setEmailTouched: (v: boolean) => void
  emailValid: boolean
  heavyMode: HeavyModeSettings
  setHeavyMode: (m: HeavyModeSettings) => void
  bulkConcurrency: number
  setBulkConcurrency: (n: number) => void
  displayTimezoneMode: DisplayTimezoneMode
  displayTimezoneCustom: string
  onReset: () => void
  onUpload: () => void
}) {
  const canUpload = props.emailValid && (!props.attributionRequired || !!props.attributionInstitutionId)
  return (
    <>
      <div style={{
        padding: '12px 16px', backgroundColor: '#eff6ff', border: '1px solid #bfdbfe',
        borderRadius: 8, fontSize: 14, color: '#1e40af',
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
      }}>
        <span>
          {props.studyGroups.length > 1
            ? `${props.studyGroups.length} studies detected · ${props.totalFileCount} files · ${formatBytes(props.totalSize)}`
            : `${props.totalFileCount} files · ${formatBytes(props.totalSize)}`}
        </span>
        <span style={{ fontSize: 12, color: '#3b82f6' }}>Project: {props.project}</span>
      </div>

      {props.attributionHint && (
        <AttributionNotice required={props.attributionRequired} text={
          props.attributionInstitutionId
            ? `Institution attribution for upload: ${props.attributionInstitutionName || props.attributionInstitutionId}`
            : props.attributionHint
        } />
      )}

      {props.studyGroups.map((group, idx) => (
        <div key={group.uid}>
          {props.studyGroups.length > 1 && (
            <h3 style={{ margin: '0 0 8px', fontSize: 14, fontWeight: 600, color: '#374151' }}>
              Study {idx + 1} of {props.studyGroups.length} ({group.files.length} files)
            </h3>
          )}
          <StudySummary
            summary={group.summary}
            displayTimezoneMode={props.displayTimezoneMode}
            displayTimezoneCustom={props.displayTimezoneCustom}
          />
        </div>
      ))}

      <TagDiffTable tags={props.tagChanges} privateTagsRemoved={props.privateTagsRemoved} />

      {props.studyGroups.length > 1 && (
        <div>
          <h3 style={{ margin: '0 0 8px', fontSize: 15, fontWeight: 600, color: '#374151' }}>
            Bulk upload — {props.studyGroups.length} studies detected
          </h3>
          <BulkStudyTable progress={{
            phase: 'uploading',
            totalStudies: props.studyGroups.length,
            studiesCompleted: 0, studiesFailed: 0, active: [],
            studies: props.studyGroups.map(g => ({
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
              value={String(props.bulkConcurrency)}
              onChange={e => props.setBulkConcurrency(Number(e.target.value))}
            >
              <option value="1">1 (sequential, gentle on network)</option>
              <option value="2">2</option>
              <option value="3">3</option>
              <option value="4">4 (fastest, may saturate uplink)</option>
            </select>
          </div>
        </div>
      )}

      <HeavyModeToggle
        value={props.heavyMode}
        onChange={props.setHeavyMode}
        studyCount={props.studyGroups.length}
        fileCount={props.totalFileCount}
        studyLooksLikeHead={props.studyGroups.some(g => studyLooksLikeHeadScan(g.files))}
      />

      {props.showEmailInput && (
        <div style={{
          padding: 16, backgroundColor: '#f9fafb',
          border: '1px solid #e5e7eb', borderRadius: 8,
        }}>
          <label htmlFor="uploader-email" style={{ display: 'block', fontSize: 14, fontWeight: 500, marginBottom: 6 }}>
            Your email <span style={{ color: '#6b7280', fontWeight: 400 }}>(optional)</span>
          </label>
          <input
            id="uploader-email"
            type="email"
            value={props.uploaderEmail}
            onChange={e => props.setUploaderEmail(e.target.value)}
            onBlur={() => props.setEmailTouched(true)}
            placeholder="you@institution.edu"
            style={{
              width: '100%', padding: '8px 12px',
              border: `1px solid ${props.emailTouched && !props.emailValid ? '#fca5a5' : '#d1d5db'}`,
              borderRadius: 6, fontSize: 14, boxSizing: 'border-box',
            }}
          />
          {props.emailTouched && !props.emailValid ? (
            <p style={{ margin: '6px 0 0', fontSize: 12, color: '#ea580c' }}>
              Please enter a valid email address.
            </p>
          ) : (
            <p style={{ margin: '6px 0 0', fontSize: 12, color: '#6b7280' }}>
              You&apos;ll receive an email when your study is approved or rejected by the reviewer.
            </p>
          )}
        </div>
      )}

      <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end' }}>
        <button
          type="button"
          onClick={props.onReset}
          style={{
            padding: '10px 24px', borderRadius: 8, border: '1px solid #d1d5db',
            backgroundColor: '#fff', cursor: 'pointer', fontSize: 14, fontWeight: 500,
          }}
        >
          Start over
        </button>
        <button
          type="button"
          onClick={props.onUpload}
          disabled={!canUpload}
          style={{
            padding: '10px 24px', borderRadius: 8, border: 'none',
            backgroundColor: canUpload ? '#2563eb' : '#93c5fd', color: '#fff',
            cursor: canUpload ? 'pointer' : 'not-allowed', fontSize: 14, fontWeight: 600,
          }}
        >
          {props.studyGroups.length > 1
            ? `Confirm & upload ${props.studyGroups.length} studies (${props.totalFileCount} files, ${formatBytes(props.totalSize)})`
            : `Confirm anonymization & upload (${props.totalFileCount} files, ${formatBytes(props.totalSize)})`}
        </button>
      </div>
    </>
  )
}

function UploadingView(props: {
  uploadProgress: { current: number; total: number }
  currentFile: string
  studyGroupCount: number
  heavyMode: HeavyModeSettings
  scrubProgress: PixelScrubProgressState
  bulkProgress: BulkProgress | null
  onCancel: () => void
}) {
  return (
    <div style={{ textAlign: 'center', padding: '48px 24px' }}>
      <p style={{ fontSize: 16, marginBottom: 8 }}>
        Anonymizing & uploading… {props.uploadProgress.current} / {props.uploadProgress.total}
      </p>
      <div style={{
        height: 8, background: '#e5e7eb', borderRadius: 4, overflow: 'hidden',
        maxWidth: 400, margin: '0 auto',
      }}>
        <div style={{
          height: '100%',
          width: `${props.uploadProgress.total ? (props.uploadProgress.current / props.uploadProgress.total) * 100 : 0}%`,
          background: '#2563eb', transition: 'width 0.2s ease',
        }} />
      </div>
      {props.currentFile && (
        <p style={{ marginTop: 8, fontSize: 12, color: '#9ca3af', fontFamily: 'monospace' }}>
          {props.currentFile.length > 48 ? '…' + props.currentFile.slice(-46) : props.currentFile}
        </p>
      )}
      {props.heavyMode.pixelScrub && (
        <div style={{ maxWidth: 500, margin: '16px auto 0' }}>
          <PixelScrubProgress state={props.scrubProgress} />
        </div>
      )}
      {props.bulkProgress && props.bulkProgress.studies.length > 1 && (
        <div style={{ maxWidth: 700, margin: '16px auto 0', textAlign: 'left' }}>
          <BulkStudyTable progress={props.bulkProgress} />
        </div>
      )}
      <button
        type="button"
        onClick={props.onCancel}
        style={{ marginTop: 16, padding: '6px 14px', borderRadius: 6, border: '1px solid #d1d5db', cursor: 'pointer' }}
      >
        Cancel
      </button>
    </div>
  )
}

function ReadyView({ uploadResults, studyGroupCount, onReset }: {
  uploadResults: UploadResult[]
  studyGroupCount: number
  onReset: () => void
}) {
  return (
    <div style={{ textAlign: 'center', padding: '48px 24px' }}>
      <div style={{ fontSize: 48, marginBottom: 16 }}>✓</div>
      <h2 style={{ margin: '0 0 12px', fontSize: 22 }}>Upload complete</h2>
      <p style={{ margin: 0, color: '#6b7280' }}>
        {uploadResults.length} of {studyGroupCount} studies uploaded successfully.
      </p>
      <button
        type="button"
        onClick={onReset}
        style={{
          marginTop: 24, padding: '10px 24px', borderRadius: 8, border: 'none',
          backgroundColor: '#2563eb', color: '#fff', fontSize: 14, fontWeight: 600,
          cursor: 'pointer',
        }}
      >
        Upload more studies
      </button>
    </div>
  )
}

// ── helpers ────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}
