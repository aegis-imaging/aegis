"""Windows Service shim for the AEGIS DIMSE bridge.

pywin32 wraps the bridge's ``service_main.cmd_run`` so the Service Control
Manager can start/stop it like any native Windows service.

Register at install time via the WiX ``<ServiceInstall>`` element; this
script is the binary the SCM invokes. Manual install for testing:

    python service_wrapper.py install
    python service_wrapper.py start
    python service_wrapper.py stop
    python service_wrapper.py remove
"""

from __future__ import annotations

import logging
import os
import socket
import sys
import threading

import servicemanager  # type: ignore
import win32event  # type: ignore
import win32service  # type: ignore
import win32serviceutil  # type: ignore

log = logging.getLogger("aegis_bridge.service")


class AEGISDimseBridgeService(win32serviceutil.ServiceFramework):
    _svc_name_ = "AEGISDimseBridge"
    _svc_display_name_ = "AEGIS DIMSE Bridge"
    _svc_description_ = (
        "Receives DICOM C-STORE associations and uploads studies to AEGIS "
        "for de-identification, QC, and downstream distribution."
    )

    def __init__(self, args):
        win32serviceutil.ServiceFramework.__init__(self, args)
        self.stop_event = win32event.CreateEvent(None, 0, 0, None)
        self._worker: threading.Thread | None = None
        socket.setdefaulttimeout(60)

    def SvcStop(self):
        self.ReportServiceStatus(win32service.SERVICE_STOP_PENDING)
        win32event.SetEvent(self.stop_event)
        # uvicorn does not expose a clean external shutdown hook; the SCM
        # will terminate the process after the configured grace window.

    def SvcDoRun(self):
        servicemanager.LogMsg(
            servicemanager.EVENTLOG_INFORMATION_TYPE,
            servicemanager.PYS_SERVICE_STARTED,
            (self._svc_name_, ""),
        )
        # Run the bridge in a daemon thread so SvcStop's SetEvent unblocks
        # the main thread's WaitForSingleObject. Process-tear-down is
        # delegated to the SCM, matching uvicorn's lifecycle.
        self._worker = threading.Thread(target=self._run_bridge, daemon=True)
        self._worker.start()
        win32event.WaitForSingleObject(self.stop_event, win32event.INFINITE)

    def _run_bridge(self) -> None:
        try:
            # Import lazily so service registration (the install/remove
            # subcommands) does not pull in the entire receiver stack.
            from src.bridge.service_main import cmd_run

            class _Args:
                pass

            cmd_run(_Args())
        except Exception:  # pragma: no cover - logged to Windows event log
            log.exception("bridge worker crashed")
            servicemanager.LogErrorMsg(
                "AEGIS DIMSE bridge worker thread crashed; see bridge.log"
            )


if __name__ == "__main__":
    if len(sys.argv) == 1:
        servicemanager.Initialize()
        servicemanager.PrepareToHostSingle(AEGISDimseBridgeService)
        servicemanager.StartServiceCtrlDispatcher()
    else:
        win32serviceutil.HandleCommandLine(AEGISDimseBridgeService)
