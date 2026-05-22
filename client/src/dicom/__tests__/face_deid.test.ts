import { describe, it, expect } from 'vitest'
import { defaceStudy, studyLooksLikeHeadScan } from '../face_deid'
import type { ParsedDicomFile } from '../../types'

function makeFile(opts: {
  modality?: string
  bodyPart?: string
  studyUid?: string
  filename?: string
}): ParsedDicomFile {
  const modality = opts.modality ?? 'MR'
  const bodyPart = opts.bodyPart ?? 'HEAD'
  return {
    filename: opts.filename ?? 'a.dcm',
    studyInstanceUid: opts.studyUid ?? '1.2.3',
    seriesInstanceUid: '1.2.3.1',
    sopInstanceUid: '1.2.3.1.1',
    modality,
    seriesDescription: '',
    tags: [{ tag: '00180015', keyword: 'BodyPartExamined', vr: 'CS', originalValue: bodyPart, anonymizedValue: bodyPart, action: 'K' }],
    arrayBuffer: new ArrayBuffer(0),
  }
}

describe('studyLooksLikeHeadScan', () => {
  it('flags head MR series with enough slices', () => {
    const files = Array.from({ length: 20 }, () => makeFile({}))
    expect(studyLooksLikeHeadScan(files)).toBe(true)
  })

  it('flags head CT', () => {
    const files = Array.from({ length: 20 }, () => makeFile({ modality: 'CT' }))
    expect(studyLooksLikeHeadScan(files)).toBe(true)
  })

  it('flags brain MR even when body part is omitted', () => {
    const files = Array.from({ length: 20 }, () => makeFile({ bodyPart: '' }))
    expect(studyLooksLikeHeadScan(files)).toBe(true)
  })

  it('rejects single-shot ultrasound', () => {
    const files = [makeFile({ modality: 'US' })]
    expect(studyLooksLikeHeadScan(files)).toBe(false)
  })

  it('rejects body MR', () => {
    const files = Array.from({ length: 20 }, () => makeFile({ bodyPart: 'CHEST' }))
    expect(studyLooksLikeHeadScan(files)).toBe(false)
  })

  it('rejects too-short series', () => {
    const files = Array.from({ length: 5 }, () => makeFile({}))
    expect(studyLooksLikeHeadScan(files)).toBe(false)
  })
})

describe('defaceStudy applicability gates', () => {
  it('returns no_op_short_series for tiny series', async () => {
    const files = [makeFile({})]
    const r = await defaceStudy('1.2.3', files)
    expect(r.status).toBe('no_op_short_series')
    expect(r.reason).toMatch(/fewer than 10/)
  })

  it('returns no_op_wrong_modality for ultrasound', async () => {
    const files = Array.from({ length: 20 }, () => makeFile({ modality: 'US' }))
    const r = await defaceStudy('1.2.3', files)
    expect(r.status).toBe('no_op_wrong_modality')
  })

  it('returns no_op_wrong_body_part for chest CT', async () => {
    const files = Array.from({ length: 20 }, () => makeFile({ modality: 'CT', bodyPart: 'CHEST' }))
    const r = await defaceStudy('1.2.3', files)
    expect(r.status).toBe('no_op_wrong_body_part')
  })

  it('returns not_implemented when tfjs-model strategy requested', async () => {
    const files = Array.from({ length: 20 }, () => makeFile({}))
    const r = await defaceStudy('1.2.3', files, { strategy: 'tfjs-model' })
    expect(r.status).toBe('not_implemented')
    expect(r.strategyUsed).toBe('tfjs-model')
  })
})
