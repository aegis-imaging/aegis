"""Basic QC backend using pydicom + numpy.

Works offline with no cloud dependencies or ML models. Suitable for local
development and basic automated QC in production as a first-pass filter.
"""

import logging
from pathlib import Path

import numpy as np

from .base import CheckResult, QCBackend, QCResult

log = logging.getLogger(__name__)


class BasicBackend(QCBackend):
    def __init__(self, snr_threshold: float = 10.0, gap_ratio_threshold: float = 2.0):
        self._snr_threshold = snr_threshold
        self._gap_ratio_threshold = gap_ratio_threshold

    @property
    def name(self) -> str:
        return "basic"

    def available(self) -> bool:
        try:
            import pydicom  # noqa: F401

            return True
        except ImportError:
            return False

    def check(self, dicom_paths: list[str]) -> QCResult:
        import pydicom

        datasets = []
        parse_errors: list[dict] = []
        for path in dicom_paths:
            try:
                ds = pydicom.dcmread(path)
                datasets.append((path, ds))
            except Exception as e:
                parse_errors.append({"file": Path(path).name, "error": str(e)})

        checks: list[CheckResult] = []

        # 1. File integrity
        checks.append(self._check_file_integrity(dicom_paths, datasets, parse_errors))

        if len(datasets) == 0:
            return QCResult(checks=checks, overall="fail")

        # 2. Slice consistency
        checks.append(self._check_slice_consistency(datasets))

        # 3. SNR estimation
        checks.append(self._check_snr(datasets))

        # 4. Coverage completeness
        checks.append(self._check_coverage(datasets))

        # 5. Missing slices
        checks.append(self._check_missing_slices(datasets))

        statuses = [c.status for c in checks]
        if "fail" in statuses:
            overall = "fail"
        elif "warn" in statuses:
            overall = "warn"
        else:
            overall = "pass"

        return QCResult(checks=checks, overall=overall)

    # ── Individual checks ──────────────────────────────────────────────────

    def _check_file_integrity(self, all_paths, parsed, errors) -> CheckResult:
        total = len(all_paths)
        ok = len(parsed)
        failed = len(errors)

        required_tags = ["Rows", "Columns", "BitsAllocated", "Modality"]
        missing_tag_files = []
        for path, ds in parsed:
            missing = [t for t in required_tags if not hasattr(ds, t)]
            if missing:
                missing_tag_files.append({"file": Path(path).name, "missing_tags": missing})

        details: dict = {
            "total_files": total,
            "parsed_ok": ok,
            "parse_failures": failed,
            "missing_tag_files": len(missing_tag_files),
        }
        if errors:
            details["errors"] = errors[:10]
        if missing_tag_files:
            details["missing_tags"] = missing_tag_files[:10]

        if failed > 0 or len(missing_tag_files) > 0:
            pct = failed / total * 100 if total > 0 else 0
            status = "fail" if pct > 10 else "warn"
            msg = f"{failed}/{total} files failed to parse, {len(missing_tag_files)} missing required tags"
        else:
            status = "pass"
            msg = f"All {total} files parsed successfully with required tags"

        return CheckResult(name="file_integrity", status=status, message=msg, details=details)

    def _check_slice_consistency(self, datasets) -> CheckResult:
        rows_set: set[int] = set()
        cols_set: set[int] = set()
        spacings: set[tuple] = set()

        for _path, ds in datasets:
            if hasattr(ds, "Rows"):
                rows_set.add(int(ds.Rows))
            if hasattr(ds, "Columns"):
                cols_set.add(int(ds.Columns))
            if hasattr(ds, "PixelSpacing"):
                ps = ds.PixelSpacing
                spacings.add((round(float(ps[0]), 4), round(float(ps[1]), 4)))

        details = {
            "unique_rows": sorted(rows_set),
            "unique_columns": sorted(cols_set),
            "unique_pixel_spacings": [list(s) for s in sorted(spacings)],
        }

        issues = []
        if len(rows_set) > 1:
            issues.append(f"inconsistent Rows: {sorted(rows_set)}")
        if len(cols_set) > 1:
            issues.append(f"inconsistent Columns: {sorted(cols_set)}")
        if len(spacings) > 1:
            issues.append(f"inconsistent PixelSpacing: {sorted(spacings)}")

        if issues:
            return CheckResult(
                name="slice_consistency",
                status="fail",
                message="; ".join(issues),
                details=details,
            )
        return CheckResult(
            name="slice_consistency",
            status="pass",
            message="All slices have consistent dimensions and spacing",
            details=details,
        )

    def _check_snr(self, datasets) -> CheckResult:
        snr_values = []
        for _path, ds in datasets:
            if not hasattr(ds, "PixelData"):
                continue
            try:
                arr = ds.pixel_array.astype(np.float64)
            except Exception:
                continue

            # Handle multi-frame or RGB
            if arr.ndim == 3:
                arr = arr[0]

            h, w = arr.shape[:2]
            corner_h = max(1, h // 10)
            corner_w = max(1, w // 10)
            corner = arr[:corner_h, :corner_w]

            noise_std = np.std(corner)
            signal_mean = np.mean(arr)

            if noise_std > 0:
                snr_values.append(signal_mean / noise_std)

        if not snr_values:
            return CheckResult(
                name="snr",
                status="warn",
                message="Could not compute SNR (no pixel data available)",
                details={"snr_values": []},
            )

        mean_snr = float(np.mean(snr_values))
        min_snr = float(np.min(snr_values))

        details = {
            "mean_snr": round(mean_snr, 2),
            "min_snr": round(min_snr, 2),
            "threshold": self._snr_threshold,
            "slices_measured": len(snr_values),
        }

        if min_snr < self._snr_threshold * 0.5:
            return CheckResult(
                name="snr",
                status="fail",
                message=f"Very low SNR: min={min_snr:.1f}, mean={mean_snr:.1f} (threshold={self._snr_threshold})",
                details=details,
            )
        elif min_snr < self._snr_threshold:
            return CheckResult(
                name="snr",
                status="warn",
                message=f"Low SNR: min={min_snr:.1f}, mean={mean_snr:.1f} (threshold={self._snr_threshold})",
                details=details,
            )
        return CheckResult(
            name="snr",
            status="pass",
            message=f"SNR acceptable: min={min_snr:.1f}, mean={mean_snr:.1f}",
            details=details,
        )

    def _check_coverage(self, datasets) -> CheckResult:
        expected_counts = {
            "HEAD": 100,
            "BRAIN": 100,
            "CHEST": 50,
            "ABDOMEN": 40,
            "PELVIS": 30,
            "SPINE": 20,
        }

        body_part = ""
        for _, ds in datasets:
            bp = getattr(ds, "BodyPartExamined", "")
            if bp:
                body_part = str(bp).upper()
                break

        count = len(datasets)
        details: dict = {"slice_count": count, "body_part": body_part}

        if body_part and body_part in expected_counts:
            expected = expected_counts[body_part]
            details["expected_min"] = expected
            ratio = count / expected
            if ratio < 0.3:
                return CheckResult(
                    name="coverage",
                    status="fail",
                    message=f"Very low slice count ({count}) for {body_part} (expected >={expected})",
                    details=details,
                )
            elif ratio < 0.6:
                return CheckResult(
                    name="coverage",
                    status="warn",
                    message=f"Potentially incomplete coverage: {count} slices for {body_part} (expected >={expected})",
                    details=details,
                )

        suffix = f" for {body_part}" if body_part else ""
        return CheckResult(
            name="coverage",
            status="pass",
            message=f"Slice count ({count}) appears adequate{suffix}",
            details=details,
        )

    def _check_missing_slices(self, datasets) -> CheckResult:
        positions = []
        for _path, ds in datasets:
            if hasattr(ds, "ImagePositionPatient"):
                pos = ds.ImagePositionPatient
                positions.append(float(pos[2]))
            elif hasattr(ds, "SliceLocation"):
                positions.append(float(ds.SliceLocation))

        if len(positions) < 3:
            return CheckResult(
                name="missing_slices",
                status="pass",
                message="Too few slices to assess gaps (< 3 with position data)",
                details={"positions_found": len(positions)},
            )

        positions.sort()
        spacings = np.diff(positions)

        if len(spacings) == 0:
            return CheckResult(
                name="missing_slices",
                status="pass",
                message="Cannot compute spacing differences",
                details={},
            )

        median_spacing = float(np.median(np.abs(spacings)))
        if median_spacing == 0:
            return CheckResult(
                name="missing_slices",
                status="warn",
                message="Median slice spacing is zero — possible localizer or single-plane acquisition",
                details={"median_spacing": 0},
            )

        gaps = []
        for i, sp in enumerate(spacings):
            ratio = abs(float(sp)) / median_spacing
            if ratio > self._gap_ratio_threshold:
                gaps.append(
                    {
                        "between_slices": [i, i + 1],
                        "spacing": round(abs(float(sp)), 3),
                        "ratio": round(ratio, 2),
                    }
                )

        details = {
            "total_slices_with_position": len(positions),
            "median_spacing_mm": round(median_spacing, 3),
            "gap_threshold_ratio": self._gap_ratio_threshold,
            "gaps_found": len(gaps),
        }
        if gaps:
            details["gaps"] = gaps[:20]

        if gaps:
            status = "fail" if len(gaps) > 3 else "warn"
            return CheckResult(
                name="missing_slices",
                status=status,
                message=f"{len(gaps)} gap(s) detected in slice positions (>{self._gap_ratio_threshold}x median spacing)",
                details=details,
            )

        return CheckResult(
            name="missing_slices",
            status="pass",
            message="No significant gaps in slice positions",
            details=details,
        )
