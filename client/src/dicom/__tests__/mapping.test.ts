import { describe, it, expect } from 'vitest'
import { createMappingCollector } from '../mapping'
import { deidentify } from '../deid'
import type { NaturalizedDataset } from '../parser'

function makeDataset(overrides: Record<string, unknown> = {}): NaturalizedDataset {
  return {
    PatientName: 'DOE^JOHN',
    PatientID: 'MRN-12345',
    StudyDate: '20240301',
    StudyInstanceUID: '1.2.3.4.5',
    SeriesInstanceUID: '1.2.3.4.5.1',
    SOPInstanceUID: '1.2.3.4.5.1.1',
    SOPClassUID: '1.2.840.10008.5.1.4.1.1.2',
    Modality: 'CT',
    ...overrides,
  } as unknown as NaturalizedDataset
}

describe('DeidResult mappings', () => {
  it('returns UID mappings', async () => {
    const ds = makeDataset()
    const result = await deidentify(ds, { salt: 'test' })
    expect(result.uidMappings.size).toBeGreaterThan(0)
    expect(result.uidMappings.has('1.2.3.4.5')).toBe(true)
    expect(result.uidMappings.get('1.2.3.4.5')!.startsWith('2.25.')).toBe(true)
  })

  it('returns PatientID mapping with pseudonym', async () => {
    const ds = makeDataset()
    const result = await deidentify(ds, { salt: 'test' })
    expect(result.patientIdMapping).toBeDefined()
    expect(result.patientIdMapping!.original).toBe('MRN-12345')
    expect(result.patientIdMapping!.replacement).toMatch(/^SUBJ-[0-9a-f]{8}$/)
  })

  it('pseudonymizes PatientName', async () => {
    const ds = makeDataset()
    await deidentify(ds, { salt: 'test' })
    expect(ds.PatientName).toMatch(/^ANON-[0-9a-f]{8}$/)
  })

  it('PatientID pseudonym is deterministic', async () => {
    const ds1 = makeDataset()
    const ds2 = makeDataset()
    const r1 = await deidentify(ds1, { salt: 'test' })
    const r2 = await deidentify(ds2, { salt: 'test' })
    expect(r1.patientIdMapping!.replacement).toBe(r2.patientIdMapping!.replacement)
  })

  it('returns dateShiftOffset when date shifting', async () => {
    const ds = makeDataset()
    const result = await deidentify(ds, { dateShift: { offsetDays: 42 } })
    expect(result.dateShiftOffset).toBe(42)
  })

  it('dateShiftOffset is undefined when not shifting', async () => {
    const ds = makeDataset()
    const result = await deidentify(ds)
    expect(result.dateShiftOffset).toBeUndefined()
  })
})

describe('MappingCollector', () => {
  it('collects UID mappings from multiple results', async () => {
    const collector = createMappingCollector()

    const ds1 = makeDataset()
    const r1 = await deidentify(ds1, { salt: 'test' })
    collector.record(r1)

    const ds2 = makeDataset({
      PatientID: 'MRN-99999',
      PatientName: 'SMITH^JANE',
      StudyInstanceUID: '9.8.7.6.5',
      SeriesInstanceUID: '9.8.7.6.5.1',
      SOPInstanceUID: '9.8.7.6.5.1.1',
    })
    const r2 = await deidentify(ds2, { salt: 'test' })
    collector.record(r2)

    const mappings = collector.getMappings()
    // Both studies' UIDs should be present
    expect(mappings.uids.has('1.2.3.4.5')).toBe(true)
    expect(mappings.uids.has('9.8.7.6.5')).toBe(true)
    // Both patient IDs
    expect(mappings.patientIds.has('MRN-12345')).toBe(true)
    expect(mappings.patientIds.has('MRN-99999')).toBe(true)
  })

  it('deduplicates same UID mapped twice', async () => {
    const collector = createMappingCollector()

    const ds1 = makeDataset()
    const r1 = await deidentify(ds1, { salt: 'test' })
    collector.record(r1)

    // Same dataset again (same UIDs)
    const ds2 = makeDataset()
    const r2 = await deidentify(ds2, { salt: 'test' })
    collector.record(r2)

    const mappings = collector.getMappings()
    // Should have only 3 UIDs (StudyInstanceUID, SeriesInstanceUID, SOPInstanceUID)
    expect(mappings.uids.size).toBe(3)
  })

  it('exports UID CSV', async () => {
    const collector = createMappingCollector()
    const ds = makeDataset()
    const r = await deidentify(ds, { salt: 'test' })
    collector.record(r)

    const csv = collector.toUidCsv()
    const lines = csv.split('\n')
    expect(lines[0]).toBe('original_uid,replacement_uid')
    expect(lines.length).toBe(4) // header + 3 UIDs
    expect(lines[1]).toContain('1.2.3.4.5,2.25.')
  })

  it('exports PatientID CSV', async () => {
    const collector = createMappingCollector()
    const ds = makeDataset()
    const r = await deidentify(ds, { salt: 'test' })
    collector.record(r)

    const csv = collector.toPatientIdCsv()
    const lines = csv.split('\n')
    expect(lines[0]).toBe('original_patient_id,replacement_patient_id,date_shift_offset')
    expect(lines.length).toBe(2) // header + 1 patient
    expect(lines[1]).toMatch(/^MRN-12345,SUBJ-[0-9a-f]{8},/)
  })

  it('includes date shift offset in PatientID CSV when shifting', async () => {
    const collector = createMappingCollector()
    const ds = makeDataset()
    const r = await deidentify(ds, { salt: 'test', dateShift: { offsetDays: -42 } })
    collector.record(r)

    const csv = collector.toPatientIdCsv()
    const lines = csv.split('\n')
    expect(lines[1]).toContain(',-42')
  })
})
