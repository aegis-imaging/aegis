"""Tests for recursive sequence de-identification."""

from tests.conftest import make_sr_dataset
from midi_b.deid.engine import DeidOptions, deidentify


def test_scrubs_phi_from_text_value():
    ds = make_sr_dataset()
    deidentify(ds, DeidOptions(salt="test-salt"))
    items = list(ds.ContentSequence)
    text = str(items[0].TextValue)
    assert "[REMOVED]" in text
    assert "John Doe" not in text


def test_preserves_non_phi_text_value():
    ds = make_sr_dataset()
    deidentify(ds, DeidOptions(salt="test-salt"))
    items = list(ds.ContentSequence)
    assert items[1].TextValue == "Normal brain MRI findings"


def test_hashes_uids_consistently():
    ds = make_sr_dataset()
    result = deidentify(ds, DeidOptions(salt="test-salt"))
    items = list(ds.ContentSequence)
    ref_uid = str(items[0].ReferencedSOPInstanceUID)
    assert ref_uid.startswith("2.25.")
    assert ref_uid != "1.2.840.113619.2.55.3.99999"
    uid = str(items[1].UID)
    assert uid.startswith("2.25.")
    assert "1.2.840.113619.2.55.3.99999" in result.uid_mappings
    assert "1.2.840.113619.2.55.3.88888" in result.uid_mappings


def test_zeros_person_name_in_nested_sequences():
    ds = make_sr_dataset()
    deidentify(ds, DeidOptions(salt="test-salt"))
    items = list(ds.ContentSequence)
    nested = list(items[0].ContentSequence)
    assert str(nested[0].PersonName) == ""


def test_recurses_into_deeply_nested_and_scrubs_phi():
    ds = make_sr_dataset()
    deidentify(ds, DeidOptions(salt="test-salt"))
    items = list(ds.ContentSequence)
    nested = list(items[0].ContentSequence)
    text = str(nested[0].TextValue)
    assert "[REMOVED]" in text
    assert "Smith" not in text


def test_non_sr_datasets_unaffected():
    from tests.conftest import make_dataset
    ds = make_dataset()
    result = deidentify(ds, DeidOptions(salt="test-salt"))
    assert ds.Modality == "CT"
    assert str(ds.StudyInstanceUID).startswith("2.25.")
    assert len(result.uid_mappings) > 0
