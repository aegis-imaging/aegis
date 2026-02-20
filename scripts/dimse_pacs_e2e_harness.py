#!/usr/bin/env python3
"""DIMSE PACS end-to-end validation harness for AEGIS.

Scenarios:
1) successful C-STORE ingest
2) transient API failure -> retry queue
3) targeted retry process recovers pending study
4) dead-letter path + replay/process recovery
"""

from __future__ import annotations

import argparse
import json
import os
import socket
import subprocess
import sys
import tempfile
import threading
import time
from dataclasses import dataclass, field
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Callable, Dict, Optional, Tuple
from urllib import error as urlerror
from urllib import parse as urlparse
from urllib import request as urlrequest


class HarnessError(RuntimeError):
    pass


@dataclass
class ScenarioResult:
    name: str
    ok: bool
    elapsed_s: float
    detail: str


@dataclass
class MockIngestState:
    mode: str = "success"  # success | fail
    lock: threading.Lock = field(default_factory=threading.Lock)
    received: list[dict] = field(default_factory=list)

    def set_mode(self, mode: str) -> None:
        if mode not in {"success", "fail"}:
            raise ValueError(f"unsupported mode: {mode}")
        with self.lock:
            self.mode = mode

    def record(self, payload: dict) -> str:
        with self.lock:
            mode = self.mode
            self.received.append(
                {
                    "ts": int(time.time()),
                    "mode": mode,
                    "study_instance_uid": payload.get("study_metadata", {}).get("study_instance_uid", ""),
                    "payload": payload,
                }
            )
            return mode

    def saw_uid(self, study_uid: str, mode: Optional[str] = None) -> bool:
        with self.lock:
            for item in self.received:
                if item["study_instance_uid"] != study_uid:
                    continue
                if mode and item["mode"] != mode:
                    continue
                return True
            return False


class MockIngestHandler(BaseHTTPRequestHandler):
    state: MockIngestState = None  # type: ignore[assignment]

    def log_message(self, fmt: str, *args) -> None:
        # Keep harness output clean.
        return

    def _write_json(self, status: int, payload: dict) -> None:
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self) -> None:  # noqa: N802
        if self.path != "/api/ingest":
            self._write_json(404, {"error": "not found"})
            return

        try:
            content_len = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            content_len = 0
        raw = self.rfile.read(content_len) if content_len > 0 else b"{}"
        try:
            payload = json.loads(raw.decode("utf-8"))
        except json.JSONDecodeError:
            self._write_json(400, {"error": "invalid json"})
            return

        mode = self.state.record(payload)
        if mode == "success":
            self._write_json(201, {"status": "ok"})
        else:
            self._write_json(503, {"error": "forced failure"})


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


def http_json(
    method: str,
    url: str,
    timeout: int,
    headers: Optional[Dict[str, str]] = None,
    payload: Optional[dict] = None,
) -> Tuple[int, dict]:
    data = None
    req_headers = dict(headers or {})
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
        req_headers["Content-Type"] = "application/json"
    req = urlrequest.Request(url=url, method=method, headers=req_headers, data=data)
    try:
        with urlrequest.urlopen(req, timeout=timeout) as resp:
            body = resp.read().decode("utf-8")
            return resp.status, json.loads(body or "{}")
    except urlerror.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(body or "{}")
        except json.JSONDecodeError:
            parsed = {"raw": body}
        return e.code, parsed


def wait_until(fn: Callable[[], bool], timeout_s: int, interval_s: float = 0.5, reason: str = "") -> None:
    end = time.time() + timeout_s
    while time.time() < end:
        if fn():
            return
        time.sleep(interval_s)
    raise HarnessError(f"timeout after {timeout_s}s waiting for condition: {reason}")


