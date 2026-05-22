"""AEGIS dimse-core — primitives shared between the cloud `dimse-receiver`
and the spoke `router`.

Modules:
  scp           DICOM C-STORE SCP factory with a per-study completion callback.
  retry_queue   Durable retry + dead-letter queue with exponential backoff.
  audit         Bounded operator/event audit log with optional JSONL persistence.
  alerts        Threshold-driven alert engine + optional webhook delivery.
  storage       Local-filesystem study storage layout (+ S3 backend).

Design: every primitive is config-free at the package level — callers wire in
their own config, callbacks, and side-effects. This is the property that
distinguishes a library from a service.
"""

from dimse_core.audit import AuditLog  # noqa: F401
from dimse_core.retry_queue import (  # noqa: F401
    RetryConfig,
    RetryItem,
    RetryQueue,
    RetryResult,
)
from dimse_core.scp import (  # noqa: F401
    SCPConfig,
    StudyAccumulator,
    create_ae,
    start_listening,
)
from dimse_core.alerts import (  # noqa: F401
    AlertConfig,
    AlertEngine,
    AlertEvent,
)
from dimse_core.storage import StudyLayout  # noqa: F401

__version__ = "0.1.0"
