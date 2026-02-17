/** Parsed DICOM tag with original and anonymized values */
export interface DicomTag {
  tag: string        // e.g., "00100010"
  keyword: string    // e.g., "PatientName"
  vr: string         // e.g., "PN"
  originalValue: string | undefined
  anonymizedValue: string | undefined
  action: TagAction
}

/** PS3.15 Annex E action codes */
export type TagAction = 'D' | 'Z' | 'X' | 'U' | 'K' | 'C'

/** Summary of a parsed DICOM study */
export interface StudySummary {
  patientName: string
  patientId: string
  studyDate: string
  studyDescription: string
  modality: string
  studyInstanceUid: string
  seriesCount: number
  imageCount: number
  bodyPart: string
}

/** A single parsed DICOM file's metadata */
export interface ParsedDicomFile {
  filename: string
  studyInstanceUid: string
  seriesInstanceUid: string
  sopInstanceUid: string
  modality: string
  seriesDescription: string
  tags: DicomTag[]
  arrayBuffer: ArrayBuffer
}

/** Result from the Web Worker */
export interface WorkerParseResult {
  type: 'progress' | 'file-parsed' | 'complete' | 'error'
  filename?: string
  file?: ParsedDicomFile
  summary?: StudySummary
  progress?: number
  total?: number
  error?: string
}
