import { parseDicomFile, serializeDataset } from '../dicom/parser'
import { deidentify, type DeidOptions } from '../dicom/deid'
import { scrubInstance, type PixelScrubOptions, type PixelScrubResult } from '../dicom/pixel_scrub'
import type { ParsedDicomFile, StudySummary } from '../types'

export interface UploadOptions {
  /** Base URL for the AEGIS API. Defaults to '' (same origin). */
  apiBaseUrl?: string
  /** Optional institution UUID for deterministic attribution at upload init. */
  institutionId?: string
  /** Called after each file is successfully uploaded. */
  onProgress?: (uploaded: number, total: number) => void
  /** Called just before each file's upload begins (filename, 0-based index, total). */
  onFileStart?: (filename: string, index: number, total: number) => void
  /** De-identification options (salt, keepPrivateTags). */
  deid?: DeidOptions
  /** Optional email address — uploader receives a confirmation when the study is processed. */
  uploaderEmail?: string
  /**
   * Heavy-mode: run in-browser pixel PHI scrub on every instance between tag
   * de-id and upload. When omitted, no pixel scrub runs (current behavior).
   * Pass `{ enabled: true }` to opt in with defaults.
   */
  pixelScrub?: PixelScrubOptions & {
    enabled?: boolean
    /** Called after each file's scrub finishes, with the per-file result. */
    onResult?: (filename: string, index: number, result: PixelScrubResult) => void
  }
}

export interface UploadResult {
  sessionId: string
  status: string
  study?: {
    studyInstanceUid: string
  }
  /**
   * Per-file pixel-scrub results when heavy mode was enabled. Omitted when
   * pixelScrub.enabled is false/undefined. Index matches `files[]`.
   */
  pixelScrubResults?: PixelScrubResult[]
}

/** Retry a fetch PUT up to maxAttempts times with exponential backoff (1s/2s/4s). */
async function putWithRetry(url: string, body: Uint8Array, maxAttempts = 3): Promise<void> {
  let delay = 1000
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    const res = await fetch(url, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/dicom' },
      body,
    })
    if (res.ok) return
    if (attempt === maxAttempts) {
      throw new Error(`File upload failed after ${maxAttempts} attempts (${res.status})`)
    }
    await new Promise(r => setTimeout(r, delay))
    delay *= 2
  }
}

/**
 * Upload a parsed DICOM study to AEGIS.
 *
 * De-identifies each file client-side (PS3.15 Annex E Basic Profile) and
 * re-serializes to DICOM bytes before uploading to the signed URLs returned
 * by the API. Patient-identifying data never leaves the browser.
 *
 * @param files    Parsed DICOM files from parseDicomFile()
 * @param projectSlug  Target project slug
 * @param summary  Study-level metadata from buildStudySummary()
 * @param options  API base URL, progress callback, de-id options
 */
export async function uploadStudy(
  files: ParsedDicomFile[],
  projectSlug: string,
  summary: StudySummary,
  options: UploadOptions = {}
): Promise<UploadResult> {
  const base = options.apiBaseUrl ?? ''
  const salt = options.deid?.salt ?? projectSlug

  // 1. Initialize the upload session
  const initRes = await fetch(`${base}/api/upload/init`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      project_slug: projectSlug,
      institution_id: options.institutionId,
      file_count: files.length,
      uploader_email: options.uploaderEmail ?? '',
      study_metadata: {
        study_instance_uid: summary.studyInstanceUid,
        modality: summary.modality,
        body_part: summary.bodyPart,
        study_description: summary.studyDescription,
        study_date: summary.studyDate,
        series_count: summary.seriesCount,
        instance_count: summary.imageCount,
      },
    }),
  })

  if (!initRes.ok) {
    let detail = ''
    try {
      const body = await initRes.json() as { error?: string }
      detail = body.error ?? ''
    } catch {
      // fall back to generic status message
    }
    throw new Error(detail || `Upload init failed (${initRes.status})`)
  }

  const initData = (await initRes.json()) as {
    session_id: string
    upload_urls: string[]
  }

  if (!initData.upload_urls || initData.upload_urls.length !== files.length) {
    throw new Error('Upload init returned unexpected URL count')
  }

  // 2. De-identify, optionally pixel-scrub, serialize, and upload each file
  const pixelScrubEnabled = options.pixelScrub?.enabled === true
  const pixelScrubResults: PixelScrubResult[] = []

  for (let i = 0; i < files.length; i++) {
    options.onFileStart?.(files[i].filename, i, files.length)

    // Re-parse from the original buffer to get a fresh mutable dataset + raw meta
    const { dataset, rawMeta } = parseDicomFile(files[i].arrayBuffer, files[i].filename)

    // Apply PS3.15 Annex E Basic Profile de-identification (tag-level)
    const { dataset: deidDataset } = await deidentify(dataset, {
      salt,
      keepPrivateTags: options.deid?.keepPrivateTags,
      retainedTags: options.deid?.retainedTags,
    })

    // Heavy mode: pixel-level PHI scrubbing (in-browser OCR + redaction).
    if (pixelScrubEnabled) {
      const r = await scrubInstance(deidDataset, options.pixelScrub)
      pixelScrubResults.push(r)
      options.pixelScrub?.onResult?.(files[i].filename, i, r)
    }

    // Re-serialize to DICOM bytes preserving the original transfer syntax.
    // When pixel scrub mutated the dataset's PixelData, those changes flow
    // through here unchanged.
    const deidBytes = serializeDataset(rawMeta, deidDataset)

    // Upload with automatic retry (up to 3 attempts, 1s/2s/4s backoff)
    await putWithRetry(initData.upload_urls[i], deidBytes)

    options.onProgress?.(i + 1, files.length)
  }

  // 3. Notify the API that all files have been uploaded
  const completeRes = await fetch(`${base}/api/upload/complete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: initData.session_id }),
  })

  if (!completeRes.ok) {
    throw new Error(`Upload completion failed (${completeRes.status})`)
  }

  const completeData = (await completeRes.json()) as {
    session_id: string
    status: string
    study?: { study_instance_uid: string }
  }

  return {
    sessionId: completeData.session_id,
    status: completeData.status,
    study: completeData.study
      ? { studyInstanceUid: completeData.study.study_instance_uid }
      : undefined,
    pixelScrubResults: pixelScrubEnabled ? pixelScrubResults : undefined,
  }
}
