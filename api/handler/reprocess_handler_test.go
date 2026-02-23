package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResetPipelineStep_StepNotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// deface is not required on a default study
	body, _ := json.Marshal(map[string]string{"step": "deface"})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reset-pipeline-step", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ResetPipelineStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestResetPipelineStep_UnknownStep(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]string{"step": "invalid_step"})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reset-pipeline-step", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ResetPipelineStep(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestResetPipelineStep_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]string{"step": "qc"})
	req := httptest.NewRequest("POST", "/api/studies/00000000-0000-0000-0000-000000000000/reset-pipeline-step", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ResetPipelineStep(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestResetPipelineStep_QCResetOK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Enable QC on the study directly
	_, err := db.ExecContext(context.Background(),
		`UPDATE studies SET qc_required = true, qc_status = 'pass' WHERE id = $1`, study.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"step": "qc"})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reset-pipeline-step", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ResetPipelineStep(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify status reset to pending
	updated, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", updated.QcStatus)
}

func TestResetPipelineStep_InFlight409(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Simulate QC in-flight
	_, err := db.ExecContext(context.Background(),
		`UPDATE studies SET qc_required = true, qc_status = 'checking' WHERE id = $1`, study.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"step": "qc"})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reset-pipeline-step", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ResetPipelineStep(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}
