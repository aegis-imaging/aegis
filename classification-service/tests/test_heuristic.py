"""Unit tests for HeuristicBackend classification logic."""

from app.backends.heuristic import HeuristicBackend, _match_body_part


backend = HeuristicBackend()


# ── Strategy 1: Direct DICOM tags ────────────────────────────────────────────


def test_classify_direct_tags_mr_head(tmp_path, make_dicom_file):
    path = make_dicom_file(tmp_dir=tmp_path, modality="MR", body_part="HEAD")
    result = backend.classify([path])
    assert result.modality == "MR"
    assert result.body_part == "HEAD"
    assert result.confidence == 0.95
    assert result.method == "dicom_tags"


def test_classify_direct_tags_ct_chest(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality="CT",
        body_part="CHEST",
        sop_class_uid="1.2.840.10008.5.1.4.1.1.2",
    )
    result = backend.classify([path])
    assert result.modality == "CT"
    assert result.body_part == "CHEST"
    assert result.confidence == 0.95


# ── Strategy 2: SOP Class UID → modality ─────────────────────────────────────


def test_classify_sop_class_ct(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        sop_class_uid="1.2.840.10008.5.1.4.1.1.2",  # CT Image Storage
        modality_value=None,  # omit Modality tag
        body_part_value=None,
        series_description="CHEST CT",
    )
    result = backend.classify([path])
    assert result.modality == "CT"
    assert result.body_part == "CHEST"
    # SOP UID finds modality, then series_description finds body_part → returns from strategy 3
    assert result.confidence == 0.75
    assert result.method == "series_description"


def test_classify_sop_class_pet(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        sop_class_uid="1.2.840.10008.5.1.4.1.1.128",  # PET Image Storage
        modality_value=None,
        body_part_value=None,
        series_description="FDG PET",
        study_description="",
        protocol_name="",
    )
    result = backend.classify([path])
    assert result.modality == "PT"
    assert result.confidence >= 0.60


def test_classify_sop_class_us(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        sop_class_uid="1.2.840.10008.5.1.4.1.1.6",  # Ultrasound
        modality_value=None,
        body_part_value=None,
        series_description="",
        study_description="",
        protocol_name="",
    )
    result = backend.classify([path])
    assert result.modality == "US"
    assert result.confidence == 0.60
    assert result.method == "sop_class"


# ── Strategy 3: SeriesDescription / ProtocolName → body part ─────────────────


def test_classify_series_description_brain(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality="MR",
        body_part_value=None,
        series_description="BRAIN T1 MPRAGE",
    )
    result = backend.classify([path])
    assert result.modality == "MR"
    assert result.body_part == "HEAD"
    assert result.confidence == 0.75
    assert result.method == "series_description"


def test_classify_series_description_chest(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality="CT",
        body_part_value=None,
        series_description="CHEST CT WITH CONTRAST",
        sop_class_uid="1.2.840.10008.5.1.4.1.1.2",
    )
    result = backend.classify([path])
    assert result.modality == "CT"
    assert result.body_part == "CHEST"
    assert result.confidence == 0.75


def test_classify_protocol_name_spine(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality="MR",
        body_part_value=None,
        series_description="",
        protocol_name="LUMBAR SPINE T2",
    )
    result = backend.classify([path])
    assert result.body_part == "SPINE"


# ── Strategy 4: StudyDescription fallback ────────────────────────────────────


def test_classify_study_description_fallback(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality_value=None,
        body_part_value=None,
        sop_class_uid="1.2.840.10008.5.1.4.1.1.4",  # MR
        series_description="",
        protocol_name="",
        study_description="ABDOMEN MRI",
    )
    result = backend.classify([path])
    assert result.body_part == "ABDOMEN"
    assert result.confidence <= 0.80


# ── Edge cases ───────────────────────────────────────────────────────────────


def test_classify_no_valid_files():
    result = backend.classify(["/nonexistent/path.dcm"])
    assert result.modality == ""
    assert result.body_part == ""
    assert result.confidence == 0.0


def test_classify_modality_only(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality_value=None,
        body_part_value=None,
        sop_class_uid="1.2.840.10008.5.1.4.1.1.7",  # Secondary Capture
        series_description="",
        protocol_name="",
        study_description="",
    )
    result = backend.classify([path])
    assert result.modality == "SC"
    assert result.body_part == ""
    assert result.confidence == 0.60


def test_classify_body_part_only(tmp_path, make_dicom_file):
    path = make_dicom_file(
        tmp_dir=tmp_path,
        modality_value=None,
        body_part_value=None,
        sop_class_uid="1.2.840.10008.5.1.4.1.1.88.11",  # Unknown SOP class
        series_description="HEAD SCAN",
        protocol_name="",
        study_description="",
    )
    result = backend.classify([path])
    assert result.body_part == "HEAD"
    assert result.confidence == 0.50
    assert result.method == "description_only"


# ── _match_body_part helper ──────────────────────────────────────────────────


def test_match_body_part_patterns():
    assert _match_body_part("brain mri") == "HEAD"
    assert _match_body_part("CRANIAL SCAN") == "HEAD"
    assert _match_body_part("thorax ct") == "CHEST"
    assert _match_body_part("lung screening") == "CHEST"
    assert _match_body_part("kidney study") == "ABDOMEN"
    assert _match_body_part("pelvis mri") == "ABDOMEN"
    assert _match_body_part("cervical spine") == "SPINE"
    assert _match_body_part("lumbar") == "SPINE"
    assert _match_body_part("knee mri") == "EXTREMITY"
    assert _match_body_part("shoulder") == "EXTREMITY"
    assert _match_body_part("thyroid ultrasound") == "NECK"
    assert _match_body_part("random text") is None
