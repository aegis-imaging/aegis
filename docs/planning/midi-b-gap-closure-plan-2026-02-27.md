# MIDI-B Gap Closure Plan

**Date:** 2026-02-27
**Status:** Draft — pending implementation
**Prerequisites:** PRs #371–#376 merged to `develop`

---

## Table of Contents

1. [Current State](#current-state)
2. [Findings: CLI Gaps](#findings-cli-gaps)
3. [Findings: Go API Supporting Integrations](#findings-go-api-supporting-integrations)
4. [Findings: MCP Server](#findings-mcp-server)
5. [Findings: Test Coverage](#findings-test-coverage)
6. [Findings: Minor Issues](#findings-minor-issues)
7. [Implementation Checklist](#implementation-checklist)

---

## Current State

### What's Merged

| Deliverable | PR | Tests | Notes |
|---|---|---|---|
| Client library: date shifting, pseudonymization, mapping CSV, text scrubbing | #371 | 80 vitest | TypeScript `@aegis/client` |
| phi-detection: pixel redaction + LLM text scrubbing endpoints | #372 | 134 pytest | `/redact` and `/scrub-text` endpoints |
| Client library: SR sequence handling | #373 | 6 new vitest (80 total) | `processSequences()` in `deid.ts` |
| Go API: pixel redaction pipeline step | #374 | Go builds clean | Core plumbing only |
| MIDI-B Python CLI tool | #375 | 79 pytest | Standalone de-identification |
| CI fix: access matrix for pixel-redaction route | #376 | — | Administrative fix |

### MIDI-B Benchmark Requirements (10 Scored Actions)

The NCI MIDI-B challenge evaluates 10 specific de-identification actions across 29,660 DICOM instances (8 modalities: MR, CT, PET, CR, DX, SR, MG, US):

**Metadata actions (8):**
1. Date shifting (consistent per patient)
2. Patient ID pseudonymization (consistent per patient)
3. Tag retention of research-useful fields
4. Free-text field scrubbing (remove PHI, preserve non-PHI)
5. UID replacement and cross-file consistency
6. Private tag removal
7. DICOM standard compliance preservation
8. SR sequence de-identification

**Pixel data actions (2):**
9. Burned-in text removal from pixel data
10. Pixel integrity preservation (no corruption of non-PHI regions)

Scoring is at both series level (primary — any instance error fails the whole series) and instance level (supplementary).

---

## Findings: CLI Gaps

### Gap 1: Pixel Redaction Not in CLI (CRITICAL)

**Impact:** 2 of 10 scored actions (burned-in text removal + pixel integrity) are completely unaddressable by the standalone CLI. US and MG modalities in the test set commonly have burned-in text overlays.

**Current state:** Pixel redaction exists only server-side:
- `phi-detection/app/backends/pixel_redact.py` — `redact_dicom_pixels(input_path, output_path, regions, padding_px=5)` zeros out bounding box regions in DICOM pixel arrays
- `phi-detection/app/backends/pixel_utils.py` — `dicom_to_pil(path)` converts DICOM to PIL Image for OCR
- `phi-detection/app/main.py` — `POST /redact` accepts `{study_uid, input_paths, output_dir}`, runs OCR detection then redaction, returns `{files_processed, files_redacted, findings}`

**Plan:** Add `--pixel-redact` flag to the CLI with two modes:
1. **Local mode** (default when `--phi-service-url` is not set): Use Tesseract OCR directly via pytesseract. Copy the detection logic from `phi-detection/app/backends/tesseract_backend.py` and the redaction logic from `pixel_redact.py` + `pixel_utils.py` into the CLI. Dependencies: `Pillow`, `pytesseract`, `numpy` (already a dependency).
2. **Remote mode** (when `--phi-service-url` is set): POST to `{phi-service-url}/redact` with file paths. This gives access to all OCR backends (Gemini, Cloud Vision, Azure Vision, Textract, Tesseract).

**Files to create/modify:**
- `midi-b/midi_b/deid/pixel_redact.py` (NEW) — copy `redact_dicom_pixels()` from `phi-detection/app/backends/pixel_redact.py`
- `midi-b/midi_b/deid/pixel_utils.py` (NEW) — copy `dicom_to_pil()`, `_apply_windowing()`, `_to_uint8()` from `phi-detection/app/backends/pixel_utils.py`
- `midi-b/midi_b/deid/pixel_detect.py` (NEW) — local Tesseract OCR detection (simplified version of `tesseract_backend.py`)
- `midi-b/midi_b/pipeline.py` (MODIFY) — add pixel redaction step after tag de-identification
- `midi-b/midi_b/cli.py` (MODIFY) — add `--pixel-redact / --no-pixel-redact` and `--phi-service-url` options
- `midi-b/requirements.txt` (MODIFY) — add `Pillow>=10.0` and `pytesseract>=0.3` as optional deps
- `midi-b/tests/test_pixel_redact.py` (NEW) — test redaction with synthetic DICOM
- `midi-b/tests/test_pixel_detect.py` (NEW) — test OCR detection with mocked Tesseract

### Gap 2: LLM Text Scrubbing Not in CLI (IMPORTANT)

**Impact:** The MIDI-B paper identified free-text processing as "the most challenging action type." Regex-only scrubbing will miss edge cases that LLM backends catch.

**Current state:** The CLI uses `midi_b/deid/text_scrub.py` (regex-only, 14 patterns + context matching). LLM scrubbing exists server-side at `phi-detection POST /scrub-text`:
- Request: `{texts: [str], context: {patient_name, patient_id, ...}}`
- Response: `{results: [{original, scrubbed, phi_found}], tool_used}`
- Backends: gemini > openai > anthropic > regex (auto-select)

**Plan:** Add `--phi-service-url` integration for text scrubbing:
- When `--phi-service-url` is set, the engine's C-action (clean) handler calls `POST {phi-service-url}/scrub-text` instead of the local regex backend
- Batch all C-action text values into a single request per file for efficiency
- Fall back to local regex if the service call fails
- No new CLI flag needed — piggybacking on `--phi-service-url` covers both pixel and text

**Files to modify:**
- `midi-b/midi_b/deid/text_scrub.py` (MODIFY) — add `RemoteTextScrubBackend` class that calls `/scrub-text`
- `midi-b/midi_b/deid/engine.py` (MODIFY) — accept an optional text scrub backend; use remote when available
- `midi-b/midi_b/pipeline.py` (MODIFY) — instantiate remote backend when `phi_service_url` is set
- `midi-b/midi_b/cli.py` (MODIFY) — add `--phi-service-url` option, pass to pipeline
- `midi-b/tests/test_text_scrub.py` (MODIFY) — add tests for remote backend (mocked HTTP)

---

## Findings: Go API Supporting Integrations

PR #374 implemented the core pixel redaction pipeline (migration, model, handler, routing, pipeline dispatch) but left out several supporting integrations that other pipeline steps have.

### Gap 3: Pipeline Funnel Missing `pixel_redacted` Stage

**File:** `api/handler/pipeline_funnel.go`

The funnel stages go: `received → classified → phi_scanned → defaced → qc_passed → bids_converted → approved → exported`. There is no `pixel_redacted` stage between `phi_scanned` and `defaced`.

**Changes needed (4 locations):**
1. **Line ~45** — Add `pixelRedacted int` variable after `phiScanned`
2. **Line ~55** — Add SQL: `COUNT(*) FILTER (WHERE pixel_redaction_status = 'complete') AS pixel_redacted,` after `phi_scanned` line
3. **Line ~67** — Add `&pixelRedacted` to `.Scan()` call after `&phiScanned`
4. **Line ~87** — Add `{"pixel_redacted", pixelRedacted},` to stages slice after `{"phi_scanned", phiScanned}`

### Gap 4: Study Diagnostics Missing Pixel Redaction

**File:** `api/handler/study_diagnostics.go`

The diagnostics endpoint evaluates blockers for classification, PHI scan, protocol, QC, BIDS, and export — but not pixel redaction. Studies stuck due to pixel redaction will not appear in "why stuck?" output.

**Changes needed (2 locations):**
1. **After line ~161** — Add `evaluateRequiredStage` call:
   ```go
   evaluateRequiredStage(study.PixelRedactionRequired, study.PixelRedactionStatus,
       "Pixel redaction", fmt.Sprintf("POST /api/studies/%s/pixel-redaction", study.StudyInstanceUID), addBlocker, addAction)
   ```
2. **Line ~221-228** — Add `"redacting": true` to the `inProgress` map

### Gap 5: Study CSV Export Missing Pixel Redaction Columns

**File:** `api/handler/study_csv.go`

CSV headers include `phi_scan_status`, `qc_status`, `bids_status`, etc. but not `pixel_redaction_required` or `pixel_redaction_status`.

**Changes needed (2 locations):**
1. **Line ~64** — Add `"pixel_redaction_status"` to header row after `"phi_scan_status"`
2. **Line ~86** — Add `st.PixelRedactionStatus` to row data after `st.PhiScanStatus`

### Gap 6: Processing Times Missing Pixel Redaction Stage

**File:** `api/handler/processing_times_query.go`

The SQL `WHERE tr.stage IN (...)` clause tracks 7 stages but not `pixel_redaction`. Audit entries `pixel_redaction.triggered` and `pixel_redaction.complete` exist (from `pixel_redaction.go`) but won't be picked up.

**Change needed (1 location):**
- **Line ~58-61** — Add `'pixel_redaction'` to the SQL IN list

### Gap 7: Compliance Report Missing Pixel Redaction Metrics

**File:** `api/handler/compliance.go`

The compliance report has a `PhiDetect` section tracking scan/flag rates but no pixel redaction metrics.

**Changes needed (4 locations):**
1. **Line ~32-36** — Add `PixelRedactionTotal`, `PixelRedacted`, `PixelRedactionFailed` fields to `compliancePhiDetect` struct
2. **Line ~108-119** — Extend the JSON variant SQL query to count pixel redaction statuses; extend `.Scan()` call
3. **Line ~225-235** — Extend the CSV variant SQL query identically
4. **Line ~302-305** — Add 3 CSV rows for `pixel_redaction_total`, `pixel_redacted`, `pixel_redaction_failed`

---

## Findings: MCP Server

### Gap 8: No `trigger_pixel_redaction` Write Tool

**Files:** `mcp-server/src/schemas.ts`, `mcp-server/src/server.ts`

All other pipeline triggers exist as MCP write tools (trigger_classification, trigger_phi_scan, trigger_protocol_check, trigger_qc_check, trigger_bids_convert, trigger_export, trigger_deface) but `trigger_pixel_redaction` is missing. AI agents cannot trigger pixel redaction.

**Changes needed (4 locations):**
1. **`schemas.ts` ~line 1222** — Add `"trigger_pixel_redaction"` to `writeToolNames` array
2. **`server.ts` tools array ~line 490** — Add tool definition with `writeInputSchema`
3. **`server.ts` dispatch ~line 4220** — Add `if (name === "trigger_pixel_redaction")` routing
4. **`server.ts` handler** — Add `handleTriggerPixelRedaction()` function following the `handleTriggerPhiScan` pattern:
   - Guard on `mcpMode === "operator"` and `enableWriteTools`
   - Lookup study by UID
   - Check `pixel_redaction_required === true` and status is `pending` or `failed`
   - POST to `/api/studies/{uid}/pixel-redaction`
   - Return `formatSuccess`

### Gap 9: `get_study_detail` Missing Pixel Redaction Fields

The `get_study_detail` MCP read tool returns the study record from the API. Since the API already includes `pixel_redaction_required` and `pixel_redaction_status` in the study JSON response (they're in the model), this likely already works. **Verify only — may not need changes.**

---

## Findings: Test Coverage

### Gap 10: No Go Tests for Pixel Redaction

No test files exist for the pixel redaction handler, model functions, or pipeline integration.

**Tests to add:**

**`api/handler/pixel_redaction_test.go`** (NEW):
- `TestTriggerPixelRedaction_MissingUID` — 400
- `TestTriggerPixelRedaction_NotFound` — 404
- `TestTriggerPixelRedaction_NotRequired` — 400
- `TestTriggerPixelRedaction_InFlight` — 409
- `TestTriggerPixelRedaction_Success` — 202

**`api/model/study_pixel_redaction_test.go`** (NEW or add to existing study_test.go):
- `TestSetPixelRedactionRequired` — sets flag + pending status
- `TestUpdatePixelRedactionStatus` — status transitions
- `TestClaimPixelRedaction` — atomic claim (pending → redacting), second claim fails

**Routing test** (add to existing routing_test.go):
- `TestRequirePixelRedactionAction` — validate `require_pixel_redaction` is accepted

---

## Findings: Minor Issues

### Gap 11: pydicom Deprecation Warnings (26 warnings)

**File:** `midi-b/tests/conftest.py` (line ~89-91)

The synthetic DICOM fixture uses `write_like_original`, `is_little_endian`, and `is_implicit_VR` which are deprecated in pydicom v3 and will be removed in v4. Currently non-blocking (tests pass) but should be cleaned up.

**Fix:** Replace with:
```python
from pydicom.uid import ExplicitVRLittleEndian
ds.file_meta.TransferSyntaxUID = ExplicitVRLittleEndian
ds.save_as(str(filepath), enforce_file_format=True)
```

### Gap 12: Pipeline Output DICOM Conformance Risk

**File:** `midi-b/midi_b/pipeline.py` (line ~117)

`ds.save_as(str(out_path), write_like_original=False)` may change the transfer syntax for compressed DICOM files (JPEG, JPEG2000). The MIDI-B benchmark penalizes non-compliant output. Should use `write_like_original=True` to preserve the original encoding, or explicitly set `enforce_file_format=True` with the original TransferSyntaxUID.

**Fix:** Change to `ds.save_as(str(out_path))` (defaults to preserving original encoding).

### Gap 13: CLI Missing `--workers` Option

**Files:** `midi-b/midi_b/cli.py`, `midi-b/midi_b/pipeline.py`

The plan specified `--workers N` (default: 4) for parallel file processing. Not implemented. This is a performance optimization only — it does not affect de-identification correctness or MIDI-B scoring.

**Priority:** Low. Defer unless processing speed becomes a bottleneck with the full 29,660-instance test set.

---

## Implementation Checklist

Order of operations, grouped by feature branch. Each group can be a single PR.

### Branch 1: `feature/midi-b-pixel-redact-cli` (Critical)

- [ ] 1. Copy `pixel_utils.py` utilities into `midi-b/midi_b/deid/pixel_utils.py` (`dicom_to_pil`, `_apply_windowing`, `_to_uint8`)
- [ ] 2. Copy `pixel_redact.py` into `midi-b/midi_b/deid/pixel_redact.py` (`redact_dicom_pixels`)
- [ ] 3. Create `midi-b/midi_b/deid/pixel_detect.py` — local Tesseract OCR detection (simplified from `tesseract_backend.py`)
- [ ] 4. Add `RemoteTextScrubBackend` to `midi-b/midi_b/deid/text_scrub.py` — calls `POST {phi_service_url}/scrub-text`
- [ ] 5. Update `midi-b/midi_b/deid/engine.py` — accept optional scrub backend parameter
- [ ] 6. Update `midi-b/midi_b/pipeline.py` — add pixel redaction step; accept `phi_service_url` and `pixel_redact` options; instantiate remote text scrub backend when URL provided
- [ ] 7. Update `midi-b/midi_b/cli.py` — add `--pixel-redact / --no-pixel-redact` and `--phi-service-url` options
- [ ] 8. Update `midi-b/requirements.txt` — add `Pillow>=10.0` and `pytesseract>=0.3` as optional dependencies
- [ ] 9. Create `midi-b/tests/test_pixel_redact.py` — test redaction with synthetic DICOM pixel data
- [ ] 10. Create `midi-b/tests/test_pixel_detect.py` — test OCR detection with mocked Tesseract
- [ ] 11. Update `midi-b/tests/test_text_scrub.py` — add tests for `RemoteTextScrubBackend` (mocked HTTP)
- [ ] 12. Update `midi-b/tests/test_pipeline.py` — add test for pixel redaction in pipeline
- [ ] 13. Fix pydicom deprecation warnings in `midi-b/tests/conftest.py` (Gap 11)
- [ ] 14. Fix `write_like_original=False` in `midi-b/midi_b/pipeline.py` (Gap 12)
- [ ] 15. Run `pytest -v` — all tests pass
- [ ] 16. Run `make lint` — clean

### Branch 2: `feature/pixel-redaction-supporting` (Go API + MCP)

- [ ] 17. Update `api/handler/pipeline_funnel.go` — add `pixel_redacted` stage (Gap 3)
- [ ] 18. Update `api/handler/study_diagnostics.go` — add `evaluateRequiredStage` + `"redacting"` to inProgress (Gap 4)
- [ ] 19. Update `api/handler/study_csv.go` — add `pixel_redaction_status` column (Gap 5)
- [ ] 20. Update `api/handler/processing_times_query.go` — add `'pixel_redaction'` to stage list (Gap 6)
- [ ] 21. Update `api/handler/compliance.go` — add pixel redaction metrics to `compliancePhiDetect` (Gap 7)
- [ ] 22. Update `mcp-server/src/schemas.ts` — add `"trigger_pixel_redaction"` to `writeToolNames` (Gap 8)
- [ ] 23. Update `mcp-server/src/server.ts` — add tool definition, dispatch routing, and handler function (Gap 8)
- [ ] 24. Verify `get_study_detail` already returns pixel redaction fields (Gap 9)
- [ ] 25. Run `cd api && go build ./... && go vet ./...` — clean
- [ ] 26. Run `cd mcp-server && npm run build` — clean
- [ ] 27. Run `make lint` — clean

### Branch 3: `feature/pixel-redaction-tests` (Go Tests)

- [ ] 28. Create `api/handler/pixel_redaction_test.go` — 5 handler tests (Gap 10)
- [ ] 29. Add pixel redaction model tests to `api/model/` — 3 model function tests (Gap 10)
- [ ] 30. Add `require_pixel_redaction` to routing test assertions (Gap 10)
- [ ] 31. Run `cd api && go test -v -count=1 ./...` — all pass
- [ ] 32. Run CI checks: `make lint`

### Final Verification

- [ ] 33. All 3 PRs merged to `develop`
- [ ] 34. CI green on `develop` (all jobs: go, go-test, python, python-test, frontend, docker, infra-guard)
- [ ] 35. Verify `midi-b` CLI end-to-end: `python -m midi_b.cli --input-dir <test> --output-dir <out> --pixel-redact --dry-run`
- [ ] 36. Verify Go API pixel redaction end-to-end: routing rule → auto-dispatch → completion

---

## Risk Assessment

| Gap | Severity | MIDI-B Score Impact | Effort |
|---|---|---|---|
| Gap 1: Pixel redaction in CLI | Critical | 2/10 scored actions unreachable | Medium (copy existing code) |
| Gap 2: LLM text scrubbing in CLI | Important | Higher accuracy on hardest action | Low (HTTP client) |
| Gap 3-7: Go API supporting integrations | Medium | No benchmark impact; operational visibility | Low (pattern replication) |
| Gap 8: MCP trigger tool | Medium | No benchmark impact; AI ops capability | Low |
| Gap 10: Go tests | Medium | No benchmark impact; regression safety | Medium |
| Gap 11-12: pydicom warnings + conformance | Low-Medium | Gap 12 could cause DICOM conformance failures | Low (one-line fixes) |
| Gap 13: CLI `--workers` | Low | Performance only | Defer |

**Recommendation:** Branch 1 (CLI pixel redaction + LLM text) is the highest-priority work for MIDI-B benchmark scoring. Branches 2 and 3 are operational quality items that can be parallelized.
