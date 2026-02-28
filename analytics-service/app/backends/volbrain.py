"""volBrain backend — online automated brain volumetry.

volBrain is a validated online pipeline for fully automated brain volume
(BV) and intracranial volume (ICV) calculations from T1 MRI scans.
This backend submits NIfTI data to the volBrain API, polls for results,
and downloads the volumetric report.

Requires:
  - ``VOLBRAIN_API_URL`` environment variable set to the API base URL
  - Network access to the volBrain service

References:
  Manjón JV, Coupé P. "volBrain: An Online MRI Brain Volumetry System."
  Frontiers in Neuroinformatics, 2016.  DOI: 10.3389/fninf.2016.00030

  https://volbrain.net
"""

from __future__ import annotations

import json
import logging
import os
import time
import urllib.error
import urllib.request

from app import config
from .base import AnalyticsBackend, AnalyticsResult
from .seg_utils import find_t1w_nifti

log = logging.getLogger(__name__)


class VolBrainBackend(AnalyticsBackend):
    """Online brain volumetry via volBrain API."""

    @property
    def name(self) -> str:
        return "volbrain"

    def available(self) -> bool:
        url = getattr(config, "VOLBRAIN_API_URL", "")
        if not url:
            return False
        # Lightweight reachability check
        try:
            req = urllib.request.Request(url, method="HEAD")
            urllib.request.urlopen(req, timeout=5)
            return True
        except Exception:
            return False

    def analyze(
        self,
        bids_dir: str,
        output_dir: str,
        study_uid: str,
        **kwargs,
    ) -> AnalyticsResult:
        start = time.time()
        out_dir = os.path.join(output_dir, "volbrain")
        os.makedirs(out_dir, exist_ok=True)

        nifti_path = find_t1w_nifti(bids_dir)
        if not nifti_path:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                error="No T1w NIfTI files found in BIDS directory",
            )

        log.info("volBrain: using %s", nifti_path)

        api_url = getattr(config, "VOLBRAIN_API_URL", "")
        api_key = getattr(config, "VOLBRAIN_API_KEY", "")
        poll_interval = int(getattr(config, "VOLBRAIN_POLL_INTERVAL", 30))
        timeout = int(getattr(config, "VOLBRAIN_TIMEOUT", 3600))

        try:
            job_id = _submit_job(api_url, api_key, nifti_path)
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"volBrain job submission failed: {e}",
            )

        log.info("volBrain: job submitted, id=%s", job_id)

        # Poll for completion
        try:
            result_data = _poll_result(
                api_url, api_key, job_id, poll_interval, timeout, start,
            )
        except TimeoutError:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"volBrain timed out after {timeout}s",
            )
        except Exception as e:
            return AnalyticsResult(
                tool=self.name,
                success=False,
                duration_seconds=time.time() - start,
                error=f"volBrain polling failed: {e}",
            )

        # Parse volumes from result
        roi_volumes = result_data.get("volumes", {})
        icv = result_data.get("icv")
        tissue_volumes = result_data.get("tissue_volumes", {})

        metrics: dict = {
            "atlas": "volbrain",
            "roi_count": len(roi_volumes),
            "roi_volumes": roi_volumes,
        }
        if icv is not None:
            metrics["icv"] = icv
        if tissue_volumes:
            metrics["tissue_volumes"] = tissue_volumes

        # Save raw API response
        raw_path = os.path.join(out_dir, "volbrain_raw.json")
        with open(raw_path, "w", encoding="utf-8") as f:
            json.dump(result_data, f, indent=2)

        summary_path = os.path.join(out_dir, "summary.json")
        with open(summary_path, "w", encoding="utf-8") as f:
            json.dump(metrics, f, indent=2)

        duration = time.time() - start
        outputs = [summary_path, raw_path]

        log.info(
            "volBrain complete for %s: %d ROIs, ICV=%.1f in %.1fs",
            study_uid,
            len(roi_volumes),
            icv if icv else 0,
            duration,
        )

        return AnalyticsResult(
            tool=self.name,
            success=True,
            duration_seconds=duration,
            outputs=outputs,
            metrics=metrics,
        )


def _submit_job(api_url: str, api_key: str, nifti_path: str) -> str:
    """Submit a NIfTI file to the volBrain API and return the job ID."""
    import mimetypes

    boundary = "----VolBrainBoundary"
    filename = os.path.basename(nifti_path)
    content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"

    with open(nifti_path, "rb") as f:
        file_data = f.read()

    # Build multipart body
    body = (
        f"--{boundary}\r\n"
        f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'
        f"Content-Type: {content_type}\r\n\r\n"
    ).encode("utf-8") + file_data + f"\r\n--{boundary}--\r\n".encode("utf-8")

    headers = {
        "Content-Type": f"multipart/form-data; boundary={boundary}",
    }
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    submit_url = f"{api_url.rstrip('/')}/api/process"
    req = urllib.request.Request(submit_url, data=body, headers=headers, method="POST")

    with urllib.request.urlopen(req, timeout=60) as resp:
        resp_data = json.loads(resp.read().decode("utf-8"))

    job_id = resp_data.get("job_id") or resp_data.get("id") or ""
    if not job_id:
        raise RuntimeError(f"No job_id in volBrain response: {resp_data}")

    return str(job_id)


def _poll_result(
    api_url: str,
    api_key: str,
    job_id: str,
    poll_interval: int,
    timeout: int,
    start_time: float,
) -> dict:
    """Poll the volBrain API until the job completes or times out."""
    status_url = f"{api_url.rstrip('/')}/api/status/{job_id}"
    headers = {}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    while True:
        elapsed = time.time() - start_time
        if elapsed > timeout:
            raise TimeoutError(f"volBrain job {job_id} timed out after {timeout}s")

        time.sleep(poll_interval)

        req = urllib.request.Request(status_url, headers=headers, method="GET")
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                data = json.loads(resp.read().decode("utf-8"))
        except urllib.error.URLError as e:
            log.warning("volBrain poll error (retrying): %s", e)
            continue

        status = data.get("status", "").lower()
        if status in ("complete", "completed", "done"):
            return data
        if status in ("error", "failed"):
            raise RuntimeError(f"volBrain job failed: {data.get('error', 'unknown')}")

        log.info("volBrain: job %s status=%s (%.0fs elapsed)", job_id, status, elapsed)
