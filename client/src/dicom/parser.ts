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
      patientName: '', patientId: '', studyDate: '',
      studyDescription: '', modality: '', studyInstanceUid: '',
      seriesCount: 0, imageCount: 0, bodyPart: '',
    }
  }

  const first = files[0]
  const firstTag = (keyword: string) =>
    first.tags.find(t => t.keyword === keyword)?.originalValue || ''

  const seriesUids = new Set(files.map(f => f.seriesInstanceUid))
  const modalities = new Set(files.map(f => f.modality).filter(Boolean))

  return {
    patientName: firstTag('PatientName'),
    patientId: firstTag('PatientID'),
    studyDate: firstTag('StudyDate'),
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

function formatValue(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined
  if (typeof value === 'string') return value
  if (typeof value === 'number') return String(value)
  if (Array.isArray(value)) return value.join(' \\ ')
  if (value instanceof ArrayBuffer || value instanceof Uint8Array) return '[Binary Data]'
  if (typeof value === 'object') return '[Sequence]'
  return String(value)
}
