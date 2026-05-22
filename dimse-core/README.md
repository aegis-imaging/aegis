# dimse-core

Shared Python primitives used by both `dimse-receiver` (cloud-side) and
`router` (spoke-side) so DIMSE listening, retry queues, audit logging, and
alerting don't get reimplemented twice.

## What's in here

| Module | Purpose |
|---|---|
| `dimse_core.scp` | DICOM C-STORE SCP factory with a per-study completion callback. The caller decides what to do with each completed study — ingest into a cloud API, ship over mTLS, run a local pipeline, whatever. |
| `dimse_core.retry_queue` | Generic retry + dead-letter queue with exponential backoff, dedup by key, and optional on-disk persistence. The caller supplies a `process_fn(item) -> RetryResult`. |
| `dimse_core.audit` | Bounded ring-buffer audit log with optional JSONL persistence. Filterable by event type on read. |
| `dimse_core.alerts` | Threshold-driven alert engine — caller declares `(name, predicate, message_fmt)` triples; engine emits events with cooldown and optional webhook delivery. |
| `dimse_core.storage` | Common on-disk study layout helper (`data_dir/dicom/{raw,clean}/<study_uid>/`). |

## Design

- **Library, not service.** No module-level config; every primitive takes config
  in via a dataclass at construction. This is what makes the same code usable
  by two services with different env namespaces.
- **No required side-effects.** Storage, ingest, shipping, alerting are all
  caller-supplied callbacks. The library never imports an HTTP framework or
  a config module.
- **Optional `pynetdicom`.** Only the SCP needs it, and even there the import
  is lazy so unit tests that mock the event objects work without it.

## Using it

```python
from dimse_core import (
    SCPConfig, create_ae, start_listening,
    RetryConfig, RetryQueue, RetryResult,
    AuditLog,
    AlertConfig, AlertEngine,
    StudyLayout,
)

# 1) Spin up an SCP
def on_study(acc):
    queue.enqueue(acc.study_instance_uid, {"acc": acc})

ae = create_ae(SCPConfig(ae_title="AEGIS", port=11112, data_dir="/var/aegis"),
               on_study)
threading.Thread(target=start_listening, args=(ae, cfg), daemon=True).start()

# 2) Drive retries
def process(item):
    ok = ship_to_cloud(item.payload["acc"])
    return RetryResult(ok=ok, error="" if ok else "cloud unreachable")

queue = RetryQueue(RetryConfig(durable_path="/var/aegis/retries.json"),
                   process_fn=process)
# Periodic worker:
while True:
    queue.process_due()
    time.sleep(5)
```

## Tests

```bash
cd dimse-core
pip install -r requirements.txt -r requirements-test.txt
pytest -v
```

## Roadmap

- The cloud `dimse-receiver` currently has its own retry queue (`app/ingest.py`)
  with the same semantics — migrating it to `dimse_core.retry_queue` is a
  follow-up PR. Behavior should match exactly; the migration is mainly about
  deleting code.
- An S3-backed `StudyLayout` variant for the cloud receiver's S3 mode (the
  receiver has its own `storage_backend.py` doing this today).
