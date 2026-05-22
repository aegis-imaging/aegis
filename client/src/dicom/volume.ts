/**
 * Compose a 3D voxel volume from a DICOM series, for analyses that need to
 * operate on the whole study at once — face de-identification being the
 * primary use case today.
 *
 * Inputs are a series' per-slice parsed datasets (we own them after
 * `parseDicomFile()`). Output is a Float32Array of shape [Z, Y, X] plus the
 * geometry metadata needed to write changes back into the right slices.
 *
 * Heuristics for slice ordering:
 *   1. ImagePositionPatient.z (sorted ascending)  — preferred, in mm
 *   2. SliceLocation                              — fallback
 *   3. InstanceNumber                             — last resort
 */

import { decodeFrames } from './pixel_decode'
import type { NaturalizedDataset } from './parser'

export interface Volume {
  /** Voxel data in z-y-x order, float32 (rescaled if slope/intercept set). */
  voxels: Float32Array
  /** Per-axis dimensions. */
  shape: { z: number; y: number; x: number }
  /** Slice-by-slice references back to the source for write-back. */
  slices: VolumeSlice[]
  /** Mean/std/min/max of the volume, computed in one pass. */
  stats: { min: number; max: number; mean: number }
}

export interface VolumeSlice {
  index: number  // z-index in the volume
  dataset: NaturalizedDataset
  rows: number
  columns: number
  /** Position along the z axis (mm) if available, else slice index. */
  zPosition: number
  /** Rescale slope (defaults to 1). */
  rescaleSlope: number
  /** Rescale intercept (defaults to 0). */
  rescaleIntercept: number
  /** Raw pixel byte view this slice owns (for write-back). */
  rawPixelBytes: Uint8Array
  bitsAllocated: number
  pixelRepresentation: number
}

export interface ComposeOptions {
  /** Override slice ordering — if omitted, we use the heuristic above. */
  customOrder?: (a: NaturalizedDataset, b: NaturalizedDataset) => number
  /** Skip rescale (treat raw voxel values as Float32 unchanged). */
  skipRescale?: boolean
}

export class VolumeShapeMismatchError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'VolumeShapeMismatchError'
  }
}

export async function composeVolume(
  datasets: NaturalizedDataset[],
  opts: ComposeOptions = {}
): Promise<Volume> {
  if (datasets.length === 0) throw new Error('composeVolume: empty input')

  const sorted = [...datasets].sort(opts.customOrder ?? defaultSliceOrder)
  const firstRows = numberAttr(sorted[0].Rows)
  const firstCols = numberAttr(sorted[0].Columns)
  if (!firstRows || !firstCols) {
    throw new Error('composeVolume: first slice has no Rows/Columns')
  }

  const slicePixels = firstRows * firstCols
  const voxels = new Float32Array(slicePixels * sorted.length)
  const slices: VolumeSlice[] = []
  let minV = Infinity
  let maxV = -Infinity
  let sum = 0

  for (let z = 0; z < sorted.length; z++) {
    const ds = sorted[z]
    const r = numberAttr(ds.Rows)
    const c = numberAttr(ds.Columns)
    if (r !== firstRows || c !== firstCols) {
      throw new VolumeShapeMismatchError(
        `Slice ${z} is ${r}x${c}, expected ${firstRows}x${firstCols}. ` +
        `Volume composition requires uniform slice geometry.`
      )
    }
    const frames = await decodeFrames(ds, { maxFrames: 1 })
    if (frames.length === 0) continue
    const frame = frames[0]

    const slope = opts.skipRescale ? 1 : (numberAttr(ds.RescaleSlope) || 1)
    const intercept = opts.skipRescale ? 0 : numberAttr(ds.RescaleIntercept)

    // The renderer in pixel_decode.ts produces RGBA luminance — for volumetric
    // analysis we need the underlying intensity. Re-read the raw bytes here so
    // signed/unsigned + bit depth are handled the same way as decode but we
    // skip the windowed display normalization.
    const intensities = rawIntensities(frame.rawPixelBytes, frame.bitsAllocated,
                                       frame.pixelRepresentation, slicePixels)
    const zBase = z * slicePixels
    for (let i = 0; i < slicePixels; i++) {
      const v = intensities[i] * slope + intercept
      voxels[zBase + i] = v
      if (v < minV) minV = v
      if (v > maxV) maxV = v
      sum += v
    }

    slices.push({
      index: z,
      dataset: ds,
      rows: r,
      columns: c,
      zPosition: extractZ(ds, z),
      rescaleSlope: slope,
      rescaleIntercept: intercept,
      rawPixelBytes: frame.rawPixelBytes,
      bitsAllocated: frame.bitsAllocated,
      pixelRepresentation: frame.pixelRepresentation,
    })
  }

  const totalVoxels = slicePixels * sorted.length
  return {
    voxels,
    shape: { z: sorted.length, y: firstRows, x: firstCols },
    slices,
    stats: {
      min: Number.isFinite(minV) ? minV : 0,
      max: Number.isFinite(maxV) ? maxV : 0,
      mean: totalVoxels > 0 ? sum / totalVoxels : 0,
    },
  }
}

