"""Basic protocol compliance backend using pydicom parameter extraction.

Compares extracted DICOM parameters against template rules with configurable
tolerances and severity levels. Supports numeric, exact match, contains_all,
and range comparison types.
"""

import logging
import math

from .base import Finding, ProtocolBackend, ProtocolResult

log = logging.getLogger(__name__)

# Severity precedence: critical > warning > info
_SEVERITY_RANK = {"critical": 3, "warning": 2, "info": 1}


class BasicBackend(ProtocolBackend):
    """Compare DICOM parameters against template rules using basic logic."""

    def __init__(self, default_tolerance: float = 5.0) -> None:
        self._default_tolerance = default_tolerance

    @property
    def name(self) -> str:
        return "basic"

    def available(self) -> bool:
        try:
            import pydicom  # noqa: F401

            return True
        except ImportError:
            return False

    def check(self, params: dict, rules: list[dict]) -> ProtocolResult:
        findings: list[Finding] = []
        worst_severity = 0

        for rule in rules:
            keyword = rule.get("tag_keyword", "")
            if not keyword:
                continue

            severity = rule.get("severity", "warning")
            expected = rule.get("target")
            match_type = rule.get("match_type", "numeric")
            tolerance_pct = rule.get("tolerance_pct", self._default_tolerance)
            tolerance_abs = rule.get("tolerance_abs")
            note = rule.get("note", "")

            actual = params.get(keyword)

            # Tag missing from DICOM
            if actual is None:
                findings.append(
                    Finding(
                        tag_keyword=keyword,
                        severity=severity,
                        status="missing",
                        message=f"{keyword} not found in DICOM headers"
                        + (f" ({note})" if note else ""),
                        expected=str(expected),
                        actual="(missing)",
                    )
                )
                worst_severity = max(worst_severity, _SEVERITY_RANK.get(severity, 0))
                continue

            if match_type == "numeric":
                finding = self._check_numeric(
                    keyword, expected, actual, tolerance_pct, tolerance_abs, severity, note
                )
            elif match_type == "exact":
                finding = self._check_exact(keyword, expected, actual, severity, note)
            elif match_type == "contains_all":
                finding = self._check_contains_all(
                    keyword, expected, actual, severity, note
                )
            elif match_type == "range":
                finding = self._check_range(keyword, expected, actual, severity, note)
            else:
                finding = self._check_numeric(
                    keyword, expected, actual, tolerance_pct, tolerance_abs, severity, note
                )

            findings.append(finding)
            if finding.status != "compliant":
                worst_severity = max(
                    worst_severity, _SEVERITY_RANK.get(severity, 0)
                )

        # Determine overall compliance
        if worst_severity >= 3:
            overall = "non_compliant"
        elif worst_severity >= 2:
            overall = "minor_deviations"
        else:
            overall = "compliant"

        return ProtocolResult(findings=findings, overall=overall)

    def _check_numeric(
        self,
        keyword: str,
        expected,
        actual,
        tolerance_pct: float,
        tolerance_abs: float | None,
        severity: str,
        note: str,
    ) -> Finding:
        """Compare numeric values with tolerance."""
        try:
            exp_val = float(expected)
        except (TypeError, ValueError):
            return self._check_exact(keyword, expected, actual, severity, note)

        # Handle list values (e.g. PixelSpacing [0.5, 0.5])
        if isinstance(actual, list):
            if isinstance(expected, list):
                return self._check_numeric_list(
                    keyword, expected, actual, tolerance_pct, tolerance_abs, severity, note
                )
            actual = actual[0] if actual else 0.0

        try:
            act_val = float(actual)
        except (TypeError, ValueError):
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="deviated",
                message=f"{keyword}: expected numeric {exp_val}, got non-numeric '{actual}'"
                + (f" ({note})" if note else ""),
                expected=str(exp_val),
                actual=str(actual),
            )

        if exp_val == 0:
            deviated = act_val != 0
            deviation_pct = None
        else:
            deviation_pct = abs(act_val - exp_val) / abs(exp_val) * 100

            if tolerance_abs is not None:
                deviated = abs(act_val - exp_val) > tolerance_abs
            else:
                deviated = deviation_pct > tolerance_pct

        if deviated:
            pct_str = f" ({deviation_pct:.1f}% deviation)" if deviation_pct is not None else ""
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="deviated",
                message=f"{keyword}: expected {exp_val}, got {act_val}{pct_str}"
                + (f" ({note})" if note else ""),
                expected=str(exp_val),
                actual=str(act_val),
                deviation_pct=deviation_pct,
            )

        return Finding(
            tag_keyword=keyword,
            severity=severity,
            status="compliant",
            message=f"{keyword}: {act_val} within tolerance of {exp_val}"
            + (f" ({note})" if note else ""),
            expected=str(exp_val),
            actual=str(act_val),
            deviation_pct=deviation_pct if deviation_pct is not None else 0.0,
        )

    def _check_numeric_list(
        self,
        keyword: str,
        expected: list,
        actual: list,
        tolerance_pct: float,
        tolerance_abs: float | None,
        severity: str,
        note: str,
    ) -> Finding:
        """Compare list of numeric values element-wise."""
        max_deviation = 0.0
        any_deviated = False

        for i, (exp, act) in enumerate(
            zip(expected, actual[: len(expected)])
        ):
            try:
                e, a = float(exp), float(act)
            except (TypeError, ValueError):
                any_deviated = True
                break
            if e == 0:
                if a != 0:
                    any_deviated = True
            else:
                dev = abs(a - e) / abs(e) * 100
                max_deviation = max(max_deviation, dev)
                if tolerance_abs is not None:
                    if abs(a - e) > tolerance_abs:
                        any_deviated = True
                elif dev > tolerance_pct:
                    any_deviated = True

        if any_deviated:
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="deviated",
                message=f"{keyword}: expected {expected}, got {actual} "
                f"(max deviation {max_deviation:.1f}%)"
                + (f" ({note})" if note else ""),
                expected=str(expected),
                actual=str(actual),
                deviation_pct=max_deviation,
            )

        return Finding(
            tag_keyword=keyword,
            severity=severity,
            status="compliant",
            message=f"{keyword}: {actual} within tolerance of {expected}"
            + (f" ({note})" if note else ""),
            expected=str(expected),
            actual=str(actual),
            deviation_pct=max_deviation,
        )

    def _check_exact(
        self, keyword: str, expected, actual, severity: str, note: str
    ) -> Finding:
        """Compare values for exact match."""
        exp_str = str(expected).upper().strip()
        act_str = str(actual).upper().strip()

        if exp_str == act_str:
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="compliant",
                message=f"{keyword}: '{actual}' matches expected"
                + (f" ({note})" if note else ""),
                expected=str(expected),
                actual=str(actual),
            )

        return Finding(
            tag_keyword=keyword,
            severity=severity,
            status="deviated",
            message=f"{keyword}: expected '{expected}', got '{actual}'"
            + (f" ({note})" if note else ""),
            expected=str(expected),
            actual=str(actual),
        )

    def _check_contains_all(
        self, keyword: str, expected, actual, severity: str, note: str
    ) -> Finding:
        """Check that actual value contains all expected values (for multi-value tags like ScanningSequence)."""
        if isinstance(expected, str):
            expected_set = {s.strip().upper() for s in expected.split("\\")}
        elif isinstance(expected, list):
            expected_set = {str(s).strip().upper() for s in expected}
        else:
            expected_set = {str(expected).strip().upper()}

        if isinstance(actual, str):
            actual_set = {s.strip().upper() for s in actual.split("\\")}
        elif isinstance(actual, list):
            actual_set = {str(s).strip().upper() for s in actual}
        else:
            actual_set = {str(actual).strip().upper()}

        missing = expected_set - actual_set
        if not missing:
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="compliant",
                message=f"{keyword}: contains all expected values {sorted(expected_set)}"
                + (f" ({note})" if note else ""),
                expected=str(sorted(expected_set)),
                actual=str(sorted(actual_set)),
            )

        return Finding(
            tag_keyword=keyword,
            severity=severity,
            status="deviated",
            message=f"{keyword}: missing {sorted(missing)} (expected {sorted(expected_set)}, got {sorted(actual_set)})"
            + (f" ({note})" if note else ""),
            expected=str(sorted(expected_set)),
            actual=str(sorted(actual_set)),
        )

    def _check_range(
        self, keyword: str, expected, actual, severity: str, note: str
    ) -> Finding:
        """Check that actual value falls within [min, max] range."""
        if not isinstance(expected, list) or len(expected) != 2:
            return self._check_exact(keyword, expected, actual, severity, note)

        try:
            lo, hi = float(expected[0]), float(expected[1])
        except (TypeError, ValueError):
            return self._check_exact(keyword, expected, actual, severity, note)

        act_val = actual
        if isinstance(actual, list):
            act_val = actual[0] if actual else 0.0

        try:
            act_num = float(act_val)
        except (TypeError, ValueError):
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="deviated",
                message=f"{keyword}: expected range [{lo}, {hi}], got non-numeric '{actual}'"
                + (f" ({note})" if note else ""),
                expected=f"[{lo}, {hi}]",
                actual=str(actual),
            )

        if lo <= act_num <= hi:
            return Finding(
                tag_keyword=keyword,
                severity=severity,
                status="compliant",
                message=f"{keyword}: {act_num} within range [{lo}, {hi}]"
                + (f" ({note})" if note else ""),
                expected=f"[{lo}, {hi}]",
                actual=str(act_num),
            )

        return Finding(
            tag_keyword=keyword,
            severity=severity,
            status="deviated",
            message=f"{keyword}: {act_num} outside range [{lo}, {hi}]"
            + (f" ({note})" if note else ""),
            expected=f"[{lo}, {hi}]",
            actual=str(act_num),
        )
