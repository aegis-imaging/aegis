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

// ── GetStudyRoutingLog ────────────────────────────────────────────────────────

func TestGetStudyRoutingLog_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/routing-log", nil)
	req.SetPathValue("studyID", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudyRoutingLog(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result []model.RoutingLogEntry
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 0)
}

func TestGetStudyRoutingLog_ReturnsEntries(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	rule := testutil.CreateTestRoutingRule(t, db, "test-rule", "auto_approve")
	entry := &model.RoutingLogEntry{
		StudyID: study.ID,
		RuleID:  rule.ID,
		Action:  "auto_approve",
		Outcome: "applied",
	}
	require.NoError(t, model.CreateRoutingLogEntry(context.Background(), db, entry))

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/routing-log", nil)
	req.SetPathValue("studyID", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudyRoutingLog(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result []model.RoutingLogEntry
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, study.ID, result[0].StudyID)
	assert.Equal(t, "auto_approve", result[0].Action)
	assert.Equal(t, "applied", result[0].Outcome)
}

func TestGetStudyRoutingLog_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies/00000000-0000-0000-0000-000000000000/routing-log", nil)
	req.SetPathValue("studyID", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.GetStudyRoutingLog(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// ── ListStudyAudit ────────────────────────────────────────────────────────────

func TestListStudyAudit_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/audit", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result []model.AuditEntry
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 0)
}

func TestListStudyAudit_ReturnsEntries(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.CreateAuditEntry(context.Background(), db,
		"study.approved", "admin@test.com", "study", study.ID, "127.0.0.1", nil))
	require.NoError(t, model.CreateAuditEntry(context.Background(), db,
		"study.shared", "admin@test.com", "study", study.ID, "127.0.0.1", nil))

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/audit", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result []model.AuditEntry
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 2)
	actions := []string{result[0].Action, result[1].Action}
	assert.ElementsMatch(t, []string{"study.approved", "study.shared"}, actions)
}

func TestListStudyAudit_OnlyScopeToStudy(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study1 := testutil.CreateTestStudy(t, db, proj.ID)
	study2 := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.CreateAuditEntry(context.Background(), db,
		"study.approved", "admin@test.com", "study", study1.ID, "127.0.0.1", nil))
	require.NoError(t, model.CreateAuditEntry(context.Background(), db,
		"study.rejected", "admin@test.com", "study", study2.ID, "127.0.0.1", nil))

	req := httptest.NewRequest("GET", "/api/studies/"+study1.ID+"/audit", nil)
	req.SetPathValue("id", study1.ID)
	rr := httptest.NewRecorder()
	srv.ListStudyAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result []model.AuditEntry
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	require.Len(t, result, 1)
	assert.Equal(t, "study.approved", result[0].Action)
}

func TestListStudyAudit_StudyNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/studies/00000000-0000-0000-0000-000000000000/audit", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ListStudyAudit(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
