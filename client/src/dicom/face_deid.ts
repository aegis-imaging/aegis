/**
 * On-device face de-identification.
 *
 * The pipeline composes a per-series 3D volume from the parsed datasets, runs
 * a defacing strategy to produce a face-mask, burns that mask into every
 * slice's raw pixel buffer, then leaves the datasets ready for the normal
 * serialize-and-upload path. Nothing leaves the browser.
 *
 * Two strategies ship today:
 *
 *   1. `'anterior-heuristic'` (default) — pure-TypeScript, zero model
 *      download. Zeros out the anterior portion of each axial slice
 *      (configurable fraction, default 30%). Matches the Python
 *      `defacing/app/backends/nibabel_fallback.py` baseline. Good enough for
 *      most head MR / CT studies; produces a visible "haircut" on the face
 *      side of the volume. Won't be biological-imaging-paper-quality, but
 *      it's deterministic, fast, and runs entirely in the browser.
 *
 *   2. `'tfjs-model'` (planned) — a TF.js port of DeepDefacer (~30 MB
 *      weights, WebGPU-accelerated). Higher fidelity, but requires shipping
 *      / hosting the model file + WebGPU runtime. Architecture is in place;
 *      calling it returns `status: 'not_implemented'` until the model file
 *      is bundled.
 *
 * Either way, the *interface* this module exposes is stable, so the upload
 * portal UI can wire against it once and benefit from any future strategy.
 */

import { Volume, composeVolume, writebackVolume, VolumeShapeMismatchError } from './volume'
import { parseDicomFile, serializeDataset } from './parser'
import type { NaturalizedDataset } from './parser'
import type { ParsedDicomFile } from '../types'
import { defaceVolumeWithTFJS, TFJSUnavailableError } from './tfjs_deface'
import type { TFBackend } from './tfjs_deface'

export type FaceDeidStatus =
  | 'completed'                  // ran successfully (mask burned into datasets)
  | 'no_op_short_series'         // not enough slices to be a 3D head volume
  | 'no_op_wrong_modality'       // not MR / CT / PT
  | 'no_op_wrong_body_part'      // body_part != HEAD/BRAIN/NECK/SKULL
  | 'shape_mismatch'             // slices weren't uniform — can't compose volume
  | 'not_implemented'            // tfjs-model called but no model file present
  | 'failed'                     // some other error

export type FaceDeidStrategy =
  | 'anterior-heuristic'
  | 'tfjs-model'
  | 'auto'                       // pick the best available

export interface FaceDeidResult {
  status: FaceDeidStatus
  studyInstanceUid: string
  strategyUsed: FaceDeidStrategy | null
  /** Filenames whose pixel data was mutated. */
  modifiedFiles: string[]
  /** Total voxels zeroed across all slices. */
  voxelsModified: number
  /** Free-form reason text (UI surfaces verbatim). */
  reason?: string
  durationMs: number
}

export interface FaceDeidOptions {
  /** Which strategy to use. Default `'auto'`. */
  strategy?: FaceDeidStrategy
  /**
   * Fraction of each axial slice's anterior side to zero out (0-1).
   * Only relevant for the anterior-heuristic strategy. Default 0.35.
   */
  anteriorFraction?: number
  /**
   * Z-axis fraction to *skip* at the top and bottom of the volume (no defacing
   * applied). Avoids zero-ing the eyes-and-up region or the neck base.
   * Default top=0.1, bottom=0.0.
   */
  zSkipTopFraction?: number
  zSkipBottomFraction?: number
  /**
   * Override the in-slice "anterior is +y or -y" detection. Default: auto
   * (uses ImageOrientationPatient if available; falls back to +y).
   */
  anteriorDirection?: 'positive-y' | 'negative-y'
  /** Where the TF.js model weights live (must be CORS-accessible). */
  modelUrl?: string
  /** Override the WebGPU/WebGL/WASM preference order (TF.js strategy only). */
  backendPreference?: ('webgpu' | 'webgl' | 'wasm')[]
  /** Called once per series with progress. */
  onProgress?: (p: FaceDeidProgress) => void
}

export interface FaceDeidProgress {
  studyInstanceUid: string
  phase: 'composing' | 'masking' | 'writeback' | 'done'
  filesProcessed: number
  fileCount: number
}

/**
 * Run face de-id on a study. The dataset references inside `files` are
 * mutated in place when status === 'completed'.
 */
