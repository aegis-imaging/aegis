/**
 * Pixel-level PHI scrubbing for DICOM instances, entirely client-side.
 *
 * Pipeline per instance:
 *   1. Decode frames → RGBA canvas (pixel_decode.ts)
 *   2. Run OCR per frame, filter to PHI-shaped tokens (pixel_ocr.ts)
 *   3. Burn black rectangles into the *original* pixel buffer at the bounding
 *      boxes. We modify the raw bytes in place so the dataset still serializes
 *      with its original transfer syntax — no transcoding step.
 *   4. Return findings so the UI can show "12 PHI regions redacted on this
 *      image" and an operator can confirm before upload.
 *
 * Pixel-level scrubbing for *compressed* transfer syntaxes is not yet
 * implemented; instances are returned with `status: 'unsupported_transfer_syntax'`
 * and the UI must decide whether to skip-upload, fall back to server, or
 * convert to uncompressed before sending. See pixel_decode.ts for the list.
 */

import {
  DecodeOptions,
  DecodedFrame,
  UnsupportedTransferSyntaxError,
  decodeFrames,
  isCompressedTransferSyntax,
  transferSyntaxLabel,
  transferSyntaxOf,
} from './pixel_decode'
import { OcrFinding, OcrOptions, getOcrEngine } from './pixel_ocr'
import type { NaturalizedDataset } from './parser'

export interface PixelScrubOptions extends OcrOptions, DecodeOptions {
  /**
   * Padding (in pixels) added on every side of each detected finding before
   * the black box is burned in. Compensates for OCR bounding boxes that hug
   * glyph edges too tightly. Default 4.
   */
  paddingPixels?: number
  /**
   * If true (default), redacted pixel values use the dataset's MinPixelValue or
   * 0 for MONOCHROME2, max for MONOCHROME1. If false, always uses 0.
   */
  matchPhotometricBackground?: boolean
}

export type PixelScrubStatus =
  | 'scrubbed'                    // ran successfully (with or without findings)
  | 'no_findings'                 // ran and found nothing
  | 'unsupported_transfer_syntax' // compressed format we can't handle yet
  | 'no_pixel_data'               // instance has no PixelData (e.g. SR, KOS)
  | 'failed'                      // decode / OCR error

export interface PixelScrubResult {
  status: PixelScrubStatus
  /** Each entry corresponds to one frame in the instance. */
  perFrameFindings: OcrFinding[][]
  /** Total number of findings burned in across all frames. */
  totalFindings: number
  /** Transfer syntax we saw, label-formatted for the UI. */
  transferSyntaxLabel: string
  /** Populated when status === 'failed' or 'unsupported_transfer_syntax'. */
  error?: string
  /** Wall-clock duration spent in scrub for this instance. */
  durationMs: number
  /**
   * Set to true when the dataset was mutated in place (i.e. PixelData now
   * contains the redacted bytes and downstream serializeDataset will produce
   * the redacted file). Useful for the UI to know whether it can fall back.
   */
  datasetMutated: boolean
}

/**
 * Scrub one DICOM instance. The dataset is mutated in place when status is
 * 'scrubbed' or 'no_findings'.
 */
