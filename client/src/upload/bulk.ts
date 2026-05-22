/**
 * Bulk upload primitive — shared by every AEGIS client surface (light web,
 * heavy web, desktop app, scripted API consumer).
 *
 * The goal is one implementation that handles the realistic "drag a parent
 * folder with N study subfolders, click Upload, get a summary" workflow:
 *
 *   - Group files by StudyInstanceUID and (optionally) PatientID for the UI
 *   - Stream parsing so a 50-study drag doesn't freeze the tab
 *   - Bounded parallelism — N studies upload concurrently (default 1, lift
 *     to 2-4 for fast links)
 *   - Skip-on-duplicate (server returns 409 → record and move on)
 *   - Per-study progress callback + cancel
 *   - Aggregated summary at the end so the UI can show "47 uploaded, 2
 *     skipped, 1 failed — see details" without losing the per-row state
 *
 * Heavy-mode options (pixelScrub / faceDeid) pass through verbatim, so a
 * bulk upload of 30 studies with face de-id "just works" — though it'll be
 * slow, the user will see the right per-study indicator and won't lose
 * progress on the rest if one study fails.
 */

import { buildStudySummary, groupByStudy, isDicomFile, parseDicomFile } from '../dicom/parser'
import { uploadStudy, type UploadOptions, type UploadResult } from './client'
import type { ParsedDicomFile, StudySummary } from '../types'

export interface BulkStudy {
  /** Stable id (StudyInstanceUID) — used for dedup, retry, and progress. */
  studyInstanceUid: string
  files: ParsedDicomFile[]
  summary: StudySummary
}

export type BulkPhase =
  | 'parsing'           // walking input files, building ParsedDicomFile objects
  | 'grouping'          // partitioning by StudyInstanceUID
  | 'uploading'         // running per-study uploads
  | 'cancelled'         // signal aborted
  | 'done'              // everything finished

export interface BulkProgress {
  phase: BulkPhase
  /** 0-based total once parsing/grouping is done. -1 while still discovering. */
  totalStudies: number
  studiesCompleted: number
  studiesFailed: number
  /** Currently in-flight study UIDs (one entry per concurrency slot). */
  active: { studyInstanceUid: string; description: string; fileCount: number }[]
  /** Detailed per-study state. */
  studies: BulkStudyState[]
}

export type BulkStudyStatus =
  | 'pending'
  | 'uploading'
  | 'completed'
  | 'duplicate'       // server said this study already exists
  | 'failed'
  | 'cancelled'

export interface BulkStudyState {
  studyInstanceUid: string
  studyDescription: string
  patientId: string
  fileCount: number
  status: BulkStudyStatus
  error?: string
  uploadResult?: UploadResult
  /** Wall-clock duration in ms for this study. */
  durationMs?: number
}

export interface BulkUploadOptions {
  /**
   * Project slug — defaults applied at upload-init time.
   * Same on every study; per-study override isn't supported here (use multiple
   * bulkUpload() calls if you need that).
   */
  projectSlug: string
  /**
   * Number of studies to upload concurrently. Default 1 (sequential).
   * 2-4 is reasonable for fast networks; >4 rarely helps because the bottleneck
   * is the cloud's per-session file-upload throughput.
   */
  concurrency?: number
  /**
   * Per-study upload options. Same shape as uploadStudy()'s, applied to every
   * study. The bulk runner wraps callbacks (onProgress, onFileStart, pixelScrub
   * onResult, faceDeid onResult) so callers can still hook them, with the
   * study UID added to the args.
   */
  uploadOptions?: Omit<UploadOptions, 'onProgress' | 'onFileStart'>
  /**
   * Called after every meaningful state transition. UI binds to this for a
   * live progress table.
   */
  onProgress?: (progress: BulkProgress) => void
  /**
   * Called when a study finishes (success, dup, fail). Useful for streaming
   * results to an audit log without waiting for the whole batch.
   */
  onStudyDone?: (study: BulkStudyState) => void
  /**
   * Abort signal — cancels the loop after the currently-in-flight studies
   * settle. We don't yank in-flight HTTP requests; this is graceful.
   */
  signal?: AbortSignal
  /**
   * Filter — return false to skip a discovered study before it uploads.
   * Useful for "exclude any study with fewer than N slices" or
   * "exclude any study where summary.patientId is empty".
   */
  filter?: (study: BulkStudy) => boolean
}