def make_dataset(study_uid: str, series_uid: str, instance_uid: str):
    try:
        from pydicom.dataset import FileDataset, FileMetaDataset
        from pydicom.uid import ExplicitVRLittleEndian
    except ModuleNotFoundError as e:
        raise HarnessError(
            "Missing pydicom dependency. Install with: "
            "cd dimse-receiver && pip install -r requirements.txt -r requirements-test.txt"
        ) from e

    file_meta = FileMetaDataset()
    file_meta.MediaStorageSOPClassUID = "1.2.840.10008.5.1.4.1.1.4"
    file_meta.MediaStorageSOPInstanceUID = instance_uid
    file_meta.TransferSyntaxUID = ExplicitVRLittleEndian

    ds = FileDataset("", {}, file_meta=file_meta, preamble=b"\x00" * 128)
    ds.SOPClassUID = file_meta.MediaStorageSOPClassUID
    ds.SOPInstanceUID = file_meta.MediaStorageSOPInstanceUID
    ds.StudyInstanceUID = study_uid
    ds.SeriesInstanceUID = series_uid
    ds.Modality = "MR"
    ds.BodyPartExamined = "HEAD"
    ds.StudyDescription = "DIMSE Harness Study"

    ds.Rows = 2
    ds.Columns = 2
    ds.BitsAllocated = 16
    ds.BitsStored = 16
    ds.HighBit = 15
    ds.PixelRepresentation = 0
    ds.SamplesPerPixel = 1
    ds.PhotometricInterpretation = "MONOCHROME2"
    ds.PixelData = (b"\x00\x00" * 4)

    ds.is_little_endian = True
    ds.is_implicit_VR = False
    return ds


def send_c_store(study_uid: str, dicom_port: int, ae_title: str, instances: int = 1) -> None:
    try:
        from pydicom.uid import generate_uid
        from pynetdicom import AE
    except ModuleNotFoundError as e:
        raise HarnessError(
            "Missing DIMSE dependencies (pydicom/pynetdicom). Install with: "
            "cd dimse-receiver && pip install -r requirements.txt -r requirements-test.txt"
        ) from e

    ae = AE(ae_title="HARNESS_SCU")
    datasets = []
    for _ in range(instances):
        ds = make_dataset(study_uid=study_uid, series_uid=generate_uid(), instance_uid=generate_uid())
        datasets.append(ds)
        ae.add_requested_context(ds.SOPClassUID)

    assoc = ae.associate("127.0.0.1", dicom_port, ae_title=ae_title)
    if not assoc.is_established:
        raise HarnessError("C-STORE association was not established")
    try:
        for ds in datasets:
            status = assoc.send_c_store(ds)
            code = getattr(status, "Status", None) if status is not None else None
            if code is None or not (code == 0x0000 or (0xB000 <= code <= 0xBFFF)):
                raise HarnessError(f"C-STORE failed with status={code}")
    finally:
        assoc.release()


def call_retry(
    base_url: str,
    path: str,
    timeout: int,
    operator_key: str,
    method: str = "GET",
) -> dict:
    url = urlparse.urljoin(base_url.rstrip("/") + "/", path.lstrip("/"))
    status, payload = http_json(method, url, timeout=timeout, headers={"x-aegis-operator-key": operator_key})
    if status != 200:
        raise HarnessError(f"{method} {path} expected 200, got {status}: {payload}")
    return payload


def wait_for_study_in_queue(base_url: str, study_uid: str, timeout: int, operator_key: str) -> None:
    def _in_queue() -> bool:
        details = call_retry(
            base_url,
            f"/ingest/retry/details?limit=10&study_instance_uid={study_uid}",
            timeout=timeout,
            operator_key=operator_key,
        )["ingest_retry"]
        return details.get("pending_total", 0) > 0

    wait_until(_in_queue, timeout_s=timeout, reason=f"study {study_uid} pending in retry queue")


