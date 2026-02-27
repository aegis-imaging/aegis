package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuditActors_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-actors-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/audit/actors", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.GetAuditActors(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}

func TestGetAuditActors_ResearcherWithProjectScopeWithoutMembershipDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-actors-denied@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/audit/actors?project_id="+proj.ID, nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.GetAuditActors(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

func TestGetAuditActors_ResearcherWithProjectScopeAllowed(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-actors-allowed@test.com", "researcher")

	require.NoError(t, model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:   proj.ID,
		AdminUserID: researcher.ID,
		Role:        "reviewer",
	}))

	req := httptest.NewRequest("GET", "/api/audit/actors?project_id="+proj.ID, nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.GetAuditActors(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "\"actors\"")
}
