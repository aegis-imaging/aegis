"""Tests for the de-identification engine — ported from client/src/dicom/__tests__/deid.test.ts."""

import re
from tests.conftest import make_dataset
from midi_b.deid.engine import DeidOptions, deidentify


def test_pseudonymizes_patient_name_and_id():
    ds = make_dataset()
    result = deidentify(ds)
    assert re.match(r"^ANON-[0-9a-f]{8}$", str(ds.PatientName))
    assert re.match(r"^SUBJ-[0-9a-f]{8}$", str(ds.PatientID))
    name_change = next(c for c in result.tag_changes if c.keyword == "PatientName")
    assert name_change.action == "Z"
    assert name_change.original_value == "DOE^JOHN"
    assert name_change.anonymized_value.startswith("ANON-")
    assert result.patient_id_mapping is not None
    assert result.patient_id_mapping["original"] == "MRN-12345"
    assert result.patient_id_mapping["replacement"].startswith("SUBJ-")


def test_removes_x_action_tags():
    ds = make_dataset()
    deidentify(ds)
    assert str(ds.ReferringPhysicianName) == ""  # Z action, not X
    assert not hasattr(ds, "InstitutionName")  # X action
    assert not hasattr(ds, "SeriesDate")  # X action
    assert not hasattr(ds, "AcquisitionDate")  # X action


def test_zeroes_study_date_and_time():
    ds = make_dataset()
    deidentify(ds)
    assert ds.StudyDate == ""
    assert ds.StudyTime == ""


def test_keeps_safe_tags():
    ds = make_dataset()
    deidentify(ds)
    assert ds.Modality == "CT"
    assert ds.Manufacturer == "SIEMENS"
    assert ds.PatientSex == "M"
    assert ds.BodyPartExamined == "CHEST"
    assert ds.Rows == 512


def test_replaces_uids_deterministically():
    ds = make_dataset()
    original_uid = ds.StudyInstanceUID
    deidentify(ds, DeidOptions(salt="test-salt"))
    assert ds.StudyInstanceUID != original_uid
    assert str(ds.StudyInstanceUID).startswith("2.25.")

    # Same input → same output
    ds2 = make_dataset()
    deidentify(ds2, DeidOptions(salt="test-salt"))
    assert ds2.StudyInstanceUID == ds.StudyInstanceUID


def test_different_salts_produce_different_uids():
    ds1 = make_dataset()
    ds2 = make_dataset()
    deidentify(ds1, DeidOptions(salt="salt-a"))
    deidentify(ds2, DeidOptions(salt="salt-b"))
    assert ds1.StudyInstanceUID != ds2.StudyInstanceUID


def test_keeps_sop_class_uid():
    ds = make_dataset()
    original = ds.SOPClassUID
    deidentify(ds)
    assert ds.SOPClassUID == original


def test_cleans_c_action_tags():
    # No PHI in description → keep
    ds1 = make_dataset(StudyDescription="CT CHEST WITH CONTRAST")
    deidentify(ds1)
    assert ds1.StudyDescription == "CT CHEST WITH CONTRAST"

    # PHI in description → scrub
    ds2 = make_dataset(StudyDescription="CT for John Smith MRN: 12345")
    deidentify(ds2)
    scrubbed = str(ds2.StudyDescription)
    assert "CT for" in scrubbed
    assert "[REMOVED]" in scrubbed
    assert "John Smith" not in scrubbed
    assert "12345" not in scrubbed


def test_respects_retained_tags():
    ds = make_dataset()
    deidentify(ds, DeidOptions(retained_tags=["StudyDate", "PatientBirthDate"]))
    assert ds.StudyDate == "20240301"
    assert ds.PatientBirthDate == "19800115"


def test_removes_private_tags():
    ds = make_dataset()
    from pydicom.tag import Tag
    tag = Tag(0x00091001)
    ds.add_new(tag, "LO", "private data")
    result = deidentify(ds)
    assert tag not in ds
    assert result.private_tags_removed > 0


def test_keeps_private_tags_when_opted_in():
    ds = make_dataset()
    ds.add_new(0x00091001, "LO", "private data")
    deidentify(ds, DeidOptions(keep_private_tags=True))
    assert ds[0x00091001].value == "private data"


def test_tag_changes_sorted_by_action_priority():
    ds = make_dataset()
    result = deidentify(ds)
    actions = [c.action for c in result.tag_changes]
    order = {"X": 0, "Z": 1, "D": 2, "U": 3, "C": 4, "K": 5}
    for i in range(1, len(actions)):
        assert order[actions[i]] >= order[actions[i - 1]]


def test_date_shifting():
    ds = make_dataset()
    deidentify(ds, DeidOptions(date_shift_offset=10))
    # StudyDate should be shifted forward by 10 days
    assert ds.StudyDate == "20240311"
    # StudyTime should be kept unchanged
    assert ds.StudyTime == "143000"
