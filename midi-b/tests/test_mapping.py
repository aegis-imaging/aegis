"""Tests for mapping CSV export."""

from midi_b.mapping import MappingCollector


def test_uid_csv_format():
    c = MappingCollector()
    c.add_uid_mappings({"1.2.3.4": "2.25.999", "1.2.3.5": "2.25.888"})
    csv = c.to_uid_csv()
    lines = csv.strip().splitlines()
    assert lines[0].strip() == "original_uid,replacement_uid"
    assert len(lines) == 3  # header + 2 rows


def test_patient_id_csv_format():
    c = MappingCollector()
    c.add_patient_id_mapping("MRN-12345", "SUBJ-aabbccdd", 42)
    csv = c.to_patient_id_csv()
    lines = csv.strip().splitlines()
    assert lines[0].strip() == "original_patient_id,replacement_patient_id,date_shift_offset"
    assert "MRN-12345" in lines[1]
    assert "SUBJ-aabbccdd" in lines[1]
    assert "42" in lines[1]


def test_uid_deduplication():
    c = MappingCollector()
    c.add_uid_mappings({"1.2.3.4": "2.25.999"})
    c.add_uid_mappings({"1.2.3.4": "2.25.999"})  # duplicate
    csv = c.to_uid_csv()
    lines = csv.strip().split("\n")
    assert len(lines) == 2  # header + 1 unique row


def test_patient_id_no_offset():
    c = MappingCollector()
    c.add_patient_id_mapping("MRN-12345", "SUBJ-aabbccdd", None)
    csv = c.to_patient_id_csv()
    assert ",," not in csv or csv.endswith(",\r\n") or csv.endswith(",\n")


def test_empty_collector():
    c = MappingCollector()
    uid_csv = c.to_uid_csv()
    assert "original_uid" in uid_csv
    lines = uid_csv.strip().split("\n")
    assert len(lines) == 1  # header only

    pid_csv = c.to_patient_id_csv()
    lines = pid_csv.strip().split("\n")
    assert len(lines) == 1  # header only
