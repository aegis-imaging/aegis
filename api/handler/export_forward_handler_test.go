package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── TriggerExport ───────────────────────────────────────────────────────────

func TestTriggerExport_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/1.2.3.no-such-uid/trigger-export", nil)
	req.SetPathValue("studyUID", "1.2.3.no-such-uid")
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestTriggerExport_NotApproved(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID) // status = "received"

	require.NoError(t, model.SetExportRequired(context.Background(), db, study.ID, true))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-export", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "only approved studies can be exported")
}

func TestTriggerExport_ExportNotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "approved"))
	// export_required remains false (default)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-export", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "export not required for this study")
}

func TestTriggerExport_AlreadyExported(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "approved"))
	require.NoError(t, model.SetExportRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.UpdateExportStatus(context.Background(), db, study.ID, "exported"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-export", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "export already exported")
}

func TestTriggerExport_AlreadyExporting(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "approved"))
	require.NoError(t, model.SetExportRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.UpdateExportStatus(context.Background(), db, study.ID, "exporting"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-export", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "export already exporting")
}

func TestTriggerExport_Success(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: fmt.Sprintf("1.2.3.trigger-export-success.%d", uniqueInt()),
		Modality:         "MR",
		InstanceCount:    1,
		Status:           "approved",
		ExportRequired:   true,
		ExportStatus:     "pending",
		DicomStore:       "raw",
		Source:           "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-export", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var result map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "exporting", result["status"])
	assert.Equal(t, study.StudyInstanceUID, result["study_uid"])
}

func TestTriggerExport_RetryAfterFailure(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := &model.Study{
		ProjectID:        proj.ID,
		StudyInstanceUID: fmt.Sprintf("1.2.3.trigger-export-retry.%d", uniqueInt()),
		Modality:         "CT",
		InstanceCount:    1,
		Status:           "approved",
		ExportRequired:   true,
		ExportStatus:     "failed",
		DicomStore:       "raw",
		Source:           "external",
	}
	require.NoError(t, model.CreateStudy(context.Background(), db, study))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/trigger-export", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerExport(rr, req)

	// Failed studies can be retried — handler resets to pending then claims.
	assert.Equal(t, http.StatusAccepted, rr.Code)
	var result map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "exporting", result["status"])
}

// ─── AuthMe ──────────────────────────────────────────────────────────────────

func TestAuthMe_NoUser(t *testing.T) {
	db := testutil.TestDB(t)
	srv, _ := dicomTestServer(t, db)

	// Call AuthMe directly without any auth middleware — no user in context.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rr := httptest.NewRecorder()

	srv.AuthMe(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "not authenticated")
}
