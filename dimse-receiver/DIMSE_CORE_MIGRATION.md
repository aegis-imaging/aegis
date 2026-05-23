# dimse-receiver → dimse-core migration

`dimse-core` is the shared library extracted in PR #458. This receiver pre-dates
it. The migration is happening **incrementally** because the receiver has 68
tests covering a 854-line ingest pipeline; touching that without breaking
behavior takes care.

## Status

| Receiver module | Status | Notes |
|---|---|---|
| `app/operator_audit.py` | ✅ migrated | Thin shim around `dimse_core.audit.AuditLog`. Public API (`record_action`, `get_actions`, `reset_actions`) unchanged. |
| `app/storage_backend.py` | not migrated | 57 lines, generic enough that a future shim around `dimse_core.storage.StudyLayout` is cheap. |
| `app/scp.py` | not migrated | ~200 lines. Same shape as `dimse_core.scp` but wired to the receiver's local `submit_ingest` / `StudyAccumulator`. Migration is a structural refactor — possible but not urgent. |
| `app/ingest.py` | not migrated | **854 lines, complex.** The retry queue + dead-letter + dedup + durable JSONL persistence is functionally identical to `dimse_core.retry_queue.RetryQueue`, but the receiver's tests pin specific JSON-state shapes that the dimse-core impl doesn't currently match byte-for-byte. Migration is a multi-step effort. |
| `app/retry_alerts.py` | not migrated | 152 lines. Maps to `dimse_core.alerts.AlertEngine` except the receiver's alert messages are built with dynamic strings that depend on both snapshot fields and config-derived thresholds — `AlertEngine.message_fmt` is .format()-on-snapshot only. Needs a small extension on `AlertEngine` or per-condition message_fn. |
| `app/sender.py` | stays receiver-only | 341 lines of outbound C-STORE / C-FIND / C-MOVE. Cloud-specific — doesn't belong in dimse-core. |

## Why migrate at all

Today the cloud receiver and the spoke `router/` both have their own SCP, retry
queue, audit log, and alert engine. They're functionally identical but
independently maintained — any improvement (bug fix, new operator action,
better backoff) has to land in two places.

Once each module is on `dimse-core`, the lock-step is automatic.

## Roadmap

### Done in this PR

- `app/operator_audit.py` → shim around `dimse_core.audit.AuditLog`
- `dimse-receiver/Dockerfile` installs `dimse-core` via editable pip install
- Tests in `tests/test_operator_audit.py` and `tests/test_main.py` still pass
  unchanged (the shim preserves the public API exactly)

### Next PR (low risk)

- `app/storage_backend.py` → shim that uses `dimse_core.storage.StudyLayout`
  for the local-filesystem path, keeping the receiver's S3 path as-is until
  dimse-core grows an S3 backend.

### Subsequent PR (medium risk)

- `app/retry_alerts.py` → either
  - (a) extend `dimse_core.alerts.AlertEngine` with a per-condition
    `message_fn(snapshot, now) -> str` callback so dynamic threshold-in-message
    formatting is supported, then shim, or
  - (b) keep `retry_alerts.py` as-is but document that the dimse-core path is
    available for new alert engines.

### Largest PR (highest risk)

- `app/scp.py` and `app/ingest.py` migration. The dimse-core SCP supports a
  swappable `write_dicom_fn` and per-study completion callback already, so
  the SCP itself is a straight swap. The retry queue requires either
  matching the JSON-state shape byte-for-byte (and a one-time migration of
  existing on-disk state) or extending dimse-core's persistence with the
  receiver's exact schema.

Until those land, the receiver's mature implementation stays in production.