/** Write voxel changes back to each slice's raw pixel bytes.
 *
 * `valueFn` receives the volume-space coordinate and returns the new voxel
 * intensity. Returning the original value is a no-op.
 *
 * Modifies the underlying DICOM PixelData buffer in place; no transcoding.
 */
export function writebackVolume(
  volume: Volume,
  valueFn: (x: number, y: number, z: number, currentValue: number) => number
): { modifiedSlices: number; modifiedVoxels: number } {
  let modifiedSlices = 0
  let modifiedVoxels = 0
  for (const slice of volume.slices) {
    const z = slice.index
    const slicePixels = slice.rows * slice.columns
    const base = z * slicePixels
    let sliceChanged = 0
    for (let y = 0; y < slice.rows; y++) {
      for (let x = 0; x < slice.columns; x++) {
        const i = y * slice.columns + x
        const current = volume.voxels[base + i]
        const next = valueFn(x, y, z, current)
        if (next !== current) {
          volume.voxels[base + i] = next
          writeIntensity(slice, i, next)
          sliceChanged++
        }
      }
    }
    if (sliceChanged > 0) {
      modifiedSlices++
      modifiedVoxels += sliceChanged
    }
  }
  return { modifiedSlices, modifiedVoxels }
}

// ── slice ordering ─────────────────────────────────────────────────────────

function defaultSliceOrder(a: NaturalizedDataset, b: NaturalizedDataset): number {
  const za = extractZ(a, 0)
  const zb = extractZ(b, 0)
  if (za !== zb) return za - zb
  const ia = numberAttr(a.InstanceNumber)
  const ib = numberAttr(b.InstanceNumber)
  return ia - ib
}

function extractZ(ds: NaturalizedDataset, fallback: number): number {
  const ipp = ds.ImagePositionPatient
  if (Array.isArray(ipp) && ipp.length >= 3) {
    const z = Number(ipp[2])
    if (Number.isFinite(z)) return z
  }
  const sloc = numberAttr(ds.SliceLocation)
  if (sloc !== 0) return sloc
  const inum = numberAttr(ds.InstanceNumber)
  return inum || fallback
}

// ── intensity I/O ──────────────────────────────────────────────────────────

function rawIntensities(
  bytes: Uint8Array,
  bitsAllocated: number,
  pixelRepresentation: number,
  pixels: number
): number[] | Int16Array | Uint16Array | Uint8Array | Int8Array {
  const bytesPerSample = Math.ceil(bitsAllocated / 8)
  if (bytesPerSample === 1) {
    return pixelRepresentation === 1
      ? new Int8Array(bytes.buffer, bytes.byteOffset, pixels)
      : new Uint8Array(bytes.buffer, bytes.byteOffset, pixels)
  }
  if (bytesPerSample === 2) {
    const view = new DataView(bytes.buffer, bytes.byteOffset, pixels * 2)
    if (pixelRepresentation === 1) {
      const out = new Int16Array(pixels)
      for (let i = 0; i < pixels; i++) out[i] = view.getInt16(i * 2, true)
      return out
    }
    const out = new Uint16Array(pixels)
    for (let i = 0; i < pixels; i++) out[i] = view.getUint16(i * 2, true)
    return out
  }
  // Anything else — return a no-op view; caller has bigger problems.
  return new Uint8Array(bytes.buffer, bytes.byteOffset, pixels)
}

function writeIntensity(slice: VolumeSlice, sliceIndex: number, intensity: number): void {
  // Reverse the slope/intercept transform when writing back.
  let raw = (intensity - slice.rescaleIntercept) / (slice.rescaleSlope || 1)
  if (!Number.isFinite(raw)) raw = 0

  const bytesPerSample = Math.ceil(slice.bitsAllocated / 8)
  const off = sliceIndex * bytesPerSample
  if (bytesPerSample === 1) {
    slice.rawPixelBytes[off] = clampToBits(raw, slice.bitsAllocated, slice.pixelRepresentation) & 0xff
    return
  }
  if (bytesPerSample === 2) {
    const w = clampToBits(raw, slice.bitsAllocated, slice.pixelRepresentation)
    slice.rawPixelBytes[off] = w & 0xff
    slice.rawPixelBytes[off + 1] = (w >> 8) & 0xff
    return
  }
  // Unusual depths — silently skip; we only support 8/16-bit storage today.
}

function clampToBits(value: number, bits: number, pixelRepresentation: number): number {
  const max = pixelRepresentation === 1 ? (1 << (bits - 1)) - 1 : (1 << bits) - 1
  const min = pixelRepresentation === 1 ? -(1 << (bits - 1)) : 0
  const rounded = Math.round(value)
  if (rounded < min) return min
  if (rounded > max) return max
  return rounded
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
