package handler_test

import (
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

func TestGetStudyByUID_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/study-uid/"+study.StudyInstanceUID, nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()
	srv.GetStudyByUID(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.Study
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, study.ID, result.ID)
	assert.Equal(t, study.StudyInstanceUID, result.StudyInstanceUID)
}

func TestGetStudyByUID_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/study-uid/9.9.9.notexist", nil)
	req.SetPathValue("studyUID", "9.9.9.notexist")
	rr := httptest.NewRecorder()
	srv.GetStudyByUID(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
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
	model.UpdateStudyStatus(context.Background(), db, study.ID, "approved")

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

func TestGetStudy_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID, nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudy(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result model.Study
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, study.ID, result.ID)
	assert.Equal(t, study.StudyInstanceUID, result.StudyInstanceUID)
}

func TestGetStudy_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetStudy(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestEvaluateRoutingRules_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/routing-rules/evaluate/"+study.ID, nil)
	req.SetPathValue("studyID", study.ID)
	rr := httptest.NewRecorder()
	srv.EvaluateRoutingRules(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Study      model.Study              `json:"study"`
		RoutingLog []model.RoutingLogEntry  `json:"routing_log"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, study.ID, result.Study.ID)
	// No rules seeded — routing log should be empty (nil or zero-length slice both accepted).
	assert.Empty(t, result.RoutingLog)
}

func TestEvaluateRoutingRules_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/routing-rules/evaluate/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("studyID", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.EvaluateRoutingRules(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestEvaluateRoutingRules_AppliesMatchingRule(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Seed a rule that matches any study (no conditions) with auto_approve action.
	testutil.CreateTestRoutingRule(t, db, "catch-all-approve", "auto_approve")

	req := httptest.NewRequest("POST", "/api/routing-rules/evaluate/"+study.ID, nil)
	req.SetPathValue("studyID", study.ID)
	rr := httptest.NewRecorder()
	srv.EvaluateRoutingRules(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result struct {
		Study      model.Study             `json:"study"`
		RoutingLog []model.RoutingLogEntry `json:"routing_log"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.NotEmpty(t, result.RoutingLog, "rule should have produced a log entry")
	assert.Equal(t, "auto_approve", result.RoutingLog[0].Action)

	// Study should now be approved.
	updated, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
}
