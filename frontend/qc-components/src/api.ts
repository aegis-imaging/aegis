// Thin fetch wrappers around the QC endpoints. Each function takes an
// optional `baseUrl` so the desktop app can point at a remote AEGIS while
// the admin-dashboard uses same-origin requests.

import type {
  AnalystThroughput,
  QCCategory,
  QCFinding,
  QCSeverity,
  QCTriageItem,
} from './types'

export interface APIClientOptions {
  /** Base URL of the AEGIS API. Default '' (same origin). */
  baseUrl?: string
  /** Auth header value, e.g. 'Bearer <api_key>'. */
  authorization?: string
  /** Custom fetch implementation (test injection). */
  fetch?: typeof fetch
}

export interface TriageFilters {
  assigned?: 'me' | 'all' | string // analyst id or "me"
  projectId?: string
  status?: string[]                 // OR-of: status values
  openOnly?: boolean
  limit?: number
}

export interface CreateFindingInput {
  category: QCCategory
  severity: QCSeverity
  body: string
  seriesUid?: string
  instanceIndex?: number
}

function joinUrl(base: string, path: string): string {
  if (!base) return path
  return base.replace(/\/$/, '') + path
}

function headers(opts?: APIClientOptions): Record<string, string> {
  const h: Record<string, string> = { 'Content-Type': 'application/json' }
  if (opts?.authorization) h.Authorization = opts.authorization
  return h
}

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let detail = ''
    try { detail = (await res.json() as { error?: string }).error ?? '' } catch {}
    throw new Error(detail || `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

export async function listTriage(filters: TriageFilters, opts?: APIClientOptions): Promise<QCTriageItem[]> {
  const params = new URLSearchParams()
  if (filters.assigned) params.set('assigned', filters.assigned)
  if (filters.projectId) params.set('project_id', filters.projectId)
  if (filters.status && filters.status.length > 0) params.set('status', filters.status.join(','))
  if (filters.openOnly) params.set('open', 'true')
  if (filters.limit) params.set('limit', String(filters.limit))
  const url = joinUrl(opts?.baseUrl ?? '', `/api/qc/triage?${params.toString()}`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { headers: headers(opts) })
  const body = await json<{ items: QCTriageItem[] }>(res)
  return body.items
}

export async function getAnalystThroughput(days: number, opts?: APIClientOptions): Promise<AnalystThroughput[]> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/qc/throughput?days=${days}`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { headers: headers(opts) })
  const body = await json<{ analysts: AnalystThroughput[] }>(res)
  return body.analysts
}

export async function listFindings(studyId: string, openOnly: boolean, opts?: APIClientOptions): Promise<QCFinding[]> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/studies/${studyId}/qc-findings${openOnly ? '?open=true' : ''}`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { headers: headers(opts) })
  const body = await json<{ findings: QCFinding[] }>(res)
  return body.findings
}

export async function createFinding(studyId: string, input: CreateFindingInput, opts?: APIClientOptions): Promise<QCFinding> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/studies/${studyId}/qc-findings`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, {
    method: 'POST',
    headers: headers(opts),
    body: JSON.stringify({
      category: input.category,
      severity: input.severity,
      body: input.body,
      series_uid: input.seriesUid,
      instance_index: input.instanceIndex,
    }),
  })
  return json<QCFinding>(res)
}

export async function resolveFinding(findingId: string, note: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/qc-findings/${findingId}/resolve`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, {
    method: 'POST',
    headers: headers(opts),
    body: JSON.stringify({ note }),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function reopenFinding(findingId: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/qc-findings/${findingId}/reopen`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { method: 'POST', headers: headers(opts) })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function deleteFinding(findingId: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/qc-findings/${findingId}`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { method: 'DELETE', headers: headers(opts) })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function assignQC(studyId: string, analystId: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/studies/${studyId}/qc/assign`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, {
    method: 'POST',
    headers: headers(opts),
    body: JSON.stringify({ analyst_id: analystId }),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function unassignQC(studyId: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/studies/${studyId}/qc/assign`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { method: 'DELETE', headers: headers(opts) })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function startQC(studyId: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/studies/${studyId}/qc/start`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { method: 'POST', headers: headers(opts) })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

export async function completeQC(studyId: string, opts?: APIClientOptions): Promise<void> {
  const url = joinUrl(opts?.baseUrl ?? '', `/api/studies/${studyId}/qc/complete`)
  const f = opts?.fetch ?? fetch
  const res = await f(url, { method: 'POST', headers: headers(opts) })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}