export interface BulkUploadResult {
  studies: BulkStudyState[]
  studiesUploaded: number
  studiesDuplicate: number
  studiesFailed: number
  durationMs: number
}

/**
 * Pre-grouped variant — take an array of already-parsed BulkStudy objects
 * (e.g. from a UI that does its own preview + tag-diff pass) and run the
 * worker-pool upload. Used by the upload-portal which has its own preview
 * stage; the one-shot bulkUpload() below uses this internally.
 */
export async function bulkUploadStudies(
  studies: BulkStudy[],
  options: BulkUploadOptions
): Promise<BulkUploadResult> {
  const startTime = nowMs()
  const onProgress = options.onProgress
  const progress: BulkProgress = {
    phase: 'uploading',
    totalStudies: studies.length,
    studiesCompleted: 0,
    studiesFailed: 0,
    active: [],
    studies: studies.map(s => ({
      studyInstanceUid: s.studyInstanceUid,
      studyDescription: s.summary.studyDescription || '(no description)',
      patientId: s.summary.patientId,
      fileCount: s.files.length,
      status: 'pending' as const,
    })),
  }
  onProgress?.(progress)
  if (studies.length === 0) {
    progress.phase = 'done'
    onProgress?.(progress)
    return finalize(progress, startTime)
  }
  await runWorkerPool(studies, progress, options)
  progress.phase = options.signal?.aborted ? 'cancelled' : 'done'
  onProgress?.(progress)
  return finalize(progress, startTime)
}

/**
 * One-shot: hand in a flat File[] (e.g. from a drag-and-drop), get back a
 * fully-orchestrated bulk upload result. Hides the parse-then-group-then-
 * upload pipeline.
 */
export async function bulkUpload(
  files: File[] | FileList,
  options: BulkUploadOptions
): Promise<BulkUploadResult> {
  const fileList = Array.from(files instanceof FileList ? files : files)

  const startTime = nowMs()
  const onProgress = options.onProgress
  const signal = options.signal

  // ── 1. Parse all files (concurrently, but in slices to avoid blocking) ──
  const progress: BulkProgress = {
    phase: 'parsing',
    totalStudies: -1,
    studiesCompleted: 0,
    studiesFailed: 0,
    active: [],
    studies: [],
  }
  onProgress?.(progress)

  const parsed: ParsedDicomFile[] = []
  for (let i = 0; i < fileList.length; i++) {
    if (signal?.aborted) {
      progress.phase = 'cancelled'
      onProgress?.(progress)
      return finalize(progress, startTime)
    }
    const file = fileList[i]
    if (!(await isDicomFile(file))) continue
    try {
      const buf = await file.arrayBuffer()
      const { parsed: p } = parseDicomFile(buf, file.name)
      parsed.push(p)
    } catch {
      // Skip unparseable files; the user sees the final count.
    }
    // Yield to the event loop every 16 files so the UI stays responsive.
    if (i % 16 === 15) await yieldToBrowser()
  }

  // ── 2. Group + filter ────────────────────────────────────────────────────
  progress.phase = 'grouping'
  onProgress?.(progress)

  const groups = groupByStudy(parsed)
  const studies: BulkStudy[] = []
  for (const [uid, studyFiles] of groups.entries()) {
    const summary = buildStudySummary(studyFiles)
    const candidate: BulkStudy = { studyInstanceUid: uid, files: studyFiles, summary }
    if (options.filter && !options.filter(candidate)) continue
    studies.push(candidate)
  }
  progress.totalStudies = studies.length
  progress.studies = studies.map(s => ({
    studyInstanceUid: s.studyInstanceUid,
    studyDescription: s.summary.studyDescription || '(no description)',
    patientId: s.summary.patientId,
    fileCount: s.files.length,
    status: 'pending',
  }))

  if (studies.length === 0) {
    progress.phase = 'done'
    onProgress?.(progress)
    return finalize(progress, startTime)
  }

  // ── 3. Concurrent per-study upload (shared worker pool) ────────────────
  progress.phase = 'uploading'
  onProgress?.(progress)
  await runWorkerPool(studies, progress, options)
  progress.phase = signal?.aborted ? 'cancelled' : 'done'
  onProgress?.(progress)
  return finalize(progress, startTime)
}

