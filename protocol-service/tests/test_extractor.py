"""Unit tests for the DICOM parameter extractor (Classic + Enhanced)."""

from app.extractor import extract_parameters, extract_from_series, ENHANCED_MR_SOP


# ── Classic DICOM extraction ─────────────────────────────────────────────────


def test_extract_classic_mr(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        repetition_time=2500.0,
        echo_time=4.2,
        flip_angle=12.0,
        slice_thickness=1.5,
    )
    result = extract_parameters(path)
    assert result.dicom_format == "classic"
    assert result.params["RepetitionTime"] == 2500.0
    assert result.params["EchoTime"] == 4.2
    assert result.params["FlipAngle"] == 12.0
    assert result.params["SliceThickness"] == 1.5
    assert result.params["MagneticFieldStrength"] == 3.0


def test_extract_classic_missing_tags(tmp_path):
    """DICOM with minimal tags — only what FileDataset requires."""
    import pydicom
    from pydicom.dataset import Dataset, FileDataset
    from pydicom.uid import generate_uid, ExplicitVRLittleEndian

    filepath = str(tmp_path / "minimal.dcm")
    file_meta = Dataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    file_meta.MediaStorageSOPInstanceUID = generate_uid()
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = FileDataset(filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128)
    ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    ds.Modality = "MR"
    ds.save_as(filepath)

    result = extract_parameters(filepath)
    assert result.dicom_format == "classic"
    assert "RepetitionTime" not in result.params
    assert "EchoTime" not in result.params


def test_extract_device_info(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        manufacturer="GE MEDICAL SYSTEMS",
        model="SIGNA Premier",
        software_version="29\\LX\\MR29.1",
    )
    result = extract_parameters(path)
    assert result.device.manufacturer == "GE MEDICAL SYSTEMS"
    assert result.device.model == "SIGNA Premier"
    assert result.device.software_version == "29\\LX\\MR29.1"


def test_extract_pixel_spacing_as_list(tmp_path, make_dicom_file):
    path = make_dicom_file(tmp_dir=tmp_path, pixel_spacing=[0.9375, 0.9375])
    result = extract_parameters(path)
    ps = result.params["PixelSpacing"]
    assert isinstance(ps, list)
    assert len(ps) == 2
    assert abs(ps[0] - 0.9375) < 0.001


def test_extract_format_detection_classic(tmp_path, make_dicom_file):
    path = make_dicom_file(tmp_dir=tmp_path)
    result = extract_parameters(path)
    assert result.dicom_format == "classic"


# ── Enhanced DICOM extraction ────────────────────────────────────────────────


def test_extract_enhanced_mr(make_enhanced_dicom):
    path = make_enhanced_dicom(
        repetition_time=3000.0,
        echo_time=30.0,
        flip_angle=90.0,
        slice_thickness=2.0,
    )
    result = extract_parameters(path)
    assert result.dicom_format == "enhanced"
    assert result.params["RepetitionTime"] == 3000.0
    assert result.params["EchoTime"] == 30.0
    assert result.params["FlipAngle"] == 90.0
    assert result.params["SliceThickness"] == 2.0


def test_extract_enhanced_perframe_override(make_enhanced_dicom):
    path = make_enhanced_dicom(echo_time=30.0, perframe_echo_time=35.0)
    result = extract_parameters(path)
    # PerFrame overrides Shared
    assert result.params["EchoTime"] == 35.0


def test_extract_format_detection_enhanced(make_enhanced_dicom):
    path = make_enhanced_dicom()
    result = extract_parameters(path)
    assert result.dicom_format == "enhanced"


# ── extract_from_series ──────────────────────────────────────────────────────


def test_extract_from_series_first_file(dicom_study):
    result = extract_from_series(dicom_study)
    assert result.params  # got params from first file
    assert result.device.manufacturer == "SIEMENS"


def test_extract_from_series_empty():
    result = extract_from_series([])
    assert result.params == {}
    assert result.dicom_format == "classic"
