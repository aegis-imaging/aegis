import { BASIC_PROFILE, isPrivateTag } from './tags'
import { DATE_TAGS, TIME_TAGS, shiftDicomDate } from './dateshift'
import { scrubFreeText } from './text_scrub'
import type { ScrubContext } from './text_scrub'
import type { TagAction, DicomTag } from '../types'
import type { NaturalizedDataset } from './parser'

/**
 * DICOM de-identification engine implementing PS3.15 Annex E Basic Profile.
 *
 * Operates on a "naturalized" dcmjs dataset (keyword-based access).
 * Returns a list of tag changes for preview and the modified dataset.
 */

/** Generate a deterministic UID from an original UID using a project-scoped salt */
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

/** Generate a deterministic pseudonym from an identifier using SHA-256 */
async function hashIdentifier(original: string, salt: string, prefix: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(salt + original)
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  const hashArray = new Uint8Array(hashBuffer)
  // Take first 4 bytes → 8 hex chars
  const hex = Array.from(hashArray.slice(0, 4))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
  return `${prefix}${hex}`
}


export interface DeidResult {
  /** Tag-level diff for preview UI */
  tagChanges: DicomTag[]
  /** Modified dataset ready for re-serialization */
  dataset: NaturalizedDataset
  /** Number of private tags removed */
  privateTagsRemoved: number
  /** Original → replacement UID mappings from this file */
  uidMappings: Map<string, string>
  /** Original PatientID → pseudonym mapping (if PatientID was present) */
  patientIdMapping?: { original: string; replacement: string }
  /** Date shift offset in days (if date shifting was applied) */
  dateShiftOffset?: number
}

export interface DeidOptions {
  /** Salt for deterministic UID hashing (should be project-scoped) */
  salt?: string
  /** Keep private tags instead of removing them */
  keepPrivateTags?: boolean
  /** DICOM keyword names to retain as-is (override Basic Profile strip/zero actions) */
  retainedTags?: string[]
  /** Date shifting: if provided, dates are shifted instead of removed/zeroed */
  dateShift?: {
    /** Day offset to apply (positive = shift forward, negative = shift back) */
    offsetDays: number
  }
}

/**
 * Apply de-identification to a naturalized dcmjs dataset in place.
 * Returns the modified dataset and a diff of all changes for UI preview.
 */
