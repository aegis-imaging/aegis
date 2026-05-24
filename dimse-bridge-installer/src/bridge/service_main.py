"""AEGIS DIMSE bridge service entry point.

Subcommands:
    run     — start the FastAPI + DIMSE SCP (default)
    pair    — exchange a pairing token for an API key (writes credentials)
    version — print the package version

The actual receiver implementation lives in ``dimse-receiver/app/`` from the
main repo; we add it to ``sys.path`` so dev mode works without symlinks. At
PyInstaller build time, ``bridge.spec`` bundles the ``app`` package as a
hidden import / extra path so the frozen binary is self-contained.
"""

from __future__ import annotations

import argparse
import logging
import os
import sys
from pathlib import Path
from typing import Optional, Sequence

log = logging.getLogger("aegis_bridge")


def _add_receiver_to_syspath() -> None:
    """Make the upstream ``dimse-receiver`` package importable.

    In dev mode (repo checkout): ``../dimse-receiver/`` relative to this file.
    In PyInstaller frozen mode: the spec file embeds the receiver as a
    pathex entry, so this is a no-op (the import would already succeed).
    """
    if getattr(sys, "frozen", False):
        return

    here = Path(__file__).resolve()
    candidates = [
        here.parents[3] / "dimse-receiver",
        here.parents[2] / "dimse-receiver",
    ]
    for c in candidates:
        if (c / "app" / "__init__.py").exists():
            if str(c) not in sys.path:
                sys.path.insert(0, str(c))
            return
    log.warning(
        "could not locate dimse-receiver package; "
        "set PYTHONPATH or build via PyInstaller"
    )


def _configure_logging(log_dir: Optional[Path]) -> None:
    handlers: list[logging.Handler] = [logging.StreamHandler(sys.stderr)]
    if log_dir is not None:
        try:
            log_dir.mkdir(parents=True, exist_ok=True)
            handlers.append(logging.FileHandler(log_dir / "bridge.log"))
        except OSError as exc:
            log.warning("could not open log dir %s: %s", log_dir, exc)
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
        handlers=handlers,
        force=True,
    )


def cmd_run(_args: argparse.Namespace) -> int:
    """Start the FastAPI app + DIMSE C-STORE SCP."""
    from src.bridge.config_loader import load

    cfg = load()
    _configure_logging(cfg.log_dir)
    log.info("loaded config: server=%s port=%d aet=%s",
             cfg.server_url or "(unset)", cfg.listen_port, cfg.aet)

    # Surface config into the receiver via env vars (the receiver reads
    # everything from os.environ in app/config.py).
    if cfg.server_url:
        os.environ.setdefault("AEGIS_SERVER_URL", cfg.server_url)
    if cfg.api_key:
        os.environ.setdefault("AEGIS_API_KEY", cfg.api_key)
    os.environ.setdefault("DIMSE_PORT", str(cfg.listen_port))
    os.environ.setdefault("DIMSE_AET", cfg.aet)

    _add_receiver_to_syspath()

    try:
        # Late import: receiver package only resolvable after sys.path mutation.
        from app.main import app as fastapi_app  # type: ignore
    except ImportError as exc:
        log.error("failed to import dimse-receiver app.main: %s", exc)
        return 2

    import uvicorn  # local import keeps cold-start fast for `version`

    health_port = int(os.environ.get("HEALTH_PORT", "8080"))
    log.info("starting uvicorn on 0.0.0.0:%d (DIMSE SCP on %d)",
             health_port, cfg.listen_port)
    uvicorn.run(fastapi_app, host="0.0.0.0", port=health_port, log_config=None)
    return 0


def cmd_pair(args: argparse.Namespace) -> int:
    """Exchange a pairing token for an API key and persist credentials."""
    from src.bridge.config_loader import load, platform_config_path
    from src.pair.pair import (
        exchange_pairing_token,
        parse_token_from_argv0,
        store_credentials,
    )

    _configure_logging(None)

    token = args.token or parse_token_from_argv0()
    server = args.server
    if not token:
        log.error("no pairing token supplied (pass --token or rename binary "
                  "to include -pair-<TOKEN>)")
        return 2
    if not server:
        log.error("no server URL supplied (pass --server https://...)")
        return 2

    try:
        result = exchange_pairing_token(server, token)
    except Exception as exc:
        log.error("pairing exchange failed: %s", exc)
        return 1

    api_key = result["api_key"]
    server_url = result.get("server_url", server)
    cfg_path = args.config or platform_config_path()
    cfg = load(cfg_path)

    try:
        store_credentials(
            server_url=server_url,
            api_key=api_key,
            config_path=cfg_path,
            listen_port=cfg.listen_port,
            aet=cfg.aet,
        )
    except Exception as exc:
        log.error("failed to store credentials: %s", exc)
        return 1
    log.info("pairing complete; server=%s config=%s", server_url, cfg_path)
    return 0


def cmd_version(_args: argparse.Namespace) -> int:
    from src.bridge import __version__
    print(f"aegis-bridge {__version__}")
    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="aegis-bridge",
        description="AEGIS DIMSE bridge — native service wrapper",
    )
    sub = p.add_subparsers(dest="command")

    sub.add_parser("run", help="Start the bridge service (default)")

    pair_p = sub.add_parser("pair", help="Exchange a pairing token for an API key")
    pair_p.add_argument("--token", help="Pairing token (otherwise read from "
                                        "the installer filename)")
    pair_p.add_argument("--server", required=True,
                        help="AEGIS control-plane base URL, e.g. "
                             "https://api.aegisimaging.ai")
    pair_p.add_argument("--config", type=Path, default=None,
                        help="Override bridge.yml path (defaults to "
                             "platform-standard location)")

    sub.add_parser("version", help="Print version and exit")
    return p


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    cmd = args.command or "run"
    handlers = {
        "run": cmd_run,
        "pair": cmd_pair,
        "version": cmd_version,
    }
    return handlers[cmd](args)


if __name__ == "__main__":
    raise SystemExit(main())
