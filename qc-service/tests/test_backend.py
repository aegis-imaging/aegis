"""Unit tests for BasicBackend QC checks."""

import numpy as np
import pytest

from app.backends.basic import BasicBackend


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _backend(**kwargs) -> BasicBackend:
    return BasicBackend(
        snr_threshold=kwargs.get("snr_threshold", 10.0),
        gap_ratio_threshold=kwargs.get("gap_ratio_threshold", 2.0),
    )


# ---------------------------------------------------------------------------
# Full pipeline tests
# ---------------------------------------------------------------------------


class TestCheckPipeline:
    """End-to-end checks on BasicBackend.check()."""

    def test_all_pass(self, dicom_study):
        """Clean study with uniform data should pass all 5 checks."""
        result = _backend().check(dicom_study)
        assert result.overall == "pass"
        assert len(result.checks) == 5
        names = [c.name for c in result.checks]
        assert names == [
            "file_integrity",
            "slice_consistency",
            "snr",
            "coverage",
            "missing_slices",
        ]
        for c in result.checks:
            assert c.status == "pass", f"{c.name} was {c.status}: {c.message}"

    def test_overall_fail_propagates(self, tmp_path, make_dicom_file):
        """If any check is fail, overall is fail."""
        # Inconsistent dimensions → slice_consistency fail
        study_uid = "1.2.3"
        p1 = make_dicom_file(tmp_path, "a.dcm", rows=64, cols=64, study_uid=study_uid)
        arr = np.full((32, 32), 500, dtype=np.uint16)
        p2 = make_dicom_file(
            tmp_path, "b.dcm", pixel_array=arr, study_uid=study_uid
        )
        result = _backend().check([p1, p2])
        assert result.overall == "fail"

    def test_no_parseable_files(self, tmp_path):
        """Non-DICOM files → file_integrity fail, only 1 check returned."""
        bad = tmp_path / "garbage.dcm"
        bad.write_text("not a dicom file")
        result = _backend().check([str(bad)])
        assert result.overall == "fail"
        assert len(result.checks) == 1
        assert result.checks[0].name == "file_integrity"


# ---------------------------------------------------------------------------
# File integrity
# ---------------------------------------------------------------------------


class TestFileIntegrity:
    def test_all_valid(self, dicom_study):
        result = _backend().check(dicom_study)
        fi = [c for c in result.checks if c.name == "file_integrity"][0]
        assert fi.status == "pass"
        assert fi.details["total_files"] == 5
        assert fi.details["parsed_ok"] == 5
        assert fi.details["parse_failures"] == 0

    def test_one_bad_file_warn(self, tmp_path, make_dicom_file, dicom_study):
        """1 out of 6 fails → <10% threshold not met, status=warn."""
        bad = tmp_path / "bad.dcm"
        bad.write_text("not dicom")
        result = _backend().check(dicom_study + [str(bad)])
        fi = [c for c in result.checks if c.name == "file_integrity"][0]
        # 1/6 = 16.7% > 10% → fail
        assert fi.status == "fail"

    def test_missing_required_tags(self, tmp_path):
        """DICOM file missing required tags → warn or fail."""
        import pydicom
        from pydicom.uid import ExplicitVRLittleEndian, generate_uid as gu

        filepath = str(tmp_path / "notags.dcm")
        file_meta = pydicom.Dataset()
        file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
        file_meta.MediaStorageSOPInstanceUID = gu()
        file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
        ds = pydicom.dataset.FileDataset(
            filepath, {}, file_meta=file_meta, preamble=b"\x00" * 128
        )
        ds.SOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
        ds.SOPInstanceUID = gu()
        # Intentionally omit Rows, Columns, BitsAllocated, Modality
        ds.save_as(filepath)

        result = _backend().check([filepath])
        fi = [c for c in result.checks if c.name == "file_integrity"][0]
        assert fi.status in ("warn", "fail")
        assert fi.details["missing_tag_files"] > 0


# ---------------------------------------------------------------------------
# Slice consistency
# ---------------------------------------------------------------------------


