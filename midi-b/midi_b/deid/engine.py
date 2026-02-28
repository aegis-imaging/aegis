"""DICOM de-identification engine implementing PS3.15 Annex E Basic Profile.

Operates on pydicom ``Dataset`` objects. Returns a result with tag changes
for preview and the modified dataset.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from pydicom.dataset import Dataset

from .dateshift import DATE_TAGS, TIME_TAGS, shift_dicom_date
from .sequences import process_sequences
from .tags import BASIC_PROFILE, is_private_tag
from .text_scrub import ScrubContext, scrub_free_text
from .uid_hash import hash_identifier, hash_uid


@dataclass
class TagChange:
    """Record of a single tag modification."""

    tag: str
    keyword: str
    vr: str
    action: str
    original_value: str | None
    anonymized_value: str | None


@dataclass
class DeidOptions:
    """Options controlling de-identification behaviour."""

    salt: str = "aegis-default-salt"
    keep_private_tags: bool = False
    retained_tags: list[str] | None = None
    date_shift_offset: int | None = None


@dataclass
class DeidResult:
    """Outcome of a de-identification run."""

    tag_changes: list[TagChange] = field(default_factory=list)
    dataset: Dataset | None = None
    private_tags_removed: int = 0
    uid_mappings: dict[str, str] = field(default_factory=dict)
    patient_id_mapping: dict[str, str] | None = None
    date_shift_offset: int | None = None


def deidentify(ds: Dataset, options: DeidOptions | None = None) -> DeidResult:
    """Apply de-identification to a pydicom Dataset **in place**.

    Returns the modified dataset and a diff of all changes for UI preview.
    """
    opts = options or DeidOptions()
    salt = opts.salt
    tag_changes: list[TagChange] = []
    private_tags_removed = 0
    uid_mappings: dict[str, str] = {}
    patient_id_mapping: dict[str, str] | None = None

    # Build scrub context from dataset
    scrub_context = ScrubContext(
        patient_name=_get_str(ds, "PatientName"),
        patient_id=_get_str(ds, "PatientID"),
        referring_physician=_get_str(ds, "ReferringPhysicianName"),
        institution_name=_get_str(ds, "InstitutionName"),
    )

    for tag, rule in BASIC_PROFILE.items():
        keyword = rule["keyword"]
        action = rule["action"]

        original_value: Any = getattr(ds, keyword, None)
        if original_value is None and action != "K":
            continue

        original_str = _format_value(original_value)

        # Retained tags override — treat as Keep
        if opts.retained_tags and keyword in opts.retained_tags:
            tag_changes.append(TagChange(
                tag=tag, keyword=keyword, vr="", action="K",
                original_value=original_str, anonymized_value=original_str,
            ))
            continue

        # Date shifting: shift date tags instead of zeroing/removing
        if opts.date_shift_offset is not None and tag in DATE_TAGS and original_str:
            shifted = shift_dicom_date(original_str, opts.date_shift_offset)
            setattr(ds, keyword, shifted)
            tag_changes.append(TagChange(
                tag=tag, keyword=keyword, vr="DA", action="Z",
                original_value=original_str, anonymized_value=shifted,
            ))
            continue

        # Time tags paired with date tags: keep unchanged when date shifting
        if opts.date_shift_offset is not None and tag in TIME_TAGS:
            tag_changes.append(TagChange(
                tag=tag, keyword=keyword, vr="TM", action="K",
                original_value=original_str, anonymized_value=original_str,
            ))
            continue

        if action == "K":
            tag_changes.append(TagChange(
                tag=tag, keyword=keyword, vr="", action="K",
                original_value=original_str, anonymized_value=original_str,
            ))

        elif action == "X":
            if hasattr(ds, keyword):
                delattr(ds, keyword)
            tag_changes.append(TagChange(
                tag=tag, keyword=keyword, vr="", action="X",
                original_value=original_str, anonymized_value=None,
            ))

        elif action == "Z":
            if tag == "00100020" and original_str:
                pseudonym = hash_identifier(original_str, salt, "SUBJ-")
                setattr(ds, keyword, pseudonym)
                patient_id_mapping = {"original": original_str, "replacement": pseudonym}
                tag_changes.append(TagChange(
                    tag=tag, keyword=keyword, vr="", action="Z",
                    original_value=original_str, anonymized_value=pseudonym,
                ))
            elif tag == "00100010" and original_str:
                pseudonym = hash_identifier(original_str, salt, "ANON-")
                setattr(ds, keyword, pseudonym)
                tag_changes.append(TagChange(
                    tag=tag, keyword=keyword, vr="PN", action="Z",
                    original_value=original_str, anonymized_value=pseudonym,
                ))
            else:
                setattr(ds, keyword, "")
                tag_changes.append(TagChange(
                    tag=tag, keyword=keyword, vr="", action="Z",
                    original_value=original_str, anonymized_value="",
                ))

        elif action == "D":
            setattr(ds, keyword, "ANONYMIZED")
            tag_changes.append(TagChange(
                tag=tag, keyword=keyword, vr="", action="D",
                original_value=original_str, anonymized_value="ANONYMIZED",
            ))

        elif action == "U":
            if original_value and isinstance(original_value, str):
                new_uid = hash_uid(original_value, salt)
                setattr(ds, keyword, new_uid)
                uid_mappings[original_value] = new_uid
                tag_changes.append(TagChange(
                    tag=tag, keyword=keyword, vr="UI", action="U",
                    original_value=original_str, anonymized_value=new_uid,
                ))

        elif action == "C":
            scrub_result = scrub_free_text(original_str or "", scrub_context)
            if scrub_result.phi_found:
                setattr(ds, keyword, scrub_result.text)
                tag_changes.append(TagChange(
                    tag=tag, keyword=keyword, vr="", action="C",
                    original_value=original_str, anonymized_value=scrub_result.text,
                ))
            else:
                tag_changes.append(TagChange(
                    tag=tag, keyword=keyword, vr="", action="C",
                    original_value=original_str, anonymized_value=original_str,
                ))

    # Process sequences (SR ContentSequence, nested UIDs, etc.)
    process_sequences(ds, scrub_context, opts.date_shift_offset, salt, uid_mappings)

    # Remove private tags (odd group numbers) unless opted out
    if not opts.keep_private_tags:
        private_tags_to_remove = []
        for elem in ds:
            if elem.tag.is_private:
                private_tags_to_remove.append(elem.tag)
        for t in private_tags_to_remove:
            del ds[t]
            private_tags_removed += 1

    # Sort: modified tags first, kept tags last
    action_order = {"X": 0, "Z": 1, "D": 2, "U": 3, "C": 4, "K": 5}
    tag_changes.sort(key=lambda c: action_order.get(c.action, 99))

    return DeidResult(
        tag_changes=tag_changes,
        dataset=ds,
        private_tags_removed=private_tags_removed,
        uid_mappings=uid_mappings,
        patient_id_mapping=patient_id_mapping,
        date_shift_offset=opts.date_shift_offset,
    )


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _get_str(ds: Dataset, keyword: str) -> str | None:
    """Safely extract a string value from a dataset element."""
    val = getattr(ds, keyword, None)
    if val is None:
        return None
    return str(val)


def _format_value(value: Any) -> str | None:
    """Format a pydicom element value to a display string."""
    if value is None:
        return None
    if isinstance(value, str):
        return value
    if isinstance(value, (int, float)):
        return str(value)
    if isinstance(value, bytes):
        return "[Binary Data]"
    if isinstance(value, (list, tuple)):
        return " \\ ".join(str(v) for v in value)
    return str(value)
