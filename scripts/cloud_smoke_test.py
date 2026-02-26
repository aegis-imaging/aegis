#!/usr/bin/env python3
"""AEGIS cloud smoke test harness.

Checks:
1) API health
2) auth identity endpoint
3) at least one enabled admin user registered (first-admin bootstrap check)
4) upload init/file/complete
5) one pipeline progression check (study detail)
6) study approve
7) export share create/redeem/download
8) DIMSE query endpoint validation (always) + optional C-FIND query (if --dimse-* args provided)
"""

from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from dataclasses import dataclass
from typing import Dict, Iterable, Optional, Tuple


class SmokeFailure(RuntimeError):
    pass


@dataclass
class StepResult:
    name: str
    ok: bool
    elapsed_s: float
    detail: str


def build_url(base_url: str, path_or_url: str) -> str:
    parsed = urllib.parse.urlparse(path_or_url)
    if parsed.scheme and parsed.netloc:
        return path_or_url
    return urllib.parse.urljoin(base_url.rstrip("/") + "/", path_or_url.lstrip("/"))


def http_request(
    method: str,
    url: str,
    headers: Optional[Dict[str, str]] = None,
    json_body: Optional[dict] = None,
    raw_body: Optional[bytes] = None,
    timeout_s: int = 30,
) -> Tuple[int, bytes, Dict[str, str]]:
    data = raw_body
    req_headers = dict(headers or {})
    if json_body is not None:
        data = json.dumps(json_body).encode("utf-8")
        req_headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url=url, method=method, data=data, headers=req_headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout_s) as resp:
            return resp.status, resp.read(), dict(resp.headers.items())
    except urllib.error.HTTPError as e:
        body = e.read()
        return e.code, body, dict(e.headers.items())


def expect_status(
    status: int,
    allowed: Iterable[int],
    step: str,
    body: bytes,
) -> None:
    if status in allowed:
        return
    detail = body.decode("utf-8", errors="replace")
    raise SmokeFailure(f"{step}: expected HTTP {sorted(set(allowed))}, got {status}. body={detail}")


def parse_json(body: bytes, step: str) -> dict:
    try:
        return json.loads(body.decode("utf-8"))
    except json.JSONDecodeError as e:
        raise SmokeFailure(f"{step}: response was not valid JSON ({e})") from e


def parse_headers(header_args: list[str]) -> Dict[str, str]:
    headers: Dict[str, str] = {}
    for raw in header_args:
        if ":" not in raw:
            raise SmokeFailure(f"invalid --admin-header value '{raw}' (expected 'Header-Name: value')")
        key, value = raw.split(":", 1)
        headers[key.strip()] = value.strip()
    return headers


def run_step(results: list[StepResult], name: str, fn) -> None:
    started = time.time()
    try:
        detail = fn()
        elapsed = time.time() - started
        results.append(StepResult(name=name, ok=True, elapsed_s=elapsed, detail=detail or "ok"))
        print(f"[PASS] {name} ({elapsed:.2f}s) - {detail or 'ok'}")
    except SmokeFailure as e:
        elapsed = time.time() - started
        results.append(StepResult(name=name, ok=False, elapsed_s=elapsed, detail=str(e)))
        print(f"[FAIL] {name} ({elapsed:.2f}s) - {e}")
        raise


