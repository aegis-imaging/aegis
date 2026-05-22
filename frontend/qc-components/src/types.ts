// Shared type definitions. Mirrors the Go API JSON shapes so any client can
// consume the same payloads (admin-dashboard, desktop app, scripted users).

export type QCSeverity = 'info' | 'minor' | 'major' | 'critical'
export type QCCategory =
  | 'phi_leak'
  | 'defacing'
  | 'motion'
  | 'protocol'
  | 'coverage'
  | 'metadata'
  | 'other'

export interface QCFinding {
  id: string
  study_id: string
  analyst_id?: string
  analyst_email: string
  analyst_name?: string
  category: QCCategory
  severity: QCSeverity
  body: string
  series_uid?: string
  instance_index?: number
  resolved_at?: string | null
  resolved_by?: string
  resolution_note?: string
  created_at: string
}

export interface QCTriageItem {
  study_id: string
  study_instance_uid: string
  modality: string
  body_part: string
  study_description: string
  project_id: string
  project_name: string
  institution_id?: string
  institution_name?: string
  status: string
  qc_status: string
  qc_assigned_to_id?: string
  qc_assigned_to_email?: string
  qc_assigned_at?: string | null
  qc_review_started_at?: string | null
  qc_review_ended_at?: string | null
  open_finding_count: number
  highest_severity_open?: QCSeverity | ''
  created_at: string
}

export interface AnalystThroughput {
  analyst_id: string
  analyst_email: string
  analyst_name: string
  studies_completed: number
  open_assignments: number
  findings_raised: number
  findings_resolved: number
  avg_review_minutes: number
  median_review_minutes: number
  studies_per_hour: number
}

/** Visual styling for a severity badge, shared across components. */
export function severityStyle(s: QCSeverity | '' | undefined) {
  switch (s) {
    case 'critical': return { color: '#b91c1c', background: '#fee2e2', label: 'critical' }
    case 'major':    return { color: '#9a3412', background: '#ffedd5', label: 'major' }
    case 'minor':    return { color: '#854d0e', background: '#fef3c7', label: 'minor' }
    case 'info':     return { color: '#1e40af', background: '#dbeafe', label: 'info' }
    default:         return { color: '#6b7280', background: '#f3f4f6', label: 'clean' }
  }
}

export const CATEGORY_LABELS: Record<QCCategory, string> = {
  phi_leak: 'PHI leak',
  defacing: 'Defacing',
  motion: 'Motion',
  protocol: 'Protocol',
  coverage: 'Coverage',
  metadata: 'Metadata',
  other: 'Other',
}