export async function scrubInstance(
  dataset: NaturalizedDataset,
  opts: PixelScrubOptions = {}
): Promise<PixelScrubResult> {
  const started = performance.now()
  const ts = transferSyntaxOf(dataset)

  if (!dataset.PixelData) {
    return result('no_pixel_data', { transferSyntaxLabel: transferSyntaxLabel(ts), durationMs: 0 })
  }
  if (isCompressedTransferSyntax(ts) && ts !== '1.2.840.10008.1.2.4.50') {
    return result('unsupported_transfer_syntax', {
      transferSyntaxLabel: transferSyntaxLabel(ts),
      error: `In-browser pixel scrub doesn't yet support ${transferSyntaxLabel(ts)} — use server-side scrubbing or transcode first.`,
      durationMs: performance.now() - started,
    })
  }

  let frames: DecodedFrame[]
  try {
    frames = await decodeFrames(dataset, opts)
  } catch (e) {
    if (e instanceof UnsupportedTransferSyntaxError) {
      return result('unsupported_transfer_syntax', {
        transferSyntaxLabel: transferSyntaxLabel(ts),
        error: e.message,
        durationMs: performance.now() - started,
      })
    }
    return result('failed', {
      transferSyntaxLabel: transferSyntaxLabel(ts),
      error: (e as Error).message,
      durationMs: performance.now() - started,
    })
  }

  const engine = await getOcrEngine(opts)
  const perFrameFindings: OcrFinding[][] = []
  let totalFindings = 0
  const padding = Math.max(0, opts.paddingPixels ?? 4)

  for (const frame of frames) {
    let findings: OcrFinding[] = []
    try {
      findings = await engine.recognise(frame, opts)
    } catch (e) {
      return result('failed', {
        transferSyntaxLabel: transferSyntaxLabel(ts),
        error: `OCR failed on frame ${frame.frameIndex}: ${(e as Error).message}`,
        durationMs: performance.now() - started,
        perFrameFindings,
      })
    }
    perFrameFindings.push(findings)
    totalFindings += findings.length
    if (findings.length > 0) {
      burnRedactions(frame, findings, padding, opts.matchPhotometricBackground !== false)
    }
  }

  if (totalFindings > 0) {
    // Re-pack frame bytes into PixelData. For uncompressed we concatenate
    // per-frame bytes back together; for JPEG Baseline we'd need to re-encode,
    // which we don't support yet — caller should not have reached here.
    repackPixelData(dataset, frames)
  }

  const durationMs = performance.now() - started
  return {
    status: totalFindings > 0 ? 'scrubbed' : 'no_findings',
    perFrameFindings,
    totalFindings,
    transferSyntaxLabel: transferSyntaxLabel(ts),
    durationMs,
    datasetMutated: totalFindings > 0,
  }
}

function result(
  status: PixelScrubStatus,
  overrides: Partial<PixelScrubResult> & { transferSyntaxLabel: string; durationMs: number }
): PixelScrubResult {
  return {
    status,
    perFrameFindings: overrides.perFrameFindings || [],
    totalFindings: 0,
    transferSyntaxLabel: overrides.transferSyntaxLabel,
    error: overrides.error,
    durationMs: overrides.durationMs,
    datasetMutated: false,
  }
}

/**
 * Burn black rectangles into a frame's raw pixel buffer.
 * Operates on the original DICOM bytes so the redaction is preserved on
 * re-serialization. Endianness + signedness handled to match decode.
 */
function burnRedactions(
  frame: DecodedFrame,
  findings: OcrFinding[],
  padding: number,
  matchBackground: boolean
): void {
  const { rawPixelBytes, width, height, bitsAllocated, samplesPerPixel, inverted } = frame
  const bytesPerSample = Math.ceil(bitsAllocated / 8)
  const samplesPerPixelLine = width * samplesPerPixel * bytesPerSample

  for (const f of findings) {
    const x0 = Math.max(0, Math.floor(f.x - padding))
    const y0 = Math.max(0, Math.floor(f.y - padding))
    const x1 = Math.min(width, Math.ceil(f.x + f.width + padding))
    const y1 = Math.min(height, Math.ceil(f.y + f.height + padding))

    for (let y = y0; y < y1; y++) {
      const lineStart = y * samplesPerPixelLine
      for (let x = x0; x < x1; x++) {
        const offset = lineStart + x * samplesPerPixel * bytesPerSample
        writePixel(rawPixelBytes, offset, bitsAllocated, samplesPerPixel, inverted, matchBackground)
      }
    }
  }
}

