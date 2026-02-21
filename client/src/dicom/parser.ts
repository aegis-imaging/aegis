import dcmjs from 'dcmjs'
import type { ParsedDicomFile, StudySummary } from '../types'
import { BASIC_PROFILE } from './tags'

const { DicomMessage, DicomMetaDictionary, DicomDict } = dcmjs.data

// dcmjs naturalized datasets are inherently loosely typed
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type NaturalizedDataset = Record<string, any>

function str(val: unknown): string {
  if (val === undefined || val === null) return ''
  return String(val)
}

/**
 * Parse a single DICOM file from an ArrayBuffer.
 * Returns the public metadata, the mutable naturalized dataset, and the raw
 * file meta — the latter two are needed for de-identification and re-serialization.
 */
export function parseDicomFile(
  arrayBuffer: ArrayBuffer,
  filename: string
): { parsed: ParsedDicomFile; dataset: NaturalizedDataset; rawMeta: Record<string, unknown> } {
  const dicomData = DicomMessage.readFile(arrayBuffer)
  const dataset = DicomMetaDictionary.naturalizeDataset(dicomData.dict)
  dataset._meta = DicomMetaDictionary.naturalizeDataset(dicomData.meta)

  // Build tag list for preview
  const tags = Object.entries(BASIC_PROFILE).map(([tag, rule]) => ({
    tag,
    keyword: rule.keyword,
    vr: '',
    originalValue: formatValue(dataset[rule.keyword]),
    anonymizedValue: undefined as string | undefined,
    action: rule.action,
  }))

  const parsed: ParsedDicomFile = {
    filename,
    studyInstanceUid: str(dataset.StudyInstanceUID),
    seriesInstanceUid: str(dataset.SeriesInstanceUID),
    sopInstanceUid: str(dataset.SOPInstanceUID),
    modality: str(dataset.Modality),
    seriesDescription: str(dataset.SeriesDescription),
    tags,
    arrayBuffer,
  }

  return { parsed, dataset, rawMeta: dicomData.meta }
}

/**
 * Re-serialize a de-identified naturalized dataset back to DICOM bytes.
 * Preserves the original file meta (TransferSyntaxUID, MediaStorageSOPClassUID, etc.)
 * from the file that was originally parsed.
 */
export function serializeDataset(
  rawMeta: Record<string, unknown>,
  dataset: NaturalizedDataset
): Uint8Array {
  // Strip the internal _meta key before denaturalizing the main dataset
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { _meta, ...mainDataset } = dataset
  const denaturalized = DicomMetaDictionary.denaturalizeDataset(mainDataset)
  const dicomFile = new DicomDict(rawMeta)
  dicomFile.dict = denaturalized
  return dicomFile.write()
}

/**
 * Build a study summary from a collection of parsed files.
 */
export function buildStudySummary(files: ParsedDicomFile[]): StudySummary {
  if (files.length === 0) {
    return {
      patientName: '', patientId: '', studyDate: '', studyTime: '', timezoneOffsetFromUtc: '', studyDateTimeIso: '',
      studyDescription: '', modality: '', studyInstanceUid: '',
      seriesCount: 0, imageCount: 0, bodyPart: '',
    }
  }

  const first = files[0]
  const firstTag = (keyword: string) =>
    first.tags.find(t => t.keyword === keyword)?.originalValue || ''

  const seriesUids = new Set(files.map(f => f.seriesInstanceUid))
  const modalities = new Set(files.map(f => f.modality).filter(Boolean))
  const studyDate = firstTag('StudyDate')
  const studyTime = firstTag('StudyTime')
  const timezoneOffsetFromUtc = firstTag('TimezoneOffsetFromUTC')
  const acquisitionDateTime = firstTag('AcquisitionDateTime')
  const studyDateTimeIso =
    dicomStudyDateTimeToIso(studyDate, studyTime, timezoneOffsetFromUtc) ||
    dicomDateTimeTagToIso(acquisitionDateTime)

  return {
    patientName: firstTag('PatientName'),
    patientId: firstTag('PatientID'),
    studyDate,
    studyTime,
    timezoneOffsetFromUtc,
    studyDateTimeIso,
    studyDescription: firstTag('StudyDescription'),
    modality: Array.from(modalities).join(', '),
    studyInstanceUid: first.studyInstanceUid,
    seriesCount: seriesUids.size,
    imageCount: files.length,
    bodyPart: firstTag('BodyPartExamined'),
  }
}

/**
 * Check if a File is likely a DICOM file.
 * DICOM Part 10 files have "DICM" at byte offset 128.
 */
