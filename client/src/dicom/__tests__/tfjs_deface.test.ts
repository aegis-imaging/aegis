import { describe, it, expect } from 'vitest'
import { defaceStudy } from '../face_deid'
import { TFJSUnavailableError } from '../tfjs_deface'
import type { ParsedDicomFile } from '../../types'

function fakeFile(opts: { mod?: string; bodyPart?: string; filename?: string } = {}): ParsedDicomFile {
  return {
    filename: opts.filename ?? 'a.dcm',
    studyInstanceUid: '1.2.3',
    seriesInstanceUid: '1.2.3.1',
    sopInstanceUid: '1.2.3.1.1',
    modality: opts.mod ?? 'MR',
    seriesDescription: '',
    tags: [{
      tag: '00180015', keyword: 'BodyPartExamined', vr: 'CS',
      originalValue: opts.bodyPart ?? 'HEAD', anonymizedValue: opts.bodyPart ?? 'HEAD',
      action: 'K',
    }],
    arrayBuffer: new ArrayBuffer(0),
  }
}

describe('defaceStudy(tfjs-model)', () => {
  it('returns not_implemented + clear reason when modelUrl is missing', async () => {
    const files = Array.from({ length: 20 }, () => fakeFile())
    const r = await defaceStudy('1.2.3', files, { strategy: 'tfjs-model' })
    expect(r.status).toBe('not_implemented')
    expect(r.strategyUsed).toBe('tfjs-model')
    expect(r.reason).toMatch(/modelUrl/)
  })

  it('still gates by modality before attempting TF.js', async () => {
    const files = Array.from({ length: 20 }, () => fakeFile({ mod: 'US' }))
    const r = await defaceStudy('1.2.3', files, {
      strategy: 'tfjs-model',
      modelUrl: 'https://example.com/m.json',
    })
    expect(r.status).toBe('no_op_wrong_modality')
  })

  it('still gates by body part before attempting TF.js', async () => {
    const files = Array.from({ length: 20 }, () => fakeFile({ bodyPart: 'CHEST' }))
    const r = await defaceStudy('1.2.3', files, {
      strategy: 'tfjs-model',
      modelUrl: 'https://example.com/m.json',
    })
    expect(r.status).toBe('no_op_wrong_body_part')
  })
})

describe('TFJSUnavailableError', () => {
  it('is an Error subclass with stable name', () => {
    const e = new TFJSUnavailableError('no backend')
    expect(e).toBeInstanceOf(Error)
    expect(e.name).toBe('TFJSUnavailableError')
    expect(e.message).toContain('no backend')
  })
})
