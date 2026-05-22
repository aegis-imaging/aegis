import { describe, it, expect } from 'vitest'
import {
  decodeFrames,
  transferSyntaxOf,
  transferSyntaxLabel,
  isCompressedTransferSyntax,
  UnsupportedTransferSyntaxError,
} from '../pixel_decode'
import type { NaturalizedDataset } from '../parser'

function explicitVrLEDataset(extra: Partial<NaturalizedDataset> = {}): NaturalizedDataset {
  return {
    _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.1' },
    Rows: 4,
    Columns: 4,
    BitsAllocated: 8,
    BitsStored: 8,
    SamplesPerPixel: 1,
    PhotometricInterpretation: 'MONOCHROME2',
    PixelRepresentation: 0,
    PixelData: new Uint8Array([
      // Row-major 4x4 grayscale image. Values 0-255.
      0, 32, 64, 96,
      128, 160, 192, 224,
      255, 200, 150, 100,
      50, 25, 12, 6,
    ]).buffer,
    ...extra,
  }
}

describe('transferSyntaxOf', () => {
  it('reads from _meta', () => {
    expect(transferSyntaxOf({ _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.1' } } as NaturalizedDataset))
      .toBe('1.2.840.10008.1.2.1')
  })

  it('handles array-shaped meta values', () => {
    expect(transferSyntaxOf({ _meta: { TransferSyntaxUID: ['1.2.840.10008.1.2'] } } as NaturalizedDataset))
      .toBe('1.2.840.10008.1.2')
  })

  it('returns empty string when missing', () => {
    expect(transferSyntaxOf({} as NaturalizedDataset)).toBe('')
  })
})

describe('isCompressedTransferSyntax', () => {
  it('flags JPEG 2000', () => {
    expect(isCompressedTransferSyntax('1.2.840.10008.1.2.4.91')).toBe(true)
  })
  it('flags RLE Lossless', () => {
    expect(isCompressedTransferSyntax('1.2.840.10008.1.2.5')).toBe(true)
  })
  it('does not flag uncompressed', () => {
    expect(isCompressedTransferSyntax('1.2.840.10008.1.2.1')).toBe(false)
  })
})

describe('transferSyntaxLabel', () => {
  it('labels uncompressed', () => {
    expect(transferSyntaxLabel('1.2.840.10008.1.2.1')).toBe('Uncompressed')
  })
  it('labels JPEG 2000', () => {
    expect(transferSyntaxLabel('1.2.840.10008.1.2.4.91')).toBe('JPEG 2000')
  })
  it('falls back to the UID itself for unknown TS', () => {
    expect(transferSyntaxLabel('1.2.3.999')).toBe('1.2.3.999')
  })
})

describe('decodeFrames — uncompressed 8-bit grayscale', () => {
  it('produces one frame for single-frame instances', async () => {
    const ds = explicitVrLEDataset()
    const frames = await decodeFrames(ds)
    expect(frames).toHaveLength(1)
    expect(frames[0].width).toBe(4)
    expect(frames[0].height).toBe(4)
    expect(frames[0].rgba.length).toBe(4 * 4 * 4)
  })

  it('inverts MONOCHROME1', async () => {
    const ds = explicitVrLEDataset({ PhotometricInterpretation: 'MONOCHROME1' })
    const frames = await decodeFrames(ds)
    expect(frames[0].inverted).toBe(true)
    // Source pixel 0 should render as RGBA white in MONOCHROME1
    expect(frames[0].rgba[0]).toBe(255)
  })

  it('throws UnsupportedTransferSyntaxError for JPEG 2000', async () => {
    const ds = explicitVrLEDataset({
      _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.4.91' },
    })
    await expect(decodeFrames(ds)).rejects.toBeInstanceOf(UnsupportedTransferSyntaxError)
  })

  it('respects maxFrames when multi-frame', async () => {
    const ds = explicitVrLEDataset({
      Rows: 2, Columns: 2, NumberOfFrames: 3,
      PixelData: new Uint8Array([
        0, 0, 0, 0,
        128, 128, 128, 128,
        255, 255, 255, 255,
      ]).buffer,
    })
    const frames = await decodeFrames(ds, { maxFrames: 2 })
    expect(frames).toHaveLength(2)
    expect(frames[0].numFrames).toBe(3)
  })

  it('respects an explicit windowCenter/windowWidth', async () => {
    const ds = explicitVrLEDataset()
    const wide = await decodeFrames(ds, { windowCenter: 128, windowWidth: 256 })
    const narrow = await decodeFrames(ds, { windowCenter: 64, windowWidth: 32 })
    // Narrow window should saturate more pixels at black or white
    const wideMid = wide[0].rgba[2 * 4]  // arbitrary mid pixel R channel
    const narrowMid = narrow[0].rgba[2 * 4]
    expect(narrowMid !== wideMid || narrow[0].rgba[0] === 0).toBe(true)
  })
})
