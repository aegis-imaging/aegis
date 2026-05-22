/**
 * On-device face de-identification — architecture stub.
 *
 * This file documents what the in-browser defacing path will look like, and
 * provides the public API surface so the upload portal can wire the UI now
 * (the toggle, the progress bar, the "not implemented" explainer). The actual
 * 3D U-Net inference is a follow-up PR — see the README at the bottom.
 *
 * Design (planned):
 *   1. Group a study's per-slice DICOM datasets by SeriesInstanceUID.
 *   2. For each MR/CT head series, sort by ImagePositionPatient.z (or
 *      InstanceNumber as fallback).
 *   3. Decode each slice's pixel data (pixel_decode.ts) and stack into a
 *      Float32Array of shape [Z, Y, X], applying the slope/intercept rescale.
 *   4. Resample to the input shape the model expects (typically 128³).
 *   5. Run the defacing model — initially a TF.js port of DeepDefacer
 *      (~30 MB weights, cacheable via the browser cache after first download).
 *      WebGPU if available, falling back to WebGL, falling back to WASM.
 *   6. Threshold the model output → 3D mask of "face" voxels.
 *   7. Upsample the mask back to the original resolution and burn it into
 *      every slice's pixel buffer (set face voxels to the dataset's
 *      configured background value).
 *   8. Mutate the original datasets in place so serializeDataset emits the
 *      defaced bytes.
 */

import type { ParsedDicomFile } from '../types'

export type FaceDeidStatus = 'not_implemented' | 'not_applicable' | 'completed' | 'failed'

export interface FaceDeidResult {
  status: FaceDeidStatus
  studyInstanceUid: string
  /** Slices that were modified in place. */
  modifiedFiles: string[]
  /** Free-form reason for non-completion (UI surfaces this verbatim). */
  reason?: string
  durationMs: number
}

export interface FaceDeidOptions {
  /**
   * Override the WebGPU/WebGL/WASM preference order.
   * Default: ['webgpu', 'webgl', 'wasm']. Useful for benchmarking.
   */
  backendPreference?: ('webgpu' | 'webgl' | 'wasm')[]
  /** URL where the model weights live (must be CORS-accessible). */
  modelUrl?: string
}

/**
 * Stub: returns 'not_implemented' for now. The UI calls this so the heavy
 * mode toggle has a stable surface to drive once the model lands.
 */
export async function defaceStudy(
  studyInstanceUid: string,
  _files: ParsedDicomFile[],
  _options: FaceDeidOptions = {}
): Promise<FaceDeidResult> {
  const started = performance.now()
  return {
    status: 'not_implemented',
    studyInstanceUid,
    modifiedFiles: [],
    reason:
      "On-device face de-identification isn't shipped yet. Skipping. " +
      'Pixel PHI scrub still ran (text-bearing burned-in regions are redacted). ' +
      "If face removal is required for this study, use the AEGIS Router (which runs " +
      'mri_deface / deepdefacer server-side on-prem) or the cloud defacing service.',
    durationMs: performance.now() - started,
  }
}

/** A small heuristic the UI can call to decide whether to even attempt face
 *  de-id for a study — once the real implementation lands. */
export function studyLooksLikeHeadScan(files: ParsedDicomFile[]): boolean {
  if (files.length < 10) return false  // single shot, not a 3D volume
  const first = files[0]
  const mod = (first.modality || '').toUpperCase()
  if (mod !== 'MR' && mod !== 'CT' && mod !== 'PT') return false
  const bp = (first.tags.find(t => t.keyword === 'BodyPartExamined')?.originalValue || '').toUpperCase()
  if (!bp) return true // no body part tag — could be head, defer
  return bp === 'HEAD' || bp === 'BRAIN' || bp === 'NECK' || bp === 'SKULL'
}