def main() -> int:
    parser = argparse.ArgumentParser(description="AEGIS cloud smoke suite")
    parser.add_argument("--base-url", required=True, help="Base API URL, e.g. https://api-dev.aegisimaging.ai")
    parser.add_argument("--project-slug", default="default", help="Project slug for upload init")
    parser.add_argument("--timeout", type=int, default=30, help="HTTP timeout in seconds (default: 30)")
    parser.add_argument("--pipeline-timeout", type=int, default=45, help="Pipeline poll timeout in seconds")
    parser.add_argument(
        "--admin-header",
        action="append",
        default=[],
        help="Admin auth header, repeatable. Example: --admin-header 'X-Goog-Authenticated-User-Email: accounts.google.com:user@example.com'",
    )
    parser.add_argument(
        "--iap-email",
        default="",
        help="Convenience for GCP IAP header injection. Sets X-Goog-Authenticated-User-Email automatically.",
    )
    parser.add_argument(
        "--azure-email",
        default="",
        help="Convenience for Azure Easy Auth header injection. Sets X-MS-CLIENT-PRINCIPAL-NAME automatically.",
    )
    parser.add_argument(
        "--dimse-ae-title",
        default="",
        help="Remote PACS AE title for optional DIMSE C-FIND smoke check (e.g. ORTHANC).",
    )
    parser.add_argument(
        "--dimse-host",
        default="",
        help="Remote PACS hostname/IP for optional DIMSE C-FIND smoke check.",
    )
    parser.add_argument(
        "--dimse-port",
        type=int,
        default=0,
        help="Remote PACS port for optional DIMSE C-FIND smoke check (e.g. 4242).",
    )
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")
    results: list[StepResult] = []
    admin_headers = parse_headers(args.admin_header)
    if args.iap_email:
        admin_headers["X-Goog-Authenticated-User-Email"] = f"accounts.google.com:{args.iap_email}"
    if args.azure_email:
        admin_headers["X-MS-CLIENT-PRINCIPAL-NAME"] = args.azure_email

    smoke_uploader = f"smoke+{int(time.time())}@example.com"
    study_uid_seed = f"2.25.{uuid.uuid4().int}"
    state: Dict[str, str] = {}

    try:
        run_step(
            results,
            "healthz",
            lambda: step_healthz(base_url, args.timeout),
        )

        run_step(
            results,
            "auth.me",
            lambda: step_auth_me(base_url, admin_headers, args.timeout),
        )

        run_step(
            results,
            "admin.users.registered",
            lambda: step_admin_users_registered(base_url, admin_headers, args.timeout),
        )

        run_step(
            results,
            "upload.init",
            lambda: step_upload_init(base_url, args.project_slug, smoke_uploader, study_uid_seed, args.timeout, state),
        )

        run_step(
            results,
            "upload.file",
            lambda: step_upload_file(args.timeout, state),
        )

        run_step(
            results,
            "upload.complete",
            lambda: step_upload_complete(base_url, args.timeout, state),
        )

        run_step(
            results,
            "pipeline.progression",
            lambda: step_pipeline_check(base_url, admin_headers, args.timeout, args.pipeline_timeout, state),
        )

        run_step(
            results,
            "study.approve",
            lambda: step_approve(base_url, admin_headers, args.timeout, state),
        )

        run_step(
            results,
            "share.create",
            lambda: step_create_share(base_url, admin_headers, args.timeout, state),
        )

        run_step(
            results,
            "share.redeem",
            lambda: step_redeem_share(base_url, args.timeout, state),
        )

        run_step(
            results,
            "share.download",
            lambda: step_download_share(base_url, args.timeout, state),
        )

        run_step(
            results,
            "dimse.query.check",
            lambda: step_dimse_query_check(
                base_url,
                admin_headers,
                args.timeout,
                args.dimse_ae_title,
                args.dimse_host,
                args.dimse_port,
            ),
        )
    except SmokeFailure:
        print_summary(results, success=False)
        return 1

    print_summary(results, success=True)
    return 0


def step_healthz(base_url: str, timeout: int) -> str:
    status, body, _ = http_request("GET", build_url(base_url, "/healthz"), timeout_s=timeout)
    expect_status(status, {200}, "healthz", body)
    payload = parse_json(body, "healthz")
    if payload.get("status") != "ok":
        raise SmokeFailure(f"healthz: expected status=ok, got {payload.get('status')!r}")
    return "status=ok"


def step_auth_me(base_url: str, admin_headers: Dict[str, str], timeout: int) -> str:
    status, body, _ = http_request("GET", build_url(base_url, "/api/auth/me"), headers=admin_headers, timeout_s=timeout)
    expect_status(status, {200}, "auth.me", body)
    payload = parse_json(body, "auth.me")
    email = payload.get("email", "")
    role = payload.get("role", "")
    if not email:
        raise SmokeFailure("auth.me: missing email in response")
    return f"email={email}, role={role or 'unknown'}"


def step_admin_users_registered(base_url: str, admin_headers: Dict[str, str], timeout: int) -> str:
    """Verify that at least one admin user is registered.

    On a fresh deployment with no admin users, every protected endpoint will
    return 403 'user not registered'. This step detects that configuration gap
    early and surfaces the fix: set FIRST_ADMIN_EMAIL env var (or seed via
    POST /api/admin-users when auth is disabled in dev mode).
    """
    status, body, _ = http_request(
        "GET",
        build_url(base_url, "/api/admin-users"),
        headers=admin_headers,
        timeout_s=timeout,
    )
    expect_status(status, {200}, "admin.users.registered", body)
    payload = parse_json(body, "admin.users.registered")
    users = payload if isinstance(payload, list) else payload.get("admin_users", [])
    enabled = [u for u in users if u.get("enabled")]
    if not enabled:
        raise SmokeFailure(
            "admin.users.registered: no enabled admin users found. "
            "Set FIRST_ADMIN_EMAIL env var on the API and restart, "
            "or seed via POST /api/admin-users (dev mode only)."
        )
    return f"count={len(enabled)} enabled admin user(s)"


