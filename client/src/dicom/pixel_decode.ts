/**
 * In-browser DICOM pixel-data decoding.
 *
 * This module turns the bytes in `dataset.PixelData` into an 8-bit grayscale
 * (or RGB) Canvas-ready image, suitable for OCR or for redaction overlay.
 *
 * Scope of this prototype:
 *   - Uncompressed transfer syntaxes (1.2.840.10008.1.2, 1.2.840.10008.1.2.1,
 *     1.2.840.10008.1.2.2 — Implicit VR LE / Explicit VR LE / Explicit VR BE)
 *   - JPEG Baseline (1.2.840.10008.1.2.4.50) via the browser's native decoder
 *
 * Compressed transfer syntaxes (JPEG 2000, JPEG-LS, RLE) are detected and
 * surfaced as a clear "unsupported transfer syntax" error so the caller can
 * fall back to server-side scrubbing for those instances. Adding WASM codecs
 * is a deliberate follow-up — it more than doubles the bundle size.
 */

import type { NaturalizedDataset } from './parser'

export interface DecodedFrame {
  /** Pixel buffer in canvas-ready RGBA order, one byte per channel. */
  rgba: Uint8ClampedArray
  width: number
  height: number
  /** True if the source was MONOCHROME1 (display-inverted vs MONOCHROME2). */
  inverted: boolean
  /**
   * Original pixel data byte view, kept so we can write the redacted version
   * back into the dataset preserving the original encoding.
   */
  rawPixelBytes: Uint8Array
  bitsAllocated: number
  samplesPerPixel: number
  pixelRepresentation: number   // 0 = unsigned, 1 = signed two's-complement
  /** Multi-frame instances expose more than one frame. */
  frameIndex: number
  numFrames: number
}

export interface DecodeOptions {
  /**
   * Override the windowed range. If omitted, we use the dataset's WindowCenter /
   * WindowWidth, falling back to the full pixel-value range.
   */
  windowCenter?: number
  windowWidth?: number
  /** Limit how many frames we decode (defaults to all). */
  maxFrames?: number
}

export class UnsupportedTransferSyntaxError extends Error {
  constructor(public readonly transferSyntaxUid: string) {
    super(`Unsupported transfer syntax for in-browser decoding: ${transferSyntaxUid}`)
    this.name = 'UnsupportedTransferSyntaxError'
  }
}

const UNCOMPRESSED_TS = new Set([
  '1.2.840.10008.1.2',       // Implicit VR Little Endian
  '1.2.840.10008.1.2.1',     // Explicit VR Little Endian
  '1.2.840.10008.1.2.2',     // Explicit VR Big Endian (rare)
])

const JPEG_BASELINE_TS = '1.2.840.10008.1.2.4.50'

const COMPRESSED_TS_LABELS: Record<string, string> = {
  '1.2.840.10008.1.2.4.51': 'JPEG Extended',
  '1.2.840.10008.1.2.4.57': 'JPEG Lossless',
  '1.2.840.10008.1.2.4.70': 'JPEG Lossless SV1',
  '1.2.840.10008.1.2.4.80': 'JPEG-LS Lossless',
  '1.2.840.10008.1.2.4.81': 'JPEG-LS Near-Lossless',
  '1.2.840.10008.1.2.4.90': 'JPEG 2000 Lossless',
  '1.2.840.10008.1.2.4.91': 'JPEG 2000',
  '1.2.840.10008.1.2.5':    'RLE Lossless',
}

/** Look up the transfer syntax UID from the file meta. */
export function transferSyntaxOf(dataset: NaturalizedDataset): string {
  const meta = (dataset._meta || {}) as Record<string, unknown>
  const tsRaw = meta.TransferSyntaxUID
  if (typeof tsRaw === 'string') return tsRaw
  if (Array.isArray(tsRaw) && typeof tsRaw[0] === 'string') return tsRaw[0]
  return ''
}

export function isCompressedTransferSyntax(ts: string): boolean {
  return ts in COMPRESSED_TS_LABELS
}

export function transferSyntaxLabel(ts: string): string {
  if (UNCOMPRESSED_TS.has(ts)) return 'Uncompressed'
  if (ts === JPEG_BASELINE_TS) return 'JPEG Baseline'
  return COMPRESSED_TS_LABELS[ts] || ts
}

/**
 * Decode all (or up to `maxFrames`) frames of a DICOM instance into RGBA
 * canvas-ready images. Async because JPEG Baseline goes through the browser's
 * async decoder.
 */
export async function decodeFrames(
  dataset: NaturalizedDataset,
  options: DecodeOptions = {}
): Promise<DecodedFrame[]> {
  const ts = transferSyntaxOf(dataset)
  if (UNCOMPRESSED_TS.has(ts)) {
    return decodeUncompressed(dataset, ts, options)
  }
  if (ts === JPEG_BASELINE_TS) {
    return decodeJpegBaseline(dataset, options)
  }
  throw new UnsupportedTransferSyntaxError(ts)
}

