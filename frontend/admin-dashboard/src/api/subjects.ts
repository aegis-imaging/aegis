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
  anon_patient_id?: string | null
  project_id: string
  study_count: number
  latest_study_date?: string
  modalities?: string[]
  demographics?: SubjectDemographics | null
}

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

export const apiListProjects = () => getJSON<Project[]>('/api/projects')

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