class TestSliceConsistency:
    def test_consistent(self, dicom_study):
        result = _backend().check(dicom_study)
        sc = [c for c in result.checks if c.name == "slice_consistency"][0]
        assert sc.status == "pass"
        assert sc.details["unique_rows"] == [64]
        assert sc.details["unique_columns"] == [64]

    def test_inconsistent_rows(self, tmp_path, make_dicom_file):
        """Different Rows across slices → fail."""
        p1 = make_dicom_file(tmp_path, "a.dcm", rows=64, cols=64)
        arr = np.full((32, 64), 500, dtype=np.uint16)
        p2 = make_dicom_file(tmp_path, "b.dcm", pixel_array=arr)
        result = _backend().check([p1, p2])
        sc = [c for c in result.checks if c.name == "slice_consistency"][0]
        assert sc.status == "fail"
        assert 32 in sc.details["unique_rows"]
        assert 64 in sc.details["unique_rows"]

    def test_inconsistent_pixel_spacing(self, tmp_path, make_dicom_file):
        """Different PixelSpacing → fail."""
        p1 = make_dicom_file(tmp_path, "a.dcm", pixel_spacing=[1.0, 1.0])
        p2 = make_dicom_file(tmp_path, "b.dcm", pixel_spacing=[0.5, 0.5])
        result = _backend().check([p1, p2])
        sc = [c for c in result.checks if c.name == "slice_consistency"][0]
        assert sc.status == "fail"
        assert len(sc.details["unique_pixel_spacings"]) == 2


# ---------------------------------------------------------------------------
# SNR estimation
# ---------------------------------------------------------------------------


class TestSNR:
    def test_high_snr_pass(self, dicom_study):
        """Default study fixture has high SNR → pass."""
        result = _backend().check(dicom_study)
        snr = [c for c in result.checks if c.name == "snr"][0]
        assert snr.status == "pass"
        assert snr.details["mean_snr"] > 10.0

    def test_low_snr_warn(self, tmp_path, make_dicom_file):
        """SNR between threshold*0.5 and threshold → warn."""
        # threshold=10 → warn range: [5, 10)
        # Need: signal_mean / noise_std ≈ 7
        # 64×64 image: corner is 6×6=36 pixels
        arr = np.full((64, 64), 70, dtype=np.uint16)
        # Set corner to values with std ≈ 10
        corner_vals = np.array([60, 65, 70, 75, 80, 55] * 6, dtype=np.uint16)[:36]
        arr[:6, :6] = corner_vals.reshape(6, 6)
        # signal_mean ≈ 70, noise_std ≈ 8.5 → SNR ≈ 8.2

        p = make_dicom_file(tmp_path, "low_snr.dcm", pixel_array=arr)
        result = _backend(snr_threshold=10.0).check([p])
        snr = [c for c in result.checks if c.name == "snr"][0]
        assert snr.status == "warn"
        assert snr.details["min_snr"] < 10.0
        assert snr.details["min_snr"] >= 5.0

    def test_very_low_snr_fail(self, tmp_path, make_dicom_file):
        """SNR below threshold*0.5 → fail."""
        # Need: signal_mean / noise_std < 5
        # Set image mean low, corner noise high
        arr = np.full((64, 64), 20, dtype=np.uint16)
        corner_vals = np.array([5, 15, 25, 35, 10, 30] * 6, dtype=np.uint16)[:36]
        arr[:6, :6] = corner_vals.reshape(6, 6)
        # signal_mean ≈ 20, noise_std ≈ ~10 → SNR ≈ 2

        p = make_dicom_file(tmp_path, "vlow_snr.dcm", pixel_array=arr)
        result = _backend(snr_threshold=10.0).check([p])
        snr = [c for c in result.checks if c.name == "snr"][0]
        assert snr.status == "fail"
        assert snr.details["min_snr"] < 5.0

    def test_no_pixel_data_warn(self, tmp_path, make_dicom_file):
        """Files without PixelData → warn (can't compute)."""
        p = make_dicom_file(
            tmp_path, "nopx.dcm", include_pixel_data=False
        )
        result = _backend().check([p])
        snr = [c for c in result.checks if c.name == "snr"][0]
        assert snr.status == "warn"
        assert "no pixel data" in snr.message.lower()


# ---------------------------------------------------------------------------
# Coverage completeness
# ---------------------------------------------------------------------------


