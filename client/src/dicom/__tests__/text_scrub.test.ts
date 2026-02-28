import { describe, it, expect } from 'vitest'
import { scrubFreeText } from '../text_scrub'
import type { ScrubContext } from '../text_scrub'

describe('scrubFreeText', () => {
  describe('pattern-based PHI detection', () => {
    it('removes SSN patterns', () => {
      const result = scrubFreeText('Patient SSN 123-45-6789 noted')
      expect(result.phiFound).toBe(true)
      expect(result.text).toContain('[REMOVED]')
      expect(result.text).not.toContain('123-45-6789')
    })

    it('removes phone numbers', () => {
      const result = scrubFreeText('Call (555) 123-4567 for results')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('555')
      expect(result.text).toContain('Call')
      expect(result.text).toContain('for results')
    })

    it('removes email addresses', () => {
      const result = scrubFreeText('Contact john.doe@hospital.com for info')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('john.doe@hospital.com')
      expect(result.text).toContain('Contact')
    })

    it('removes MRN patterns', () => {
      const result = scrubFreeText('MRN: 12345 CHEST CT')
      expect(result.phiFound).toBe(true)
      expect(result.text).toContain('[REMOVED]')
      expect(result.text).toContain('CHEST CT')
    })

    it('removes accession number patterns', () => {
      const result = scrubFreeText('ACC: A12345 study performed')
      expect(result.phiFound).toBe(true)
      expect(result.text).toContain('[REMOVED]')
      expect(result.text).toContain('study performed')
    })

    it('removes US date formats', () => {
      const result = scrubFreeText('Exam on 01/15/2024 showed normal findings')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('01/15/2024')
      expect(result.text).toContain('normal findings')
    })

    it('removes written date formats', () => {
      const result = scrubFreeText('Study from January 15, 2024 reviewed')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('January 15, 2024')
    })

    it('removes ISO date formats', () => {
      const result = scrubFreeText('Date: 2024-01-15 findings normal')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('2024-01-15')
    })

    it('removes street addresses', () => {
      const result = scrubFreeText('Located at 123 Main Street in the city')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('123 Main Street')
      expect(result.text).toContain('Located at')
    })

    it('removes title-name patterns', () => {
      const result = scrubFreeText('Brain MRI with contrast, Dr. Smith ordering')
      expect(result.phiFound).toBe(true)
      expect(result.text).toContain('Brain MRI with contrast,')
      expect(result.text).toContain('[REMOVED]')
      expect(result.text).toContain('ordering')
    })

    it('removes hospital name patterns', () => {
      const result = scrubFreeText('Performed at General Hospital radiology dept')
      expect(result.phiFound).toBe(true)
      expect(result.text).toContain('[REMOVED]')
      expect(result.text).toContain('radiology dept')
    })

    it('removes IP addresses', () => {
      const result = scrubFreeText('Source: 192.168.1.100 workstation')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('192.168.1.100')
    })

    it('removes URLs', () => {
      const result = scrubFreeText('See https://hospital.com/patient/123 for details')
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('https://hospital.com/patient/123')
    })

    it('removes Patient prefix patterns', () => {
      const result = scrubFreeText('Patient: John Smith had a CT scan')
      expect(result.phiFound).toBe(true)
      expect(result.text).toContain('[REMOVED]')
    })
  })

  describe('medical term preservation', () => {
    it('preserves standard imaging descriptions with no PHI', () => {
      const result = scrubFreeText('CT CHEST WITH CONTRAST')
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('CT CHEST WITH CONTRAST')
    })

    it('preserves MRI sequence descriptions', () => {
      const result = scrubFreeText('T1 MPRAGE SAGITTAL')
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('T1 MPRAGE SAGITTAL')
    })

    it('preserves anatomy terms', () => {
      const result = scrubFreeText('AXIAL 3MM BRAIN')
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('AXIAL 3MM BRAIN')
    })

    it('preserves modality references', () => {
      const result = scrubFreeText('MRI HEAD WITHOUT CONTRAST')
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('MRI HEAD WITHOUT CONTRAST')
    })

    it('preserves common findings', () => {
      const result = scrubFreeText('Normal chest radiograph bilateral')
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('Normal chest radiograph bilateral')
    })
  })

  describe('context-based matching', () => {
    const context: ScrubContext = {
      patientName: 'DOE^JOHN',
      patientId: 'MRN-12345',
      referringPhysician: 'SMITH^ALICE',
      institutionName: 'General Hospital',
    }

    it('removes patient name tokens from text', () => {
      const result = scrubFreeText('CT scan for Doe, reviewed by team', context)
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toMatch(/doe/i)
      expect(result.text).toContain('CT scan for')
    })

    it('removes patient ID from text', () => {
      const result = scrubFreeText('Patient MRN-12345 CT CHEST', context)
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('MRN-12345')
      expect(result.text).toContain('CT CHEST')
    })

    it('removes referring physician name from text', () => {
      const result = scrubFreeText('Ordered by Smith for brain MRI', context)
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toMatch(/smith/i)
      expect(result.text).toContain('brain MRI')
    })

    it('removes institution name from text', () => {
      const result = scrubFreeText('Performed at General Hospital radiology', context)
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toContain('General Hospital')
    })

    it('handles case-insensitive name matching', () => {
      const result = scrubFreeText('Study for JOHN DOE chest CT', context)
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toMatch(/john/i)
      expect(result.text).not.toMatch(/doe/i)
      expect(result.text).toContain('chest CT')
    })
  })

  describe('combined pattern + context', () => {
    it('removes both pattern PHI and context PHI', () => {
      const context: ScrubContext = { patientName: 'DOE^JOHN' }
      const result = scrubFreeText('CT for John Doe, SSN 123-45-6789', context)
      expect(result.phiFound).toBe(true)
      expect(result.text).not.toMatch(/john/i)
      expect(result.text).not.toMatch(/doe/i)
      expect(result.text).not.toContain('123-45-6789')
      expect(result.text).toContain('CT for')
    })
  })

  describe('edge cases', () => {
    it('returns empty string for empty input', () => {
      const result = scrubFreeText('')
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('')
    })

    it('handles null-ish values gracefully', () => {
      const result = scrubFreeText(undefined as unknown as string)
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('')
    })

    it('collapses consecutive [REMOVED] tokens', () => {
      const context: ScrubContext = { patientName: 'DOE^JOHN' }
      const result = scrubFreeText('John Doe CT scan', context)
      expect(result.phiFound).toBe(true)
      // "John" and "Doe" are adjacent → should collapse to single [REMOVED]
      expect(result.text).not.toContain('[REMOVED] [REMOVED]')
    })

    it('handles empty context gracefully', () => {
      const result = scrubFreeText('Normal CT scan', {})
      expect(result.phiFound).toBe(false)
      expect(result.text).toBe('Normal CT scan')
    })
  })
})