export async function isDicomFile(file: File): Promise<boolean> {
  if (file.name.startsWith('.')) return false
  const lowerName = file.name.toLowerCase()
  if (lowerName.endsWith('.xml') || lowerName.endsWith('.json') ||
      lowerName.endsWith('.txt') || lowerName.endsWith('.pdf') ||
      lowerName.endsWith('.jpg') || lowerName.endsWith('.png')) {
    return false
  }

  if (file.size < 132) return false
  const header = await file.slice(128, 132).arrayBuffer()
  const magic = new Uint8Array(header)
  return magic[0] === 0x44 && magic[1] === 0x49 &&
         magic[2] === 0x43 && magic[3] === 0x4D // "DICM"
}

/**
 * Group parsed DICOM files by StudyInstanceUID.
 * Returns a Map where keys are study UIDs and values are arrays of files.
 */
export function groupByStudy(files: ParsedDicomFile[]): Map<string, ParsedDicomFile[]> {
  const groups = new Map<string, ParsedDicomFile[]>()
  for (const file of files) {
    const uid = file.studyInstanceUid
    const group = groups.get(uid)
    if (group) {
      group.push(file)
    } else {
      groups.set(uid, [file])
    }
  }
  return groups
}

function dicomStudyDateTimeToIso(studyDate: string, studyTime: string, offset: string): string {
  const dateParts = parseDicomDate(studyDate)
  if (!dateParts) return ''

  const timeParts = parseDicomTime(studyTime) ?? { hh: 0, mm: 0, ss: 0 }
  const parsedOffset = parseDicomOffset(offset)
  if (!parsedOffset) return ''

  return (
    `${pad4(dateParts.y)}-${pad2(dateParts.m)}-${pad2(dateParts.d)}` +
    `T${pad2(timeParts.hh)}:${pad2(timeParts.mm)}:${pad2(timeParts.ss)}` +
    `${parsedOffset.slice(0, 3)}:${parsedOffset.slice(3)}`
  )
}

function dicomDateTimeTagToIso(value: string): string {
  if (!value) return ''
  const match = value.match(/^(\d{4})(\d{2})(\d{2})(\d{2})?(\d{2})?(\d{2})?(?:\.\d+)?([+-]\d{4})?$/)
  if (!match) return ''

  const y = Number(match[1])
  const m = Number(match[2])
  const d = Number(match[3])
  const hh = Number(match[4] ?? '0')
  const mm = Number(match[5] ?? '0')
  const ss = Number(match[6] ?? '0')
  const offset = parseDicomOffset(match[7] ?? '')
  if (!isValidDate(y, m, d) || !isValidTime(hh, mm, ss) || !offset) return ''

  return `${pad4(y)}-${pad2(m)}-${pad2(d)}T${pad2(hh)}:${pad2(mm)}:${pad2(ss)}${offset.slice(0, 3)}:${offset.slice(3)}`
}

function parseDicomDate(value: string): { y: number; m: number; d: number } | null {
  if (!value || !/^\d{8}$/.test(value)) return null
  const y = Number(value.slice(0, 4))
  const m = Number(value.slice(4, 6))
  const d = Number(value.slice(6, 8))
  return isValidDate(y, m, d) ? { y, m, d } : null
}

function parseDicomTime(value: string): { hh: number; mm: number; ss: number } | null {
  if (!value) return null
  const main = value.split('.')[0]
  if (!/^\d{2}(\d{2})?(\d{2})?$/.test(main)) return null

  const hh = Number(main.slice(0, 2))
  const mm = main.length >= 4 ? Number(main.slice(2, 4)) : 0
  const ss = main.length >= 6 ? Number(main.slice(4, 6)) : 0

  return isValidTime(hh, mm, ss) ? { hh, mm, ss } : null
}

function parseDicomOffset(value: string): string | null {
  if (!value || !/^[+-]\d{4}$/.test(value)) return null
  const hh = Number(value.slice(1, 3))
  const mm = Number(value.slice(3, 5))
  if (hh > 23 || mm > 59) return null
  return value
}

function isValidDate(y: number, m: number, d: number): boolean {
  if (m < 1 || m > 12 || d < 1 || d > 31) return false
  const dt = new Date(Date.UTC(y, m - 1, d))
  return dt.getUTCFullYear() === y && dt.getUTCMonth() === m - 1 && dt.getUTCDate() === d
}

function isValidTime(hh: number, mm: number, ss: number): boolean {
  return hh >= 0 && hh <= 23 && mm >= 0 && mm <= 59 && ss >= 0 && ss <= 59
}

function pad2(n: number): string {
  return String(n).padStart(2, '0')
}

function pad4(n: number): string {
  return String(n).padStart(4, '0')
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