// ── Uncompressed ────────────────────────────────────────────────────────────

function decodeUncompressed(
  dataset: NaturalizedDataset,
  ts: string,
  options: DecodeOptions
): DecodedFrame[] {
  const rows = numberAttr(dataset.Rows)
  const cols = numberAttr(dataset.Columns)
  const bitsAllocated = numberAttr(dataset.BitsAllocated) || 8
  const bitsStored = numberAttr(dataset.BitsStored) || bitsAllocated
  const samplesPerPixel = numberAttr(dataset.SamplesPerPixel) || 1
  const photometric = String(dataset.PhotometricInterpretation || 'MONOCHROME2')
  const pixelRepresentation = numberAttr(dataset.PixelRepresentation) || 0
  const inverted = photometric === 'MONOCHROME1'
  const numFrames = numberAttr(dataset.NumberOfFrames) || 1
  const maxFrames = options.maxFrames ?? numFrames
  const framesToDecode = Math.min(numFrames, maxFrames)

  const rawPixelBytes = pixelDataBytes(dataset)
  const bytesPerSample = Math.ceil(bitsAllocated / 8)
  const samplesPerFrame = rows * cols * samplesPerPixel
  const frameByteSize = samplesPerFrame * bytesPerSample
  const bigEndian = ts === '1.2.840.10008.1.2.2'

  const wc = options.windowCenter ?? numberAttr(dataset.WindowCenter)
  const ww = options.windowWidth ?? numberAttr(dataset.WindowWidth)

  const frames: DecodedFrame[] = []
  for (let f = 0; f < framesToDecode; f++) {
    const frameOffset = f * frameByteSize
    const frameSlice = rawPixelBytes.subarray(frameOffset, frameOffset + frameByteSize)
    let pixelValues: number[] | Int16Array | Uint16Array | Uint8Array

    if (samplesPerPixel === 1 && bytesPerSample === 1) {
      pixelValues = pixelRepresentation === 1
        ? new Int8Array(frameSlice.buffer, frameSlice.byteOffset, samplesPerFrame) as unknown as number[]
        : frameSlice
    } else if (samplesPerPixel === 1 && bytesPerSample === 2) {
      pixelValues = read16(frameSlice, samplesPerFrame, bigEndian, pixelRepresentation === 1)
    } else if (samplesPerPixel === 3 && bytesPerSample === 1) {
      // RGB / YBR — handle in renderer.
      pixelValues = frameSlice
    } else {
      // Unusual; fall back to raw bytes and let downstream OCR cope.
      pixelValues = frameSlice
    }

    const rgba = renderToRGBA({
      pixelValues, rows, cols, samplesPerPixel, bitsStored,
      photometric, windowCenter: wc, windowWidth: ww,
      pixelRepresentation,
    })

    frames.push({
      rgba, width: cols, height: rows, inverted,
      rawPixelBytes: new Uint8Array(frameSlice),
      bitsAllocated, samplesPerPixel, pixelRepresentation,
      frameIndex: f, numFrames,
    })
  }
  return frames
}

// ── JPEG Baseline (via the browser's decoder) ──────────────────────────────

async function decodeJpegBaseline(
  dataset: NaturalizedDataset,
  options: DecodeOptions
): Promise<DecodedFrame[]> {
  // For JPEG transfer syntaxes, dcmjs naturalizes PixelData as an array of
  // ArrayBuffers — one per frame (one per fragment for some).
  const rows = numberAttr(dataset.Rows)
  const cols = numberAttr(dataset.Columns)
  const bitsAllocated = numberAttr(dataset.BitsAllocated) || 8
  const samplesPerPixel = numberAttr(dataset.SamplesPerPixel) || 3
  const pixelRepresentation = numberAttr(dataset.PixelRepresentation) || 0
  const numFrames = numberAttr(dataset.NumberOfFrames) || 1
  const maxFrames = options.maxFrames ?? numFrames

  const frameBuffers = pixelDataFrames(dataset)
  const out: DecodedFrame[] = []
  const count = Math.min(maxFrames, frameBuffers.length || numFrames)
  for (let i = 0; i < count; i++) {
    const buf = frameBuffers[i] || frameBuffers[0]
    const blob = new Blob([buf], { type: 'image/jpeg' })
    const bitmap = await createImageBitmap(blob)
    const canvas = document.createElement('canvas')
    canvas.width = bitmap.width || cols
    canvas.height = bitmap.height || rows
    const ctx = canvas.getContext('2d', { willReadFrequently: true })!
    ctx.drawImage(bitmap, 0, 0)
    const img = ctx.getImageData(0, 0, canvas.width, canvas.height)
    out.push({
      rgba: img.data,
      width: img.width,
      height: img.height,
      inverted: false,
      rawPixelBytes: new Uint8Array(buf),
      bitsAllocated,
      samplesPerPixel,
      pixelRepresentation,
      frameIndex: i,
      numFrames,
    })
  }
  return out
}

