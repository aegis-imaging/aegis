package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DemoGenerate calls the synth-service which is not available in tests.
// The handler exits early with 503 when SynthServiceURL is unconfigured.

func TestDemoGenerate_ServiceUnconfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/demo/generate", nil)
	rr := httptest.NewRecorder()
	srv.DemoGenerate(rr, req)

	// SynthServiceURL is not configured in test server → 503.
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestDemoStudyStatus_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/demo/study/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("studyID", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.DemoStudyStatus(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDemoStudyStatus_OK(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/demo/study/"+study.ID, nil)
	req.SetPathValue("studyID", study.ID)
	rr := httptest.NewRecorder()
	srv.DemoStudyStatus(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, study.ID, resp["study_id"])
	assert.Equal(t, study.StudyInstanceUID, resp["study_uid"])
	assert.Equal(t, "received", resp["status"])
	assert.Equal(t, "raw", resp["dicom_store"])
}

func TestDemoStudyStatus_MissingPathValue(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// SetPathValue not called → PathValue("studyID") returns ""
	req := httptest.NewRequest("GET", "/api/demo/study/", nil)
	rr := httptest.NewRecorder()
	srv.DemoStudyStatus(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
