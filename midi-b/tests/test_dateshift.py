"""Tests for date shifting utilities."""

from midi_b.deid.dateshift import (
    DATE_TAGS,
    TIME_TAGS,
    generate_date_offset,
    shift_dicom_date,
    shift_dicom_datetime,
)


def test_shift_forward():
    assert shift_dicom_date("20240301", 10) == "20240311"


def test_shift_backward():
    assert shift_dicom_date("20240310", -10) == "20240229"  # 2024 is leap year


def test_shift_across_year():
    assert shift_dicom_date("20241231", 1) == "20250101"


def test_shift_negative_across_year():
    assert shift_dicom_date("20250101", -1) == "20241231"


def test_shift_zero():
    assert shift_dicom_date("20240301", 0) == "20240301"


def test_shift_invalid_returns_unchanged():
    assert shift_dicom_date("not-a-date", 10) == "not-a-date"
    assert shift_dicom_date("12345", 10) == "12345"


def test_shift_datetime():
    result = shift_dicom_datetime("20240301143000.000+0500", 10)
    assert result == "20240311143000.000+0500"


def test_shift_datetime_short():
    assert shift_dicom_datetime("2024", 10) == "2024"


def test_generate_offset_deterministic():
    o1 = generate_date_offset("MRN-12345", "salt")
    o2 = generate_date_offset("MRN-12345", "salt")
    assert o1 == o2


def test_generate_offset_within_range():
    offset = generate_date_offset("MRN-12345", "salt", max_days=365)
    assert -365 <= offset <= 365


def test_generate_offset_different_patients():
    o1 = generate_date_offset("MRN-12345", "salt")
    o2 = generate_date_offset("MRN-99999", "salt")
    assert o1 != o2


def test_generate_offset_different_salts():
    o1 = generate_date_offset("MRN-12345", "salt-a")
    o2 = generate_date_offset("MRN-12345", "salt-b")
    assert o1 != o2


def test_date_tags_set():
    assert "00080020" in DATE_TAGS  # StudyDate
    assert "00100030" in DATE_TAGS  # PatientBirthDate


def test_time_tags_set():
    assert "00080030" in TIME_TAGS  # StudyTime
    assert "00080033" in TIME_TAGS  # ContentTime


def test_shift_leap_year():
    # Feb 28 2024 + 1 = Feb 29 (leap year)
    assert shift_dicom_date("20240228", 1) == "20240229"
    # Feb 28 2023 + 1 = Mar 01 (non-leap year)
    assert shift_dicom_date("20230228", 1) == "20230301"
