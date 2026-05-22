"""Thin wrapper around dimse_core.scp that uses the router's on-disk layout.

The actual SCP machinery (pynetdicom, per-association state, transfer
syntaxes, study aggregation) lives in dimse_core. Here we just adapt it to
the router-specific `studies/<study_uid>/{raw,deid}/` layout and re-export
StudyAccumulator under the original name so callers don't need to change.
"""
from __future__ import annotations

import logging
from typing import Callable

from dimse_core import scp as core_scp
from dimse_core.scp import StudyAccumulator  # re-export

from app import storage
from app.config import RouterConfig

log = logging.getLogger(__name__)

__all__ = ["StudyAccumulator", "create_ae", "start_listening"]


def _router_write_dicom(_cfg: core_scp.SCPConfig, study_uid: str, file_index: int, data: bytes) -> str:
    """Persist instances under the router's `studies/<study_uid>/raw/` layout.

    Returns the relative storage key for SCP bookkeeping.
    """
    paths = storage.layout(_cfg.data_dir, study_uid)
    storage.write_raw_bytes(paths, file_index, data)
    return f"studies/{study_uid}/raw/{file_index}.dcm"


def _router_next_index(_cfg: core_scp.SCPConfig, study_uid: str) -> int:
    paths = storage.layout(_cfg.data_dir, study_uid)
    return storage.next_raw_index(paths)


def _to_core_cfg(cfg: RouterConfig) -> core_scp.SCPConfig:
    return core_scp.SCPConfig(
        ae_title=cfg.dimse_ae_title,
        port=cfg.dimse_port,
        max_associations=cfg.dimse_max_associations,
        data_dir=cfg.data_dir,
        write_dicom_fn=_router_write_dicom,
    )


def create_ae(cfg: RouterConfig, on_study_complete: Callable[[StudyAccumulator], None]):
    """Build the pynetdicom AE configured for the router."""
    return core_scp.create_ae(
        _to_core_cfg(cfg),
        on_study_complete,
        next_index_fn=_router_next_index,
    )


def start_listening(ae, cfg: RouterConfig | None = None) -> None:
    """Blocking listen loop. Run in a daemon thread.

    For backwards compatibility the existing caller invokes this with only the
    AE; we recover the config from env when not provided.
    """
    if cfg is None:
        cfg = RouterConfig.load()
    core_scp.start_listening(ae, _to_core_cfg(cfg))
