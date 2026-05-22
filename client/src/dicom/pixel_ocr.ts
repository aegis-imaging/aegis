/**
 * Lazy-loaded OCR engine.
 *
 * The point of "lazy" is that tesseract.js + its language data is ~10-12 MB.
 * The standard upload flow (tag de-id only) should never download it. We pull
 * the engine in only when the user opts into heavy mode for the first time.
 *
 * Hosted assets are pinned to a specific CDN version below so the bundle is
 * deterministic across deploys; for offline / on-prem operation a follow-up
 * PR should self-host the WASM + language data.
 */

import type { DecodedFrame } from './pixel_decode'

export interface OcrFinding {
  /** Recognised text, trimmed. */
  text: string
  /** Confidence score 0-1 (rescaled from Tesseract's 0-100). */
  confidence: number
  /** Axis-aligned bounding box in pixel coordinates of the source frame. */
  x: number
  y: number
  width: number
  height: number
}

export interface OcrOptions {
  /** Drop findings with `confidence < minConfidence` (default 0.4). */
  minConfidence?: number
  /** Drop findings shorter than this many characters after trim (default 3). */
  minLength?: number
  /**
   * If true, only findings that *look like* PHI patterns are returned. Useful
   * when the image legitimately has small-print labels you don't want to redact.
   * Default true. Disable for "scrub anything that's text" behaviour.
   */
  phiPatternsOnly?: boolean
  /**
   * Tesseract worker pool size. 1 is fine for typical interactive use; bump
   * to 2-4 only when running large studies in batch.
   */
  workers?: number
  /** Tesseract data + worker URLs. Useful for self-hosted deployment. */
  langPath?: string
  workerPath?: string
  corePath?: string
}

interface OcrEngineHandle {
  recognise(frame: DecodedFrame, opts: OcrOptions): Promise<OcrFinding[]>
  terminate(): Promise<void>
}

let _engineSingleton: Promise<OcrEngineHandle> | null = null

/**
 * Reset the engine singleton — primarily for tests so each `vi.mock` reload
 * starts fresh.
 */
export function _resetEngineForTests(): void {
  _engineSingleton = null
}

/**
 * Get (or lazily load) the OCR engine.
 * The engine is shared across calls so we only pay the dictionary-load cost
 * once per session.
 */
export async function getOcrEngine(opts: OcrOptions = {}): Promise<OcrEngineHandle> {
  if (_engineSingleton) return _engineSingleton

  _engineSingleton = (async () => {
    // Dynamic import keeps tesseract.js out of the default chunk. The
    // package is a *peer* dep so consumers that don't use heavy mode never
    // install it; routing the import through a variable string keeps the
    // TypeScript compiler from requiring its type declarations at build
    // time (only the consumer that opts in pays the install cost).
    const pkg = 'tesseract.js'
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const tess: any = await import(/* @vite-ignore */ pkg)
    const workerOpts: Record<string, string> = {}
    if (opts.langPath) workerOpts.langPath = opts.langPath
    if (opts.workerPath) workerOpts.workerPath = opts.workerPath
    if (opts.corePath) workerOpts.corePath = opts.corePath

    // OEM 1 = LSTM_ONLY (best accuracy with modern Tesseract).
    const worker = await tess.createWorker('eng', 1, workerOpts)
    await worker.setParameters({
      // PSM 11 = SPARSE_TEXT. Right for medical images: scattered burned-in
      // labels rather than full paragraphs of body copy.
      tessedit_pageseg_mode: '11',
    })

    return {
      async recognise(frame: DecodedFrame, callOpts: OcrOptions): Promise<OcrFinding[]> {
        const canvas = framesToCanvas(frame)
        const { data } = await worker.recognize(canvas, undefined, { blocks: false })
        return wordsToFindings(data, callOpts)
      },
      async terminate(): Promise<void> {
        await worker.terminate()
      },
    }
  })()

  return _engineSingleton
}

function framesToCanvas(frame: DecodedFrame): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = frame.width
  canvas.height = frame.height
  const ctx = canvas.getContext('2d', { willReadFrequently: true })!
  const img = new ImageData(frame.rgba, frame.width, frame.height)
  ctx.putImageData(img, 0, 0)
  return canvas
}

// dcmjs's tesseract.js types are missing some shape; describe what we use.
interface TesseractWord {
  text: string
  confidence: number
  bbox: { x0: number; y0: number; x1: number; y1: number }
}
interface TesseractData {
  words?: TesseractWord[]
  text?: string
}

function wordsToFindings(data: TesseractData, opts: OcrOptions): OcrFinding[] {
  const minConf = opts.minConfidence ?? 0.4
  const minLen = opts.minLength ?? 3
  const out: OcrFinding[] = []
  for (const w of data.words || []) {
    const text = (w.text || '').trim()
    if (text.length < minLen) continue
    const confidence = (w.confidence || 0) / 100
    if (confidence < minConf) continue
    if (opts.phiPatternsOnly !== false && !looksLikePhi(text)) continue
    out.push({
      text, confidence,
      x: w.bbox.x0,
      y: w.bbox.y0,
      width: Math.max(1, w.bbox.x1 - w.bbox.x0),
      height: Math.max(1, w.bbox.y1 - w.bbox.y0),
    })
  }
  return out
}

// Pragmatic PHI heuristics. The goal isn't a perfect classifier — we have a
// human-review step downstream — but rather to keep us from carpet-bombing
// every label in the image. False positives here cause unnecessary redactions;
// false negatives let PHI through, so when in doubt we redact.
const RX_DATE        = /\b(19|20)\d{2}[-/.]?\d{1,2}[-/.]?\d{1,2}\b/
const RX_TIME        = /\b\d{1,2}:\d{2}(:\d{2})?\b/
const RX_DIGIT_RUN   = /\b\d{4,}\b/                  // MRN, accession, phone
const RX_SSN         = /\b\d{3}-\d{2}-\d{4}\b/
const RX_NAME_CAPS   = /\b[A-Z]{2,}(?:\s*\^?\s*[A-Z]{2,})+\b/  // SMITH^JANE
const RX_NAME_MIXED  = /\b[A-Z][a-z]{2,}\s+[A-Z][a-z]{2,}\b/   // John Smith
const RX_MR_ACC      = /\b(MRN|ACC#?|ACCESSION|PT|PATIENT|DOB|PHONE|TEL|ADDR|ADDRESS|EMAIL)\b/i
const RX_EMAIL       = /\b[\w.+-]+@[\w-]+\.[a-z]{2,}\b/i

const PHI_PATTERNS: RegExp[] = [
  RX_DATE, RX_TIME, RX_DIGIT_RUN, RX_SSN,
  RX_NAME_CAPS, RX_NAME_MIXED, RX_MR_ACC, RX_EMAIL,
]

/** Exported for tests + downstream tooling. */
export function looksLikePhi(text: string): boolean {
  return PHI_PATTERNS.some(re => re.test(text))
}
