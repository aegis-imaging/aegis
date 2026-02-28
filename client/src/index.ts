/**
 * @aegis/client — DICOM anonymization and upload library
 *
 * Provides browser-side DICOM parsing, PS3.15 Annex E de-identification,
 * and secure upload to an AEGIS API endpoint.
 *
 * Usage:
 *   import { isDicomFile, parseDicomFile, buildStudySummary, deidentify, uploadStudy } from '@aegis/client'
 */

export const VERSION = '0.1.0'

// Types
export type { TagAction, DicomTag, StudySummary, ParsedDicomFile } from './types'

// DICOM parsing
export { parseDicomFile, buildStudySummary, isDicomFile, serializeDataset, groupByStudy } from './dicom/parser'
export type { NaturalizedDataset } from './dicom/parser'

// De-identification
export { deidentify } from './dicom/deid'
export type { DeidResult, DeidOptions } from './dicom/deid'

// Date shifting
export { shiftDicomDate, shiftDicomDateTime, generateDateOffset } from './dicom/dateshift'

// De-identification mapping
export { createMappingCollector } from './dicom/mapping'
export type { DeidMappings, MappingCollector } from './dicom/mapping'

// Free-text PHI scrubbing
export { scrubFreeText } from './dicom/text_scrub'
export type { ScrubContext, ScrubResult } from './dicom/text_scrub'

// Tag profile
export { BASIC_PROFILE, getTagRule, isPrivateTag, ACTION_LABELS } from './dicom/tags'

// Upload
export { uploadStudy } from './upload/client'
export type { UploadOptions, UploadResult } from './upload/client'
