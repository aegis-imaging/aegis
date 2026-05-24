/**
 * DICOM anonymization helpers wrapping the existing @aegis/client library.
 *
 * The Tauri app reads file bytes via the Rust `read_dicom_bytes` IPC command
 * (returning a Vec<u8> serialised as a number[] / Uint8Array on the JS side)
 * and feeds them into parseDicomFile() one at a time. Each iteration:
 *
 *   read_dicom_bytes  -> Uint8Array
 *   parseDicomFile    -> { dataset, rawMeta, ... }
 *   deidentify        -> mutated dataset
 *   serializeDataset  -> Uint8Array (anonymized DICOM)
 *   uploadFile        -> PUT to signed URL
 *
 * After each upload the JS reference is dropped so V8 can GC the buffer before
 * we read the next file. Holding 1000 ArrayBuffers will OOM the Tauri webview.
 */

import {
  parseDicomFile,
  buildStudySummary,
  isDicomFile,
  deidentify,
  serializeDataset,
  groupByStudy,
  type ParsedDicomFile,
  type StudySummary,
} from '@aegis/client'

export interface PreviewFile {
  /** Absolute path on the host filesystem. */
  path: string
  /** Final segment of the path — used as the filename in upload init. */
  filename: string
  size_bytes: number
}

export interface ParsedStudy {
  uid: string
  files: ParsedDicomFile[]
  summary: StudySummary
}

/** Build a ParsedDicomFile from raw bytes + a filename. */
export function parseBytes(bytes: Uint8Array, filename: string): ParsedDicomFile | null {
  const arrayBuffer = bytes.buffer.slice(
    bytes.byteOffset,
    bytes.byteOffset + bytes.byteLength,
  ) as ArrayBuffer
  if (!isDicomFile(arrayBuffer)) {
    return null
  }
  const parsed = parseDicomFile(arrayBuffer, filename)
  return parsed
}

/**
 * Group a flat list of parsed files into per-study summaries. We rely on
 * @aegis/client's groupByStudy + buildStudySummary for parity with the
 * web upload portal.
 */
export function summarizeStudies(parsed: ParsedDicomFile[]): ParsedStudy[] {
  const grouped = groupByStudy(parsed)
  return Array.from(grouped.entries()).map(([uid, files]) => ({
    uid,
    files,
    summary: buildStudySummary(files),
  }))
}

/**
 * De-identify a single parsed file and return the serialised DICOM bytes
 * (preserving the original transfer syntax).
 */
export async function deidentifyToBytes(
  parsed: ParsedDicomFile,
  salt: string,
): Promise<Uint8Array> {
  // Re-parse from the original buffer to get a fresh mutable dataset. This
  // mirrors the web upload portal — running deidentify() on a dataset that
  // has already been serialised once can stomp pixel data.
  const { dataset, rawMeta } = parseDicomFile(parsed.arrayBuffer, parsed.filename)
  const { dataset: deidDataset } = await deidentify(dataset, { salt })
  return serializeDataset(rawMeta, deidDataset)
}