class TestCoverage:
    def test_adequate_head(self, tmp_path, make_dicom_file):
        """100+ slices for HEAD → pass."""
        paths = [
            make_dicom_file(
                tmp_path, f"s{i}.dcm", body_part="HEAD", instance_number=i
            )
            for i in range(120)
        ]
        result = _backend().check(paths)
        cov = [c for c in result.checks if c.name == "coverage"][0]
        assert cov.status == "pass"

    def test_low_count_warn(self, tmp_path, make_dicom_file):
        """50 slices for HEAD (50/100 = 0.5, < 0.6) → warn."""
        paths = [
            make_dicom_file(
                tmp_path, f"s{i}.dcm", body_part="HEAD", instance_number=i
            )
            for i in range(50)
        ]
        result = _backend().check(paths)
        cov = [c for c in result.checks if c.name == "coverage"][0]
        assert cov.status == "warn"

    def test_very_low_count_fail(self, tmp_path, make_dicom_file):
        """20 slices for HEAD (20/100 = 0.2, < 0.3) → fail."""
        paths = [
            make_dicom_file(
                tmp_path, f"s{i}.dcm", body_part="HEAD", instance_number=i
            )
            for i in range(20)
        ]
        result = _backend().check(paths)
        cov = [c for c in result.checks if c.name == "coverage"][0]
        assert cov.status == "fail"

    def test_unknown_body_part_pass(self, tmp_path, make_dicom_file):
        """Unknown body part → pass (no expected count)."""
        paths = [
            make_dicom_file(
                tmp_path, f"s{i}.dcm", body_part="FOOT", instance_number=i
            )
            for i in range(5)
        ]
        result = _backend().check(paths)
        cov = [c for c in result.checks if c.name == "coverage"][0]
        assert cov.status == "pass"


# ---------------------------------------------------------------------------
# Missing slices
# ---------------------------------------------------------------------------


class TestMissingSlices:
    def test_uniform_spacing_pass(self, dicom_study):
        """Uniform 1mm spacing → no gaps → pass."""
        result = _backend().check(dicom_study)
        ms = [c for c in result.checks if c.name == "missing_slices"][0]
        assert ms.status == "pass"
        assert ms.details["gaps_found"] == 0

    def test_gap_detected_warn(self, tmp_path, make_dicom_file):
        """One large gap → warn (<=3 gaps)."""
        # Positions: 0, 1, 2, 3, 10 → gap between 3 and 10 is 7mm vs median ~1mm
        positions = [0.0, 1.0, 2.0, 3.0, 10.0]
        paths = []
        for i, z in enumerate(positions):
            p = make_dicom_file(
                tmp_path,
                f"s{i}.dcm",
                image_position=[0.0, 0.0, z],
                slice_location=z,
                instance_number=i + 1,
            )
            paths.append(p)
        result = _backend().check(paths)
        ms = [c for c in result.checks if c.name == "missing_slices"][0]
        assert ms.status == "warn"
        assert ms.details["gaps_found"] >= 1

    def test_many_gaps_fail(self, tmp_path, make_dicom_file):
        """>3 gaps → fail."""
        # Positions with many gaps: 0, 1, 10, 11, 20, 21, 30, 31, 40, 41
        # Median spacing ≈ 1, gaps of 9 at indices 1→2, 3→4, 5→6, 7→8 = 4 gaps
        positions = [0, 1, 10, 11, 20, 21, 30, 31, 40, 41]
        paths = []
        for i, z in enumerate(positions):
            p = make_dicom_file(
                tmp_path,
                f"s{i}.dcm",
                image_position=[0.0, 0.0, float(z)],
                instance_number=i + 1,
            )
            paths.append(p)
        result = _backend().check(paths)
        ms = [c for c in result.checks if c.name == "missing_slices"][0]
        assert ms.status == "fail"
        assert ms.details["gaps_found"] > 3

    def test_too_few_slices_pass(self, tmp_path, make_dicom_file):
        """<3 slices with position data → pass (can't assess)."""
        paths = [
            make_dicom_file(
                tmp_path, f"s{i}.dcm", image_position=[0.0, 0.0, float(i)]
            )
            for i in range(2)
        ]
        result = _backend().check(paths)
        ms = [c for c in result.checks if c.name == "missing_slices"][0]
        assert ms.status == "pass"
        assert "too few" in ms.message.lower()

    def test_slice_location_fallback(self, tmp_path, make_dicom_file):
        """Uses SliceLocation when ImagePositionPatient is absent."""
        positions = [0.0, 1.0, 2.0, 3.0, 4.0]
        paths = []
        for i, z in enumerate(positions):
            p = make_dicom_file(
                tmp_path,
                f"s{i}.dcm",
                slice_location=z,
                instance_number=i + 1,
            )
            paths.append(p)
        result = _backend().check(paths)
        ms = [c for c in result.checks if c.name == "missing_slices"][0]
        assert ms.status == "pass"
        assert ms.details["total_slices_with_position"] == 5
