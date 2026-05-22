"""Audit log — thin re-export of dimse_core.AuditLog.

Was its own implementation; the implementation moved to dimse_core so the
cloud `dimse-receiver` can use the same primitive in a future migration.
"""
from dimse_core.audit import AuditLog as _CoreAuditLog


class AuditLog(_CoreAuditLog):
    """Router-flavoured AuditLog — same API as dimse_core.AuditLog with router
    defaults (2000 entries retained, JSONL persistence on)."""

    def __init__(self, path: str, max_entries: int = 2000) -> None:
        super().__init__(max_entries=max_entries, persist_path=path)
