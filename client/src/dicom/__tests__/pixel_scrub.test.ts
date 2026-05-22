import { describe, it, expect, beforeEach, vi } from 'vitest'
import { scrubInstance, scrubStudyDatasets } from '../pixel_scrub'
import * as pixelOcr from '../pixel_ocr'
import type { NaturalizedDataset } from '../parser'

function uncompressedDataset(extra: Partial<NaturalizedDataset> = {}): NaturalizedDataset {
  return {
    _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.1' },
    Rows: 8,
    Columns: 8,
    BitsAllocated: 8,
    BitsStored: 8,
    SamplesPerPixel: 1,
    PhotometricInterpretation: 'MONOCHROME2',
    PixelRepresentation: 0,
    // 8x8 grayscale: top-left quadrant nonzero so a "redaction" change is observable.
    PixelData: new Uint8Array([
      200, 200, 200, 200, 0, 0, 0, 0,
      200, 200, 200, 200, 0, 0, 0, 0,
      200, 200, 200, 200, 0, 0, 0, 0,
      200, 200, 200, 200, 0, 0, 0, 0,
      0,   0,   0,   0,   0, 0, 0, 0,
      0,   0,   0,   0,   0, 0, 0, 0,
      0,   0,   0,   0,   0, 0, 0, 0,
      0,   0,   0,   0,   0, 0, 0, 0,
    ]).buffer,
    ...extra,
  }
}

const mockEngine = (findings: pixelOcr.OcrFinding[]) => ({
  recognise: vi.fn().mockResolvedValue(findings),
  terminate: vi.fn().mockResolvedValue(undefined),
})

beforeEach(() => {
  pixelOcr._resetEngineForTests()
  // Patch the dynamic import in pixel_ocr to a fake engine. Each test wires
  // its own getOcrEngine return so we can drive findings.
})

describe('scrubInstance', () => {
  it('returns no_pixel_data for SR-style datasets', async () => {
    const ds: NaturalizedDataset = { _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.1' } }
    const r = await scrubInstance(ds)
    expect(r.status).toBe('no_pixel_data')
  })

  it('returns unsupported_transfer_syntax for JPEG 2000', async () => {
    const ds = uncompressedDataset({ _meta: { TransferSyntaxUID: '1.2.840.10008.1.2.4.91' } })
    const r = await scrubInstance(ds)
    expect(r.status).toBe('unsupported_transfer_syntax')
    expect(r.transferSyntaxLabel).toBe('JPEG 2000')
  })

  it('returns no_findings when OCR returns nothing', async () => {
    vi.spyOn(pixelOcr, 'getOcrEngine').mockResolvedValue(mockEngine([]) as never)
    const ds = uncompressedDataset()
    const r = await scrubInstance(ds)
    expect(r.status).toBe('no_findings')
    expect(r.totalFindings).toBe(0)
    expect(r.datasetMutated).toBe(false)
  })

  it('burns a redaction at the finding bounding box and mutates PixelData', async () => {
    vi.spyOn(pixelOcr, 'getOcrEngine').mockResolvedValue(mockEngine([
      { text: 'JANE^DOE', confidence: 0.95, x: 0, y: 0, width: 3, height: 2 },
    ]) as never)
    const ds = uncompressedDataset()
    const beforeBytes = new Uint8Array((ds.PixelData as ArrayBuffer).slice(0))
    const r = await scrubInstance(ds, { paddingPixels: 0 })
    expect(r.status).toBe('scrubbed')
    expect(r.totalFindings).toBe(1)
    expect(r.datasetMutated).toBe(true)
    const after = new Uint8Array(ds.PixelData as ArrayBuffer)
    // Top-left 3x2 region should be zero now.
    for (let y = 0; y < 2; y++) {
      for (let x = 0; x < 3; x++) {
        expect(after[y * 8 + x]).toBe(0)
      }
    }
    // Pixel just outside the box should be unchanged.
    expect(after[2 * 8 + 3]).toBe(beforeBytes[2 * 8 + 3])
  })

  it('handles multiple findings per frame', async () => {
    vi.spyOn(pixelOcr, 'getOcrEngine').mockResolvedValue(mockEngine([
      { text: 'SMITH', confidence: 0.9, x: 0, y: 0, width: 2, height: 2 },
      { text: '12345', confidence: 0.85, x: 5, y: 4, width: 2, height: 2 },
    ]) as never)
    const ds = uncompressedDataset()
    const r = await scrubInstance(ds, { paddingPixels: 0 })
    expect(r.totalFindings).toBe(2)
    expect(r.perFrameFindings[0]).toHaveLength(2)
  })

  it('respects padding around bounding boxes', async () => {
    vi.spyOn(pixelOcr, 'getOcrEngine').mockResolvedValue(mockEngine([
      { text: 'X', confidence: 0.9, x: 2, y: 2, width: 1, height: 1 },
    ]) as never)
    const ds = uncompressedDataset()
    await scrubInstance(ds, { paddingPixels: 2 })
    const after = new Uint8Array(ds.PixelData as ArrayBuffer)
    // With padding=2, region [0-4) x [0-4) should be zero (clamped at edges).
    for (let y = 0; y < 4; y++) {
      for (let x = 0; x < 4; x++) {
        expect(after[y * 8 + x]).toBe(0)
      }
    }
  })
})

describe('scrubStudyDatasets', () => {
  it('aggregates findings across files and calls onProgress per file', async () => {
    vi.spyOn(pixelOcr, 'getOcrEngine').mockResolvedValue(mockEngine([
      { text: 'NAME', confidence: 0.9, x: 0, y: 0, width: 2, height: 2 },
    ]) as never)
    const datasets = [uncompressedDataset(), uncompressedDataset(), uncompressedDataset()]
    const progressCalls: number[] = []
    const summary = await scrubStudyDatasets('1.2.3', datasets, {
      paddingPixels: 0,
      onProgress: p => { progressCalls.push(p.fileIndex) },
    })
    expect(summary.filesScrubbed).toBe(3)
    expect(summary.totalFindings).toBe(3)
    expect(progressCalls).toEqual([0, 1, 2])
  })

  it('stops early when onProgress returns false', async () => {
    vi.spyOn(pixelOcr, 'getOcrEngine').mockResolvedValue(mockEngine([]) as never)
    const datasets = [uncompressedDataset(), uncompressedDataset(), uncompressedDataset()]
    const summary = await scrubStudyDatasets('1.2.3', datasets, {
      onProgress: p => p.fileIndex < 1, // cancel after the 2nd
    })
    expect(summary.perFile).toHaveLength(2)
  })

  it('continues past failures and counts them', async () => {
    let call = 0
    vi.spyOn(pixelOcr, 'getOcrEngine').mockImplementation(async () => ({
      recognise: vi.fn().mockImplementation(async () => {
        if (call++ === 1) throw new Error('OCR boom')
        return []
      }),
      terminate: vi.fn(),
    } as never))
    const datasets = [uncompressedDataset(), uncompressedDataset(), uncompressedDataset()]
    const summary = await scrubStudyDatasets('1.2.3', datasets)
    expect(summary.filesFailed).toBe(1)
    expect(summary.perFile).toHaveLength(3)
  })
})