// ── Render helpers ─────────────────────────────────────────────────────────

interface RenderArgs {
  pixelValues: ArrayLike<number>
  rows: number
  cols: number
  samplesPerPixel: number
  bitsStored: number
  photometric: string
  windowCenter?: number
  windowWidth?: number
  pixelRepresentation: number
}

function renderToRGBA(a: RenderArgs): Uint8ClampedArray {
  const size = a.rows * a.cols
  const rgba = new Uint8ClampedArray(size * 4)

  if (a.samplesPerPixel === 1) {
    const { center, width } = chooseWindow(a)
    const min = center - width / 2
    const inv = a.photometric === 'MONOCHROME1'
    for (let i = 0; i < size; i++) {
      const v = a.pixelValues[i] ?? 0
      let scaled = ((v - min) / width) * 255
      if (scaled < 0) scaled = 0
      else if (scaled > 255) scaled = 255
      const lum = inv ? 255 - scaled : scaled
      const j = i * 4
      rgba[j] = lum
      rgba[j + 1] = lum
      rgba[j + 2] = lum
      rgba[j + 3] = 255
    }
  } else if (a.samplesPerPixel === 3) {
    // Interleaved RGB (RGB or YBR_FULL); the latter would need colour-space
    // conversion. Most ultrasound + SC images are RGB already.
    for (let i = 0; i < size; i++) {
      const j = i * 4
      const k = i * 3
      rgba[j] = a.pixelValues[k] ?? 0
      rgba[j + 1] = a.pixelValues[k + 1] ?? 0
      rgba[j + 2] = a.pixelValues[k + 2] ?? 0
      rgba[j + 3] = 255
    }
  }
  return rgba
}

function chooseWindow(a: RenderArgs): { center: number; width: number } {
  if (typeof a.windowCenter === 'number' && typeof a.windowWidth === 'number' && a.windowWidth > 0) {
    return { center: a.windowCenter, width: a.windowWidth }
  }
  // Fallback: cover full bit-stored range. For signed types, centre on 0.
  const range = Math.pow(2, a.bitsStored)
  if (a.pixelRepresentation === 1) {
    return { center: 0, width: range }
  }
  return { center: range / 2, width: range }
}

// ── PixelData unpacking ────────────────────────────────────────────────────

/** Returns a flat Uint8Array view of the *first* PixelData element. */
export function pixelDataBytes(dataset: NaturalizedDataset): Uint8Array {
  const raw = dataset.PixelData
  if (!raw) throw new Error('PixelData not present in dataset')
  // dcmjs may give an array of ArrayBuffers or a single ArrayBuffer.
  if (Array.isArray(raw)) {
    if (raw.length === 0) throw new Error('PixelData array is empty')
    return toBytes(raw[0])
  }
  return toBytes(raw)
}

/** Returns an array of frame ArrayBuffers (for encapsulated formats). */
export function pixelDataFrames(dataset: NaturalizedDataset): ArrayBuffer[] {
  const raw = dataset.PixelData
  if (!raw) return []
  if (Array.isArray(raw)) {
    return raw.map(toBytes).map(b => b.buffer.slice(b.byteOffset, b.byteOffset + b.byteLength))
  }
  const b = toBytes(raw)
  return [b.buffer.slice(b.byteOffset, b.byteOffset + b.byteLength)]
}

function toBytes(v: unknown): Uint8Array {
  if (v instanceof Uint8Array) return v
  if (v instanceof ArrayBuffer) return new Uint8Array(v)
  if (ArrayBuffer.isView(v)) return new Uint8Array(v.buffer, v.byteOffset, v.byteLength)
  throw new Error('Unsupported PixelData representation')
}

function read16(
  bytes: Uint8Array, count: number, bigEndian: boolean, signed: boolean
): Int16Array | Uint16Array {
  const buf = new ArrayBuffer(count * 2)
  const view = new DataView(buf)
  for (let i = 0; i < count; i++) {
    const lo = bytes[i * 2]
    const hi = bytes[i * 2 + 1]
    const word = bigEndian ? ((lo << 8) | hi) : ((hi << 8) | lo)
    view.setInt16(i * 2, signed ? toSigned16(word) : word, true)
  }
  return signed ? new Int16Array(buf) : new Uint16Array(buf)
}

function toSigned16(n: number): number {
  return n & 0x8000 ? n - 0x10000 : n
}

function numberAttr(v: unknown): number {
  if (typeof v === 'number') return v
  if (typeof v === 'string') {
    const n = parseFloat(v)
    return Number.isFinite(n) ? n : 0
  }
  if (Array.isArray(v) && v.length > 0) return numberAttr(v[0])
  return 0
}
