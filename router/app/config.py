"""Router configuration loaded from env vars.

Single source of truth — every other module imports from here. Designed so a
spoke operator only has to set a small number of vars to get a working install;
the defaults are conservative.
"""

from __future__ import annotations

import os
from dataclasses import dataclass


def _env(name: str, default: str = "") -> str:
    return os.environ.get(name, default).strip()


def _env_int(name: str, default: int) -> int:
    raw = _env(name, "")
    if not raw:
        return default
    try:
        return int(raw)
    except ValueError:
        return default


def _env_bool(name: str, default: bool) -> bool:
    raw = _env(name, "").lower()
    if raw in ("1", "true", "yes", "on"):
        return True
    if raw in ("0", "false", "no", "off"):
        return False
    return default


@dataclass(frozen=True)
class RouterConfig:
    # Identity
    site_name: str
    site_id: str

    # DICOM SCP (incoming from PACS)
    dimse_ae_title: str
    dimse_port: int
    dimse_max_associations: int

    # HTTP server (incoming web uploads + operator API)
    http_host: str
    http_port: int

    # Local storage
    data_dir: str
    quarantine_dir: str

    # Pipeline sidecars (each empty == disabled)
    defacing_url: str
    phi_detection_url: str
    qc_url: str
    classification_url: str
    protocol_url: str
    bids_url: str
    analytics_url: str
    sct_url: str
    synth_url: str

    # MIDI-B tag de-id options
    midi_b_salt: str
    midi_b_date_shift: bool
    midi_b_date_shift_max_days: int
    midi_b_keep_private_tags: bool
    midi_b_retained_tags: tuple[str, ...]

    # Cloud forwarder (mTLS)
    cloud_url: str
    cloud_project_slug: str
    cloud_institution_slug: str
    cloud_client_cert: str  # path
    cloud_client_key: str  # path
    cloud_ca_cert: str  # path; empty == use system trust store

    # Retry / quarantine policy
    ship_retry_initial_seconds: int
    ship_retry_max_seconds: int
    ship_retry_max_attempts: int
    quarantine_alert_url: str  # optional webhook for cloud control plane

    # Operator API auth
    operator_api_key: str  # if empty, operator endpoints are unauthenticated (dev only)

    @classmethod
    def load(cls) -> "RouterConfig":
        retained_raw = _env("MIDI_B_RETAINED_TAGS", "")
        retained = tuple(t.strip() for t in retained_raw.split(",") if t.strip())
        return cls(
            site_name=_env("ROUTER_SITE_NAME", "unset"),
            site_id=_env("ROUTER_SITE_ID", "unset"),
            dimse_ae_title=_env("DIMSE_AE_TITLE", "AEGIS_ROUTER"),
            dimse_port=_env_int("DIMSE_PORT", 11112),
            dimse_max_associations=_env_int("DIMSE_MAX_ASSOCIATIONS", 10),
            http_host=_env("HTTP_HOST", "0.0.0.0"),
            http_port=_env_int("HTTP_PORT", 8080),
            data_dir=_env("ROUTER_DATA_DIR", "/var/lib/aegis-router"),
            quarantine_dir=_env("ROUTER_QUARANTINE_DIR", "/var/lib/aegis-router/quarantine"),
            defacing_url=_env("DEFACING_SERVICE_URL", ""),
            phi_detection_url=_env("PHI_DETECTION_SERVICE_URL", ""),
            qc_url=_env("QC_SERVICE_URL", ""),
            classification_url=_env("CLASSIFICATION_SERVICE_URL", ""),
            protocol_url=_env("PROTOCOL_SERVICE_URL", ""),
            bids_url=_env("BIDS_SERVICE_URL", ""),
            analytics_url=_env("ANALYTICS_SERVICE_URL", ""),
            sct_url=_env("SCT_SERVICE_URL", ""),
            synth_url=_env("SYNTH_SERVICE_URL", ""),
            midi_b_salt=_env("MIDI_B_SALT", "aegis-router-default-CHANGE-ME"),
            midi_b_date_shift=_env_bool("MIDI_B_DATE_SHIFT", True),
            midi_b_date_shift_max_days=_env_int("MIDI_B_DATE_SHIFT_MAX_DAYS", 365),
            midi_b_keep_private_tags=_env_bool("MIDI_B_KEEP_PRIVATE_TAGS", False),
            midi_b_retained_tags=retained,
            cloud_url=_env("CLOUD_RECEIVER_URL", ""),
            cloud_project_slug=_env("CLOUD_PROJECT_SLUG", "default"),
            cloud_institution_slug=_env("CLOUD_INSTITUTION_SLUG", ""),
            cloud_client_cert=_env("CLOUD_CLIENT_CERT", ""),
            cloud_client_key=_env("CLOUD_CLIENT_KEY", ""),
            cloud_ca_cert=_env("CLOUD_CA_CERT", ""),
            ship_retry_initial_seconds=_env_int("SHIP_RETRY_INITIAL_SECONDS", 15),
            ship_retry_max_seconds=_env_int("SHIP_RETRY_MAX_SECONDS", 600),
            ship_retry_max_attempts=_env_int("SHIP_RETRY_MAX_ATTEMPTS", 8),
            quarantine_alert_url=_env("QUARANTINE_ALERT_URL", ""),
            operator_api_key=_env("OPERATOR_API_KEY", ""),
        )

    def enabled_sidecars(self) -> dict[str, str]:
        """Return only the sidecars that have a URL set (== enabled)."""
        candidates = {
            "defacing": self.defacing_url,
            "phi_detection": self.phi_detection_url,
            "qc": self.qc_url,
            "classification": self.classification_url,
            "protocol": self.protocol_url,
            "bids": self.bids_url,
            "analytics": self.analytics_url,
            "sct": self.sct_url,
            "synth": self.synth_url,
        }
        return {name: url for name, url in candidates.items() if url}

    def cloud_forwarding_configured(self) -> bool:
        return bool(self.cloud_url and self.cloud_client_cert and self.cloud_client_key)