def step_upload_init(
    base_url: str,
    project_slug: str,
    uploader_email: str,
    study_uid: str,
    timeout: int,
    state: Dict[str, str],
) -> str:
    payload = {
        "project_slug": project_slug,
        "file_count": 1,
        "uploader_email": uploader_email,
        "study_metadata": {
            "study_instance_uid": study_uid,
            "modality": "MRI",
            "body_part": "HEAD",
            "study_description": "Cloud smoke upload",
            "series_count": 1,
            "instance_count": 1,
        },
    }
    status, body, _ = http_request(
        "POST",
        build_url(base_url, "/api/upload/init"),
        json_body=payload,
        timeout_s=timeout,
    )
    expect_status(status, {200}, "upload.init", body)
    resp = parse_json(body, "upload.init")
    session_id = resp.get("session_id")
    upload_urls = resp.get("upload_urls") or []
    if not session_id or not upload_urls:
        raise SmokeFailure("upload.init: missing session_id or upload_urls")
    state["session_id"] = session_id
    state["upload_url"] = upload_urls[0]
    return f"session_id={session_id}"


def step_upload_file(timeout: int, state: Dict[str, str]) -> str:
    upload_url = state.get("upload_url")
    if not upload_url:
        raise SmokeFailure("upload.file: missing upload_url state")
    synthetic_dicom = b"DICM-SMOKE-" + uuid.uuid4().hex.encode("ascii")
    status, body, _ = http_request(
        "PUT",
        upload_url,
        headers={"Content-Type": "application/dicom"},
        raw_body=synthetic_dicom,
        timeout_s=timeout,
    )
    expect_status(status, {200}, "upload.file", body)
    return f"bytes={len(synthetic_dicom)}"


def step_upload_complete(base_url: str, timeout: int, state: Dict[str, str]) -> str:
    session_id = state.get("session_id")
    if not session_id:
        raise SmokeFailure("upload.complete: missing session_id state")
    status, body, _ = http_request(
        "POST",
        build_url(base_url, "/api/upload/complete"),
        json_body={"session_id": session_id},
        timeout_s=timeout,
    )
    expect_status(status, {200}, "upload.complete", body)
    resp = parse_json(body, "upload.complete")
    study = resp.get("study") or {}
    study_id = study.get("id")
    study_uid = study.get("study_instance_uid")
    if not study_id or not study_uid:
        raise SmokeFailure("upload.complete: missing study id/uid in response")
    state["study_id"] = study_id
    state["study_uid"] = study_uid
    return f"study_id={study_id}"


def step_pipeline_check(
    base_url: str,
    admin_headers: Dict[str, str],
    timeout: int,
    pipeline_timeout: int,
    state: Dict[str, str],
) -> str:
    study_id = state.get("study_id")
    if not study_id:
        raise SmokeFailure("pipeline.progression: missing study_id state")
    deadline = time.time() + pipeline_timeout
    latest_status = ""
    while time.time() < deadline:
        status, body, _ = http_request(
            "GET",
            build_url(base_url, f"/api/studies/{study_id}"),
            headers=admin_headers,
            timeout_s=timeout,
        )
        expect_status(status, {200}, "pipeline.progression", body)
        study = parse_json(body, "pipeline.progression")
        latest_status = study.get("status", "")
        if latest_status:
            if study.get("source") not in ("external", "internal"):
                raise SmokeFailure("pipeline.progression: study source missing/invalid")
            return f"status={latest_status}"
        time.sleep(2)
    raise SmokeFailure(f"pipeline.progression: timed out waiting for status (last={latest_status!r})")


def step_approve(base_url: str, admin_headers: Dict[str, str], timeout: int, state: Dict[str, str]) -> str:
    study_id = state.get("study_id")
    if not study_id:
        raise SmokeFailure("study.approve: missing study_id state")
    status, body, _ = http_request(
        "POST",
        build_url(base_url, f"/api/studies/{study_id}/approve"),
        headers=admin_headers,
        json_body={},
        timeout_s=timeout,
    )
    if status == 200:
        return "approved"
    if status == 400 and b"already approved" in body:
        return "already approved"
    expect_status(status, {200}, "study.approve", body)
    return "approved"


def step_create_share(base_url: str, admin_headers: Dict[str, str], timeout: int, state: Dict[str, str]) -> str:
    study_id = state.get("study_id")
    if not study_id:
        raise SmokeFailure("share.create: missing study_id state")
    payload = {
        "recipient_email": f"smoke-recipient+{int(time.time())}@example.com",
        "note": "cloud smoke test",
        "expiry_hours": 1,
    }
    status, body, _ = http_request(
        "POST",
        build_url(base_url, f"/api/studies/{study_id}/share"),
        headers=admin_headers,
        json_body=payload,
        timeout_s=timeout,
    )
    expect_status(status, {201}, "share.create", body)
    resp = parse_json(body, "share.create")
    export_url = resp.get("export_url", "")
    if not export_url:
        raise SmokeFailure("share.create: missing export_url")
    token = export_url.rstrip("/").split("/")[-1]
    if not token:
        raise SmokeFailure("share.create: could not parse token from export_url")
    state["export_token"] = token
    return f"share_id={resp.get('id', 'unknown')}"


