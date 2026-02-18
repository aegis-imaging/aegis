import { parseDicomFile, serializeDataset } from '../dicom/parser'
import { deidentify, type DeidOptions } from '../dicom/deid'
import type { ParsedDicomFile, StudySummary } from '../types'

export interface UploadOptions {
  /** Base URL for the AEGIS API. Defaults to '' (same origin). */
  apiBaseUrl?: string
  /** Called after each file is successfully uploaded. */
  onProgress?: (uploaded: number, total: number) => void
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
    throw new Error(`Upload init failed (${initRes.status})`)
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

    const putRes = await fetch(initData.upload_urls[i], {
      method: 'PUT',
      headers: { 'Content-Type': 'application/dicom' },
      body: deidBytes,
    })

    if (!putRes.ok) {
      throw new Error(`File upload failed at index ${i} (${putRes.status})`)
    }

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