async function runWorkerPool(
  studies: BulkStudy[],
  progress: BulkProgress,
  options: BulkUploadOptions
): Promise<void> {
  const onProgress = options.onProgress
  const signal = options.signal
  const concurrency = clamp(options.concurrency ?? 1, 1, 8)
  let nextIndex = 0
  const lock = makeMutex()

  async function nextWork(): Promise<BulkStudy | null> {
    return lock.run(async () => {
      if (signal?.aborted) return null
      while (nextIndex < studies.length) {
        const s = studies[nextIndex++]
        const state = progress.studies.find(p => p.studyInstanceUid === s.studyInstanceUid)!
        if (state.status !== 'pending') continue
        return s
      }
      return null
    })
  }

  async function worker(slot: number): Promise<void> {
    while (true) {
      const study = await nextWork()
      if (!study) return
      const state = progress.studies.find(s => s.studyInstanceUid === study.studyInstanceUid)!
      state.status = 'uploading'
      progress.active[slot] = {
        studyInstanceUid: study.studyInstanceUid,
        description: state.studyDescription,
        fileCount: state.fileCount,
      }
      onProgress?.(progress)

      const studyStart = nowMs()
      try {
        const r = await uploadStudy(study.files, options.projectSlug, study.summary, options.uploadOptions)
        state.status = 'completed'
        state.uploadResult = r
        state.durationMs = nowMs() - studyStart
        progress.studiesCompleted++
      } catch (e) {
        const msg = (e as Error).message || String(e)
        state.durationMs = nowMs() - studyStart
        if (isDuplicateError(msg)) {
          state.status = 'duplicate'
        } else {
          state.status = 'failed'
          state.error = msg
          progress.studiesFailed++
        }
      } finally {
        progress.active[slot] = { studyInstanceUid: '', description: '', fileCount: 0 }
        options.onStudyDone?.(state)
        onProgress?.(progress)
      }
      if (signal?.aborted) {
        for (const s of progress.studies) {
          if (s.status === 'pending') s.status = 'cancelled'
        }
        return
      }
    }
  }

  progress.active = Array(concurrency).fill(0).map(() =>
    ({ studyInstanceUid: '', description: '', fileCount: 0 }))
  await Promise.all(Array(concurrency).fill(0).map((_, slot) => worker(slot)))
}

// ── helpers ────────────────────────────────────────────────────────────────

function finalize(progress: BulkProgress, startTime: number): BulkUploadResult {
  const studiesUploaded = progress.studies.filter(s => s.status === 'completed').length
  const studiesDuplicate = progress.studies.filter(s => s.status === 'duplicate').length
  const studiesFailed = progress.studies.filter(s => s.status === 'failed').length
  return {
    studies: progress.studies,
    studiesUploaded,
    studiesDuplicate,
    studiesFailed,
    durationMs: nowMs() - startTime,
  }
}

function isDuplicateError(msg: string): boolean {
  return /409|already exists|duplicate/i.test(msg)
}

function clamp(v: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, v))
}

function nowMs(): number {
  return typeof performance !== 'undefined' && typeof performance.now === 'function'
    ? performance.now()
    : Date.now()
}

function yieldToBrowser(): Promise<void> {
  return new Promise(resolve => {
    if (typeof requestAnimationFrame === 'function') {
      requestAnimationFrame(() => resolve())
    } else {
      setTimeout(resolve, 0)
    }
  })
}

interface Mutex {
  run<T>(fn: () => Promise<T>): Promise<T>
}

function makeMutex(): Mutex {
  let chain: Promise<unknown> = Promise.resolve()
  return {
    run<T>(fn: () => Promise<T>): Promise<T> {
      const next = chain.then(() => fn())
      chain = next.catch(() => undefined)
      return next
    },
  }
}
