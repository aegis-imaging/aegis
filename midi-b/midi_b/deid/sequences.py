"""Recursive DICOM sequence traversal for de-identification.

Handles SR ContentSequence trees and any other nested sequence data.
In pydicom, sequences are ``Sequence`` objects containing ``Dataset`` items.
"""

from __future__ import annotations

import re
from typing import TYPE_CHECKING

from pydicom.sequence import Sequence as DicomSequence

from .dateshift import shift_dicom_date
from .text_scrub import ScrubContext, scrub_free_text
from .uid_hash import hash_uid

if TYPE_CHECKING:
    from pydicom.dataset import Dataset


def process_sequences(
    ds: Dataset,
    scrub_context: ScrubContext,
    date_shift_offset: int | None,
    salt: str,
    uid_mappings: dict[str, str],
) -> None:
    """Walk all sequence elements in *ds* and de-identify their contents."""
    for elem in ds:
        if elem.keyword in ("PixelData",):
            continue
        if elem.VR != "SQ" or elem.value is None:
            continue
        seq: DicomSequence = elem.value
        for item in seq:
            _process_sequence_item(item, scrub_context, date_shift_offset, salt, uid_mappings)


def _process_sequence_item(
    item: Dataset,
    scrub_context: ScrubContext,
    date_shift_offset: int | None,
    salt: str,
    uid_mappings: dict[str, str],
) -> None:
    """De-identify a single sequence item (nested dataset)."""
    for elem in list(item):
        # Recurse into nested sequences
        if elem.VR == "SQ" and elem.value is not None:
            for nested in elem.value:
                _process_sequence_item(nested, scrub_context, date_shift_offset, salt, uid_mappings)
            continue

        value = elem.value
        if value is None:
            continue

        # Convert PersonName to string for comparison
        val_str = str(value) if value is not None else ""

        keyword = elem.keyword or ""

        # TextValue — scrub free text for PHI
        if keyword == "TextValue" and isinstance(value, str):
            result = scrub_free_text(value, scrub_context)
            if result.phi_found:
                elem.value = result.text
            continue

        # PersonName — zero out
        if elem.VR == "PN" or (keyword.endswith("Name") and "^" in val_str):
            elem.value = ""
            continue

        # UID tags — apply consistent hash mapping
        if isinstance(value, str) and (
            keyword.endswith("UID") or keyword == "UID"
        ) and re.match(r"^[012]\.\d", value):
            if value in uid_mappings:
                elem.value = uid_mappings[value]
            else:
                new_uid = hash_uid(value, salt)
                uid_mappings[value] = new_uid
                elem.value = new_uid
            continue

        # Date tags within sequences — shift if date shifting enabled
        if (
            date_shift_offset is not None
            and isinstance(value, str)
            and re.match(r"^\d{8}$", value)
            and "date" in keyword.lower()
        ):
            elem.value = shift_dicom_date(value, date_shift_offset)
            continue
