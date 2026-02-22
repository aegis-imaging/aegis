package handler_test

import (
	"context"
	"database/sql"
	"fmt"
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

// pipelineServer creates a Server with PipelineAuto=true and optional sidecar URLs.
func pipelineServer(t *testing.T, db *sql.DB, classifyURL, phiURL, qcURL, bidsURL string) *handler.Server {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Port:                     "0",
		StorageMode:              "local",
		LocalStorageDir:          tmpDir,
		APIBaseURL:               "http://localhost:8080",
		PipelineAuto:             true, // enable auto dispatch
		AuthEnabled:              false,
		DevUserEmail:             "test@aegis.local",
		AllowedOrigins:           []string{"http://localhost:3000"},
		ClassificationServiceURL: classifyURL,
		PhiDetectionServiceURL:   phiURL,
		QcServiceURL:             qcURL,
		BidsServiceURL:           bidsURL,
	}
	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg)
}

// pollClassifyStatus waits for classification_status to reach expected or times out.
func pollClassifyStatus(t *testing.T, db *sql.DB, studyID, expected string) {
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

// uniquePipelineUID returns a unique study UID for pipeline tests.
func uniquePipelineUID() string {
	return fmt.Sprintf("1.2.3.pipeline-test.%d", uniqueInt())
}

// ─── AdvancePipeline: no-op cases ────────────────────────────────────────────

func TestAdvancePipeline_NoOpWhenDisabled(t *testing.T) {
	db := testutil.TestDB(t)
	// PipelineAuto=false (default TestServer)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := &model.Study{
		ProjectID:              proj.ID,
		StudyInstanceUID:       uniquePipelineUID(),
		Modality:               "MR",
		InstanceCount:          1,
		Status:                 "received",
		DicomStore:             "raw",
		Source:                 "external",
		ClassificationRequired: true,
		ClassificationStatus:   "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	// Status should be unchanged — pipeline is disabled.
	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", refreshed.ClassificationStatus)
}

func TestAdvancePipeline_NoOpForApprovedStudy(t *testing.T) {
	db := testutil.TestDB(t)
	srv := pipelineServer(t, db, "http://fake-classify:9999", "", "", "")
	proj := testutil.SeedProject(t, db)

	study := &model.Study{
		ProjectID:              proj.ID,
		StudyInstanceUID:       uniquePipelineUID(),
		Modality:               "MR",
		InstanceCount:          1,
		Status:                 "approved",
		DicomStore:             "raw",
		Source:                 "external",
		ClassificationRequired: true,
		ClassificationStatus:   "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	// Approved is a terminal state — no dispatch.
	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", refreshed.ClassificationStatus)
}

func TestAdvancePipeline_NoOpForRejectedStudy(t *testing.T) {
	db := testutil.TestDB(t)
	srv := pipelineServer(t, db, "http://fake-classify:9999", "http://fake-phi:9999", "", "")
	proj := testutil.SeedProject(t, db)

	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: uniquePipelineUID(),
		Modality:         "MR",
		InstanceCount:    1,
		Status:           "rejected",
		DicomStore:       "raw",
		Source:           "external",
		PhiScanRequired:  true,
		PhiScanStatus:    "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	// Rejected is a terminal state — no dispatch.
	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", refreshed.PhiScanStatus)
}

// ─── AdvancePipeline: Phase 0 (Classification) ───────────────────────────────

func TestAdvancePipeline_ClassificationPendingNoURL_BlocksPhase1(t *testing.T) {
	db := testutil.TestDB(t)
	// No classification URL — Phase 0 logs warning and returns early.
	// Phi scan should NOT be dispatched even if it's pending and URL is set.
	srv := pipelineServer(t, db, "", "http://fake-phi:9999", "", "")
	proj := testutil.SeedProject(t, db)

	study := &model.Study{
		ProjectID:              proj.ID,
		StudyInstanceUID:       uniquePipelineUID(),
		Modality:               "MR",
		InstanceCount:          1,
		Status:                 "received",
		DicomStore:             "raw",
		Source:                 "external",
		ClassificationRequired: true,
		ClassificationStatus:   "pending",
		PhiScanRequired:        true,
		PhiScanStatus:          "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	// Phase 0 returned early — classification still pending, phi scan untouched.
	assert.Equal(t, "pending", refreshed.ClassificationStatus)
	assert.Equal(t, "pending", refreshed.PhiScanStatus)
}

func TestAdvancePipeline_ClassificationClassifying_BlocksPhase1(t *testing.T) {
	db := testutil.TestDB(t)
	srv := pipelineServer(t, db, "http://fake-classify:9999", "http://fake-phi:9999", "", "")
	proj := testutil.SeedProject(t, db)

	study := &model.Study{
		ProjectID:              proj.ID,
		StudyInstanceUID:       uniquePipelineUID(),
		Modality:               "MR",
		InstanceCount:          1,
		Status:                 "received",
		DicomStore:             "raw",
		Source:                 "external",
		ClassificationRequired: true,
		ClassificationStatus:   "classifying", // already in progress
		PhiScanRequired:        true,
		PhiScanStatus:          "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	// Still classifying — Phase 1 blocked.
	assert.Equal(t, "classifying", refreshed.ClassificationStatus)
	assert.Equal(t, "pending", refreshed.PhiScanStatus)
}

func TestAdvancePipeline_ClassificationPendingWithURL_Claims(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)

	// Mock classification service.
	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Return a minimal classification response.
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"study_uid":"test","status":"complete","modality":"MR","body_part":"HEAD","confidence":0.95,"tool_used":"heuristic"}`)
	}))
	defer svc.Close()

	srv := pipelineServer(t, db, svc.URL, "", "", "")

	study := &model.Study{
		ProjectID:              proj.ID,
		StudyInstanceUID:       uniquePipelineUID(),
		Modality:               "MR",
		InstanceCount:          1,
		Status:                 "received",
		DicomStore:             "raw",
		Source:                 "external",
		ClassificationRequired: true,
		ClassificationStatus:   "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	// AdvancePipeline claimed (pending → classifying) then launched goroutine.
	// No DICOM files on disk → goroutine sets "failed". Poll for the terminal state.
	pollClassifyStatus(t, db, study.ID, "failed")
}

// ─── AdvancePipeline: Phase 1 (no classification required) ───────────────────

func TestAdvancePipeline_Phase1_PhiScanNoURL_NoDispatch(t *testing.T) {
	db := testutil.TestDB(t)
	// No phi detection URL — phi scan should stay pending.
	srv := pipelineServer(t, db, "", "", "", "")
	proj := testutil.SeedProject(t, db)

	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: uniquePipelineUID(),
		Modality:         "MR",
		InstanceCount:    1,
		Status:           "received",
		DicomStore:       "raw",
		Source:           "external",
		PhiScanRequired:  true,
		PhiScanStatus:    "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", refreshed.PhiScanStatus)
}

// ─── AdvancePipeline: Phase 2 (defacing gate) ────────────────────────────────

func TestAdvancePipeline_Phase2_BlockedByDefacing(t *testing.T) {
	db := testutil.TestDB(t)
	srv := pipelineServer(t, db, "", "", "http://fake-qc:9999", "http://fake-bids:9999")
	proj := testutil.SeedProject(t, db)

	// Defacing required and not yet complete (status=defacing, not defaced).
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: uniquePipelineUID(),
		Modality:         "MR",
		InstanceCount:    1,
		Status:           "defacing",
		DefacingRequired: true,
		DicomStore:       "raw",
		Source:           "external",
		QcRequired:       true,
		QcStatus:         "pending",
		BidsRequired:     true,
		BidsStatus:       "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	srv.AdvancePipeline(context.Background(), study.ID)

	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	// Phase 2 blocked — QC and BIDS should not be dispatched.
	assert.Equal(t, "pending", refreshed.QcStatus)
	assert.Equal(t, "pending", refreshed.BidsStatus)
}

func TestAdvancePipeline_Phase2_UnblockedAfterDefaced(t *testing.T) {
	db := testutil.TestDB(t)
	// QC URL not set → QC dispatch no-ops, but the blocking gate should pass.
	srv := pipelineServer(t, db, "", "", "", "")
	proj := testutil.SeedProject(t, db)

	// Defacing required and complete (status=defaced).
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: uniquePipelineUID(),
		Modality:         "MR",
		InstanceCount:    1,
		Status:           "defaced",
		DefacingRequired: true,
		DicomStore:       "clean",
		Source:           "external",
		QcRequired:       true,
		QcStatus:         "pending",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	// This should not panic or error — it reaches Phase 2, then no-ops (no QC URL).
	srv.AdvancePipeline(context.Background(), study.ID)

	refreshed, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	// No QC URL → QC stays pending, but the gate was passed (no block).
	assert.Equal(t, "pending", refreshed.QcStatus)
}
