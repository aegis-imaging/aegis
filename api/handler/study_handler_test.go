package handler_test

import (
	"context"
	"database/sql"
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

func TestListStudies_WithLabelFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	// Label s1 only
	_, err := model.AddStudyLabel(context.Background(), db, s1.ID, "cohort-A", "admin@test.local")
	require.NoError(t, err)

	// Filter by label — should return only s1
	req := httptest.NewRequest("GET", "/api/studies?label=cohort-A", nil)
	rr := httptest.NewRecorder()
	srv.ListStudies(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var result struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, s1.ID, result.Studies[0].ID)

	// Filter by label that doesn't match either study
	req2 := httptest.NewRequest("GET", "/api/studies?label=cohort-Z", nil)
	rr2 := httptest.NewRecorder()
	srv.ListStudies(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code)

	var result2 struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&result2))
	assert.Equal(t, 0, result2.Total)

	// No label filter returns both
	req3 := httptest.NewRequest("GET", "/api/studies", nil)
	rr3 := httptest.NewRecorder()
	srv.ListStudies(rr3, req3)
	var result3 struct{ Total int `json:"total"` }
	json.NewDecoder(rr3.Body).Decode(&result3)
	assert.Equal(t, 2, result3.Total)

	_ = s2
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

func TestDeleteStudy_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("DELETE", "/api/studies/"+study.ID, nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.DeleteStudy(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Study should no longer exist.
	_, err := model.GetStudyByID(context.Background(), db, study.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteStudy_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("DELETE", "/api/studies/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.DeleteStudy(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestRecordStudyView_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/viewed", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.RecordStudyView(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var result map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "ok", result["status"])

	// Verify audit entry was created.
	entries, err := model.ListAuditEntriesForStudy(context.Background(), db, study.ID)
	require.NoError(t, err)
	found := false
	for _, e := range entries {
		if e.Action == "study.viewed" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected study.viewed audit entry")
}
