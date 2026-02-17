import { BASIC_PROFILE, isPrivateTag } from './tags'
import type { TagAction, DicomTag } from '../types'

/**
 * DICOM de-identification engine implementing PS3.15 Annex E Basic Profile.
 *
 * Operates on a "naturalized" dcmjs dataset (keyword-based access).
 * Returns a list of tag changes for preview and the modified dataset.
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type NaturalizedDataset = Record<string, any>

/** Generate a deterministic UID from an original UID using a project-scoped key */
async function hashUid(originalUid: string, salt: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(salt + originalUid)
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  const hashArray = new Uint8Array(hashBuffer)

  // Convert to DICOM UID format: 2.25.<decimal from first 16 bytes>
  // 2.25 is the UUID-derived UID root
  let decimal = BigInt(0)
  for (let i = 0; i < 16; i++) {
    decimal = (decimal << BigInt(8)) | BigInt(hashArray[i])
  }
  const uid = `2.25.${decimal.toString()}`

  // DICOM UIDs max 64 chars
  return uid.substring(0, 64)
}

/** Check if a string value might contain PHI (simple heuristic) */
function mightContainPhi(value: string): boolean {
  if (!value || typeof value !== 'string') return false

  // Check for patterns that suggest PHI
  const phiPatterns = [
    /\b\d{3}-\d{2}-\d{4}\b/,     // SSN
    /\b\d{3}[-.)]\d{3}[-.)]\d{4}/, // Phone
    /\b[A-Z][a-z]+\s+[A-Z][a-z]+/, // Person name (Title Case)
    /\b\d+\s+\w+\s+(st|ave|rd|blvd|dr|ln|ct)\b/i, // Address
    /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z]{2,}\b/i, // Email
    /\bMRN\s*[:#]?\s*\d+/i,        // Medical record number
  ]

  return phiPatterns.some(pattern => pattern.test(value))
}

export interface DeidResult {
  /** Tag-level diff for preview UI */
  tagChanges: DicomTag[]
  /** Modified dataset ready for export */
  dataset: NaturalizedDataset
  /** Number of private tags removed */
  privateTagsRemoved: number
}

export interface DeidOptions {
  /** Salt for deterministic UID hashing (should be project-scoped) */
  salt?: string
  /** Keep private tags instead of removing them */
  keepPrivateTags?: boolean
}

/**
 * Apply de-identification to a naturalized dcmjs dataset.
 * Returns the modified dataset and a diff of all changes.
 */
export async function deidentify(
  dataset: NaturalizedDataset,
  options: DeidOptions = {}
): Promise<DeidResult> {
  const salt = options.salt || 'aegis-default-salt'
  const tagChanges: DicomTag[] = []
  let privateTagsRemoved = 0

  // Build reverse lookup: keyword -> tag number
  const keywordToTag: Record<string, string> = {}
  for (const [tag, rule] of Object.entries(BASIC_PROFILE)) {
    keywordToTag[rule.keyword] = tag
  }

  // Process tags defined in the basic profile
  for (const [tag, rule] of Object.entries(BASIC_PROFILE)) {
    const keyword = rule.keyword
    const originalValue = dataset[keyword]

    // Skip if tag not present and action isn't 'K'
    if (originalValue === undefined && rule.action !== 'K') continue

    const originalStr = formatValue(originalValue)

    switch (rule.action) {
      case 'K':
        // Keep as-is
        tagChanges.push({
          tag, keyword, vr: '', action: 'K',
          originalValue: originalStr,
          anonymizedValue: originalStr,
        })
        break

      case 'X':
        // Remove entirely
        delete dataset[keyword]
        tagChanges.push({
          tag, keyword, vr: '', action: 'X',
          originalValue: originalStr,
          anonymizedValue: undefined,
        })
        break

      case 'Z':
        // Replace with zero-length value
        dataset[keyword] = ''
        tagChanges.push({
          tag, keyword, vr: '', action: 'Z',
          originalValue: originalStr,
          anonymizedValue: '',
        })
        break

      case 'D':
        // Replace with dummy value
        dataset[keyword] = 'ANONYMIZED'
        tagChanges.push({
          tag, keyword, vr: '', action: 'D',
          originalValue: originalStr,
          anonymizedValue: 'ANONYMIZED',
        })
        break

      case 'U': {
        // Replace UID with deterministic hash
        if (originalValue && typeof originalValue === 'string') {
          const newUid = await hashUid(originalValue, salt)
          dataset[keyword] = newUid
          tagChanges.push({
            tag, keyword, vr: 'UI', action: 'U',
            originalValue: originalStr,
            anonymizedValue: newUid,
          })
        }
        break
      }

      case 'C':
        // Clean: remove if might contain PHI, otherwise keep
        if (mightContainPhi(originalStr || '')) {
          delete dataset[keyword]
          tagChanges.push({
            tag, keyword, vr: '', action: 'C',
            originalValue: originalStr,
            anonymizedValue: undefined,
          })
        } else {
          tagChanges.push({
            tag, keyword, vr: '', action: 'C',
            originalValue: originalStr,
            anonymizedValue: originalStr,
          })
        }
        break
    }
  }

  // Remove all private tags (odd group numbers) unless opted out
  if (!options.keepPrivateTags) {
    const keysToRemove: string[] = []
    for (const key of Object.keys(dataset)) {
      // dcmjs naturalized datasets use keywords, but we also check
      // if any raw tag-format keys exist
      if (typeof key === 'string' && /^[0-9A-Fa-f]{8}$/.test(key) && isPrivateTag(key)) {
        keysToRemove.push(key)
        privateTagsRemoved++
      }
    }
    for (const key of keysToRemove) {
      delete dataset[key]
    }
  }

  // Sort tag changes: modified first, then kept
  tagChanges.sort((a, b) => {
    const order: Record<TagAction, number> = { X: 0, Z: 1, D: 2, U: 3, C: 4, K: 5 }
    return order[a.action] - order[b.action]
  })

  return { tagChanges, dataset, privateTagsRemoved }
}

/** Format a DICOM value to a display string */
function formatValue(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined
  if (typeof value === 'string') return value
  if (typeof value === 'number') return String(value)
  if (Array.isArray(value)) return value.join(' \\ ')
  if (value instanceof ArrayBuffer || value instanceof Uint8Array) return '[Binary Data]'
  if (typeof value === 'object') return '[Sequence]'
  return String(value)
}
