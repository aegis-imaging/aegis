"""DICOM date shifting utilities."""

from __future__ import annotations

import hashlib
from datetime import datetime, timedelta

# Tags whose values are dates that should be shifted.
DATE_TAGS: set[str] = {
    "00080012",  # InstanceCreationDate
    "00080020",  # StudyDate
    "00080021",  # SeriesDate
    "00080022",  # AcquisitionDate
    "00080023",  # ContentDate
    "00100030",  # PatientBirthDate
    "00400244",  # PerformedProcedureStepStartDate
}

# Tags whose values are times paired with the above dates — kept unchanged.
TIME_TAGS: set[str] = {
    "00080013",  # InstanceCreationTime
    "00080030",  # StudyTime
    "00080031",  # SeriesTime
    "00080032",  # AcquisitionTime
    "00080033",  # ContentTime
    "00400245",  # PerformedProcedureStepStartTime
}


def shift_dicom_date(date_str: str, offset_days: int) -> str:
    """Shift a DICOM DA value (YYYYMMDD) by *offset_days*.

    Returns the shifted date in YYYYMMDD format.
    If the input is not a valid 8-digit date, returns it unchanged.
    """
    if len(date_str) != 8 or not date_str.isdigit():
        return date_str
    try:
        dt = datetime(int(date_str[:4]), int(date_str[4:6]), int(date_str[6:8]))
    except ValueError:
        return date_str
    shifted = dt + timedelta(days=offset_days)
    return shifted.strftime("%Y%m%d")


def shift_dicom_datetime(dt_str: str, offset_days: int) -> str:
    """Shift a DICOM DT value (YYYYMMDD + optional time/tz suffix).

    Only the date portion (first 8 chars) is shifted; the rest is preserved.
    """
    if len(dt_str) < 8:
        return dt_str
    date_part = shift_dicom_date(dt_str[:8], offset_days)
    return date_part + dt_str[8:]


def generate_date_offset(patient_id: str, salt: str, max_days: int = 365) -> int:
    """Generate a deterministic date-shift offset for a patient.

    Uses SHA-256 of ``salt + patient_id``; first 4 bytes as unsigned 32-bit
    integer mapped to ``[-max_days, +max_days]``.
    """
    data = (salt + patient_id).encode("utf-8")
    digest = hashlib.sha256(data).digest()
    raw = int.from_bytes(digest[:4], byteorder="big")
    span = max_days * 2 + 1
    return (raw % span) - max_days