export async function deidentify(
  dataset: NaturalizedDataset,
  options: DeidOptions = {}
): Promise<DeidResult> {
  const salt = options.salt || 'aegis-default-salt'
  const tagChanges: DicomTag[] = []
  let privateTagsRemoved = 0
  const uidMappings = new Map<string, string>()
  let patientIdMapping: { original: string; replacement: string } | undefined

  // Build scrub context from dataset for token-level PHI scrubbing in C-action fields
  const scrubContext: ScrubContext = {
    patientName: formatValue(dataset.PatientName) ?? undefined,
    patientId: formatValue(dataset.PatientID) ?? undefined,
    referringPhysician: formatValue(dataset.ReferringPhysicianName) ?? undefined,
    institutionName: formatValue(dataset.InstitutionName) ?? undefined,
  }

  for (const [tag, rule] of Object.entries(BASIC_PROFILE)) {
    const keyword = rule.keyword
    const originalValue = dataset[keyword]

    if (originalValue === undefined && rule.action !== 'K') continue

    const originalStr = formatValue(originalValue)

    // If the tag is in the project's retained list, treat it as Keep regardless
    // of the Basic Profile action.
    if (options.retainedTags?.includes(keyword)) {
      tagChanges.push({
        tag, keyword, vr: '', action: 'K',
        originalValue: originalStr,
        anonymizedValue: originalStr,
      })
      continue
    }

    // Date shifting: when enabled, shift date tags instead of zeroing/removing
    if (options.dateShift && DATE_TAGS.has(tag) && originalStr) {
      const shifted = shiftDicomDate(originalStr, options.dateShift.offsetDays)
      dataset[keyword] = shifted
      tagChanges.push({
        tag, keyword, vr: 'DA', action: 'Z',
        originalValue: originalStr,
        anonymizedValue: shifted,
      })
      continue
    }

    // Time tags associated with date tags: keep unchanged when date shifting
    if (options.dateShift && TIME_TAGS.has(tag)) {
      tagChanges.push({
        tag, keyword, vr: 'TM', action: 'K',
        originalValue: originalStr,
        anonymizedValue: originalStr,
      })
      continue
    }

    switch (rule.action) {
      case 'K':
        tagChanges.push({
          tag, keyword, vr: '', action: 'K',
          originalValue: originalStr,
          anonymizedValue: originalStr,
        })
        break

      case 'X':
        delete dataset[keyword]
        tagChanges.push({
          tag, keyword, vr: '', action: 'X',
          originalValue: originalStr,
          anonymizedValue: undefined,
        })
        break

      case 'Z': {
        // PatientID and PatientName get deterministic pseudonyms instead of empty
        if (tag === '00100020' && originalStr) {
          const pseudonym = await hashIdentifier(originalStr, salt, 'SUBJ-')
          dataset[keyword] = pseudonym
          patientIdMapping = { original: originalStr, replacement: pseudonym }
          tagChanges.push({
            tag, keyword, vr: '', action: 'Z',
            originalValue: originalStr,
            anonymizedValue: pseudonym,
          })
        } else if (tag === '00100010' && originalStr) {
          const pseudonym = await hashIdentifier(originalStr, salt, 'ANON-')
          dataset[keyword] = pseudonym
          tagChanges.push({
            tag, keyword, vr: 'PN', action: 'Z',
            originalValue: originalStr,
            anonymizedValue: pseudonym,
          })
        } else {
          dataset[keyword] = ''
          tagChanges.push({
            tag, keyword, vr: '', action: 'Z',
            originalValue: originalStr,
            anonymizedValue: '',
          })
        }
        break
      }

      case 'D':
        dataset[keyword] = 'ANONYMIZED'
        tagChanges.push({
          tag, keyword, vr: '', action: 'D',
          originalValue: originalStr,
          anonymizedValue: 'ANONYMIZED',
        })
        break

      case 'U': {
        if (originalValue && typeof originalValue === 'string') {
          const newUid = await hashUid(originalValue, salt)
          dataset[keyword] = newUid
          uidMappings.set(originalValue, newUid)
          tagChanges.push({
            tag, keyword, vr: 'UI', action: 'U',
            originalValue: originalStr,
            anonymizedValue: newUid,
          })
        }
        break
      }

      case 'C': {
        const scrubResult = scrubFreeText(originalStr || '', scrubContext)
        if (scrubResult.phiFound) {
          dataset[keyword] = scrubResult.text
          tagChanges.push({
            tag, keyword, vr: '', action: 'C',
            originalValue: originalStr,
            anonymizedValue: scrubResult.text,
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
  }

  // Recursively process sequences (SR ContentSequence, nested UIDs, etc.)
  await processSequences(dataset, scrubContext, options, salt, uidMappings)

  // Remove all private tags (odd group numbers) unless opted out
  if (!options.keepPrivateTags) {
    const keysToRemove: string[] = []
    for (const key of Object.keys(dataset)) {
      if (typeof key === 'string' && /^[0-9A-Fa-f]{8}$/.test(key) && isPrivateTag(key)) {
        keysToRemove.push(key)
        privateTagsRemoved++
      }
    }
    for (const key of keysToRemove) {
      delete dataset[key]
    }
  }

  // Sort: modified tags first, kept tags last
  tagChanges.sort((a, b) => {
    const order: Record<TagAction, number> = { X: 0, Z: 1, D: 2, U: 3, C: 4, K: 5 }
    return order[a.action] - order[b.action]
  })

  return {
    tagChanges,
    dataset,
    privateTagsRemoved,
    uidMappings,
    patientIdMapping,
    dateShiftOffset: options.dateShift?.offsetDays,
  }
}

/**
 * Recursively process DICOM sequences for de-identification.
 * Handles SR ContentSequence trees and any other nested sequence data.
 *
 * In dcmjs naturalized datasets, sequences are arrays of plain objects.
 * Each object is a sequence item (nested dataset).
 */
async function processSequences(
  dataset: NaturalizedDataset,
  scrubContext: ScrubContext,
  options: DeidOptions,
  salt: string,
  uidMappings: Map<string, string>
): Promise<void> {
  for (const key of Object.keys(dataset)) {
    if (key === '_meta' || key === 'PixelData') continue
    const value = dataset[key]
    if (!Array.isArray(value) || value.length === 0) continue
    if (typeof value[0] !== 'object' || value[0] === null) continue

    // Array of objects → sequence items
    for (const item of value) {
      await processSequenceItem(item as Record<string, unknown>, scrubContext, options, salt, uidMappings)
    }
  }
}

async function processSequenceItem(
  item: Record<string, unknown>,
  scrubContext: ScrubContext,
  options: DeidOptions,
  salt: string,
  uidMappings: Map<string, string>
): Promise<void> {
  for (const [key, value] of Object.entries(item)) {
    if (value === undefined || value === null) continue

    // Recurse into nested sequences (array of objects)
    if (Array.isArray(value) && value.length > 0 &&
        typeof value[0] === 'object' && value[0] !== null) {
      for (const nestedItem of value) {
        await processSequenceItem(nestedItem as Record<string, unknown>, scrubContext, options, salt, uidMappings)
      }
      continue
    }

    if (typeof value !== 'string') continue

    // TextValue (0040,A160) — scrub free text for PHI
    if (key === 'TextValue') {
      const result = scrubFreeText(value, scrubContext)
      if (result.phiFound) {
        item[key] = result.text
      }
      continue
    }

    // PersonName — zero out
    if (key === 'PersonName' || (key.endsWith('Name') && value.includes('^'))) {
      item[key] = ''
      continue
    }

    // UID tags — apply consistent hash mapping
    if ((key.endsWith('UID') || key === 'UID') && /^[012]\.\d/.test(value)) {
      if (uidMappings.has(value)) {
        item[key] = uidMappings.get(value)!
      } else {
        const newUid = await hashUid(value, salt)
        uidMappings.set(value, newUid)
        item[key] = newUid
      }
      continue
    }

    // Date tags within sequences — shift if date shifting enabled
    if (options.dateShift && /^\d{8}$/.test(value) && key.toLowerCase().includes('date')) {
      item[key] = shiftDicomDate(value, options.dateShift.offsetDays)
      continue
    }
  }
}

function formatValue(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined
  if (typeof value === 'string') return value
  if (typeof value === 'number') return String(value)
  if (Array.isArray(value)) return value.join(' \\ ')
  if (value instanceof ArrayBuffer || value instanceof Uint8Array) return '[Binary Data]'
  if (typeof value === 'object') return '[Sequence]'
  return String(value)
}
