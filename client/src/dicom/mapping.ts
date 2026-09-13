/**
 * De-identification mapping collector and CSV exporter.
 *
 * Accumulates UID and PatientID mappings across multiple deidentify() calls
 * and exports them as CSV files matching the MIDI-B submission format.
 */

import type { DeidResult } from './deid'

export interface DeidMappings {
  /** Original UID → replacement UID */
  uids: Map<string, string>
  /** Original PatientID → pseudonym */
  patientIds: Map<string, string>
  /** PatientID → date shift offset in days (if date shifting was applied) */
  dateOffsets: Map<string, number>
}

export interface MappingCollector {
  /** Record mappings from a single deidentify() result */
  record(result: DeidResult): void
  /** Get all accumulated mappings */
  getMappings(): DeidMappings
  /** Export UID mappings as CSV: "original_uid,replacement_uid" */
  toUidCsv(): string
  /** Export PatientID mappings as CSV: "original_patient_id,replacement_patient_id,date_shift_offset" */
  toPatientIdCsv(): string
}

export function createMappingCollector(): MappingCollector {
  const uids = new Map<string, string>()
  const patientIds = new Map<string, string>()
  const dateOffsets = new Map<string, number>()

  return {
    record(result: DeidResult): void {
      // Collect UID mappings
      if (result.uidMappings) {
        for (const [original, replacement] of result.uidMappings) {
          uids.set(original, replacement)
        }
      }

      // Collect PatientID mapping
      if (result.patientIdMapping) {
        patientIds.set(result.patientIdMapping.original, result.patientIdMapping.replacement)
      }

      // Collect date shift offset
      if (result.dateShiftOffset !== undefined && result.patientIdMapping) {
        dateOffsets.set(result.patientIdMapping.original, result.dateShiftOffset)
      }
    },

    getMappings(): DeidMappings {
      return {
        uids: new Map(uids),
        patientIds: new Map(patientIds),
        dateOffsets: new Map(dateOffsets),
      }
    },

    toUidCsv(): string {
      const lines = ['original_uid,replacement_uid']
      for (const [original, replacement] of uids) {
        lines.push(`${original},${replacement}`)
      }
      return lines.join('\n')
    },

    toPatientIdCsv(): string {
      const lines = ['original_patient_id,replacement_patient_id,date_shift_offset']
      for (const [original, replacement] of patientIds) {
        const offset = dateOffsets.get(original) ?? ''
        lines.push(`${original},${replacement},${offset}`)
      }
      return lines.join('\n')
    },
  }
}
