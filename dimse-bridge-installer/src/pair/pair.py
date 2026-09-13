"""Pairing token exchange against the AEGIS control plane.

The installer is named ``aegis-dimse-bridge-1.0.0-pair-<TOKEN>.<ext>``. On
first launch, the bridge inspects its own executable filename, extracts the
token, and exchanges it for a long-lived API key via
``POST /api/install/pair``. The token is single-use server-side.
"""

from __future__ import annotations

import logging
import re
import sys
from pathlib import Path
from typing import Optional

import requests

log = logging.getLogger(__name__)

# Matches ``-pair-<token>`` anywhere in the basename stem. The token is
# restricted to URL-safe characters that the API generates.
_TOKEN_RE = re.compile(r"-pair-([A-Za-z0-9_-]+)")

DEFAULT_TIMEOUT_SECS = 30


def parse_token_from_argv0() -> Optional[str]:
    """Look at ``sys.executable`` (PyInstaller frozen binary path) and
    return the embedded pairing token, if any.

    Returns ``None`` if the executable name does not contain ``-pair-<token>``.
    """
    # When running as a PyInstaller --onedir bundle, sys.executable points
    # to the actual binary that was launched (which is what the installer
    # filename renames to during install on some platforms). For dev mode
    # (python -m), sys.executable is the Python interpreter, which won't
    # match — that's fine; ``aegis-bridge pair --token X`` is the manual
    # fallback.
    exe = Path(sys.executable).name
    match = _TOKEN_RE.search(exe)
    if match:
        return match.group(1)

    # Also check argv[0] in case the binary was renamed post-install.
    if sys.argv:
        argv0 = Path(sys.argv[0]).name
        match = _TOKEN_RE.search(argv0)
        if match:
            return match.group(1)
    return None


def exchange_pairing_token(server_url: str, token: str,
                           timeout: int = DEFAULT_TIMEOUT_SECS) -> dict:
    """POST the pairing token to ``{server_url}/api/install/pair``.

    Returns the parsed JSON body, expected to be
    ``{"api_key": "...", "server_url": "..."}``.

    Raises ``requests.HTTPError`` on non-2xx responses.
    """
    if not server_url:
        raise ValueError("server_url is required")
    if not token:
        raise ValueError("token is required")

    url = server_url.rstrip("/") + "/api/install/pair"
    log.info("exchanging pairing token at %s", url)
    resp = requests.post(
        url,
        json={"pairing_token": token},
        timeout=timeout,
        headers={"User-Agent": "aegis-dimse-bridge-installer/1.0"},
    )
    resp.raise_for_status()
    payload = resp.json()
    if not isinstance(payload, dict):
        raise ValueError(f"unexpected response shape: {type(payload).__name__}")
    if "api_key" not in payload:
        raise ValueError("response missing api_key")
    return payload


def store_credentials(server_url: str, api_key: str,
                      config_path: Path,
                      listen_port: int = 11112,
                      aet: str = "AEGISBRIDGE") -> None:
    """Persist credentials: API key in OS keyring, server_url in yaml.

    Importing here (rather than at module top) keeps this module testable
    without the runtime config_loader being importable.
    """
    from src.bridge.config_loader import (
        KEYRING_ACCOUNT,
        KEYRING_SERVICE,
        write_yaml,
    )

    try:
        import keyring  # type: ignore
    except ImportError as exc:  # pragma: no cover
        raise RuntimeError(
            "keyring library not installed; cannot store API key securely"
        ) from exc

    keyring.set_password(KEYRING_SERVICE, KEYRING_ACCOUNT, api_key)
    write_yaml(config_path, server_url=server_url, listen_port=listen_port, aet=aet)
    log.info("credentials stored (api_key in keyring, server_url in %s)", config_path)
