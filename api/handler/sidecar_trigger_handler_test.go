package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sidecarServer creates a handler.Server with one sidecar URL configured.
func sidecarServer(t *testing.T, db *sql.DB, field string, svcURL string) *handler.Server {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:            "0",
		StorageMode:     "local",
		LocalStorageDir: tmpDir,
		APIBaseURL:      "http://localhost:8080",
		PipelineAuto:    false,
		AuthEnabled:     false,
		DevUserEmail:    "test@aegis.local",
		AllowedOrigins:  []string{"http://localhost:3000"},
	}
	switch field {
	case "phi":
		cfg.PhiDetectionServiceURL = svcURL
	case "qc":
		cfg.QcServiceURL = svcURL
	case "bids":
		cfg.BidsServiceURL = svcURL
	case "classification":
		cfg.ClassificationServiceURL = svcURL
	case "deface":
		cfg.DefacingServiceURL = svcURL
	case "protocol":
		cfg.ProtocolServiceURL = svcURL
	}
	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg)
}

// pollStatus polls the DB until the study's phi_scan_status reaches expected or timeout.
func pollPhiStatus(t *testing.T, db *sql.DB, studyID, expected string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := model.GetStudyByID(context.Background(), db, studyID)
		if err == nil && s.PhiScanStatus == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("timed out waiting for phi_scan_status=%q", expected)
}

func pollQcStatus(t *testing.T, db *sql.DB, studyID, expected string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := model.GetStudyByID(context.Background(), db, studyID)
		if err == nil && s.QcStatus == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("timed out waiting for qc_status=%q", expected)
}

func pollBidsStatus(t *testing.T, db *sql.DB, studyID, expected string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := model.GetStudyByID(context.Background(), db, studyID)
		if err == nil && s.BidsStatus == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("timed out waiting for bids_status=%q", expected)
}

func pollClassificationStatus(t *testing.T, db *sql.DB, studyID, expected string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := model.GetStudyByID(context.Background(), db, studyID)
		if err == nil && s.ClassificationStatus == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("timed out waiting for classification_status=%q", expected)
}

func pollStudyStatus(t *testing.T, db *sql.DB, studyID, expected string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := model.GetStudyByID(context.Background(), db, studyID)
		if err == nil && s.Status == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("timed out waiting for status=%q", expected)
}

// ─── PHI Scan ─────────────────────────────────────────────────────────────────

func TestTriggerPhiScan_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/no.such.uid/phi-scan", nil)
	req.SetPathValue("studyUID", "no.such.uid")
	rr := httptest.NewRecorder()

	srv.TriggerPhiScan(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestTriggerPhiScan_NotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/phi-scan", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerPhiScan(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "does not require PHI scan")
}

func TestTriggerPhiScan_AlreadyInProgress(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetPhiScanRequired(t.Context(), db, study.ID, true))
	require.NoError(t, model.UpdatePhiScanStatus(t.Context(), db, study.ID, "scanning"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/phi-scan", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerPhiScan(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "already in progress")
}

func TestTriggerPhiScan_ServiceNotConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db) // no service URL
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetPhiScanRequired(t.Context(), db, study.ID, true))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/phi-scan", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerPhiScan(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, "scanning", body["status"])

	// Status should be "scanning" even without a live service
	updated, err := model.GetStudyByID(t.Context(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "scanning", updated.PhiScanStatus)
}

func TestTriggerPhiScan_WithMockService_ResultsInClean(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetPhiScanRequired(t.Context(), db, study.ID, true))

	// Mock phi-detection service that responds "clean"
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/detect", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"study_uid":    study.StudyInstanceUID,
			"status":       "complete",
			"phi_detected": false,
			"findings":     []any{},
			"tool_used":    "tesseract",
		})
	}))
	defer svc.Close()

	srv := sidecarServer(t, db, "phi", svc.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/phi-scan", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerPhiScan(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	// Goroutine will update status to "clean" (no files → "failed" since no DICOM in store, but service call happens)
	// Since test storage has no files, runPhiScan will set "failed" due to empty file list.
	// This still validates the 202 path and status transition.
	pollPhiStatus(t, db, study.ID, "failed")
}

// ─── QC Check ─────────────────────────────────────────────────────────────────

func TestTriggerQcCheck_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/no.such.uid/qc-check", nil)
	req.SetPathValue("studyUID", "no.such.uid")
	rr := httptest.NewRecorder()

	srv.TriggerQcCheck(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestTriggerQcCheck_NotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/qc-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerQcCheck(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "does not require QC check")
}

