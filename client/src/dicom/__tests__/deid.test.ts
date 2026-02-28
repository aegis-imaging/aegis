import { describe, it, expect } from 'vitest'
import { deidentify } from '../deid'
import type { NaturalizedDataset } from '../parser'

/** Create a minimal dataset that exercises the main action types */
function makeDataset(overrides: Record<string, unknown> = {}): NaturalizedDataset {
  return {
    PatientName: 'DOE^JOHN',
    PatientID: 'MRN-12345',
    PatientBirthDate: '19800115',
    PatientSex: 'M',
    ReferringPhysicianName: 'SMITH^ALICE',
    InstitutionName: 'General Hospital',
    StudyDate: '20240301',
    StudyTime: '143000',
    SeriesDate: '20240301',
    AcquisitionDate: '20240301',
    ContentDate: '20240301',
    AccessionNumber: 'ACC-99999',
    StudyInstanceUID: '1.2.840.113619.2.55.3.12345',
    SeriesInstanceUID: '1.2.840.113619.2.55.3.12345.1',
    SOPInstanceUID: '1.2.840.113619.2.55.3.12345.1.1',
    SOPClassUID: '1.2.840.10008.5.1.4.1.1.2',
    Modality: 'CT',
    Manufacturer: 'SIEMENS',
    StudyDescription: 'CT CHEST WITH CONTRAST',
    SeriesDescription: 'AXIAL 3MM',
    BodyPartExamined: 'CHEST',
    Rows: 512,
    Columns: 512,
    ...overrides,
  } as unknown as NaturalizedDataset
}

describe('deidentify', () => {
  it('pseudonymizes PatientName and PatientID', async () => {
    const ds = makeDataset()
    const result = await deidentify(ds)
    // PatientName → ANON-<hash8>, PatientID → SUBJ-<hash8>
    expect(ds.PatientName).toMatch(/^ANON-[0-9a-f]{8}$/)
    expect(ds.PatientID).toMatch(/^SUBJ-[0-9a-f]{8}$/)
    // Changes should be recorded
    const nameChange = result.tagChanges.find(c => c.keyword === 'PatientName')
    expect(nameChange).toBeDefined()
    expect(nameChange!.action).toBe('Z')
    expect(nameChange!.originalValue).toBe('DOE^JOHN')
    expect(nameChange!.anonymizedValue).toMatch(/^ANON-/)
    // PatientID mapping should be populated
    expect(result.patientIdMapping).toBeDefined()
    expect(result.patientIdMapping!.original).toBe('MRN-12345')
    expect(result.patientIdMapping!.replacement).toMatch(/^SUBJ-/)
  })

  it('removes tags marked X', async () => {
    const ds = makeDataset()
    await deidentify(ds)
    // ReferringPhysicianName is Z (zeroed), not X
    expect(ds.ReferringPhysicianName).toBe('')
    // InstitutionName is X (removed)
    expect(ds.InstitutionName).toBeUndefined()
    expect(ds.SeriesDate).toBeUndefined()
    expect(ds.AcquisitionDate).toBeUndefined()
  })

  it('zeroes StudyDate and StudyTime', async () => {
    const ds = makeDataset()
    await deidentify(ds)
    expect(ds.StudyDate).toBe('')
    expect(ds.StudyTime).toBe('')
  })

  it('keeps safe tags (K action)', async () => {
    const ds = makeDataset()
    await deidentify(ds)
    expect(ds.Modality).toBe('CT')
    expect(ds.Manufacturer).toBe('SIEMENS')
    expect(ds.PatientSex).toBe('M')
    expect(ds.BodyPartExamined).toBe('CHEST')
    expect(ds.Rows).toBe(512)
  })

  it('replaces UIDs deterministically', async () => {
    const ds = makeDataset()
    const originalStudyUid = ds.StudyInstanceUID as string
    const result = await deidentify(ds, { salt: 'test-salt' })

    expect(ds.StudyInstanceUID).not.toBe(originalStudyUid)
    expect((ds.StudyInstanceUID as string).startsWith('2.25.')).toBe(true)

    // Same input → same output (deterministic)
    const ds2 = makeDataset()
    await deidentify(ds2, { salt: 'test-salt' })
    expect(ds2.StudyInstanceUID).toBe(ds.StudyInstanceUID)
  })

  it('uses different UIDs with different salts', async () => {
    const ds1 = makeDataset()
    const ds2 = makeDataset()
    await deidentify(ds1, { salt: 'salt-a' })
    await deidentify(ds2, { salt: 'salt-b' })
    expect(ds1.StudyInstanceUID).not.toBe(ds2.StudyInstanceUID)
  })

  it('keeps SOPClassUID unchanged', async () => {
    const ds = makeDataset()
    const original = ds.SOPClassUID
    await deidentify(ds)
    expect(ds.SOPClassUID).toBe(original)
  })

  it('cleans C-action tags conditionally', async () => {
    // No PHI in description → keep
    const ds1 = makeDataset({ StudyDescription: 'CT CHEST WITH CONTRAST' })
    await deidentify(ds1)
    expect(ds1.StudyDescription).toBe('CT CHEST WITH CONTRAST')

    // PHI in description → scrub tokens, preserving non-PHI
    const ds2 = makeDataset({ StudyDescription: 'CT for John Smith MRN: 12345' })
    await deidentify(ds2)
    const scrubbed = ds2.StudyDescription as string
    expect(scrubbed).toContain('CT for')
    expect(scrubbed).toContain('[REMOVED]')
    expect(scrubbed).not.toContain('John Smith')
    expect(scrubbed).not.toContain('12345')
  })

  it('respects retainedTags option', async () => {
    const ds = makeDataset()
    await deidentify(ds, { retainedTags: ['StudyDate', 'PatientBirthDate'] })
    expect(ds.StudyDate).toBe('20240301')
    expect(ds.PatientBirthDate).toBe('19800115')
  })

  it('removes private tags by default', async () => {
    const ds = makeDataset({ '00091001': 'private data' })
    const result = await deidentify(ds)
    expect((ds as Record<string, unknown>)['00091001']).toBeUndefined()
    expect(result.privateTagsRemoved).toBeGreaterThan(0)
  })

  it('keeps private tags when keepPrivateTags is true', async () => {
    const ds = makeDataset({ '00091001': 'private data' })
    await deidentify(ds, { keepPrivateTags: true })
    expect((ds as Record<string, unknown>)['00091001']).toBe('private data')
  })

  it('returns tagChanges sorted by action priority', async () => {
    const ds = makeDataset()
    const result = await deidentify(ds)
    const actions = result.tagChanges.map(c => c.action)
    const order: Record<string, number> = { X: 0, Z: 1, D: 2, U: 3, C: 4, K: 5 }
    for (let i = 1; i < actions.length; i++) {
      expect(order[actions[i]]).toBeGreaterThanOrEqual(order[actions[i - 1]])
    }
  })
})
