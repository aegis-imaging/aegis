# Trigger Handler Implementation Validation

**Status**: ✅ Complete and validated

## Summary

All async processing trigger handlers have been implemented following a consistent pattern. The `TriggerClassification` handler is correctly implemented and ready for use.

## Implementation Pattern

All trigger handlers (`TriggerDeface`, `TriggerPhiScan`, `TriggerQcCheck`, `TriggerBidsConversion`, `TriggerClassification`, `TriggerProtocolCheck`, `TriggerExport`) follow the same pattern:

### 1. Precondition Validation
- Extract study UID from request path parameter
- Fetch study by UID (404 if not found)
- Verify study has the required flag enabled (e.g., `ClassificationRequired=true`)
- Verify processing is not already in progress (409 Conflict if yes)

### 2. State Transition
- Update processing status to "in-progress" state (e.g., `classification_status = "classifying"`)
- Create audit entry recording the trigger action with actor email and client IP

### 3. Service Availability Check
- Check if the service URL environment variable is configured
- If not configured: log warning, return 202 Accepted with queued message (will be processed later)
- If configured: proceed to async dispatch

### 4. Async Dispatch
- Launch goroutine to execute `run*` function (e.g., `runClassification`)
- Return 202 Accepted immediately

### 5. Background Processing (in goroutine)
- List DICOM files from appropriate store (`raw` or `clean`)
- Call Python service with study UID and file paths
- Handle service response and update study status
- Create completion audit entry with results/errors
- For classification: re-evaluate routing rules if metadata was updated
- Advance pipeline to dispatch next eligible services

## Implemented Triggers

| Handler | Route | Service URL | Status Field | Precondition | Notes |
|---------|-------|-------------|--------------|--------------|-------|
| `TriggerDeface` | `POST /api/deface/{studyUID}` | `DEFACING_SERVICE_URL` | `status` | `DefacingRequired=true` | Updates `dicom_store` from `raw` to `clean` |
| `TriggerPhiScan` | `POST /api/studies/{studyUID}/phi-scan` | `PHI_DETECTION_SERVICE_URL` | `phi_scan_status` | `PhiScanRequired=true` | Detects burned-in PHI via OCR |
| `TriggerQcCheck` | `POST /api/studies/{studyUID}/qc-check` | `QC_SERVICE_URL` | `qc_status` | `QcRequired=true` | Validates image quality |
| `TriggerBidsConversion` | `POST /api/studies/{studyUID}/bids-convert` | `BIDS_SERVICE_URL` | `bids_status` | `BidsRequired=true` | Converts to NIfTI/BIDS format |
| `TriggerClassification` | `POST /api/studies/{studyUID}/classify` | `CLASSIFICATION_SERVICE_URL` | `classification_status` | `ClassificationRequired=true` | Fills modality/body_part, re-evaluates routing |
| `TriggerProtocolCheck` | `POST /api/studies/{studyUID}/protocol-check` | `PROTOCOL_SERVICE_URL` | `protocol_status` | `ProtocolRequired=true` | Validates MRI acquisition parameters |
| `TriggerExport` | `POST /api/studies/{studyUID}/trigger-export` | N/A (forward destinations) | `export_status` | `ExportRequired=true` | Forwards to DICOMweb/DIMSE destinations |

## Route Registration (main.go)

All trigger routes are registered with the `adminOnly` middleware:

```go
// Async processing triggers — admin-initiated pipeline actions.
mux.HandleFunc("POST /api/deface/{studyUID}", adminOnly(srv.TriggerDeface))
mux.HandleFunc("POST /api/studies/{studyUID}/phi-scan", adminOnly(srv.TriggerPhiScan))
mux.HandleFunc("POST /api/studies/{studyUID}/qc-check", adminOnly(srv.TriggerQcCheck))
mux.HandleFunc("POST /api/studies/{studyUID}/bids-convert", adminOnly(srv.TriggerBidsConversion))
mux.HandleFunc("POST /api/studies/{studyUID}/classify", adminOnly(srv.TriggerClassification))
mux.HandleFunc("POST /api/studies/{studyUID}/protocol-check", adminOnly(srv.TriggerProtocolCheck))
mux.HandleFunc("POST /api/studies/{studyUID}/trigger-export", adminOnly(srv.TriggerExport))
```

## TriggerClassification Specifics

### Unique Behavior
- On successful classification, re-evaluates routing rules to allow metadata-dependent rules to fire
- Updates study `modality` and `body_part` if classification confidence >= 0.5
- Only updates currently empty fields (doesn't overwrite existing values)

### Response Pattern

**Success (202 Accepted)** - Service configured:
```json
{
  "status": "classifying",
  "message": "Classification pipeline started"
}
```

**Queued (202 Accepted)** - Service not configured:
```json
{
  "status": "classifying",
  "message": "Study queued for classification — set CLASSIFICATION_SERVICE_URL to enable processing"
}
```

**Error (400 Bad Request)**:
```json
{
  "error": "study does not require classification"
}
```

**Error (409 Conflict)**:
```json
{
  "error": "classification already in progress"
}
```

## Audit Trail

Classification trigger creates two audit entries:

1. **Trigger event** (`classification.triggered`):
   - Actor email and IP captured
   - Study UID included

2. **Completion event** (`classification.complete` or `classification.failed`):
   - Tool used (heuristic/google_vision/aws_rekognition)
   - Results (modality, body_part, confidence)
   - Metadata update flag
   - Processing duration

## Error Handling

- **Study not found**: 404 Not Found
- **Study doesn't require classification**: 400 Bad Request
- **Classification in progress**: 409 Conflict
- **Service unavailable**: Queued as pending, can be processed later
- **Service error**: 202 Accepted returned to caller, status set to "failed"
- **Network error**: Logged, status set to "failed"

## Security

- All trigger endpoints require `adminOnly` middleware (role-based access control)
- Viewers cannot trigger processing (403 Forbidden)
- Audit trail captures all trigger attempts with actor identity

## Testing

All tests pass (11 packages, ~120 tests):
- ✅ Config tests
- ✅ Digest scheduler tests
- ✅ Email template tests
- ✅ Handler tests
- ✅ Importer tests
- ✅ Middleware/auth tests
- ✅ Model/database tests
- ✅ Routing engine tests
- ✅ Storage tests

## Integration with Pipeline

The trigger handlers integrate with the automated processing pipeline (`AdvancePipeline`):

1. Classification runs first (Phase 0) — re-evaluates routing
2. Other services dispatched based on updated requirements
3. Each handler advances the pipeline after completion
4. Dependency graph ensures correct processing order

## Configuration

To enable classification processing, set:
```bash
CLASSIFICATION_SERVICE_URL=http://classification-service:8080  # or on-prem URL
```

Without this, classification studies are queued but not processed until either:
- The environment variable is set and service restarted
- Admin manually re-triggers via dashboard

## Precondition Rationale

The `ClassificationRequired=true` precondition ensures:
- Classification is only triggered when routing rules or admin intent requires it
- Manual triggers cannot bypass system requirements
- Prevents unnecessary processing of already-classified studies
- Aligns with other trigger handlers' patterns

Studies reach `ClassificationRequired=true` through:
1. Explicit `require_classification` routing rule action
2. Auto-pipeline dispatch when configured
3. Manual admin trigger via dashboard (requires precondition)

## Completion Criteria

✅ TriggerClassification fully implemented
✅ Follows consistent pattern with other triggers
✅ Route registered with admin-only middleware
✅ Audit trail captures all actions
✅ Error handling complete
✅ All tests passing
✅ Documentation complete
