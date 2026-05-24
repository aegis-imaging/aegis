"""Bridge configuration loader.

Reads non-secret config from ``bridge.yml`` at a platform-standard path and
pulls the API key from the OS keyring. Secrets are deliberately NOT persisted
to the yaml file.
"""

from __future__ import annotations

import logging
import os
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

import yaml

try:
    import keyring  # type: ignore
except ImportError:  # pragma: no cover - keyring is a hard runtime dep
    keyring = None  # type: ignore

log = logging.getLogger(__name__)

KEYRING_SERVICE = "aegis-dimse-bridge"
KEYRING_ACCOUNT = "api-key"


def platform_config_path() -> Path:
    """Return the platform-standard path for ``bridge.yml``.

    Linux:   /etc/aegis/bridge.yml
    macOS:   /Library/Application Support/AEGIS/bridge.yml
    Windows: %ProgramData%\\AEGIS\\bridge.yml
    """
    if sys.platform == "darwin":
        return Path("/Library/Application Support/AEGIS/bridge.yml")
    if sys.platform == "win32":
        return Path(os.path.expandvars(r"%ProgramData%\AEGIS\bridge.yml"))
    return Path("/etc/aegis/bridge.yml")


def platform_log_dir() -> Path:
    """Return the platform-standard log directory."""
    if sys.platform == "darwin":
        return Path("/Library/Logs/AEGIS")
    if sys.platform == "win32":
        return Path(os.path.expandvars(r"%ProgramData%\AEGIS\logs"))
    return Path("/var/log/aegis")


@dataclass
class BridgeConfig:
    """Resolved configuration for the bridge service."""

    server_url: str = ""
    listen_port: int = 11112
    aet: str = "AEGISBRIDGE"
    log_dir: Path = field(default_factory=platform_log_dir)
    api_key: Optional[str] = None
    config_path: Optional[Path] = None


def _read_api_key() -> Optional[str]:
    if keyring is None:
        log.warning("keyring library not available; api_key will be unset")
        return None
    try:
        return keyring.get_password(KEYRING_SERVICE, KEYRING_ACCOUNT)
    except Exception as exc:  # pragma: no cover - depends on host keyring
        log.warning("failed to read api_key from keyring: %s", exc)
        return None


def load(config_path: Optional[Path] = None) -> BridgeConfig:
    """Load bridge configuration from yaml + keyring.

    A missing yaml file is tolerated; the bridge will run with defaults but
    will refuse to upload anything until ``server_url`` and ``api_key`` are
    set (i.e. until ``aegis-bridge pair`` has been run).
    """
    path = config_path or platform_config_path()
    data: dict = {}
    if path.exists():
        try:
            with path.open("r", encoding="utf-8") as fh:
                loaded = yaml.safe_load(fh) or {}
            if isinstance(loaded, dict):
                data = loaded
            else:
                log.warning("bridge.yml at %s is not a mapping; ignoring", path)
        except (OSError, yaml.YAMLError) as exc:
            log.warning("failed to read %s: %s", path, exc)
    else:
        log.info("no bridge.yml at %s; using defaults", path)

    log_dir = data.get("log_dir")
    cfg = BridgeConfig(
        server_url=str(data.get("server_url", "") or ""),
        listen_port=int(data.get("listen_port", 11112)),
        aet=str(data.get("aet", "AEGISBRIDGE")),
        log_dir=Path(log_dir) if log_dir else platform_log_dir(),
        api_key=_read_api_key(),
        config_path=path,
    )
    return cfg


def write_yaml(path: Path, server_url: str, listen_port: int = 11112,
               aet: str = "AEGISBRIDGE") -> None:
    """Persist non-secret config to ``bridge.yml``.

    Secrets (api_key) are stored separately in the OS keyring.
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {
        "server_url": server_url,
        "listen_port": listen_port,
        "aet": aet,
    }
    with path.open("w", encoding="utf-8") as fh:
        yaml.safe_dump(payload, fh, default_flow_style=False, sort_keys=True)
