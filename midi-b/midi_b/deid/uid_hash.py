"""Deterministic UID hashing and identifier pseudonymization."""

from __future__ import annotations

import hashlib


def hash_uid(original_uid: str, salt: str) -> str:
    """Generate a deterministic UID from an original UID using a project-scoped salt.

    Uses SHA-256, first 16 bytes as big-endian decimal, formatted as ``2.25.<decimal>``.
    Result truncated to 64 characters (DICOM UID max length).
    """
    data = (salt + original_uid).encode("utf-8")
    digest = hashlib.sha256(data).digest()

    # First 16 bytes → big-endian unsigned integer
    decimal = int.from_bytes(digest[:16], byteorder="big")
    uid = f"2.25.{decimal}"

    return uid[:64]


def hash_identifier(original: str, salt: str, prefix: str) -> str:
    """Generate a deterministic pseudonym from an identifier using SHA-256.

    Takes first 4 bytes → 8 hex characters, prepends *prefix*.
    """
    data = (salt + original).encode("utf-8")
    digest = hashlib.sha256(data).digest()
    hex8 = digest[:4].hex()
    return f"{prefix}{hex8}"
