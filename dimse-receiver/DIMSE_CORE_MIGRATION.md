# dimse-receiver → dimse-core migration

`dimse-core` is the shared library extracted in PR #458. This receiver pre-dates
it. The migration is happening **incrementally** because the receiver has 68
tests covering a 854-line ingest pipeline; touching that without breaking
behavior takes care.

## Status

| Receiver module | Status | Notes |
|---|---|---|
| `app/operator_audit.py` | ✅ migrated | Thin shim around `dimse_core.audit.AuditLog`. Public API (`record_action`, `get_actions`, `reset_actions`) unchanged. |
| `app/storage_backend.py` | ✅ migrated | Local-filesystem path now goes through `dimse_core.storage.StudyLayout`. S3 path stays receiver-local (no S3 backend in dimse-core yet). Public API (`write_dicom`, `next_file_index`) unchanged. |
| `app/scp.py` | not migrated | ~200 lines. Same shape as `dimse_core.scp` but wired to the receiver's local `submit_ingest` / `StudyAccumulator`. Migration is a structural refactor — possible but not urgent. |
| `app/ingest.py` | deferred | **854 lines, deeply test-coupled.** See [Why ingest.py is deferred](#why-ingestpy-is-deferred) below. |
| `app/retry_alerts.py` | not migrated | 152 lines. Maps to `dimse_core.alerts.AlertEngine` except the receiver's alert messages are built with dynamic strings that depend on both snapshot fields and config-derived thresholds — `AlertEngine.message_fmt` is .format()-on-snapshot only. Needs a small extension on `AlertEngine` or per-condition message_fn. |
| `app/sender.py` | stays receiver-only | 341 lines of outbound C-STORE / C-FIND / C-MOVE. Cloud-specific — doesn't belong in dimse-core. |

## Why migrate at all

Today the cloud receiver and the spoke `router/` both have their own SCP, retry
queue, audit log, and alert engine. They're functionally identical but
independently maintained — any improvement (bug fix, new operator action,
better backoff) has to land in two places.

Once each module is on `dimse-core`, the lock-step is automatic.

## Roadmap

### Done so far

- `app/operator_audit.py` → shim around `dimse_core.audit.AuditLog`
- `app/storage_backend.py` → local path now goes through
  `dimse_core.storage.StudyLayout` (S3 path unchanged). Gains atomic
  `.tmp`-swap writes from the library for free.
- `dimse-receiver/Dockerfile` installs `dimse-core` via editable pip install
- Tests in `tests/test_operator_audit.py`, `tests/test_main.py`, and
  `tests/test_scp.py` (including the S3-mode test) still pass unchanged
  because the shims preserve the public API exactly.

### Subsequent PR (medium risk)

- `app/retry_alerts.py` → either
  - (a) extend `dimse_core.alerts.AlertEngine` with a per-condition
    `message_fn(snapshot, now) -> str` callback so dynamic threshold-in-message
    formatting is supported, then shim, or
  - (b) keep `retry_alerts.py` as-is but document that the dimse-core path is
    available for new alert engines.

### Largest PR (highest risk) — `app/scp.py`

The dimse-core SCP supports a swappable `write_dicom_fn` and per-study
completion callback already, so the SCP itself is a straight swap. This is
worth picking up once `ingest.py` is settled (it touches `submit_ingest` /
`StudyAccumulator` which are still receiver-owned).

Until those land, the receiver's mature implementation stays in production.

## Why ingest.py is deferred

A close read against `dimse_core.retry_queue.RetryQueue` turned up enough
structural mismatch that an honest "migrate ingest.py" PR is bigger and
riskier than the value it delivers right now:

- **Tests poke internals.** `tests/test_ingest.py` reads and writes
  `ingest_module._retry_queue[i].next_attempt_at`,
  `ingest_module._retry_queue[i].acc.file_count`, and
  `ingest_module._retry_queue[i].acc.series_uids` directly. dimse-core's
  `_pending` is a `list[RetryItem]` with an opaque `payload: dict`, not a
  `list[QueuedIngest]` carrying a real `StudyAccumulator`. Bridging this
  needs a wrapper class with property setters that mutate the underlying
  dict — possible but it's adapter glue, not shared code.
- **JSON state shape differs.** The receiver's durable file uses the key
  `retry_queue`; dimse-core uses `pending`. A migration needs either a
  one-shot file rename at startup (we'd want to keep reading both for one
  release) or to teach dimse-core the receiver's shape — and the per-item
  shape differs too (`acc: {study_instance_uid, modality, body_part,
  series_uids: [sorted]}` vs `payload: {...}`).
- **Metrics surface is wider.** `retry_snapshot()` exposes 11 fields the
  receiver's `/healthz`, `/ingest/retry`, and admin proxy rely on
  (`deduped_total`, `dead_letter_deduped_total`, `replayed_total`,
  `cleared_pending_total`, `cleared_dead_letter_total`, age fields, next-due
  fields, etc.). Some are not in `RetryQueue.snapshot()` today.
- **`_merge_accumulator` semantics are receiver-specific.** Max file_count,
  union of `series_uids: set`, fill-empty-fields for modality/body_part —
  maps to dimse-core's `merge_payload_fn` callback but the receiver mutates
  a `StudyAccumulator` dataclass in place, not a dict.
- **Production code is stable.** 68 tests, runs in cloud + would run
  identically in the router. The dimse-core copy is currently consumed only
  by the router stub.

**Recommendation:** keep `ingest.py` receiver-local for now and revisit
once one of these holds:

1. The router actually depends on `RetryQueue` in production traffic and we
   need lock-step bug fixes.
2. dimse-core grows a "StudyAccumulator payload + receiver-flavored
   snapshot" mode (or the receiver agrees to switch its public JSON shape).
3. We're rewriting `tests/test_ingest.py` for unrelated reasons and can
   absorb the internals→public-API churn in the same PR.

Sharing the **backoff math** alone (a 10-line helper) is too small to be
worth a PR on its own; if/when `scp.py` migrates, we can fold that in.
