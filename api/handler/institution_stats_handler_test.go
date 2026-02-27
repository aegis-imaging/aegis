package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstitutionStats_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	inst := testutil.CreateTestInstitution(t, db, "Stats Test Inst")

	req := httptest.NewRequest("GET", "/api/institutions/"+inst.ID+"/stats", nil)
	req.SetPathValue("id", inst.ID)
	rr := httptest.NewRecorder()
	srv.GetInstitutionStats(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var stats model.InstitutionStats
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&stats))
	assert.Equal(t, inst.ID, stats.InstitutionID)
	assert.Equal(t, 0, stats.TotalStudies)
	assert.Nil(t, stats.LastStudyAt)
}

func TestGetInstitutionStats_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/institutions/nonexistent/stats", nil)
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.GetInstitutionStats(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetInstitutionStats_ResearcherRequiresProjectScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "Researcher Scope Inst")
	researcher := testutil.CreateTestAdminUser(t, db, "inst-stats-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/institutions/"+inst.ID+"/stats", nil)
	req.SetPathValue("id", inst.ID)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.GetInstitutionStats(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}
