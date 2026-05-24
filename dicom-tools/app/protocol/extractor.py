"""Unified DICOM parameter extraction for Classic and Enhanced MR images.

Classic DICOM: one file per slice, acquisition parameters at the top level.
Enhanced DICOM: one file per volume (multi-frame), acquisition parameters in
SharedFunctionalGroupsSequence (5200,9229) and PerFrameFunctionalGroupsSequence
(5200,9230) nested sequences.

This module returns a flat dict of parameter name -> value regardless of format.
"""

import logging
from dataclasses import dataclass, field

import pydicom

log = logging.getLogger(__name__)

# Enhanced MR Image Storage SOP Class UID
ENHANCED_MR_SOP = "1.2.840.10008.5.1.4.1.1.4.1"

# Mapping from nested Enhanced DICOM sequences to the tags they contain.
# Each entry: (sequence keyword, [(tag keyword in sequence, output key)])
_ENHANCED_SHARED_MAPPINGS: list[tuple[str, list[tuple[str, str]]]] = [
    (
        "MRTimingAndRelatedParametersSequence",
        [
            ("RepetitionTime", "RepetitionTime"),
            ("FlipAngle", "FlipAngle"),
        ],
    ),
    (
        "MREchoSequence",
        [
            ("EffectiveEchoTime", "EchoTime"),
        ],
    ),
    (
        "MRModifierSequence",
        [
            ("InversionTimes", "InversionTime"),
        ],
    ),
    (
        "MRFOVGeometrySequence",
        [
            ("InPlanePhaseEncodingDirection", "InPlanePhaseEncodingDirection"),
        ],
    ),
    (
        "MRAveragesSequence",
        [
            ("NumberOfAverages", "NumberOfAverages"),
        ],
    ),
    (
        "PixelMeasuresSequence",
        [
            ("PixelSpacing", "PixelSpacing"),
            ("SliceThickness", "SliceThickness"),
            ("SpacingBetweenSlices", "SpacingBetweenSlices"),
        ],
    ),
    (
        "MRDiffusionSequence",
        [
            ("DiffusionBValue", "DiffusionBValue"),
        ],
    ),
]

# Top-level Classic DICOM tags to extract.
_CLASSIC_TAGS: list[str] = [
    "RepetitionTime",
    "EchoTime",
    "InversionTime",
    "FlipAngle",
    "SliceThickness",
    "SpacingBetweenSlices",
    "PixelBandwidth",
    "NumberOfAverages",
    "EchoTrainLength",
    "MagneticFieldStrength",
    "Rows",
    "Columns",
    "PixelSpacing",
    "InPlanePhaseEncodingDirection",
    "ScanningSequence",
    "SequenceVariant",
    "MRAcquisitionType",
    "AcquisitionMatrix",
    "ImagingFrequency",
    "ImagedNucleus",
]

# Device identification tags (same for Classic and Enhanced).
_DEVICE_TAGS: list[str] = [
    "Manufacturer",
    "ManufacturerModelName",
    "SoftwareVersions",
    "Modality",
    "ProtocolName",
    "SeriesDescription",
    "SequenceName",
    "BodyPartExamined",
    "MagneticFieldStrength",
]


@dataclass
class DeviceInfo:
    manufacturer: str = ""
    model: str = ""
    software_version: str = ""


@dataclass
class ExtractionResult:
    """Parameters extracted from a DICOM file or series."""

    params: dict = field(default_factory=dict)
    device: DeviceInfo = field(default_factory=DeviceInfo)
    dicom_format: str = "classic"  # "classic" | "enhanced"
    series_description: str = ""
    protocol_name: str = ""


def _safe_value(ds: pydicom.Dataset, keyword: str):
    """Get a tag value, returning None if missing."""
    if keyword not in ds:
        return None
    elem = ds[keyword]
    val = elem.value
    if val is None:
        return None
    # Convert pydicom Sequence to None (handled separately)
    if elem.VR == "SQ":
        return None
    # Convert MultiValue to list
    if hasattr(val, "__iter__") and not isinstance(val, (str, bytes)):
        return [float(v) if isinstance(v, (int, float)) else str(v) for v in val]
    if isinstance(val, (int, float)):
        return float(val)
    return str(val)


