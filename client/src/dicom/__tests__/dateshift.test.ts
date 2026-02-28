import { describe, it, expect } from 'vitest'
import { shiftDicomDate, shiftDicomDateTime, generateDateOffset } from '../dateshift'
import { deidentify } from '../deid'
import type { NaturalizedDataset } from '../parser'

describe('shiftDicomDate', () => {
  it('shifts a date forward', () => {
    expect(shiftDicomDate('20240301', 10)).toBe('20240311')
  })

  it('shifts a date backward', () => {
    expect(shiftDicomDate('20240310', -10)).toBe('20240229') // 2024 is a leap year
  })

  it('shifts across month boundary', () => {
    expect(shiftDicomDate('20240128', 5)).toBe('20240202')
  })

  it('shifts across year boundary', () => {
    expect(shiftDicomDate('20241225', 10)).toBe('20250104')
  })

  it('shifts backward across year boundary', () => {
    expect(shiftDicomDate('20240105', -10)).toBe('20231226')
  })

  it('handles leap year correctly', () => {
    expect(shiftDicomDate('20240228', 1)).toBe('20240229')
    expect(shiftDicomDate('20240229', 1)).toBe('20240301')
    // Non-leap year
    expect(shiftDicomDate('20230228', 1)).toBe('20230301')
  })

  it('returns original for empty/short strings', () => {
    expect(shiftDicomDate('', 5)).toBe('')
    expect(shiftDicomDate('2024', 5)).toBe('2024')
  })

  it('returns original for unparseable dates', () => {
    expect(shiftDicomDate('ABCDEFGH', 5)).toBe('ABCDEFGH')
  })

  it('handles large offsets', () => {
    // +365 days from Jan 1
    expect(shiftDicomDate('20240101', 366)).toBe('20250101') // 2024 is leap year = 366 days
  })
})

describe('shiftDicomDateTime', () => {
  it('shifts date component and preserves time', () => {
    expect(shiftDicomDateTime('20240301143000.000000', 10)).toBe('20240311143000.000000')
  })

  it('preserves timezone offset', () => {
    expect(shiftDicomDateTime('20240301143000.000000+0500', -5)).toBe('20240225143000.000000+0500')
  })

  it('handles bare date (no time)', () => {
    expect(shiftDicomDateTime('20240301', 1)).toBe('20240302')
  })

  it('returns original for empty strings', () => {
    expect(shiftDicomDateTime('', 5)).toBe('')
  })
})

describe('generateDateOffset', () => {
  it('returns a number within range', async () => {
    const offset = await generateDateOffset('PATIENT-001', 'test-salt', 365)
    expect(offset).toBeGreaterThanOrEqual(-365)
    expect(offset).toBeLessThanOrEqual(365)
  })

  it('is deterministic (same input → same output)', async () => {
    const a = await generateDateOffset('PATIENT-001', 'salt-a')
    const b = await generateDateOffset('PATIENT-001', 'salt-a')
    expect(a).toBe(b)
  })

  it('produces different offsets for different patients', async () => {
    const a = await generateDateOffset('PATIENT-001', 'salt')
    const b = await generateDateOffset('PATIENT-002', 'salt')
    // Statistically extremely unlikely to be equal (1 in 731 chance)
    expect(a).not.toBe(b)
  })

  it('produces different offsets for different salts', async () => {
    const a = await generateDateOffset('PATIENT-001', 'salt-a')
    const b = await generateDateOffset('PATIENT-001', 'salt-b')
    expect(a).not.toBe(b)
  })

  it('respects custom maxDays', async () => {
    const offset = await generateDateOffset('PATIENT-001', 'salt', 30)
    expect(offset).toBeGreaterThanOrEqual(-30)
    expect(offset).toBeLessThanOrEqual(30)
  })
})

describe('deidentify with dateShift', () => {
  function makeDataset(): NaturalizedDataset {
    return {
      PatientName: 'DOE^JOHN',
      PatientID: 'MRN-12345',
      StudyDate: '20240301',
      StudyTime: '143000',
      SeriesDate: '20240301',
      AcquisitionDate: '20240301',
      ContentDate: '20240301',
      StudyInstanceUID: '1.2.3.4.5',
      SeriesInstanceUID: '1.2.3.4.5.1',
      SOPInstanceUID: '1.2.3.4.5.1.1',
      SOPClassUID: '1.2.840.10008.5.1.4.1.1.2',
      Modality: 'CT',
    } as unknown as NaturalizedDataset
  }

  it('shifts dates when dateShift option is provided', async () => {
    const ds = makeDataset()
    await deidentify(ds, { dateShift: { offsetDays: 10 } })
    expect(ds.StudyDate).toBe('20240311')
    expect(ds.SeriesDate).toBe('20240311')
    expect(ds.AcquisitionDate).toBe('20240311')
    expect(ds.ContentDate).toBe('20240311')
  })

  it('preserves time tags when date shifting', async () => {
    const ds = makeDataset()
    await deidentify(ds, { dateShift: { offsetDays: 10 } })
    expect(ds.StudyTime).toBe('143000')
  })

  it('still zeroes/removes dates without dateShift option', async () => {
    const ds = makeDataset()
    await deidentify(ds)
    expect(ds.StudyDate).toBe('')       // Z action
    expect(ds.SeriesDate).toBeUndefined() // X action
  })

  it('negative offset shifts dates backward', async () => {
    const ds = makeDataset()
    await deidentify(ds, { dateShift: { offsetDays: -30 } })
    expect(ds.StudyDate).toBe('20240131')
  })
})
