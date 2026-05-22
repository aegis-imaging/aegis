import { describe, it, expect, vi, beforeEach } from 'vitest'
import { bulkUploadStudies } from '../bulk'
import * as uploadClient from '../client'
import type { BulkStudy, BulkStudyState } from '../bulk'

function fakeStudy(uid: string, fileCount = 1): BulkStudy {
  return {
    studyInstanceUid: uid,
    files: Array.from({ length: fileCount }, (_, i) => ({
      filename: `${uid}-${i}.dcm`,
      studyInstanceUid: uid,
      seriesInstanceUid: `${uid}.1`,
      sopInstanceUid: `${uid}.1.${i}`,
      modality: 'MR',
      seriesDescription: '',
      tags: [],
      arrayBuffer: new ArrayBuffer(0),
    })),
    summary: {
      patientName: 'X', patientId: `p-${uid}`, studyDate: '', studyTime: '',
      timezoneOffsetFromUtc: '', studyDateTimeIso: '',
      studyDescription: `study ${uid}`, modality: 'MR',
      studyInstanceUid: uid, seriesCount: 1, imageCount: fileCount, bodyPart: 'HEAD',
    },
  }
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('bulkUploadStudies', () => {
  it('uploads each study sequentially when concurrency=1', async () => {
    const order: string[] = []
    vi.spyOn(uploadClient, 'uploadStudy').mockImplementation((async (_files: unknown, _slug: unknown, summary: { studyInstanceUid: string }) => {
      order.push(summary.studyInstanceUid)
      await new Promise(r => setTimeout(r, 5))
      return { sessionId: 's', status: 'completed' }
    }) as never)
    const studies = [fakeStudy('A'), fakeStudy('B'), fakeStudy('C')]
    const result = await bulkUploadStudies(studies, { projectSlug: 'p', concurrency: 1 })
    expect(result.studiesUploaded).toBe(3)
    expect(result.studiesFailed).toBe(0)
    expect(order).toEqual(['A', 'B', 'C'])
  })

  it('runs concurrent uploads when concurrency>1', async () => {
    let inflight = 0
    let peakInflight = 0
    vi.spyOn(uploadClient, 'uploadStudy').mockImplementation(async () => {
      inflight++
      peakInflight = Math.max(peakInflight, inflight)
      await new Promise(r => setTimeout(r, 20))
      inflight--
      return { sessionId: 's', status: 'completed' }
    })
    const studies = ['A', 'B', 'C', 'D', 'E'].map(s => fakeStudy(s))
    const result = await bulkUploadStudies(studies, { projectSlug: 'p', concurrency: 3 })
    expect(result.studiesUploaded).toBe(5)
    expect(peakInflight).toBeGreaterThanOrEqual(2)
    expect(peakInflight).toBeLessThanOrEqual(3)
  })

  it('classifies a 409 error as duplicate, not failure', async () => {
    vi.spyOn(uploadClient, 'uploadStudy').mockRejectedValue(new Error('409: duplicate StudyInstanceUID'))
    const studies = [fakeStudy('A')]
    const result = await bulkUploadStudies(studies, { projectSlug: 'p' })
    expect(result.studiesUploaded).toBe(0)
    expect(result.studiesDuplicate).toBe(1)
    expect(result.studiesFailed).toBe(0)
    expect(result.studies[0].status).toBe('duplicate')
  })

  it('continues past one failing study', async () => {
    let call = 0
    vi.spyOn(uploadClient, 'uploadStudy').mockImplementation(async () => {
      if (call++ === 1) throw new Error('boom')
      return { sessionId: 's', status: 'completed' }
    })
    const studies = [fakeStudy('A'), fakeStudy('B'), fakeStudy('C')]
    const result = await bulkUploadStudies(studies, { projectSlug: 'p' })
    expect(result.studiesUploaded).toBe(2)
    expect(result.studiesFailed).toBe(1)
    expect(result.studies[1].status).toBe('failed')
    expect(result.studies[1].error).toMatch(/boom/)
  })

  it('emits progress callbacks for every state transition', async () => {
    vi.spyOn(uploadClient, 'uploadStudy').mockResolvedValue({ sessionId: 's', status: 'completed' })
    const phases: string[] = []
    const studies = [fakeStudy('A'), fakeStudy('B')]
    await bulkUploadStudies(studies, {
      projectSlug: 'p',
      concurrency: 1,
      onProgress: p => phases.push(p.phase),
    })
    expect(phases[0]).toBe('uploading')
    expect(phases[phases.length - 1]).toBe('done')
  })

  it('calls onStudyDone exactly once per study with its final state', async () => {
    vi.spyOn(uploadClient, 'uploadStudy').mockResolvedValue({ sessionId: 's', status: 'completed' })
    const finals: BulkStudyState[] = []
    const studies = [fakeStudy('A'), fakeStudy('B')]
    await bulkUploadStudies(studies, {
      projectSlug: 'p',
      onStudyDone: s => finals.push({ ...s }),
    })
    expect(finals.map(s => s.studyInstanceUid).sort()).toEqual(['A', 'B'])
    expect(finals.every(s => s.status === 'completed')).toBe(true)
  })

  it('handles empty input gracefully', async () => {
    const r = await bulkUploadStudies([], { projectSlug: 'p' })
    expect(r.studies).toEqual([])
    expect(r.studiesUploaded).toBe(0)
  })
})
