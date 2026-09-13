"""Token-level free-text PHI scrubbing (regex and remote backends).

Ported from ``client/src/dicom/text_scrub.ts`` and
``phi-detection/app/backends/text_scrub.py`` (RegexTextScrubBackend).
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass

_REPLACEMENT = "[REMOVED]"

# ---------------------------------------------------------------------------
# PHI detection patterns
# ---------------------------------------------------------------------------
_PHI_PATTERNS: list[tuple[re.Pattern[str], str]] = [
    (re.compile(r"\b\d{3}-\d{2}-\d{4}\b"), "SSN"),
    (re.compile(r"\b\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b"), "phone"),
    (re.compile(r"\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b", re.I), "email"),
    (re.compile(r"\b(?:MRN|MR#|MED\s*REC)\s*[:#]?\s*\d+\b", re.I), "MRN"),
    (re.compile(r"\b(?:ACC|ACCESSION)\s*[:#]?\s*[A-Z0-9\-]+\b", re.I), "accession"),
    (re.compile(r"\b(?:0?[1-9]|1[0-2])[/\-](?:0?[1-9]|[12]\d|3[01])[/\-](?:19|20)?\d{2}\b"), "date_us"),
    (
        re.compile(
            r"\b(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|"
            r"Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)"
            r"\s+\d{1,2},?\s*\d{4}\b",
            re.I,
        ),
        "date_written",
    ),
    (re.compile(r"\b(?:19|20)\d{2}[/\-](?:0?[1-9]|1[0-2])[/\-](?:0?[1-9]|[12]\d|3[01])\b"), "date_iso"),
    (
        re.compile(
            r"\b\d+\s+\w+(?:\s+\w+)?\s+(?:street|avenue|road|boulevard|drive|lane|"
            r"court|way|place|circle|parkway)\b",
            re.I,
        ),
        "address",
    ),
    (re.compile(r"\b[A-Z]{2}\s+\d{5}(?:-\d{4})?\b"), "zip_state"),
    (re.compile(r"\b(?:Dr|Mr|Mrs|Ms|Prof|Rev)\.?\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*"), "title_name"),
    (
        re.compile(
            r"\b\w+(?:\s+\w+)*\s+(?:Hospital|Clinic|Medical\s+Center|Health\s+System|"
            r"Health\s+Center|Healthcare|University\s+Hospital)\b",
            re.I,
        ),
        "hospital",
    ),
    (re.compile(r"\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b"), "ip_address"),
    (re.compile(r"\bhttps?://[^\s]+", re.I), "url"),
    (re.compile(r"\b(?:Patient|Pt)\s*[:#]?\s*[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*"), "patient_prefix"),
]

# ---------------------------------------------------------------------------
# Medical terms that should NOT be flagged as person names
# ---------------------------------------------------------------------------
MEDICAL_TERMS: set[str] = {
    # anatomy
    "head", "brain", "chest", "abdomen", "pelvis", "spine", "neck", "shoulder",
    "knee", "hip", "ankle", "wrist", "elbow", "foot", "hand", "finger", "toe",
    "lung", "heart", "liver", "kidney", "pancreas", "spleen", "colon", "rectum",
    "skull", "femur", "tibia", "fibula", "humerus", "radius", "ulna", "clavicle",
    # imaging
    "axial", "sagittal", "coronal", "oblique", "scout", "topogram", "localizer",
    "contrast", "gadolinium", "iodine", "injection", "bolus", "delay", "series",
    "sequence", "protocol", "acquisition", "reconstruction", "slice", "volume",
    "phase", "dynamic", "perfusion", "diffusion",
    # modality
    "ct", "mri", "mr", "pet", "spect", "ultrasound", "xray", "mammography",
    "fluoroscopy", "angiography", "tomography",
    # common words
    "with", "without", "and", "the", "for", "of", "in", "on", "at", "to",
    "pre", "post", "follow", "up", "routine", "standard", "stat", "urgent",
    "bilateral", "unilateral", "left", "right", "anterior", "posterior",
    "superior", "inferior", "medial", "lateral", "proximal", "distal",
    # sequences
    "flair", "stir", "fiesta", "mprage", "space", "blade", "propeller",
    "dwi", "adc", "swi", "tof", "bold", "dti", "epi", "t1w", "t2w", "t1", "t2", "pd", "ir",
    # findings
    "normal", "abnormal", "acute", "chronic", "benign", "malignant",
    "fracture", "lesion", "mass", "nodule", "cyst", "stenosis", "occlusion",
}


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

@dataclass
class ScrubContext:
    """Known identifiers from the DICOM dataset, used for context matching."""

    patient_name: str | None = None
    patient_id: str | None = None
    referring_physician: str | None = None
    institution_name: str | None = None


@dataclass
class ScrubResult:
    text: str
    phi_found: bool


def scrub_free_text(value: str, context: ScrubContext | None = None) -> ScrubResult:
    """Scrub PHI from a free-text DICOM field value.

    Returns the scrubbed string with PHI tokens replaced by ``[REMOVED]``,
    plus a flag indicating whether any PHI was detected.
    """
    if not value:
        return ScrubResult(text="", phi_found=False)

    spans: list[tuple[int, int]] = []

    # --- context-based matching ---
    if context:
        for name_field in (context.patient_name, context.referring_physician):
            if name_field:
                for token in _name_to_tokens(name_field):
                    if token.lower() not in MEDICAL_TERMS:
                        pat = re.compile(r"\b" + re.escape(token) + r"\b", re.I)
                        for m in pat.finditer(value):
                            spans.append((m.start(), m.end()))
        if context.patient_id:
            pat = re.compile(r"\b" + re.escape(context.patient_id) + r"\b", re.I)
            for m in pat.finditer(value):
                spans.append((m.start(), m.end()))
        if context.institution_name:
            pat = re.compile(r"\b" + re.escape(context.institution_name) + r"\b", re.I)
            for m in pat.finditer(value):
                spans.append((m.start(), m.end()))

    # --- pattern-based matching ---
    for pattern, _label in _PHI_PATTERNS:
        for m in pattern.finditer(value):
            spans.append((m.start(), m.end()))

    if not spans:
        return ScrubResult(text=value, phi_found=False)

    merged = _merge_spans(spans)
    parts: list[str] = []
    cursor = 0
    for start, end in merged:
        parts.append(value[cursor:start])
        parts.append(_REPLACEMENT)
        cursor = end
    parts.append(value[cursor:])
    result = "".join(parts)

    # Collapse consecutive [REMOVED] tokens
    result = re.sub(r"(\[REMOVED\](?:\s+\[REMOVED\])+)", _REPLACEMENT, result)

    return ScrubResult(text=result, phi_found=True)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _name_to_tokens(name: str) -> list[str]:
    """Split a DICOM person name into searchable tokens."""
    tokens = re.split(r"[\^,\s]+", name)
    return [t.strip() for t in tokens if len(t.strip()) > 1]


def _merge_spans(spans: list[tuple[int, int]]) -> list[tuple[int, int]]:
    """Sort and merge overlapping or adjacent spans."""
    if not spans:
        return []
    sorted_spans = sorted(spans, key=lambda s: s[0])
    merged: list[tuple[int, int]] = [sorted_spans[0]]
    for start, end in sorted_spans[1:]:
        prev_start, prev_end = merged[-1]
        if start <= prev_end:
            merged[-1] = (prev_start, max(prev_end, end))
        else:
            merged.append((start, end))
    return merged


# ---------------------------------------------------------------------------
# Remote text scrub backend (calls phi-detection /scrub-text endpoint)
# ---------------------------------------------------------------------------

_remote_log = logging.getLogger(__name__)


class RemoteTextScrubBackend:
    """Text scrubbing backend that delegates to a remote phi-detection service.

    Falls back to the local regex backend if the service call fails.
    """

    def __init__(self, service_url: str, timeout: float = 10.0):
        self.service_url = service_url.rstrip("/")
        self.timeout = timeout

    def scrub(self, value: str, context: ScrubContext | None = None) -> ScrubResult:
        """Scrub PHI from text using the remote service, with regex fallback."""
        if not value:
            return ScrubResult(text="", phi_found=False)
        try:
            return self._call_remote(value, context)
        except Exception as e:
            _remote_log.warning("Remote scrub failed, falling back to regex: %s", e)
            return scrub_free_text(value, context)

    def scrub_batch(
        self, texts: list[str], context: ScrubContext | None = None,
    ) -> list[ScrubResult]:
        """Scrub multiple texts in a single HTTP call."""
        if not texts:
            return []
        try:
            return self._call_remote_batch(texts, context)
        except Exception as e:
            _remote_log.warning("Remote batch scrub failed, falling back to regex: %s", e)
            return [scrub_free_text(t, context) for t in texts]

    def _call_remote(self, value: str, context: ScrubContext | None) -> ScrubResult:
        results = self._call_remote_batch([value], context)
        return results[0]

    def _call_remote_batch(
        self, texts: list[str], context: ScrubContext | None,
    ) -> list[ScrubResult]:
        import urllib.request
        import json

        ctx_dict = None
        if context:
            ctx_dict = {}
            if context.patient_name:
                ctx_dict["patient_name"] = context.patient_name
            if context.patient_id:
                ctx_dict["patient_id"] = context.patient_id
            if context.referring_physician:
                ctx_dict["referring_physician"] = context.referring_physician
            if context.institution_name:
                ctx_dict["institution_name"] = context.institution_name

        payload = json.dumps({"texts": texts, "context": ctx_dict or None}).encode()
        req = urllib.request.Request(
            f"{self.service_url}/scrub-text",
            data=payload,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=self.timeout) as resp:
            body = json.loads(resp.read())

        results: list[ScrubResult] = []
        for item in body.get("results", []):
            results.append(ScrubResult(
                text=item.get("scrubbed", ""),
                phi_found=item.get("phi_found", False),
            ))

        # Pad with regex fallback if the service returned fewer results
        while len(results) < len(texts):
            results.append(scrub_free_text(texts[len(results)]))

        return results
