"""Unit tests for BasicBackend protocol compliance checking."""

from app.backends.basic import BasicBackend


backend = BasicBackend(default_tolerance=5.0)


# ── Numeric match type ───────────────────────────────────────────────────────


def test_numeric_compliant():
    params = {"RepetitionTime": 2000.0}
    rules = [{"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
              "severity": "critical", "tolerance_pct": 5.0}]
    result = backend.check(params, rules)
    assert result.overall == "compliant"
    assert result.findings[0].status == "compliant"


def test_numeric_deviated():
    params = {"RepetitionTime": 2500.0}
    rules = [{"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
              "severity": "critical", "tolerance_pct": 5.0}]
    result = backend.check(params, rules)
    assert result.overall == "non_compliant"
    assert result.findings[0].status == "deviated"
    assert result.findings[0].deviation_pct > 20.0


def test_numeric_within_tolerance():
    params = {"RepetitionTime": 2098.0}  # 4.9% deviation
    rules = [{"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
              "severity": "warning", "tolerance_pct": 5.0}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "compliant"


def test_numeric_tolerance_abs():
    params = {"EchoTime": 3.8}  # 0.3 absolute deviation from 3.5
    rules = [{"tag_keyword": "EchoTime", "target": 3.5, "match_type": "numeric",
              "severity": "warning", "tolerance_pct": 1.0, "tolerance_abs": 0.5}]
    result = backend.check(params, rules)
    # Absolute tolerance wins: |3.8 - 3.5| = 0.3 < 0.5
    assert result.findings[0].status == "compliant"


def test_numeric_zero_target():
    params = {"InversionTime": 0.0}
    rules = [{"tag_keyword": "InversionTime", "target": 0.0, "match_type": "numeric",
              "severity": "info"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "compliant"


def test_numeric_zero_target_deviated():
    params = {"InversionTime": 5.0}
    rules = [{"tag_keyword": "InversionTime", "target": 0.0, "match_type": "numeric",
              "severity": "warning"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "deviated"


# ── Exact match type ─────────────────────────────────────────────────────────


def test_exact_match():
    params = {"MRAcquisitionType": "3D"}
    rules = [{"tag_keyword": "MRAcquisitionType", "target": "3D", "match_type": "exact",
              "severity": "critical"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "compliant"


def test_exact_mismatch():
    params = {"MRAcquisitionType": "2D"}
    rules = [{"tag_keyword": "MRAcquisitionType", "target": "3D", "match_type": "exact",
              "severity": "critical"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "deviated"
    assert result.overall == "non_compliant"


# ── Contains_all match type ──────────────────────────────────────────────────


def test_contains_all_compliant():
    params = {"ScanningSequence": ["GR", "IR"]}
    rules = [{"tag_keyword": "ScanningSequence", "target": ["GR", "IR"],
              "match_type": "contains_all", "severity": "warning"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "compliant"


def test_contains_all_missing():
    params = {"ScanningSequence": ["GR"]}
    rules = [{"tag_keyword": "ScanningSequence", "target": ["GR", "IR"],
              "match_type": "contains_all", "severity": "warning"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "deviated"


# ── Range match type ─────────────────────────────────────────────────────────


def test_range_within():
    params = {"FlipAngle": 90.0}
    rules = [{"tag_keyword": "FlipAngle", "target": [85.0, 95.0], "match_type": "range",
              "severity": "warning"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "compliant"


def test_range_outside():
    params = {"FlipAngle": 70.0}
    rules = [{"tag_keyword": "FlipAngle", "target": [85.0, 95.0], "match_type": "range",
              "severity": "critical"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "deviated"
    assert result.overall == "non_compliant"


# ── Missing tag ──────────────────────────────────────────────────────────────


def test_missing_tag():
    params = {}  # No tags at all
    rules = [{"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
              "severity": "critical"}]
    result = backend.check(params, rules)
    assert result.findings[0].status == "missing"
    assert result.overall == "non_compliant"


# ── Severity aggregation ────────────────────────────────────────────────────


def test_overall_severity_aggregation():
    params = {
        "RepetitionTime": 2000.0,  # compliant
        "EchoTime": 999.0,         # deviated (critical)
        "FlipAngle": 9.1,          # compliant
    }
    rules = [
        {"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
         "severity": "info", "tolerance_pct": 5.0},
        {"tag_keyword": "EchoTime", "target": 3.5, "match_type": "numeric",
         "severity": "critical", "tolerance_pct": 5.0},
        {"tag_keyword": "FlipAngle", "target": 9.0, "match_type": "numeric",
         "severity": "warning", "tolerance_pct": 5.0},
    ]
    result = backend.check(params, rules)
    assert result.overall == "non_compliant"  # critical trumps all


def test_overall_minor_deviations():
    params = {"RepetitionTime": 2000.0, "EchoTime": 3.8}
    rules = [
        {"tag_keyword": "RepetitionTime", "target": 2000.0, "match_type": "numeric",
         "severity": "info", "tolerance_pct": 5.0},
        {"tag_keyword": "EchoTime", "target": 3.5, "match_type": "numeric",
         "severity": "warning", "tolerance_pct": 1.0},  # 8.6% > 1% → deviated
    ]
    result = backend.check(params, rules)
    assert result.overall == "minor_deviations"