def step_redeem_share(base_url: str, timeout: int, state: Dict[str, str]) -> str:
    token = state.get("export_token")
    if not token:
        raise SmokeFailure("share.redeem: missing export_token state")
    status, body, _ = http_request("GET", build_url(base_url, f"/api/export/{token}"), timeout_s=timeout)
    expect_status(status, {200}, "share.redeem", body)
    resp = parse_json(body, "share.redeem")
    download_url = resp.get("download_url", "")
    if not download_url:
        raise SmokeFailure("share.redeem: missing download_url")
    state["download_url"] = download_url
    return f"study_uid={resp.get('study_uid', 'unknown')}"


def step_download_share(base_url: str, timeout: int, state: Dict[str, str]) -> str:
    token = state.get("export_token")
    if not token:
        raise SmokeFailure("share.download: missing export_token state")
    status, body, headers = http_request("GET", build_url(base_url, f"/api/export/{token}/download"), timeout_s=timeout)
    expect_status(status, {200}, "share.download", body)
    content_type = next((v for k, v in headers.items() if k.lower() == "content-type"), "")
    if "zip" not in content_type.lower():
        raise SmokeFailure(f"share.download: expected zip content type, got {content_type!r}")
    if len(body) == 0:
        raise SmokeFailure("share.download: empty response body")
    return f"bytes={len(body)}"


def step_dimse_query_check(
    base_url: str,
    admin_headers: Dict[str, str],
    timeout: int,
    dimse_ae_title: str,
    dimse_host: str,
    dimse_port: int,
) -> str:
    """Verify the DIMSE query endpoint.

    Phase 1 (always): Send a request with a missing required field — the
    API must return 400 (validation), confirming the route is registered
    and the handler is reachable.

    Phase 2 (when --dimse-* args are all provided): Send a properly-formed
    C-FIND query and expect 200 with a ``matches`` list (possibly empty) or
    503 when the DIMSE receiver sidecar is not configured on this deployment.
    A 502/timeout from the sidecar is also surfaced as informational (not a
    hard failure) so the smoke suite does not fail on deployments where the
    remote PACS is unreachable.
    """
    query_url = build_url(base_url, "/api/dimse/query")

    # Phase 1: validation check — POST with no body fields → expect 400.
    status, body, _ = http_request(
        "POST", query_url, headers=admin_headers, json_body={}, timeout_s=timeout
    )
    if status != 400:
        raise SmokeFailure(
            f"dimse.query.check: expected 400 for missing required fields, got {status}; body={body[:200]!r}"
        )

    # Phase 2: live query (only when all three DIMSE params are provided).
    if not (dimse_ae_title and dimse_host and dimse_port > 0):
        return "validation=400 (live query skipped — pass --dimse-ae-title/--dimse-host/--dimse-port to enable)"

    status2, body2, _ = http_request(
        "POST",
        query_url,
        headers=admin_headers,
        json_body={
            "ae_title": dimse_ae_title,
            "host": dimse_host,
            "port": dimse_port,
            "query_level": "STUDY",
            "query_params": {},
        },
        timeout_s=timeout,
    )

    if status2 == 503:
        return "validation=400, live=503 (DIMSE receiver not configured on this deployment)"
    if status2 in (502, 504):
        return f"validation=400, live={status2} (DIMSE sidecar unreachable — informational)"
    if status2 != 200:
        raise SmokeFailure(
            f"dimse.query.check: live query returned {status2}; body={body2[:200]!r}"
        )

    resp2 = parse_json(body2, "dimse.query.check.live")
    matches = resp2.get("matches", [])
    count = resp2.get("count", len(matches))
    return f"validation=400, live=200 count={count}"


def print_summary(results: list[StepResult], success: bool) -> None:
    print("\n=== Cloud Smoke Summary ===")
    for r in results:
        status = "PASS" if r.ok else "FAIL"
        print(f"- {status}: {r.name} ({r.elapsed_s:.2f}s) - {r.detail}")
    total = sum(r.elapsed_s for r in results)
    print(f"Result: {'PASS' if success else 'FAIL'} ({total:.2f}s, steps={len(results)})")


if __name__ == "__main__":
    sys.exit(main())
