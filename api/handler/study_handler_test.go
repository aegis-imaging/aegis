package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStudies_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
		Limit   int           `json:"limit"`
		Offset  int           `json:"offset"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Studies, 1)
}

func TestListStudies_WithStatusFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies?status=approved", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	json.NewDecoder(rr.Body).Decode(&result)
	assert.Equal(t, 0, result.Total, "no approved studies yet")
}

func TestApproveStudy_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/approve", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ApproveStudy(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result map[string]string
	json.NewDecoder(rr.Body).Decode(&result)
	assert.Equal(t, "approved", result["status"])
}

func TestApproveStudy_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/studies/nonexistent/approve", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.ApproveStudy(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestApproveStudy_AlreadyApproved(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	model.UpdateStudyStatus(nil, db, study.ID, "approved")

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/approve", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ApproveStudy(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRejectStudy_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reject", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.RejectStudy(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result map[string]string
	json.NewDecoder(rr.Body).Decode(&result)
	assert.Equal(t, "rejected", result["status"])
}