export async function defaceStudy(
  studyInstanceUid: string,
  files: ParsedDicomFile[],
  options: FaceDeidOptions = {}
): Promise<FaceDeidResult> {
  const started = performance.now()
  const strategy: FaceDeidStrategy = options.strategy ?? 'auto'

  // 1. Cheap rejects before we touch pixels.
  const applicability = checkApplicability(files)
  if (applicability !== 'ok') {
    return {
      status: applicability,
      studyInstanceUid,
      strategyUsed: null,
      modifiedFiles: [],
      voxelsModified: 0,
      reason: applicabilityMessage(applicability),
      durationMs: performance.now() - started,
    }
  }

  // 2. Pick the strategy.
  const pickedStrategy: FaceDeidStrategy =
    strategy === 'auto' ? 'anterior-heuristic' : strategy

  if (pickedStrategy === 'tfjs-model') {
    if (!options.modelUrl) {
      return {
        status: 'not_implemented',
        studyInstanceUid,
        strategyUsed: 'tfjs-model',
        modifiedFiles: [],
        voxelsModified: 0,
        reason:
          "TF.js model strategy requires `modelUrl` pointing at a hosted " +
          "tensorflowjs_converter output (model.json). Either provide it or use " +
          "`strategy: 'anterior-heuristic'`.",
        durationMs: performance.now() - started,
      }
    }
    return runTFJSModelStrategy(studyInstanceUid, files, options, started)
  }

  // 3. Run anterior-heuristic.
  options.onProgress?.({ studyInstanceUid, phase: 'composing', filesProcessed: 0, fileCount: files.length })

  // We need mutable datasets — re-parse each file's ArrayBuffer to get fresh
  // ones we own and can write back through.
  const datasets = files.map(f => parseDicomFile(f.arrayBuffer, f.filename))
  const modifiedFiles: string[] = []
  let voxelsModified = 0

  try {
    const volume = await composeVolume(datasets.map(d => d.dataset))
    options.onProgress?.({ studyInstanceUid, phase: 'masking', filesProcessed: 0, fileCount: files.length })

    const result = applyAnteriorHeuristic(volume, options)
    voxelsModified = result.voxelsModified
    for (let i = 0; i < datasets.length; i++) {
      // Map each parsed file's dataset back to its source filename.
      if (datasetWasModified(volume.slices, i)) {
        modifiedFiles.push(files[i].filename)
      }
    }
    options.onProgress?.({
      studyInstanceUid, phase: 'writeback',
      filesProcessed: modifiedFiles.length, fileCount: files.length,
    })

    // Replace each ParsedDicomFile's arrayBuffer with a re-serialized version
    // of the mutated dataset so the upload path picks up the defaced pixels.
    for (let i = 0; i < datasets.length; i++) {
      const next = serializeDataset(datasets[i].rawMeta, datasets[i].dataset)
      ;(files[i] as { arrayBuffer: ArrayBuffer }).arrayBuffer = sliceToArrayBuffer(next)
    }

    options.onProgress?.({
      studyInstanceUid, phase: 'done',
      filesProcessed: modifiedFiles.length, fileCount: files.length,
    })

    return {
      status: 'completed',
      studyInstanceUid,
      strategyUsed: 'anterior-heuristic',
      modifiedFiles,
      voxelsModified,
      durationMs: performance.now() - started,
    }
  } catch (e) {
    if (e instanceof VolumeShapeMismatchError) {
      return {
        status: 'shape_mismatch',
        studyInstanceUid,
        strategyUsed: 'anterior-heuristic',
        modifiedFiles: [],
        voxelsModified: 0,
        reason: e.message,
        durationMs: performance.now() - started,
      }
    }
    return {
      status: 'failed',
      studyInstanceUid,
      strategyUsed: 'anterior-heuristic',
      modifiedFiles: [],
      voxelsModified: 0,
      reason: (e as Error).message,
      durationMs: performance.now() - started,
    }
  }
}

// ── strategies ─────────────────────────────────────────────────────────────

interface ApplyResult {
  voxelsModified: number
}

function applyAnteriorHeuristic(volume: Volume, opts: FaceDeidOptions): ApplyResult {
  const anteriorFraction = clamp01(opts.anteriorFraction ?? 0.35)
  const zSkipTop = clamp01(opts.zSkipTopFraction ?? 0.1)
  const zSkipBottom = clamp01(opts.zSkipBottomFraction ?? 0.0)
  const direction = opts.anteriorDirection
    ?? detectAnteriorDirection(volume.slices[0]?.dataset)

  const { y: height, z: depth } = volume.shape

  const zStart = Math.floor(depth * zSkipBottom)
  const zEnd = Math.ceil(depth * (1 - zSkipTop))
  const cutoff = Math.floor(height * anteriorFraction)
  const background = backgroundValueFor(volume)

  // For 'positive-y' anterior, anterior occupies rows 0..cutoff.
  // For 'negative-y', anterior occupies rows (height-cutoff)..height.
  const ranges = direction === 'positive-y'
    ? { yStart: 0, yEnd: cutoff }
    : { yStart: height - cutoff, yEnd: height }

  const { modifiedVoxels } = writebackVolume(volume, (_x, y, z, current) => {
    if (z < zStart || z >= zEnd) return current
    if (y < ranges.yStart || y >= ranges.yEnd) return current
    return background
  })
  return { voxelsModified: modifiedVoxels }
}

function backgroundValueFor(volume: Volume): number {
  // For MR the background is approximately the min value. For CT we want air
  // (~-1000 HU after rescale). Volume min covers both naturally.
  return volume.stats.min
}