def wait_for_study_dead_letter(base_url: str, study_uid: str, timeout: int, operator_key: str) -> None:
    def _dead_letter() -> bool:
        details = call_retry(
            base_url,
            f"/ingest/retry/details?limit=10&study_instance_uid={study_uid}",
            timeout=timeout,
            operator_key=operator_key,
        )["ingest_retry"]
        return details.get("dead_letter_total", 0) > 0

    wait_until(_dead_letter, timeout_s=timeout, reason=f"study {study_uid} in dead-letter")


def run_scenario(results: list[ScenarioResult], name: str, fn: Callable[[], str]) -> None:
    started = time.time()
    try:
        detail = fn()
        elapsed = time.time() - started
        results.append(ScenarioResult(name=name, ok=True, elapsed_s=elapsed, detail=detail))
        print(f"[PASS] {name} ({elapsed:.2f}s) - {detail}")
    except Exception as e:
        elapsed = time.time() - started
        results.append(ScenarioResult(name=name, ok=False, elapsed_s=elapsed, detail=str(e)))
        print(f"[FAIL] {name} ({elapsed:.2f}s) - {e}")
        raise


def print_summary(results: list[ScenarioResult], success: bool) -> None:
    print("\n=== DIMSE PACS E2E Summary ===")
    for item in results:
        status = "PASS" if item.ok else "FAIL"
        print(f"- {status}: {item.name} ({item.elapsed_s:.2f}s) - {item.detail}")
    total = sum(item.elapsed_s for item in results)
    print(f"Result: {'PASS' if success else 'FAIL'} ({total:.2f}s, scenarios={len(results)})")


def tail_file(path: Path, max_lines: int = 80) -> str:
    if not path.exists():
        return "(no log file)"
    lines = path.read_text(errors="replace").splitlines()
    return "\n".join(lines[-max_lines:])


