/**
 * Token-level free-text PHI scrubbing for DICOM de-identification.
 *
 * Instead of binary keep/delete decisions on entire fields, this module
 * replaces individual PHI tokens with [REMOVED] while preserving non-PHI
 * tokens. This is critical for MIDI-B benchmark scoring which evaluates
 * both <text_removed> (PHI removed) and <text_retained> (non-PHI preserved).
 */

export interface ScrubContext {
  /** Known patient name (e.g., "DOE^JOHN" or "John Doe") */
  patientName?: string
  /** Known patient ID (e.g., "MRN-12345") */
  patientId?: string
  /** Known referring physician name */
  referringPhysician?: string
  /** Known institution name */
  institutionName?: string
}

export interface ScrubResult {
  /** Scrubbed text with PHI tokens replaced by [REMOVED] */
  text: string
  /** Whether any PHI was found and removed */
  phiFound: boolean
}

const REPLACEMENT = '[REMOVED]'

// ---------- Pattern-based PHI detection ----------

const SSN_PATTERN = /\b\d{3}-\d{2}-\d{4}\b/g
const PHONE_PATTERN = /\b\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b/g
const EMAIL_PATTERN = /\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b/gi
const MRN_PATTERN = /\b(?:MRN|MR#|MED\s*REC)\s*[:#]?\s*\d+\b/gi
const ACCESSION_PATTERN = /\b(?:ACC|ACCESSION)\s*[:#]?\s*[A-Z0-9\-]+\b/gi
const DATE_US_PATTERN = /\b(?:0?[1-9]|1[0-2])[/\-](?:0?[1-9]|[12]\d|3[01])[/\-](?:19|20)?\d{2}\b/g
const DATE_WRITTEN_PATTERN = /\b(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\s+\d{1,2},?\s*\d{4}\b/gi
const DATE_ISO_PATTERN = /\b(?:19|20)\d{2}[/\-](?:0?[1-9]|1[0-2])[/\-](?:0?[1-9]|[12]\d|3[01])\b/g
const STREET_ADDRESS_PATTERN = /\b\d+\s+\w+(?:\s+\w+)?\s+(?:street|avenue|road|boulevard|drive|lane|court|way|place|circle|parkway)\b/gi
const ZIP_WITH_STATE_PATTERN = /\b[A-Z]{2}\s+\d{5}(?:-\d{4})?\b/g
const TITLE_NAME_PATTERN = /\b(?:Dr|Mr|Mrs|Ms|Prof|Rev)\.?\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*/g
const HOSPITAL_PATTERN = /\b\w+(?:\s+\w+)*\s+(?:Hospital|Clinic|Medical\s+Center|Health\s+System|Health\s+Center|Healthcare|University\s+Hospital)\b/gi
const IP_ADDRESS_PATTERN = /\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/g
const URL_PATTERN = /\bhttps?:\/\/[^\s]+/gi
const PATIENT_PREFIX_PATTERN = /\b(?:Patient|Pt)\s*[:#]?\s*[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*/g

// Common medical/radiological terms that should NOT be flagged as person names
const MEDICAL_TERMS = new Set([
  // Anatomy
  'head', 'brain', 'chest', 'abdomen', 'pelvis', 'spine', 'neck', 'shoulder',
  'knee', 'hip', 'ankle', 'wrist', 'elbow', 'foot', 'hand', 'finger', 'toe',
  'lung', 'heart', 'liver', 'kidney', 'pancreas', 'spleen', 'colon', 'rectum',
  'skull', 'femur', 'tibia', 'fibula', 'humerus', 'radius', 'ulna', 'clavicle',
  // Imaging terms
  'axial', 'sagittal', 'coronal', 'oblique', 'scout', 'topogram', 'localizer',
  'contrast', 'gadolinium', 'iodine', 'injection', 'bolus', 'delay',
  'series', 'sequence', 'protocol', 'acquisition', 'reconstruction',
  'slice', 'volume', 'phase', 'dynamic', 'perfusion', 'diffusion',
  // Modality
  'ct', 'mri', 'mr', 'pet', 'spect', 'ultrasound', 'xray', 'mammography',
  'fluoroscopy', 'angiography', 'tomography',
  // Common procedure words
  'with', 'without', 'and', 'the', 'for', 'of', 'in', 'on', 'at', 'to',
  'pre', 'post', 'follow', 'up', 'routine', 'standard', 'stat', 'urgent',
  'bilateral', 'unilateral', 'left', 'right', 'anterior', 'posterior',
  'superior', 'inferior', 'medial', 'lateral', 'proximal', 'distal',
  // Sequence types
  'flair', 'stir', 'fiesta', 'mprage', 'space', 'blade', 'propeller',
  'dwi', 'adc', 'swi', 'tof', 'bold', 'dti', 'epi',
  't1w', 't2w', 't1', 't2', 'pd', 'ir',
  // Common findings/descriptions
  'normal', 'abnormal', 'acute', 'chronic', 'benign', 'malignant',
  'fracture', 'lesion', 'mass', 'nodule', 'cyst', 'stenosis', 'occlusion',
])

/** All pattern-based detections */
const PHI_PATTERNS: RegExp[] = [
  SSN_PATTERN,
  PHONE_PATTERN,
  EMAIL_PATTERN,
  MRN_PATTERN,
  ACCESSION_PATTERN,
  DATE_US_PATTERN,
  DATE_WRITTEN_PATTERN,
  DATE_ISO_PATTERN,
  STREET_ADDRESS_PATTERN,
  ZIP_WITH_STATE_PATTERN,
  TITLE_NAME_PATTERN,
  HOSPITAL_PATTERN,
  IP_ADDRESS_PATTERN,
  URL_PATTERN,
  PATIENT_PREFIX_PATTERN,
]

/**
 * Normalize a DICOM name (e.g., "DOE^JOHN^^MD") into tokens suitable for matching.
 * Splits on ^, whitespace, commas; returns lowercase tokens.
 */
function nameToTokens(name: string): string[] {
  return name
    .split(/[\^,\s]+/)
    .map(t => t.trim().toLowerCase())
    .filter(t => t.length > 1) // skip single-char initials
}

/**
 * Check if a word is a known medical term (case-insensitive).
 */
function isMedicalTerm(word: string): boolean {
  return MEDICAL_TERMS.has(word.toLowerCase())
}

/**
 * Scrub PHI tokens from free-text while preserving non-PHI content.
 *
 * Strategy:
 *   1. First, check for exact context matches (patient name, ID, institution)
 *   2. Then apply regex patterns for common PHI types
 *   3. Replace each matched span with [REMOVED]
 *   4. Preserve all non-matched text
 */
export function scrubFreeText(value: string, context: ScrubContext = {}): ScrubResult {
  if (!value || typeof value !== 'string') {
    return { text: value ?? '', phiFound: false }
  }

  // Collect all spans to redact as [start, end] pairs
  const spans: Array<[number, number]> = []

  // --- Context-based matching ---

  // Patient name tokens
  if (context.patientName) {
    const tokens = nameToTokens(context.patientName)
    for (const token of tokens) {
      if (isMedicalTerm(token)) continue
      const regex = new RegExp(`\\b${escapeRegex(token)}\\b`, 'gi')
      let m: RegExpExecArray | null
      while ((m = regex.exec(value)) !== null) {
        spans.push([m.index, m.index + m[0].length])
      }
    }
  }

  // Patient ID
  if (context.patientId) {
    const regex = new RegExp(`\\b${escapeRegex(context.patientId)}\\b`, 'gi')
    let m: RegExpExecArray | null
    while ((m = regex.exec(value)) !== null) {
      spans.push([m.index, m.index + m[0].length])
    }
  }

  // Referring physician name tokens
  if (context.referringPhysician) {
    const tokens = nameToTokens(context.referringPhysician)
    for (const token of tokens) {
      if (isMedicalTerm(token)) continue
      const regex = new RegExp(`\\b${escapeRegex(token)}\\b`, 'gi')
      let m: RegExpExecArray | null
      while ((m = regex.exec(value)) !== null) {
        spans.push([m.index, m.index + m[0].length])
      }
    }
  }

  // Institution name
  if (context.institutionName) {
    const regex = new RegExp(`\\b${escapeRegex(context.institutionName)}\\b`, 'gi')
    let m: RegExpExecArray | null
    while ((m = regex.exec(value)) !== null) {
      spans.push([m.index, m.index + m[0].length])
    }
  }

  // --- Pattern-based matching ---
  for (const pattern of PHI_PATTERNS) {
    // Reset lastIndex for global patterns
    const regex = new RegExp(pattern.source, pattern.flags)
    let m: RegExpExecArray | null
    while ((m = regex.exec(value)) !== null) {
      spans.push([m.index, m.index + m[0].length])
      // Prevent infinite loop on zero-length matches
      if (m[0].length === 0) break
    }
  }

  if (spans.length === 0) {
    return { text: value, phiFound: false }
  }

  // Merge overlapping spans
  const merged = mergeSpans(spans)

  // Build scrubbed string
  let result = ''
  let cursor = 0
  for (const [start, end] of merged) {
    result += value.substring(cursor, start)
    result += REPLACEMENT
    cursor = end
  }
  result += value.substring(cursor)

  // Collapse multiple consecutive [REMOVED] tokens with only whitespace between
  result = result.replace(/(\[REMOVED\](?:\s+\[REMOVED\])+)/g, REPLACEMENT)

  return { text: result, phiFound: true }
}

/** Merge overlapping/adjacent [start, end] spans */
function mergeSpans(spans: Array<[number, number]>): Array<[number, number]> {
  if (spans.length === 0) return []
  const sorted = [...spans].sort((a, b) => a[0] - b[0])
  const merged: Array<[number, number]> = [sorted[0]]
  for (let i = 1; i < sorted.length; i++) {
    const last = merged[merged.length - 1]
    if (sorted[i][0] <= last[1]) {
      last[1] = Math.max(last[1], sorted[i][1])
    } else {
      merged.push(sorted[i])
    }
  }
  return merged
}

/** Escape special regex characters in a string */
function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