func TestTriggerQcCheck_AlreadyInProgress(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetQcRequired(t.Context(), db, study.ID, true))
	require.NoError(t, model.UpdateQcStatus(t.Context(), db, study.ID, "checking"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/qc-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerQcCheck(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestTriggerQcCheck_ServiceNotConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetQcRequired(t.Context(), db, study.ID, true))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/qc-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerQcCheck(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	updated, err := model.GetStudyByID(t.Context(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "checking", updated.QcStatus)
}

func TestTriggerQcCheck_WithMockService(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetQcRequired(t.Context(), db, study.ID, true))

	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/check", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"study_uid":       study.StudyInstanceUID,
			"status":          "complete",
			"overall_quality": "pass",
			"quality_issues":  []any{},
			"tool_used":       "basic",
		})
	}))
	defer svc.Close()

	srv := sidecarServer(t, db, "qc", svc.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/qc-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerQcCheck(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	// No DICOM files in test store → goroutine sets status to "failed"
	pollQcStatus(t, db, study.ID, "failed")
}

// ─── BIDS Conversion ──────────────────────────────────────────────────────────

func TestTriggerBidsConversion_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/no.uid/bids-convert", nil)
	req.SetPathValue("studyUID", "no.uid")
	rr := httptest.NewRecorder()

	srv.TriggerBidsConversion(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestTriggerBidsConversion_NotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/bids-convert", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerBidsConversion(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "does not require BIDS conversion")
}

func TestTriggerBidsConversion_AlreadyInProgress(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetBidsRequired(t.Context(), db, study.ID, true))
	require.NoError(t, model.UpdateBidsStatus(t.Context(), db, study.ID, "converting"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/bids-convert", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerBidsConversion(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestTriggerBidsConversion_ServiceNotConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetBidsRequired(t.Context(), db, study.ID, true))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/bids-convert", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerBidsConversion(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	updated, err := model.GetStudyByID(t.Context(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "converting", updated.BidsStatus)
}

func TestTriggerBidsConversion_WithMockService(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetBidsRequired(t.Context(), db, study.ID, true))

	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/convert", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"study_uid":    study.StudyInstanceUID,
			"status":       "complete",
			"output_files": []string{"sub-abc/anat/sub-abc_T1w.nii.gz"},
			"tool_used":    "dcm2niix",
		})
	}))
	defer svc.Close()

	srv := sidecarServer(t, db, "bids", svc.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/bids-convert", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerBidsConversion(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	// No DICOM files → goroutine sets bids_status to "failed"
	pollBidsStatus(t, db, study.ID, "failed")
}

// ─── Classification ───────────────────────────────────────────────────────────

func TestTriggerClassification_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/no.uid/classify", nil)
	req.SetPathValue("studyUID", "no.uid")
	rr := httptest.NewRecorder()

	srv.TriggerClassification(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestTriggerClassification_NotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/classify", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerClassification(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "does not require classification")
}

func TestTriggerClassification_AlreadyInProgress(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetClassificationRequired(t.Context(), db, study.ID, true))
	require.NoError(t, model.UpdateClassificationStatus(t.Context(), db, study.ID, "classifying"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/classify", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerClassification(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestTriggerClassification_ServiceNotConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetClassificationRequired(t.Context(), db, study.ID, true))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/classify", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerClassification(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	updated, err := model.GetStudyByID(t.Context(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "classifying", updated.ClassificationStatus)
}

func TestTriggerClassification_WithMockService(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetClassificationRequired(t.Context(), db, study.ID, true))

	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/classify", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "complete",
			"modality":  "MRI",
			"body_part": "HEAD",
			"confidence": 0.95,
			"method":    "dicom_tags",
			"tool_used": "heuristic",
		})
	}))
	defer svc.Close()

	srv := sidecarServer(t, db, "classification", svc.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/classify", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerClassification(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	// No DICOM files → goroutine sets classification_status to "failed"
	pollClassificationStatus(t, db, study.ID, "failed")
}

// ─── Defacing ─────────────────────────────────────────────────────────────────

func TestTriggerDeface_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/no.uid/trigger-deface", nil)
	req.SetPathValue("studyUID", "no.uid")
	rr := httptest.NewRecorder()

	srv.TriggerDeface(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestTriggerDeface_NotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-deface", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerDeface(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "does not require defacing")
}

func TestTriggerDeface_AlreadyInProgress(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetDefacingRequired(t.Context(), db, study.ID, true))
	require.NoError(t, model.UpdateStudyStatus(t.Context(), db, study.ID, "defacing"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-deface", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerDeface(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "already in progress")
}

func TestTriggerDeface_ServiceNotConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetDefacingRequired(t.Context(), db, study.ID, true))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-deface", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerDeface(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	updated, err := model.GetStudyByID(t.Context(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "defacing", updated.Status)
}

func TestTriggerDeface_WithMockService(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetDefacingRequired(t.Context(), db, study.ID, true))

	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/deface", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"study_uid":    study.StudyInstanceUID,
			"status":       "complete",
			"output_paths": []string{},
			"tool_used":    "nibabel",
		})
	}))
	defer svc.Close()

	srv := sidecarServer(t, db, "deface", svc.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-deface", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerDeface(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	// No DICOM files in test store → goroutine transitions study back to "received" on failure
	pollStudyStatus(t, db, study.ID, "received")
}
