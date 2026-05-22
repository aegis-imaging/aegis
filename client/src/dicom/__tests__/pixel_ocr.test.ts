import { describe, it, expect } from 'vitest'
import { looksLikePhi } from '../pixel_ocr'

describe('looksLikePhi', () => {
  describe('positive matches', () => {
    it('flags date-shaped tokens', () => {
      expect(looksLikePhi('2024-03-15')).toBe(true)
      expect(looksLikePhi('2024/03/15')).toBe(true)
      expect(looksLikePhi('20240315')).toBe(true)
    })

    it('flags clock times', () => {
      expect(looksLikePhi('14:30')).toBe(true)
      expect(looksLikePhi('14:30:00')).toBe(true)
    })

    it('flags long digit runs (MRN, accession, phone)', () => {
      expect(looksLikePhi('1234567')).toBe(true)
      expect(looksLikePhi('MRN: 8675309')).toBe(true)
    })

    it('flags SSN-shaped strings', () => {
      expect(looksLikePhi('123-45-6789')).toBe(true)
    })

    it('flags DICOM name format (LAST^FIRST)', () => {
      expect(looksLikePhi('SMITH^JANE')).toBe(true)
      expect(looksLikePhi('DOE^JOHN^A')).toBe(true)
    })

    it('flags mixed-case names', () => {
      expect(looksLikePhi('John Smith')).toBe(true)
      expect(looksLikePhi('Jane Doe')).toBe(true)
    })

    it('flags PHI labels even without values', () => {
      expect(looksLikePhi('MRN')).toBe(true)
      expect(looksLikePhi('ACC#')).toBe(true)
      expect(looksLikePhi('DOB')).toBe(true)
      expect(looksLikePhi('PATIENT')).toBe(true)
    })

    it('flags emails', () => {
      expect(looksLikePhi('jane@hospital.org')).toBe(true)
    })
  })

  describe('negative matches', () => {
    it('does not flag short anatomic labels', () => {
      expect(looksLikePhi('AX')).toBe(false)
      expect(looksLikePhi('SAG')).toBe(false)
      expect(looksLikePhi('TRA')).toBe(false)
    })

    it('does not flag protocol names', () => {
      expect(looksLikePhi('T1w')).toBe(false)
      expect(looksLikePhi('FLAIR')).toBe(false)
    })

    it('does not flag scanner orientation markers', () => {
      expect(looksLikePhi('R')).toBe(false)
      expect(looksLikePhi('L')).toBe(false)
      expect(looksLikePhi('A')).toBe(false)
    })

    it('does not flag generic short text', () => {
      expect(looksLikePhi('the')).toBe(false)
      expect(looksLikePhi('and')).toBe(false)
    })
  })
})
