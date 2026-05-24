/**
 * Thin HTTP client for the AEGIS API.
 *
 * Wraps fetch() with the Bearer token from the keyring credential, and exposes
 * the handful of endpoints the desktop uploader needs (projects list +
 * upload init/file/complete). Anonymization happens in JS via @aegis/client;
 * uploadStudy() from that lib is unsuitable here because (a) it cannot inject
 * Authorization headers and (b) it pulls file bytes from a browser File.
 */

export interface Project {
  id: string
  name: string
  slug: string
  description?: string
}

export interface UploadInitResponse {
  session_id: string
  upload_urls: string[]
}

export interface UploadCompleteResponse {
  session_id: string
  status: string
  study?: { study_instance_uid: string }
}

export interface UploadInitInput {
  projectSlug: string
  fileCount: number
  uploaderEmail?: string
  studyMetadata: {
    studyInstanceUid: string
    modality: string
    bodyPart: string
    studyDescription: string
    studyDate: string
    seriesCount: number
    imageCount: number
  }
}

export class AegisApi {
  private readonly serverUrl: string
  private readonly apiKey: string

  constructor(serverUrl: string, apiKey: string) {
    this.serverUrl = serverUrl.replace(/\/+$/, '')
    this.apiKey = apiKey
  }

  private url(path: string): string {
    return `${this.serverUrl}${path.startsWith('/') ? '' : '/'}${path}`
  }

  private authHeaders(extra: Record<string, string> = {}): Record<string, string> {
    return {
      Authorization: `Bearer ${this.apiKey}`,
      ...extra,
    }
  }

  async listProjects(): Promise<Project[]> {
    const res = await fetch(this.url('/api/projects'), {
      method: 'GET',
      headers: this.authHeaders(),
    })
    if (!res.ok) {
      throw new Error(`List projects failed (${res.status})`)
    }
    const body = await res.json()
    // The API returns either { projects: [...] } or a bare array depending on
    // version. Accept both.
    if (Array.isArray(body)) return body as Project[]
    if (Array.isArray(body?.projects)) return body.projects as Project[]
    return []
  }

  async uploadInit(input: UploadInitInput): Promise<UploadInitResponse> {
    const res = await fetch(this.url('/api/upload/init'), {
      method: 'POST',
      headers: this.authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({
        project_slug: input.projectSlug,
        file_count: input.fileCount,
        uploader_email: input.uploaderEmail ?? '',
        study_metadata: {
          study_instance_uid: input.studyMetadata.studyInstanceUid,
          modality: input.studyMetadata.modality,
          body_part: input.studyMetadata.bodyPart,
          study_description: input.studyMetadata.studyDescription,
          study_date: input.studyMetadata.studyDate,
          series_count: input.studyMetadata.seriesCount,
          instance_count: input.studyMetadata.imageCount,
        },
      }),
    })
    if (!res.ok) {
      let detail = ''
      try {
        const body = await res.json()
        detail = (body as { error?: string }).error ?? ''
      } catch {
        // ignore
      }
      throw new Error(detail || `Upload init failed (${res.status})`)
    }
    return (await res.json()) as UploadInitResponse
  }

  /**
   * PUT a single file to a signed upload URL. Retries 3x with 1s/2s/4s
   * backoff before surfacing the error. Signed URLs typically don't need
   * the Bearer token, but the API also accepts authenticated PUTs to its
   * own /api/upload/file/{session}/{index} fallback path, so we include
   * the header for the fallback case.
   */
  async uploadFile(url: string, bytes: Uint8Array): Promise<void> {
    let delay = 1000
    const maxAttempts = 3
    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      const isSigned = !url.startsWith(this.serverUrl)
      const headers: Record<string, string> = { 'Content-Type': 'application/dicom' }
      if (!isSigned) {
        headers.Authorization = `Bearer ${this.apiKey}`
      }
      const res = await fetch(url, {
        method: 'PUT',
        headers,
        body: bytes,
      })
      if (res.ok) return
      if (attempt === maxAttempts) {
        throw new Error(`File upload failed after ${maxAttempts} attempts (${res.status})`)
      }
      await new Promise(r => setTimeout(r, delay))
      delay *= 2
    }
  }

  async uploadComplete(sessionId: string): Promise<UploadCompleteResponse> {
    const res = await fetch(this.url('/api/upload/complete'), {
      method: 'POST',
      headers: this.authHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ session_id: sessionId }),
    })
    if (!res.ok) {
      throw new Error(`Upload completion failed (${res.status})`)
    }
    return (await res.json()) as UploadCompleteResponse
  }
}
