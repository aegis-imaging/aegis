import { describe, it, expect } from 'vitest'
import { composeVolume, writebackVolume, VolumeShapeMismatchError } from '../volume'
import type { NaturalizedDataset } from '../parser'

function uncompressedSlice(opts: {
  uid?: string
  z?: number
  rows?: number
  cols?: number
  bitsAllocated?: number
  pixelRepresentation?: number
  fill?: number
  rescaleSlope?: number
  rescaleIntercept?: number
  instanceNumber?: number
}): NaturalizedDataset {
  const rows = opts.rows ?? 4
  const cols = opts.cols ?? 4
  const ba = opts.bitsAllocated ?? 8
  const fill = opts.fill ?? 100
  const bytesPerSample = Math.ceil(ba / 8)
  const buf = new ArrayBuffer(rows * cols * bytesPerSample)
  if (bytesPerSample === 1) {
    new Uint8Array(buf).fill(fill)
  } else {
    new Uint16Array(buf).fill(fill)
  }
  return {
    _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.1' },
    Rows: rows,
    Columns: cols,
    BitsAllocated: ba,
    BitsStored: ba,
    SamplesPerPixel: 1,
    PhotometricInterpretation: 'MONOCHROME2',
    PixelRepresentation: opts.pixelRepresentation ?? 0,
    PixelData: buf,
    ImagePositionPatient: opts.z !== undefined ? [0, 0, opts.z] : undefined,
    InstanceNumber: opts.instanceNumber,
    RescaleSlope: opts.rescaleSlope,
    RescaleIntercept: opts.rescaleIntercept,
    SOPInstanceUID: opts.uid ?? `1.2.3.${opts.z ?? Math.random()}`,
  } as NaturalizedDataset
}

describe('composeVolume', () => {
  it('builds a Float32Array of shape [Z, Y, X]', async () => {
    const slices = [
      uncompressedSlice({ z: 0, fill: 10 }),
      uncompressedSlice({ z: 1, fill: 20 }),
      uncompressedSlice({ z: 2, fill: 30 }),
    ]
    const v = await composeVolume(slices)
    expect(v.shape).toEqual({ z: 3, y: 4, x: 4 })
    expect(v.voxels.length).toBe(48)
    // First slice (z=0) all 10s
    for (let i = 0; i < 16; i++) expect(v.voxels[i]).toBe(10)
    // Middle slice all 20s
    for (let i = 16; i < 32; i++) expect(v.voxels[i]).toBe(20)
  })

  it('sorts by ImagePositionPatient.z ascending', async () => {
    const slices = [
      uncompressedSlice({ z: 2, fill: 30 }),
      uncompressedSlice({ z: 0, fill: 10 }),
      uncompressedSlice({ z: 1, fill: 20 }),
    ]
    const v = await composeVolume(slices)
    expect(v.voxels[0]).toBe(10)            // z=0
    expect(v.voxels[16]).toBe(20)           // z=1
    expect(v.voxels[32]).toBe(30)           // z=2
  })

  it('falls back to InstanceNumber when no position', async () => {
    const slices = [
      uncompressedSlice({ instanceNumber: 3, fill: 30 }),
      uncompressedSlice({ instanceNumber: 1, fill: 10 }),
      uncompressedSlice({ instanceNumber: 2, fill: 20 }),
    ]
    const v = await composeVolume(slices)
    expect(v.voxels[0]).toBe(10)
    expect(v.voxels[16]).toBe(20)
    expect(v.voxels[32]).toBe(30)
  })

  it('applies rescale slope and intercept', async () => {
    const slices = [
      uncompressedSlice({ z: 0, fill: 100, rescaleSlope: 2, rescaleIntercept: -50 }),
    ]
    const v = await composeVolume(slices)
    expect(v.voxels[0]).toBe(100 * 2 + -50)
  })

  it('skips rescale when asked', async () => {
    const slices = [
      uncompressedSlice({ z: 0, fill: 100, rescaleSlope: 5, rescaleIntercept: 1000 }),
    ]
    const v = await composeVolume(slices, { skipRescale: true })
    expect(v.voxels[0]).toBe(100)
  })

  it('throws VolumeShapeMismatchError for inconsistent geometry', async () => {
    const slices = [
      uncompressedSlice({ z: 0, rows: 4, cols: 4 }),
      uncompressedSlice({ z: 1, rows: 8, cols: 4 }),
    ]
    await expect(composeVolume(slices)).rejects.toBeInstanceOf(VolumeShapeMismatchError)
  })

  it('computes min/max/mean over the volume', async () => {
    const slices = [
      uncompressedSlice({ z: 0, fill: 10 }),
      uncompressedSlice({ z: 1, fill: 20 }),
    ]
    const v = await composeVolume(slices)
    expect(v.stats.min).toBe(10)
    expect(v.stats.max).toBe(20)
    expect(v.stats.mean).toBe(15)
  })
})

describe('writebackVolume', () => {
  it('writes new values back into raw pixel bytes (8-bit)', async () => {
    const slices = [uncompressedSlice({ z: 0, fill: 100 })]
    const v = await composeVolume(slices)
    const { modifiedSlices, modifiedVoxels } = writebackVolume(v, (x, y) => {
      if (y < 2) return 0  // zero top half
      return v.voxels[y * 4 + x]  // unchanged
    })
    expect(modifiedSlices).toBe(1)
    expect(modifiedVoxels).toBe(8)
    // Check the raw bytes — top half should be 0
    const bytes = new Uint8Array(slices[0].PixelData as ArrayBuffer)
    expect(bytes[0]).toBe(0)
    expect(bytes[7]).toBe(0)
    expect(bytes[8]).toBe(100)  // unchanged row 2 col 0
  })

  it('writes new values back to 16-bit pixel buffers correctly', async () => {
    const slices = [uncompressedSlice({ z: 0, fill: 1000, bitsAllocated: 16 })]
    const v = await composeVolume(slices)
    writebackVolume(v, (_x, _y) => 0)
    const view = new Uint16Array(slices[0].PixelData as ArrayBuffer)
    expect(view[0]).toBe(0)
    expect(view[15]).toBe(0)
  })

  it('reverses rescale slope/intercept on writeback', async () => {
    // With slope=2, intercept=-50, raw value 100 → 150 in volume space.
    // Writing back 200 in volume space should produce raw value 125.
    const slices = [uncompressedSlice({
      z: 0, fill: 100, rescaleSlope: 2, rescaleIntercept: -50, bitsAllocated: 8,
    })]
    const v = await composeVolume(slices)
    writebackVolume(v, () => 200)
    const bytes = new Uint8Array(slices[0].PixelData as ArrayBuffer)
    expect(bytes[0]).toBe(125)  // (200 - -50) / 2
  })

  it('only writes voxels that actually changed', async () => {
    const slices = [uncompressedSlice({ z: 0, fill: 50 })]
    const v = await composeVolume(slices)
    const { modifiedVoxels } = writebackVolume(v, (_x, _y, _z, current) => current)
    expect(modifiedVoxels).toBe(0)
  })
})
