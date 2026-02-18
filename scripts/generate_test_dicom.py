#!/usr/bin/env python3
"""Generate synthetic DICOM files for AEGIS upload portal testing.

Creates a realistic-looking brain MRI study with multiple series and instances.
All data is synthetic — no real patient information.
"""

import os
import numpy as np
from pydicom.dataset import Dataset, FileDataset
from pydicom.uid import generate_uid, ExplicitVRLittleEndian
from pydicom.sequence import Sequence
import datetime

OUTPUT_DIR = os.path.join(os.path.dirname(__file__), '..', 'test-data', 'brain-mri')

# Synthetic patient info (all fake — for testing de-identification)
PATIENT_NAME = 'DOE^JOHN^Q'
PATIENT_ID = 'MRN-123456'
PATIENT_BIRTH_DATE = '19750315'
PATIENT_SEX = 'M'

# Institution info (fake)
INSTITUTION_NAME = 'Springfield General Hospital'
INSTITUTION_ADDRESS = '742 Evergreen Terrace, Springfield, IL 62704'
REFERRING_PHYSICIAN = 'SMITH^JANE^M^DR'
PERFORMING_PHYSICIAN = 'JONES^ROBERT^A^DR'
OPERATORS_NAME = 'TECH^MARY'
STATION_NAME = 'MR-SCANNER-01'
DEPARTMENT_NAME = 'Radiology'

# Study-level UIDs
STUDY_INSTANCE_UID = generate_uid()
STUDY_DATE = '20240115'
STUDY_TIME = '093045'
STUDY_DESCRIPTION = 'BRAIN MRI W/O CONTRAST'
ACCESSION_NUMBER = 'ACC-2024-00789'

# Series definitions
SERIES = [
    {
        'description': 'T1 SAGITTAL',
        'modality': 'MR',
        'series_number': 1,
        'num_instances': 5,
        'rows': 64,
        'cols': 64,
        'slice_thickness': 1.0,
        'repetition_time': 2000.0,
        'echo_time': 3.5,
        'flip_angle': 9.0,
        'magnetic_field': 3.0,
        'protocol_name': 'T1_SAG_MPRAGE',
        'body_part': 'HEAD',
    },
    {
        'description': 'T2 FLAIR AXIAL',
        'modality': 'MR',
        'series_number': 2,
        'num_instances': 5,
        'rows': 64,
        'cols': 64,
        'slice_thickness': 3.0,
        'repetition_time': 9000.0,
        'echo_time': 120.0,
        'flip_angle': 150.0,
        'magnetic_field': 3.0,
        'protocol_name': 'T2_FLAIR_AX',
        'body_part': 'HEAD',
    },
    {
        'description': 'DWI AXIAL',
        'modality': 'MR',
        'series_number': 3,
        'num_instances': 3,
        'rows': 64,
        'cols': 64,
        'slice_thickness': 5.0,
        'repetition_time': 5000.0,
        'echo_time': 90.0,
        'flip_angle': 90.0,
        'magnetic_field': 3.0,
        'protocol_name': 'DWI_AX_B1000',
        'body_part': 'HEAD',
    },
]