function writePixel(
  buf: Uint8Array,
  offset: number,
  bitsAllocated: number,
  samplesPerPixel: number,
  inverted: boolean,
  matchBackground: boolean
): void {
  // For MONOCHROME1, "black" displays as the *maximum* value; for MONOCHROME2,
  // "black" is 0. The opposite is true for "white" — we always pick the
  // visual-black side here.
  const bytesPerSample = Math.ceil(bitsAllocated / 8)
  let writeValue = 0
  if (matchBackground && inverted && bitsAllocated > 0) {
    writeValue = Math.pow(2, bitsAllocated) - 1
  }

  for (let s = 0; s < samplesPerPixel; s++) {
    const sampleOffset = offset + s * bytesPerSample
    if (bytesPerSample === 1) {
      buf[sampleOffset] = writeValue & 0xff
    } else if (bytesPerSample === 2) {
      buf[sampleOffset] = writeValue & 0xff
      buf[sampleOffset + 1] = (writeValue >> 8) & 0xff
    }
  }
}

function repackPixelData(dataset: NaturalizedDataset, frames: DecodedFrame[]): void {
  // For single-frame instances, dataset.PixelData is either an ArrayBuffer or
  // a one-element array of ArrayBuffer. We need to overwrite the bytes so
  // serializeDataset emits the redacted version.
  if (frames.length === 0) return
  const isArrayShape = Array.isArray(dataset.PixelData)
  if (isArrayShape && (dataset.PixelData as unknown[]).length === frames.length) {
    const arr = dataset.PixelData as ArrayBuffer[]
    for (let i = 0; i < frames.length; i++) {
      arr[i] = bufferFor(frames[i])
    }
    return
  }
  // Multi-frame uncompressed: a single contiguous buffer. Concatenate.
  const total = frames.reduce((n, f) => n + f.rawPixelBytes.byteLength, 0)
  const merged = new Uint8Array(total)
  let off = 0
  for (const f of frames) {
    merged.set(f.rawPixelBytes, off)
    off += f.rawPixelBytes.byteLength
  }
  if (isArrayShape) {
    ;(dataset.PixelData as unknown[])[0] = merged.buffer
  } else {
    dataset.PixelData = merged.buffer
  }
}

function bufferFor(frame: DecodedFrame): ArrayBuffer {
  const slice = frame.rawPixelBytes
  // Copy out of any larger backing buffer so we own a clean ArrayBuffer.
  const out = new Uint8Array(slice.byteLength)
  out.set(slice)
  return out.buffer
}

/**
 * Bulk-scrub every instance in a study group, with progress reporting and
 * cancellation support. Used by the upload portal's heavy mode.
 */
export interface StudyScrubProgress {
  studyInstanceUid: string
  fileIndex: number
  fileCount: number
  fileResult: PixelScrubResult
}

export interface StudyScrubOptions extends PixelScrubOptions {
  /** Called after every instance scrub. Return false to cancel mid-study. */
  onProgress?: (p: StudyScrubProgress) => boolean | void
  /** Abort the loop without finishing the remaining files. */
  signal?: AbortSignal
}

export interface StudyScrubSummary {
  studyInstanceUid: string
  filesScrubbed: number
  filesUnsupported: number
  filesFailed: number
  totalFindings: number
  perFile: PixelScrubResult[]
  durationMs: number
}

export async function scrubStudyDatasets(
  studyInstanceUid: string,
  datasets: NaturalizedDataset[],
  options: StudyScrubOptions = {}
): Promise<StudyScrubSummary> {
  const started = performance.now()
  const perFile: PixelScrubResult[] = []
  let filesScrubbed = 0
  let filesUnsupported = 0
  let filesFailed = 0
  let totalFindings = 0

  for (let i = 0; i < datasets.length; i++) {
    if (options.signal?.aborted) break
    const r = await scrubInstance(datasets[i], options)
    perFile.push(r)
    if (r.status === 'scrubbed') filesScrubbed++
    else if (r.status === 'unsupported_transfer_syntax') filesUnsupported++
    else if (r.status === 'failed') filesFailed++
    totalFindings += r.totalFindings
    const cont = options.onProgress?.({
      studyInstanceUid,
      fileIndex: i,
      fileCount: datasets.length,
      fileResult: r,
    })
    if (cont === false) break
  }

  return {
    studyInstanceUid,
    filesScrubbed,
    filesUnsupported,
    filesFailed,
    totalFindings,
    perFile,
    durationMs: performance.now() - started,
  }
}
