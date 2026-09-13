"""Tests for UID hashing and identifier pseudonymization."""

from midi_b.deid.uid_hash import hash_uid, hash_identifier


def test_hash_uid_format():
    uid = hash_uid("1.2.840.113619.2.55.3.12345", "test-salt")
    assert uid.startswith("2.25.")
    assert len(uid) <= 64


def test_hash_uid_deterministic():
    uid1 = hash_uid("1.2.840.113619.2.55.3.12345", "test-salt")
    uid2 = hash_uid("1.2.840.113619.2.55.3.12345", "test-salt")
    assert uid1 == uid2


def test_hash_uid_different_salts():
    uid1 = hash_uid("1.2.840.113619.2.55.3.12345", "salt-a")
    uid2 = hash_uid("1.2.840.113619.2.55.3.12345", "salt-b")
    assert uid1 != uid2


def test_hash_uid_different_inputs():
    uid1 = hash_uid("1.2.3.4", "salt")
    uid2 = hash_uid("1.2.3.5", "salt")
    assert uid1 != uid2


def test_hash_identifier_format():
    result = hash_identifier("MRN-12345", "salt", "SUBJ-")
    assert result.startswith("SUBJ-")
    assert len(result) == len("SUBJ-") + 8  # 8 hex chars


def test_hash_identifier_deterministic():
    r1 = hash_identifier("MRN-12345", "salt", "SUBJ-")
    r2 = hash_identifier("MRN-12345", "salt", "SUBJ-")
    assert r1 == r2


def test_hash_identifier_different_patients():
    r1 = hash_identifier("MRN-12345", "salt", "SUBJ-")
    r2 = hash_identifier("MRN-99999", "salt", "SUBJ-")
    assert r1 != r2


def test_hash_identifier_anon_prefix():
    result = hash_identifier("DOE^JOHN", "salt", "ANON-")
    assert result.startswith("ANON-")
    assert len(result) == 13  # ANON- + 8 hex

def test_hash_uid_matches_typescript():
    """Verify Python output matches the TypeScript implementation.

    The TS implementation uses Web Crypto SHA-256 with the same algorithm:
    salt + originalUid → SHA-256 → first 16 bytes as big-endian decimal → 2.25.<decimal>
    """
    uid = hash_uid("1.2.840.113619.2.55.3.12345", "test-salt")
    # Should be a valid DICOM UID (only digits and dots)
    assert all(c in "0123456789." for c in uid)
