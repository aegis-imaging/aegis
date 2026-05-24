// Typed clients for the XNAT-style nav endpoints. These are intentionally
// thin: they describe the response shape and centralize fetch options
// (credentials, error handling) so page components stay declarative.

export type Project = {
  id: string
  name: string
  slug: string
  description: string
  archived: boolean
  restricted: boolean
  member_count?: number
  created_at: string
  updated_at: string
}

export type SubjectDemographics = {
  id: string
  subject_id: string
  project_id: string
  sex: string
  age_at_scan?: number
  diagnosis: string
  education_years?: number
  mmse_score?: number
  moca_score?: number
  cdr_global?: number
  apoe_genotype: string
  notes: string
}

export type SubjectAggregate = {
  subject_id: string
  project_id: string
  study_count: number
  latest_study_date?: string
  modalities?: string[]
  demographics?: SubjectDemographics | null
}

// Per-pipeline-stage status string. AEGIS uses the same shape across all
// stages: '' (not yet required), 'pending', 'running' / 'scanning' /
// 'analyzing' (stage-specific in-flight verbs), 'complete', 'partial',
// 'failed'. The status display chooses an icon + color from this string.
export type PipelineStatus = string

export type StudyStub = {
  id: string
  project_id: string
  study_instance_uid: string
  subject_id?: string | null
  modality?: string
  body_part?: string
  study_description?: string
  study_date?: string | null
  status: string
  source: string
  instance_count: number
  created_at: string
  updated_at: string

  // Pipeline stages. Each has a `*_required` boolean and a `*_status`
  // string. Routing rules decide which stages run on each study.
  phi_scan_required?: boolean
  phi_scan_status?: PipelineStatus
  qc_required?: boolean
  qc_status?: PipelineStatus
  bids_required?: boolean
  bids_status?: PipelineStatus
  classification_required?: boolean
  classification_status?: PipelineStatus
  protocol_required?: boolean
  protocol_status?: PipelineStatus
  export_required?: boolean
  export_status?: PipelineStatus
  analytics_required?: boolean
  analytics_status?: PipelineStatus
}

export type ProjectSubjectsResponse = {
  project_id: string
  subjects: SubjectAggregate[]
  total: number
}

export type ProjectSubjectResponse = SubjectAggregate & {
  studies: StudyStub[]
}

export type SearchHit = {
  kind: 'project' | 'subject' | 'study'
  project_id?: string
  subject_id?: string
  study_id?: string
  label: string
  secondary?: string
}

export type SearchResponse = {
  query: string
  projects: SearchHit[]
  subjects: SearchHit[]
  studies: SearchHit[]
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: 'include' })
  if (!res.ok) {
    throw new Error(`${url}: HTTP ${res.status}`)
  }
  return (await res.json()) as T
}

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    // Try to surface the server's error message so the form can show it.
    let detail = `HTTP ${res.status}`
    try {
      const j = await res.json()
      if (j && typeof j === 'object' && 'error' in j && typeof j.error === 'string') {
        detail = j.error
      }
    } catch {
      // body wasn't JSON; fall back to status code
    }
    throw new Error(detail)
  }
  return (await res.json()) as T
}

export const apiListProjects = () => getJSON<Project[]>('/api/projects')

export type CreateProjectRequest = {
  name: string
  slug?: string
  description?: string
}

export const apiCreateProject = (req: CreateProjectRequest) =>
  postJSON<Project>('/api/projects', req)

export const apiGetProject = (projectId: string) =>
  getJSON<Project>(`/api/projects/${encodeURIComponent(projectId)}`)

export const apiListProjectSubjects = (projectId: string) =>
  getJSON<ProjectSubjectsResponse>(
    `/api/projects/${encodeURIComponent(projectId)}/subjects`,
  )

export const apiGetProjectSubject = (projectId: string, subjectId: string) =>
  getJSON<ProjectSubjectResponse>(
    `/api/projects/${encodeURIComponent(projectId)}/subjects/${encodeURIComponent(subjectId)}`,
  )

export const apiSearch = (q: string) =>
  getJSON<SearchResponse>(`/api/search?q=${encodeURIComponent(q)}`)

export type CurrentUser = {
  id: string
  email: string
  name: string
  role: 'admin' | 'viewer' | 'researcher'
}

export const apiGetMe = () => getJSON<CurrentUser>('/api/auth/me')
