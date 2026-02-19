"""
Tests for app.pipeline — series grouping, defacing eligibility, and pipeline orchestration.
"""

import os
import shutil
from pathlib import Path
from unittest.mock import MagicMock

import pydicom
from pydicom.uid import generate_uid

from app.pipeline import group_by_series, should_deface_series, run_pipeline


# ---------------------------------------------------------------------------
# group_by_series
# ---------------------------------------------------------------------------

class TestGroupBySeries:
    """Tests for group_by_series()."""

    def test_single_series(self, dicom_series_25):
        """All files with the same SeriesInstanceUID land in one group."""
        groups = group_by_series(dicom_series_25)
        assert len(groups) == 1
        uid = list(groups.keys())[0]
        assert len(groups[uid]) == 25

    def test_multiple_series(self, multi_series_study):
        """Two distinct SeriesInstanceUIDs produce two groups."""
        paths_a, paths_b = multi_series_study
        all_paths = paths_a + paths_b
        groups = group_by_series(all_paths)
        assert len(groups) == 2
        sizes = sorted(len(v) for v in groups.values())
        assert sizes == [25, 25]

    def test_unreadable_files(self, tmp_path):
        """Non-DICOM files are grouped under the 'unknown' key."""
        junk = str(tmp_path / "garbage.dcm")
        with open(junk, "w") as f:
            f.write("this is not a DICOM file")

        groups = group_by_series([junk])
        assert "unknown" in groups
        assert len(groups["unknown"]) == 1


# ---------------------------------------------------------------------------
# should_deface_series
# ---------------------------------------------------------------------------

class TestShouldDefaceSeries:
    """Tests for should_deface_series()."""

    def test_too_few_slices(self, dicom_series_5):
        """5 slices is below the 20-slice threshold -- returns False."""
        assert should_deface_series(dicom_series_5) is False

    def test_enough_slices(self, dicom_series_25):
        """25 slices exceeds the 20-slice threshold -- returns True."""
        assert should_deface_series(dicom_series_25) is True

    def test_secondary_capture_skipped(self, make_dicom_file, tmp_path):
        """
        SOPClassUID for Secondary Capture (1.2.840.10008.5.1.4.1.1.7)
        should return False even with 25 slices.
        """
        series_uid = generate_uid()
        study_uid = generate_uid()
        sc_dir = str(tmp_path / "sc_series")
        os.makedirs(sc_dir, exist_ok=True)

        paths = []
        for i in range(25):
            p = make_dicom_file(
                sc_dir,
                filename=f"{i:04d}.dcm",
                series_uid=series_uid,
                study_uid=study_uid,
                sop_class_uid="1.2.840.10008.5.1.4.1.1.7",  # Secondary Capture
                instance_number=i + 1,
                image_position=(0.0, 0.0, float(i)),
            )
            paths.append(p)

        assert should_deface_series(paths) is False


# ---------------------------------------------------------------------------
# run_pipeline
# ---------------------------------------------------------------------------

class TestRunPipeline:
    """Tests for run_pipeline()."""

    def test_passthrough_below_threshold(self, dicom_series_5, tmp_path):
        """
        Series with < 20 slices are passed through (copied to output)
        without calling the defacing backend.
        """
        output_dir = str(tmp_path / "output")
        mock_backend = MagicMock()
        mock_backend.name = "mock"
        mock_backend.deface.return_value = []

        result = run_pipeline(dicom_series_5, output_dir, mock_backend)

        # Backend should never be called for a sub-threshold series
        mock_backend.deface.assert_not_called()

        # All 5 files should appear in the output
        assert len(result) == 5
        for p in result:
            assert os.path.exists(p)

    def test_with_mock_backend(self, dicom_series_25, tmp_path):
        """
        A qualifying 25-slice series triggers the defacing backend.
        Verify that backend.deface() is called exactly once.
        """
        output_dir = str(tmp_path / "output")

        def fake_deface(series_dir, out_dir):
            """Copy inputs to outputs (simulating a no-op deface)."""
            Path(out_dir).mkdir(parents=True, exist_ok=True)
            paths = []
            for f in sorted(Path(series_dir).glob("*.dcm")):
                dst = str(Path(out_dir) / f.name)
                shutil.copy2(str(f), dst)
                paths.append(dst)
            return paths

        mock_backend = MagicMock()
        mock_backend.name = "mock"
        mock_backend.deface.side_effect = fake_deface

        result = run_pipeline(dicom_series_25, output_dir, mock_backend)

        mock_backend.deface.assert_called_once()
        assert len(result) == 25
        for p in result:
            assert os.path.exists(p)

    def test_backend_failure_passthrough(self, dicom_series_25, tmp_path):
        """
        When the backend raises an exception, the pipeline falls back to
        copying the original files (passthrough).
        """
        output_dir = str(tmp_path / "output")

        mock_backend = MagicMock()
        mock_backend.name = "mock-fail"
        mock_backend.deface.side_effect = RuntimeError("backend crashed")

        result = run_pipeline(dicom_series_25, output_dir, mock_backend)

        # Files should still be copied as passthrough
        assert len(result) == 25
        for p in result:
            assert os.path.exists(p)

    def test_multi_series_pipeline(self, multi_series_study, tmp_path):
        """
        Pipeline processes each series independently. Two qualifying series
        should each trigger backend.deface() once (total 2 calls).
        """
        paths_a, paths_b = multi_series_study
        all_paths = paths_a + paths_b
        output_dir = str(tmp_path / "output")

        def fake_deface(series_dir, out_dir):
            Path(out_dir).mkdir(parents=True, exist_ok=True)
            paths = []
            for f in sorted(Path(series_dir).glob("*.dcm")):
                dst = str(Path(out_dir) / f.name)
                shutil.copy2(str(f), dst)
                paths.append(dst)
            return paths

        mock_backend = MagicMock()
        mock_backend.name = "mock"
        mock_backend.deface.side_effect = fake_deface

        result = run_pipeline(all_paths, output_dir, mock_backend)

        assert mock_backend.deface.call_count == 2
        assert len(result) == 50