def main() -> int:
    parser = argparse.ArgumentParser(description="DIMSE PACS E2E harness")
    parser.add_argument("--timeout", type=int, default=30, help="HTTP/poll timeout seconds")
    parser.add_argument("--dimse-ae-title", default="AEGIS", help="DIMSE receiver AE title")
    parser.add_argument("--workdir", default="dimse-receiver", help="DIMSE receiver app working directory")
    parser.add_argument(
        "--receiver-python",
        default=sys.executable,
        help="Python executable used to run dimse-receiver uvicorn (default: current interpreter)",
    )
    parser.add_argument("--keep-logs", action="store_true", help="Keep harness temp directory with logs")
    args = parser.parse_args()

    results: list[ScenarioResult] = []
    temp_dir_obj = tempfile.TemporaryDirectory(prefix="aegis-dimse-e2e-")
    temp_dir = Path(temp_dir_obj.name)
    dimse_log = temp_dir / "dimse-receiver.log"

    mock_port = free_port()
    dimse_http_port = free_port()
    dimse_dicom_port = free_port()
    operator_key = "harness-operator-key"

    mock_state = MockIngestState()
    MockIngestHandler.state = mock_state
    mock_server = ThreadingHTTPServer(("127.0.0.1", mock_port), MockIngestHandler)
    mock_thread = threading.Thread(target=mock_server.serve_forever, daemon=True)
    mock_thread.start()
    print(f"Mock ingest server listening on 127.0.0.1:{mock_port}")

    preflight = subprocess.run(
        [
            args.receiver_python,
            "-c",
            "import fastapi,uvicorn,pydicom,pynetdicom",
        ],
        capture_output=True,
        text=True,
    )
    if preflight.returncode != 0:
        msg = (preflight.stderr or preflight.stdout).strip()
        raise HarnessError(
            "receiver runtime missing required packages (fastapi/uvicorn/pydicom/pynetdicom). "
            "Install with: cd dimse-receiver && pip install -r requirements.txt -r requirements-test.txt "
            f"(detail: {msg})"
        )

    log_handle = dimse_log.open("w", encoding="utf-8")
    env = os.environ.copy()
    env.update(
        {
            "DIMSE_AE_TITLE": args.dimse_ae_title,
            "DIMSE_PORT": str(dimse_dicom_port),
            "DIMSE_DATA_DIR": str(temp_dir / "data"),
            "API_URL": f"http://127.0.0.1:{mock_port}",
            "DIMSE_PROJECT_SLUG": "default",
            "DIMSE_INGEST_TIMEOUT": "3",
            "DIMSE_INGEST_RETRY_INTERVAL": "1",
            "DIMSE_INGEST_RETRY_BACKOFF_MULTIPLIER": "1.0",
            "DIMSE_INGEST_RETRY_MAX_INTERVAL": "1",
            "DIMSE_INGEST_MAX_ATTEMPTS": "2",
            "DIMSE_OPERATOR_API_KEY": operator_key,
        }
    )

    proc = subprocess.Popen(
        [args.receiver_python, "-m", "uvicorn", "app.main:app", "--host", "127.0.0.1", "--port", str(dimse_http_port)],
        cwd=args.workdir,
        env=env,
        stdout=log_handle,
        stderr=subprocess.STDOUT,
    )

    base_url = f"http://127.0.0.1:{dimse_http_port}"

    try:
        startup_deadline = time.time() + args.timeout
        while time.time() < startup_deadline:
            if proc.poll() is not None:
                raise HarnessError(
                    f"dimse-receiver exited before healthz ready (code={proc.returncode}); "
                    f"check dependencies and log output"
                )
            try:
                status, _ = http_json("GET", f"{base_url}/healthz", timeout=2)
                if status == 200:
                    break
            except Exception:
                pass
            time.sleep(0.5)
        else:
            raise HarnessError("timed out waiting for dimse-receiver /healthz")
        print(f"DIMSE receiver ready (http={dimse_http_port}, dicom={dimse_dicom_port})")

        try:
            from pydicom.uid import generate_uid
        except ModuleNotFoundError as e:
            raise HarnessError(
                "Missing pydicom dependency. Install with: "
                "cd dimse-receiver && pip install -r requirements.txt -r requirements-test.txt"
            ) from e

        def scenario_success() -> str:
            study_uid = generate_uid()
            mock_state.set_mode("success")
            send_c_store(study_uid=study_uid, dicom_port=dimse_dicom_port, ae_title=args.dimse_ae_title, instances=2)
            wait_until(
                lambda: mock_state.saw_uid(study_uid, mode="success"),
                timeout_s=args.timeout,
                reason="mock ingest saw successful study",
            )
            snap = call_retry(base_url, "/ingest/retry", timeout=args.timeout, operator_key=operator_key)["ingest_retry"]
            if snap.get("pending", 0) != 0 or snap.get("dead_letter", 0) != 0:
                raise HarnessError(f"expected empty retry/dead-letter after success, got {snap}")
            return f"study_uid={study_uid}"

        def scenario_transient_failure_to_retry() -> str:
            study_uid = generate_uid()
            mock_state.set_mode("fail")
            send_c_store(study_uid=study_uid, dicom_port=dimse_dicom_port, ae_title=args.dimse_ae_title)
            wait_for_study_in_queue(base_url, study_uid, timeout=args.timeout, operator_key=operator_key)
            return f"study_uid={study_uid}"

        def scenario_replay_process_recovery() -> str:
            study_uid = generate_uid()
            mock_state.set_mode("fail")
            send_c_store(study_uid=study_uid, dicom_port=dimse_dicom_port, ae_title=args.dimse_ae_title)
            wait_for_study_in_queue(base_url, study_uid, timeout=args.timeout, operator_key=operator_key)

            mock_state.set_mode("success")
            process = call_retry(
                base_url,
                f"/ingest/retry/process/{study_uid}",
                timeout=args.timeout,
                operator_key=operator_key,
                method="POST",
            )["ingest_retry"]
            if process.get("result") != "ok":
                raise HarnessError(f"expected targeted process result=ok, got {process}")
            wait_until(
                lambda: mock_state.saw_uid(study_uid, mode="success"),
                timeout_s=args.timeout,
                reason="mock ingest saw recovered study",
            )
            details = call_retry(
                base_url,
                f"/ingest/retry/details?limit=10&study_instance_uid={study_uid}",
                timeout=args.timeout,
                operator_key=operator_key,
            )["ingest_retry"]
            if details.get("pending_total", 0) != 0 or details.get("dead_letter_total", 0) != 0:
                raise HarnessError(f"expected study cleared from retry/dead-letter, got {details}")
            return f"study_uid={study_uid}"

        def scenario_dead_letter_and_operator_recovery() -> str:
            study_uid = generate_uid()
            mock_state.set_mode("fail")
            send_c_store(study_uid=study_uid, dicom_port=dimse_dicom_port, ae_title=args.dimse_ae_title)
            wait_for_study_in_queue(base_url, study_uid, timeout=args.timeout, operator_key=operator_key)

            for _ in range(4):
                result = call_retry(
                    base_url,
                    f"/ingest/retry/process/{study_uid}",
                    timeout=args.timeout,
                    operator_key=operator_key,
                    method="POST",
                )["ingest_retry"]
                if result.get("result") == "dead_letter":
                    break
            wait_for_study_dead_letter(base_url, study_uid, timeout=args.timeout, operator_key=operator_key)

            health = http_json("GET", f"{base_url}/healthz", timeout=args.timeout)[1]
            reasons = health.get("degraded_reasons", [])
            if "dead_letter_nonzero" not in reasons:
                raise HarnessError(f"expected dead_letter_nonzero in health degraded reasons, got {reasons}")

            mock_state.set_mode("success")
            replay = call_retry(
                base_url,
                f"/ingest/retry/replay/{study_uid}",
                timeout=args.timeout,
                operator_key=operator_key,
                method="POST",
            )["ingest_retry"]
            if replay.get("moved", 0) != 1:
                raise HarnessError(f"expected moved=1 on replay, got {replay}")

            final = call_retry(
                base_url,
                f"/ingest/retry/process/{study_uid}",
                timeout=args.timeout,
                operator_key=operator_key,
                method="POST",
            )["ingest_retry"]
            if final.get("result") != "ok":
                raise HarnessError(f"expected final targeted process result=ok, got {final}")

            wait_until(
                lambda: mock_state.saw_uid(study_uid, mode="success"),
                timeout_s=args.timeout,
                reason="mock ingest saw replayed/recovered study",
            )

            details = call_retry(
                base_url,
                f"/ingest/retry/details?limit=10&study_instance_uid={study_uid}",
                timeout=args.timeout,
                operator_key=operator_key,
            )["ingest_retry"]
            if details.get("pending_total", 0) != 0 or details.get("dead_letter_total", 0) != 0:
                raise HarnessError(f"expected no remaining retry/dead-letter entries, got {details}")
            return f"study_uid={study_uid}"

        run_scenario(results, "success_c_store_ingest", scenario_success)
        run_scenario(results, "transient_failure_to_retry_queue", scenario_transient_failure_to_retry)
        run_scenario(results, "process_controls_restore_ingestion", scenario_replay_process_recovery)
        run_scenario(results, "dead_letter_path_and_recovery", scenario_dead_letter_and_operator_recovery)

    except Exception as e:
        print_summary(results, success=False)
        print("\n=== dimse-receiver log tail ===")
        print(tail_file(dimse_log))
        print(f"\nHarness failed: {e}")
        return 1
    finally:
        if proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
        log_handle.close()
        mock_server.shutdown()
        mock_server.server_close()
        if args.keep_logs:
            print(f"Harness logs/data kept at: {temp_dir}")
        else:
            temp_dir_obj.cleanup()

    print_summary(results, success=True)
    return 0


if __name__ == "__main__":
    sys.exit(main())
