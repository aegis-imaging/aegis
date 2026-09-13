/**
 * DICOM date shifting utilities for de-identification.
 *
 * Instead of removing or zeroing dates (which destroys temporal relationships),
 * date shifting applies a consistent per-patient day offset to all dates.
 * This preserves relative timing between studies while preventing re-identification.
 */

/** Known DICOM date tags (DA VR) that should be shifted */
export const DATE_TAGS: Set<string> = new Set([
  '00080012', // InstanceCreationDate
  '00080020', // StudyDate
  '00080021', // SeriesDate
  '00080022', // AcquisitionDate
  '00080023', // ContentDate
  '00100030', // PatientBirthDate
  '00400244', // PerformedProcedureStepStartDate
])

/** Known DICOM time tags (TM VR) associated with date tags — kept unchanged */
export const TIME_TAGS: Set<string> = new Set([
  '00080013', // InstanceCreationTime
  '00080030', // StudyTime
  '00080031', // SeriesTime
  '00080032', // AcquisitionTime
  '00080033', // ContentTime
  '00400245', // PerformedProcedureStepStartTime
])

/**
 * Shift a DICOM DA (YYYYMMDD) value by the given number of days.
 * Returns the shifted date in YYYYMMDD format.
 * Returns the original string unchanged if it cannot be parsed.
 */
export function shiftDicomDate(dateStr: string, offsetDays: number): string {
  if (!dateStr || dateStr.length < 8) return dateStr

  const year = parseInt(dateStr.substring(0, 4), 10)
  const month = parseInt(dateStr.substring(4, 6), 10) - 1 // JS months are 0-based
  const day = parseInt(dateStr.substring(6, 8), 10)

  if (isNaN(year) || isNaN(month) || isNaN(day)) return dateStr

  const date = new Date(Date.UTC(year, month, day))
  date.setUTCDate(date.getUTCDate() + offsetDays)

  const y = date.getUTCFullYear().toString().padStart(4, '0')
  const m = (date.getUTCMonth() + 1).toString().padStart(2, '0')
  const d = date.getUTCDate().toString().padStart(2, '0')
  return `${y}${m}${d}`
}

/**
 * Shift a DICOM DT (YYYYMMDDHHMMSS.FFFFFF&ZZXX) value by the given number of days.
 * Only the date portion (first 8 chars) is shifted; the time and timezone are preserved.
 * Returns the original string unchanged if it cannot be parsed.
 */
export function shiftDicomDateTime(dtStr: string, offsetDays: number): string {
  if (!dtStr || dtStr.length < 8) return dtStr

  const datePart = dtStr.substring(0, 8)
  const rest = dtStr.substring(8)
  const shifted = shiftDicomDate(datePart, offsetDays)

  return shifted + rest
}

/**
 * Generate a deterministic day offset from a patient ID and salt.
 *
 * Uses SHA-256 of (salt + patientId), takes the first 4 bytes as an unsigned
 * integer, and maps it to the range [-maxDays, +maxDays].
 *
 * The same patient ID and salt always produce the same offset, ensuring
 * consistency across all files for the same patient.
 *
 * NOTE: This function is async because it uses the Web Crypto API (crypto.subtle).
 */
export async function generateDateOffset(
  patientId: string,
  salt: string,
  maxDays: number = 365
): Promise<number> {
  const encoder = new TextEncoder()
  const data = encoder.encode(salt + patientId)
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  const view = new DataView(hashBuffer)

  // Read first 4 bytes as unsigned 32-bit integer
  const raw = view.getUint32(0)

  // Map to [-maxDays, +maxDays]
  const range = maxDays * 2 + 1
  const offset = (raw % range) - maxDays

  return offset
}
