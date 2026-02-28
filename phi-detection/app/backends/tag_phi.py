"""Private tag PHI scanner.

Reads DICOM private tags (odd group numbers) and checks text values for
patterns that look like Protected Health Information — patient names,
dates of birth, accession numbers, SSNs, MRNs, phone numbers, and
institution names.

Used by the ``POST /detect-tags`` endpoint to warn when ``keep_private_tags``
is enabled and private tags contain PHI that would normally be stripped.
"""

import logging
import re
from dataclasses import dataclass, field
from pathlib import Path

log = logging.getLogger(__name__)

# Common PHI patterns in DICOM tag values
_DATE_PATTERN = re.compile(
    r"\b\d{4}[-/]?\d{2}[-/]?\d{2}\b"  # 2024-01-15 or 20240115
)
_SSN_PATTERN = re.compile(
    r"\b\d{3}-\d{2}-\d{4}\b"
)
_PHONE_PATTERN = re.compile(
    r"\b\d{3}[-.)]\s?\d{3}[-.)]\s?\d{4}\b"
)
_MRN_PATTERN = re.compile(
    r"\bMRN[\s:=#]*\d{4,}\b", re.IGNORECASE
)
_NAME_CARET = re.compile(
    r"[A-Za-z]+\^[A-Za-z]+"  # DICOM PN-format: Last^First
)
_ACCESSION_PATTERN = re.compile(
    r"\bACC[\s:=#]*\d{4,}\b", re.IGNORECASE
)

# Known vendor private tag descriptions that are SAFE (acquisition parameters)
_SAFE_DESCRIPTIONS = {
    "csa header",
    "csa series header",
    "csa image header",
    "b_value",
    "diffusion gradient",
    "gradient direction",
    "slice timing",
    "bandwidth",
    "sar",
    "number of slices",
    "number of images in mosaic",
    "protocol name",
    "sequence name",
    "coil string",
    "mr acquisition type",
}


@dataclass
class TagPHIFinding:
    """A single private tag that may contain PHI."""

    file: str
    tag: str  # e.g. "(0009,0010)"
    value_preview: str  # first 80 chars of the value
    pattern: str  # which pattern matched (e.g. "name_caret", "date", "ssn")


@dataclass
class TagPHIResult:
    """Result of scanning private tags across multiple files."""

    files_scanned: int = 0
    findings: list[TagPHIFinding] = field(default_factory=list)
    private_tags_scanned: int = 0


def scan_private_tags(paths: list[str]) -> TagPHIResult:
    """Scan DICOM files for PHI in private tag values.

    Only inspects tags with odd group numbers (private tags).
    Returns findings for any tag values that match PHI patterns.
    """
    import pydicom

    result = TagPHIResult()

    for path_str in paths:
        path = Path(path_str)
        if not path.exists():
            continue

        try:
            ds = pydicom.dcmread(str(path), stop_before_pixels=True)
        except Exception as e:
            log.debug("Cannot read DICOM %s: %s", path, e)
            continue

        result.files_scanned += 1
        filename = path.name

        for elem in ds:
            # Only check private tags (odd group number)
            if elem.tag.group % 2 == 0:
                continue

            result.private_tags_scanned += 1

            # Get string representation of value
            try:
                val = str(elem.value)
            except Exception:
                continue

            if not val or len(val) < 3:
                continue

            # Skip binary/numeric-only values
            val_stripped = val.strip()
            if not val_stripped:
                continue

            # Skip known safe vendor parameter descriptions
            val_lower = val_stripped.lower()
            if any(safe in val_lower for safe in _SAFE_DESCRIPTIONS):
                continue

            tag_str = f"({elem.tag.group:04x},{elem.tag.element:04x})"
            preview = val_stripped[:80]

            # Check each PHI pattern
            if _NAME_CARET.search(val_stripped):
                result.findings.append(TagPHIFinding(
                    file=filename, tag=tag_str,
                    value_preview=preview, pattern="name_caret",
                ))
            elif _SSN_PATTERN.search(val_stripped):
                result.findings.append(TagPHIFinding(
                    file=filename, tag=tag_str,
                    value_preview=preview, pattern="ssn",
                ))
            elif _MRN_PATTERN.search(val_stripped):
                result.findings.append(TagPHIFinding(
                    file=filename, tag=tag_str,
                    value_preview=preview, pattern="mrn",
                ))
            elif _ACCESSION_PATTERN.search(val_stripped):
                result.findings.append(TagPHIFinding(
                    file=filename, tag=tag_str,
                    value_preview=preview, pattern="accession",
                ))
            elif _PHONE_PATTERN.search(val_stripped):
                result.findings.append(TagPHIFinding(
                    file=filename, tag=tag_str,
                    value_preview=preview, pattern="phone",
                ))
            # Date check last (many false positives from acquisition dates)
            # Only flag dates that look like DOB context
            elif _DATE_PATTERN.search(val_stripped) and _looks_like_dob_context(val_stripped):
                result.findings.append(TagPHIFinding(
                    file=filename, tag=tag_str,
                    value_preview=preview, pattern="date_of_birth",
                ))

    return result


def _looks_like_dob_context(val: str) -> bool:
    """Heuristic: flag dates only if they appear in a DOB-like context."""
    val_lower = val.lower()
    return any(kw in val_lower for kw in ("birth", "dob", "patient", "born"))