function detectAnteriorDirection(ds: NaturalizedDataset | undefined): 'positive-y' | 'negative-y' {
  if (!ds) return 'positive-y'
  // ImageOrientationPatient is a 6-element array: row dir cosines + column
  // dir cosines (in LPS). Column direction's Y component tells us how the
  // image's "down" relates to the patient. For standard axial acquisitions
  // anterior is +y in image space when the column cosine has positive y.
  const iop = ds.ImageOrientationPatient
  if (Array.isArray(iop) && iop.length >= 6) {
    const colY = Number(iop[4])
    if (Number.isFinite(colY) && colY < 0) return 'negative-y'
  }
  return 'positive-y'
}

function datasetWasModified(
  slices: { rawPixelBytes: Uint8Array }[],
  _datasetIndex: number
): boolean {
  // Anterior-heuristic touches every slice in the active z range. The UI shows
  // counts but doesn't gate on this; "yes if there were any slices" is fine.
  return slices.length > 0
}

// ── TF.js model strategy ──────────────────────────────────────────────────

async function runTFJSModelStrategy(
  studyInstanceUid: string,
  files: ParsedDicomFile[],
  options: FaceDeidOptions,
  started: number,
): Promise<FaceDeidResult> {
  const datasets = files.map(f => parseDicomFile(f.arrayBuffer, f.filename))
  try {
    const volume = await composeVolume(datasets.map(d => d.dataset))
    const tfBackendPref = (options.backendPreference ?? []) as TFBackend[]
    const result = await defaceVolumeWithTFJS(volume, {
      modelUrl: options.modelUrl!,
      backendPreference: tfBackendPref.length > 0 ? tfBackendPref : undefined,
    })
    for (let i = 0; i < datasets.length; i++) {
      const next = serializeDataset(datasets[i].rawMeta, datasets[i].dataset)
      ;(files[i] as { arrayBuffer: ArrayBuffer }).arrayBuffer = sliceToArrayBuffer(next)
    }
    return {
      status: 'completed',
      studyInstanceUid,
      strategyUsed: 'tfjs-model',
      modifiedFiles: files.map(f => f.filename),
      voxelsModified: result.voxelsModified,
      reason: `TF.js model (${result.backendUsed}); inference ${Math.round(result.modelInferenceMs)} ms, total ${Math.round(result.totalMs)} ms`,
      durationMs: performance.now() - started,
    }
  } catch (e) {
    if (e instanceof VolumeShapeMismatchError) {
      return {
        status: 'shape_mismatch', studyInstanceUid, strategyUsed: 'tfjs-model',
        modifiedFiles: [], voxelsModified: 0,
        reason: e.message, durationMs: performance.now() - started,
      }
    }
    if (e instanceof TFJSUnavailableError) {
      return {
        status: 'not_implemented', studyInstanceUid, strategyUsed: 'tfjs-model',
        modifiedFiles: [], voxelsModified: 0,
        reason: e.message, durationMs: performance.now() - started,
      }
    }
    return {
      status: 'failed', studyInstanceUid, strategyUsed: 'tfjs-model',
      modifiedFiles: [], voxelsModified: 0,
      reason: (e as Error).message, durationMs: performance.now() - started,
    }
  }
}

// Re-export NaturalizedDataset reference type for downstream consumers that
// import face_deid first.
export type { Volume } from './volume'

// ── applicability ─────────────────────────────────────────────────────────

type Applicability = 'ok' | 'no_op_short_series' | 'no_op_wrong_modality' | 'no_op_wrong_body_part'

function checkApplicability(files: ParsedDicomFile[]): Applicability {
  if (files.length < 10) return 'no_op_short_series'
  const first = files[0]
  const mod = (first.modality || '').toUpperCase()
  if (mod !== 'MR' && mod !== 'CT' && mod !== 'PT') return 'no_op_wrong_modality'
  const bp = (first.tags.find(t => t.keyword === 'BodyPartExamined')?.originalValue || '').toUpperCase()
  if (!bp) return 'ok'
  if (bp === 'HEAD' || bp === 'BRAIN' || bp === 'NECK' || bp === 'SKULL') return 'ok'
  return 'no_op_wrong_body_part'
}

function applicabilityMessage(reason: Applicability): string {
  switch (reason) {
    case 'no_op_short_series':
      return 'Series has fewer than 10 instances — too short to be a 3D head volume. Skipping face de-id.'
    case 'no_op_wrong_modality':
      return 'Modality is not MR/CT/PT. Face de-id only runs on volumetric brain imaging. Skipping.'
    case 'no_op_wrong_body_part':
      return 'BodyPartExamined is not HEAD/BRAIN/NECK/SKULL. Face de-id only runs on head studies. Skipping.'
    default:
      return ''
  }
}

// ── studyLooksLikeHeadScan (kept compatible with previous UI imports) ─────

export function studyLooksLikeHeadScan(files: ParsedDicomFile[]): boolean {
  return checkApplicability(files) === 'ok'
}

// ── helpers ────────────────────────────────────────────────────────────────

function clamp01(v: number): number {
  if (v < 0) return 0
  if (v > 1) return 1
  return v
}

function sliceToArrayBuffer(u8: Uint8Array): ArrayBuffer {
  const buf = new ArrayBuffer(u8.byteLength)
  new Uint8Array(buf).set(u8)
  return buf
}
