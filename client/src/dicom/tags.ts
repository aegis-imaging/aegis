import type { TagAction } from '../types'

/**
 * DICOM PS3.15 Annex E Basic Profile — tag action table.
 *
 * Action codes:
 *   D = replace with dummy value
 *   Z = replace with zero-length value
 *   X = remove entirely
 *   U = replace UID (deterministic hash to preserve linkage)
 *   K = keep (safe to retain)
 *   C = clean (remove if might contain PHI)
 */

interface TagRule {
  keyword: string
  action: TagAction
  description?: string
}

/** Tag rules keyed by DICOM tag (GGGGEEEE format, no 'x' prefix) */
export const BASIC_PROFILE: Record<string, TagRule> = {
  // --- Patient identifiers ---
  '00100010': { keyword: 'PatientName', action: 'Z' },
  '00100020': { keyword: 'PatientID', action: 'Z' },
  '00100030': { keyword: 'PatientBirthDate', action: 'Z' },
  '00100032': { keyword: 'PatientBirthTime', action: 'X' },
  '00100040': { keyword: 'PatientSex', action: 'K' },
  '00101000': { keyword: 'OtherPatientIDs', action: 'X' },
  '00101001': { keyword: 'OtherPatientNames', action: 'X' },
  '00101002': { keyword: 'OtherPatientIDsSequence', action: 'X' },
  '00100050': { keyword: 'PatientInsurancePlanCodeSequence', action: 'X' },
  '00101040': { keyword: 'PatientAddress', action: 'X' },
  '00101060': { keyword: 'PatientMotherBirthName', action: 'X' },
  '00102160': { keyword: 'EthnicGroup', action: 'K' },
  '001021F0': { keyword: 'PatientReligiousPreference', action: 'X' },
  '00102201': { keyword: 'PatientSpeciesDescription', action: 'C' },
  '00102292': { keyword: 'PatientBreedDescription', action: 'C' },
  '00104000': { keyword: 'PatientComments', action: 'X' },

  // --- Physician / operator identifiers ---
  '00080090': { keyword: 'ReferringPhysicianName', action: 'Z' },
  '00080092': { keyword: 'ReferringPhysicianAddress', action: 'X' },
  '00080094': { keyword: 'ReferringPhysicianTelephoneNumbers', action: 'X' },
  '00080096': { keyword: 'ReferringPhysicianIdentificationSequence', action: 'X' },
  '00081048': { keyword: 'PhysiciansOfRecord', action: 'X' },
  '00081049': { keyword: 'PhysiciansOfRecordIdentificationSequence', action: 'X' },
  '00081050': { keyword: 'PerformingPhysicianName', action: 'X' },
  '00081052': { keyword: 'PerformingPhysicianIdentificationSequence', action: 'X' },
  '00081060': { keyword: 'NameOfPhysiciansReadingStudy', action: 'X' },
  '00081062': { keyword: 'PhysiciansReadingStudyIdentificationSequence', action: 'X' },
  '00081070': { keyword: 'OperatorsName', action: 'X' },
  '00081072': { keyword: 'OperatorIdentificationSequence', action: 'X' },

  // --- Institution identifiers ---
  '00080080': { keyword: 'InstitutionName', action: 'X' },
  '00080081': { keyword: 'InstitutionAddress', action: 'X' },
  '00080082': { keyword: 'InstitutionCodeSequence', action: 'X' },
  '00081010': { keyword: 'StationName', action: 'X' },
  '00081040': { keyword: 'InstitutionalDepartmentName', action: 'X' },
  '00081041': { keyword: 'InstitutionalDepartmentTypeCodeSequence', action: 'X' },

  // --- Dates ---
  '00080012': { keyword: 'InstanceCreationDate', action: 'X' },
  '00080013': { keyword: 'InstanceCreationTime', action: 'X' },
  '00080020': { keyword: 'StudyDate', action: 'Z' },
  '00080021': { keyword: 'SeriesDate', action: 'X' },
  '00080022': { keyword: 'AcquisitionDate', action: 'X' },
  '00080023': { keyword: 'ContentDate', action: 'X' },
  '00080030': { keyword: 'StudyTime', action: 'Z' },
  '00080031': { keyword: 'SeriesTime', action: 'X' },
  '00080032': { keyword: 'AcquisitionTime', action: 'X' },
  '00080033': { keyword: 'ContentTime', action: 'X' },
  '00080050': { keyword: 'AccessionNumber', action: 'Z' },

  // --- UIDs (deterministic hash replacement) ---
  '0020000D': { keyword: 'StudyInstanceUID', action: 'U' },
  '0020000E': { keyword: 'SeriesInstanceUID', action: 'U' },
  '00080018': { keyword: 'SOPInstanceUID', action: 'U' },
  '00080016': { keyword: 'SOPClassUID', action: 'K' },
  '00020010': { keyword: 'TransferSyntaxUID', action: 'K' },
  '00200052': { keyword: 'FrameOfReferenceUID', action: 'U' },
  '00081115': { keyword: 'ReferencedSeriesSequence', action: 'U' },
  '00081155': { keyword: 'ReferencedSOPInstanceUID', action: 'U' },

  // --- Study / series description (clean, may contain PHI) ---
  '00081030': { keyword: 'StudyDescription', action: 'C' },
  '0008103E': { keyword: 'SeriesDescription', action: 'C' },
  '00081080': { keyword: 'AdmittingDiagnosesDescription', action: 'X' },
  '00081084': { keyword: 'AdmittingDiagnosesCodeSequence', action: 'X' },
  '00102180': { keyword: 'Occupation', action: 'X' },
  '001021B0': { keyword: 'AdditionalPatientHistory', action: 'X' },
  '00380010': { keyword: 'AdmissionID', action: 'X' },
  '00380500': { keyword: 'PatientState', action: 'X' },
  '00084000': { keyword: 'IdentifyingComments', action: 'X' },

  // --- Request attributes ---
  '00321032': { keyword: 'RequestingPhysician', action: 'X' },
  '00321033': { keyword: 'RequestingService', action: 'X' },
  '00321060': { keyword: 'RequestedProcedureDescription', action: 'C' },
  '00400006': { keyword: 'ScheduledPerformingPhysicianName', action: 'X' },
  '00400244': { keyword: 'PerformedProcedureStepStartDate', action: 'X' },
  '00400245': { keyword: 'PerformedProcedureStepStartTime', action: 'X' },
  '00400253': { keyword: 'PerformedProcedureStepID', action: 'X' },
  '00400254': { keyword: 'PerformedProcedureStepDescription', action: 'C' },
  '00401001': { keyword: 'RequestedProcedureID', action: 'X' },

  // --- Device / protocol (keep for research utility) ---
  '00080060': { keyword: 'Modality', action: 'K' },
  '00080070': { keyword: 'Manufacturer', action: 'K' },
  '00081090': { keyword: 'ManufacturerModelName', action: 'K' },
  '00181000': { keyword: 'DeviceSerialNumber', action: 'X' },
  '00181020': { keyword: 'SoftwareVersions', action: 'K' },
  '00181030': { keyword: 'ProtocolName', action: 'C' },

  // --- Image parameters (keep for research) ---
  '00180050': { keyword: 'SliceThickness', action: 'K' },
  '00180080': { keyword: 'RepetitionTime', action: 'K' },
  '00180081': { keyword: 'EchoTime', action: 'K' },
  '00180082': { keyword: 'InversionTime', action: 'K' },
  '00180083': { keyword: 'NumberOfAverages', action: 'K' },
  '00180084': { keyword: 'ImagingFrequency', action: 'K' },
  '00180085': { keyword: 'ImagedNucleus', action: 'K' },
  '00180087': { keyword: 'MagneticFieldStrength', action: 'K' },
  '00180088': { keyword: 'SpacingBetweenSlices', action: 'K' },
  '00180095': { keyword: 'PixelBandwidth', action: 'K' },
  '00181310': { keyword: 'AcquisitionMatrix', action: 'K' },
  '00181312': { keyword: 'InPlanePhaseEncodingDirection', action: 'K' },
  '00181314': { keyword: 'FlipAngle', action: 'K' },
  '00280010': { keyword: 'Rows', action: 'K' },
  '00280011': { keyword: 'Columns', action: 'K' },
  '00280030': { keyword: 'PixelSpacing', action: 'K' },
  '00280100': { keyword: 'BitsAllocated', action: 'K' },
  '00280101': { keyword: 'BitsStored', action: 'K' },
  '00280102': { keyword: 'HighBit', action: 'K' },
  '00200032': { keyword: 'ImagePositionPatient', action: 'K' },
  '00200037': { keyword: 'ImageOrientationPatient', action: 'K' },
  '00200013': { keyword: 'InstanceNumber', action: 'K' },
  '00201041': { keyword: 'SliceLocation', action: 'K' },

  // --- Pixel data (keep) ---
  '7FE00010': { keyword: 'PixelData', action: 'K' },

  // --- Body part (keep for routing) ---
  '00180015': { keyword: 'BodyPartExamined', action: 'K' },

  // --- Structured Report (SR) tags ---
  // ContentSequence structure is preserved; items within are recursively de-identified.
  '0040A730': { keyword: 'ContentSequence', action: 'K' },
  '0040A160': { keyword: 'TextValue', action: 'C' },
  '0040A123': { keyword: 'PersonName', action: 'Z' },
  '0040A124': { keyword: 'UID', action: 'U' },
  '00081199': { keyword: 'ReferencedSOPSequence', action: 'K' },
}

/** Look up tag rule by DICOM tag number (GGGGEEEE format) */
export function getTagRule(tag: string): TagRule | undefined {
  return BASIC_PROFILE[tag.toUpperCase()]
}

/** Check if a tag is a private tag (odd group number) */
export function isPrivateTag(tag: string): boolean {
  const group = parseInt(tag.substring(0, 4), 16)
  return group % 2 === 1
}

/** Human-readable label for action codes */
export const ACTION_LABELS: Record<TagAction, string> = {
  D: 'Replace with dummy',
  Z: 'Zero-length',
  X: 'Remove',
  U: 'Replace UID',
  K: 'Keep',
  C: 'Clean',
}
