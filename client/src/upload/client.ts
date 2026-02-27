import { parseDicomFile, serializeDataset } from '../dicom/parser'
import { deidentify, type DeidOptions } from '../dicom/deid'
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
}

export interface UploadResult {
  sessionId: string
  status: string
  study?: {
    studyInstanceUid: string
  }
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

  // 2. De-identify, serialize, and upload each file
  for (let i = 0; i < files.length; i++) {
    options.onFileStart?.(files[i].filename, i, files.length)

    // Re-parse from the original buffer to get a fresh mutable dataset + raw meta
    const { dataset, rawMeta } = parseDicomFile(files[i].arrayBuffer, files[i].filename)

    // Apply PS3.15 Annex E Basic Profile de-identification
    const { dataset: deidDataset } = await deidentify(dataset, {
      salt,
      keepPrivateTags: options.deid?.keepPrivateTags,
      retainedTags: options.deid?.retainedTags,
    })

    // Re-serialize to DICOM bytes preserving the original transfer syntax
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
  }
}