def _extract_device_info(ds: pydicom.Dataset) -> DeviceInfo:
    """Extract scanner manufacturer, model, and software version."""
    manufacturer = str(getattr(ds, "Manufacturer", "") or "").strip()
    model = str(getattr(ds, "ManufacturerModelName", "") or "").strip()
    sw = getattr(ds, "SoftwareVersions", "") or ""
    if hasattr(sw, "__iter__") and not isinstance(sw, str):
        sw = "\\".join(str(v) for v in sw)
    return DeviceInfo(
        manufacturer=manufacturer,
        model=model,
        software_version=str(sw).strip(),
    )


def _extract_classic(ds: pydicom.Dataset) -> dict:
    """Extract acquisition parameters from a Classic (legacy) DICOM dataset."""
    params: dict = {}
    for keyword in _CLASSIC_TAGS:
        val = _safe_value(ds, keyword)
        if val is not None:
            params[keyword] = val
    return params


def _extract_enhanced(ds: pydicom.Dataset) -> dict:
    """Extract acquisition parameters from an Enhanced MR DICOM dataset.

    Reads from SharedFunctionalGroupsSequence first, then checks
    PerFrameFunctionalGroupsSequence frame 0 for per-frame overrides.
    Also reads top-level tags that are present in Enhanced MR.
    """
    params: dict = {}

    # Top-level tags that exist in both Classic and Enhanced
    for keyword in [
        "MagneticFieldStrength",
        "Rows",
        "Columns",
        "ScanningSequence",
        "SequenceVariant",
        "MRAcquisitionType",
        "PixelBandwidth",
        "EchoTrainLength",
        "AcquisitionMatrix",
        "ImagingFrequency",
        "ImagedNucleus",
    ]:
        val = _safe_value(ds, keyword)
        if val is not None:
            params[keyword] = val

    # Extract from SharedFunctionalGroupsSequence
    shared_seq = getattr(ds, "SharedFunctionalGroupsSequence", None)
    if shared_seq and len(shared_seq) > 0:
        shared = shared_seq[0]
        for seq_keyword, tag_mappings in _ENHANCED_SHARED_MAPPINGS:
            nested = getattr(shared, seq_keyword, None)
            if nested and len(nested) > 0:
                item = nested[0]
                for src_keyword, dst_key in tag_mappings:
                    val = _safe_value(item, src_keyword)
                    if val is not None:
                        params[dst_key] = val

    # Check PerFrameFunctionalGroupsSequence frame 0 for overrides
    perframe_seq = getattr(ds, "PerFrameFunctionalGroupsSequence", None)
    if perframe_seq and len(perframe_seq) > 0:
        frame0 = perframe_seq[0]
        for seq_keyword, tag_mappings in _ENHANCED_SHARED_MAPPINGS:
            nested = getattr(frame0, seq_keyword, None)
            if nested and len(nested) > 0:
                item = nested[0]
                for src_keyword, dst_key in tag_mappings:
                    val = _safe_value(item, src_keyword)
                    if val is not None:
                        params[dst_key] = val

    return params


def extract_parameters(dicom_path: str) -> ExtractionResult:
    """Extract acquisition parameters and device info from a single DICOM file.

    Automatically detects Classic vs Enhanced DICOM format.
    """
    try:
        ds = pydicom.dcmread(dicom_path, stop_before_pixels=True)
    except Exception as e:
        log.warning("Failed to read DICOM file %s: %s", dicom_path, e)
        return ExtractionResult()

    device = _extract_device_info(ds)
    sop_class = str(getattr(ds, "SOPClassUID", "") or "")
    is_enhanced = sop_class == ENHANCED_MR_SOP
    dicom_format = "enhanced" if is_enhanced else "classic"

    if is_enhanced:
        params = _extract_enhanced(ds)
    else:
        params = _extract_classic(ds)

    protocol_name = str(getattr(ds, "ProtocolName", "") or "").strip()
    series_desc = str(getattr(ds, "SeriesDescription", "") or "").strip()

    return ExtractionResult(
        params=params,
        device=device,
        dicom_format=dicom_format,
        series_description=series_desc,
        protocol_name=protocol_name,
    )


def extract_from_series(dicom_paths: list[str]) -> ExtractionResult:
    """Extract parameters from a series of DICOM files.

    Reads the first valid file for device info and parameters (Classic),
    or the single multi-frame file (Enhanced). For Classic DICOM, additional
    files are read only if the first is missing expected tags.
    """
    if not dicom_paths:
        return ExtractionResult()

    # Try the first file
    result = extract_parameters(dicom_paths[0])

    # If we got no params from the first file, try the next few
    if not result.params and len(dicom_paths) > 1:
        for path in dicom_paths[1:5]:
            alt = extract_parameters(path)
            if alt.params:
                result = alt
                break

    return result