def create_dicom_file(
    series_info: dict,
    instance_number: int,
    series_uid: str,
) -> FileDataset:
    """Create a single synthetic DICOM file."""

    sop_instance_uid = generate_uid()
    filename = f'IM-{series_info["series_number"]:04d}-{instance_number:04d}.dcm'
    filepath = os.path.join(OUTPUT_DIR, filename)

    # File meta info
    file_meta = Dataset()
    file_meta.MediaStorageSOPClassUID = '1.2.840.10008.5.1.4.1.1.4'  # MR Image Storage
    file_meta.MediaStorageSOPInstanceUID = sop_instance_uid
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b'\x00' * 128)

    # Patient module
    ds.PatientName = PATIENT_NAME
    ds.PatientID = PATIENT_ID
    ds.PatientBirthDate = PATIENT_BIRTH_DATE
    ds.PatientSex = PATIENT_SEX
    ds.PatientAddress = '456 Oak Street, Springfield, IL 62704'
    ds.OtherPatientIDs = 'ALT-789012'
    ds.EthnicGroup = 'Caucasian'

    # General study module
    ds.StudyInstanceUID = STUDY_INSTANCE_UID
    ds.StudyDate = STUDY_DATE
    ds.StudyTime = STUDY_TIME
    ds.StudyDescription = STUDY_DESCRIPTION
    ds.AccessionNumber = ACCESSION_NUMBER
    ds.ReferringPhysicianName = REFERRING_PHYSICIAN
    ds.StudyID = 'STUDY-001'

    # Institution
    ds.InstitutionName = INSTITUTION_NAME
    ds.InstitutionAddress = INSTITUTION_ADDRESS
    ds.InstitutionalDepartmentName = DEPARTMENT_NAME
    ds.StationName = STATION_NAME

    # Physician / operator
    ds.PerformingPhysicianName = PERFORMING_PHYSICIAN
    ds.OperatorsName = OPERATORS_NAME
    ds.PhysiciansOfRecord = REFERRING_PHYSICIAN

    # General series module
    ds.SeriesInstanceUID = series_uid
    ds.SeriesDate = STUDY_DATE
    ds.SeriesTime = STUDY_TIME
    ds.SeriesDescription = series_info['description']
    ds.SeriesNumber = series_info['series_number']
    ds.Modality = series_info['modality']
    ds.BodyPartExamined = series_info['body_part']
    ds.ProtocolName = series_info['protocol_name']

    # SOP common
    ds.SOPClassUID = '1.2.840.10008.5.1.4.1.1.4'  # MR Image Storage
    ds.SOPInstanceUID = sop_instance_uid
    ds.InstanceCreationDate = datetime.date.today().strftime('%Y%m%d')
    ds.InstanceCreationTime = datetime.datetime.now().strftime('%H%M%S')

    # General image module
    ds.InstanceNumber = instance_number
    ds.ContentDate = STUDY_DATE
    ds.ContentTime = STUDY_TIME

    # Image pixel module
    rows = series_info['rows']
    cols = series_info['cols']
    ds.Rows = rows
    ds.Columns = cols
    ds.BitsAllocated = 16
    ds.BitsStored = 12
    ds.HighBit = 11
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = 'MONOCHROME2'

    # Synthetic pixel data (gradient with noise)
    np.random.seed(series_info['series_number'] * 100 + instance_number)
    pixel_array = np.random.randint(0, 2048, (rows, cols), dtype=np.uint16)
    # Add a circle to make it look brain-like
    y, x = np.ogrid[-rows//2:rows//2, -cols//2:cols//2]
    mask = x*x + y*y <= (rows//3)**2
    pixel_array[mask] += 1000
    pixel_array = np.clip(pixel_array, 0, 4095).astype(np.uint16)
    ds.PixelData = pixel_array.tobytes()

    # MR-specific parameters
    ds.SliceThickness = series_info['slice_thickness']
    ds.RepetitionTime = series_info['repetition_time']
    ds.EchoTime = series_info['echo_time']
    ds.FlipAngle = series_info['flip_angle']
    ds.MagneticFieldStrength = series_info['magnetic_field']
    ds.SpacingBetweenSlices = series_info['slice_thickness']
    ds.PixelSpacing = [0.5, 0.5]
    ds.ImagePositionPatient = [0.0, 0.0, float(instance_number * series_info['slice_thickness'])]
    ds.ImageOrientationPatient = [1.0, 0.0, 0.0, 0.0, 1.0, 0.0]
    ds.SliceLocation = float(instance_number * series_info['slice_thickness'])

    # Manufacturer info
    ds.Manufacturer = 'SIEMENS'
    ds.ManufacturerModelName = 'MAGNETOM Prisma'
    ds.SoftwareVersions = 'syngo MR E11'
    ds.DeviceSerialNumber = 'SN-12345'

    return ds


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    total_files = 0
    for series_info in SERIES:
        series_uid = generate_uid()
        for i in range(1, series_info['num_instances'] + 1):
            ds = create_dicom_file(series_info, i, series_uid)
            ds.save_as(ds.filename)
            total_files += 1
            print(f'  Created: {os.path.basename(ds.filename)}')

    print(f'\nGenerated {total_files} DICOM files in {OUTPUT_DIR}')
    print(f'  Patient: {PATIENT_NAME}')
    print(f'  Study: {STUDY_DESCRIPTION}')
    print(f'  Series: {len(SERIES)}')
    print(f'  Institution: {INSTITUTION_NAME}')


if __name__ == '__main__':
    main()
