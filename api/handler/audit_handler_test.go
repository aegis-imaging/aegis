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

// auditPageResponse mirrors handler.auditResponse for test decoding.
type auditPageResponse struct {
	Entries []model.AuditEntry `json:"entries"`
	Total   int                `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
}

func TestListAudit_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/audit", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 0, result.Total)
	assert.Len(t, result.Entries, 0)
	assert.Equal(t, 100, result.Limit) // default
	assert.Equal(t, 0, result.Offset)
}

func TestListAudit_ReturnsEntries(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	model.CreateAuditEntry(t.Context(), db, "study.approved", "admin@test.com", "study", "s1", "127.0.0.1", nil)
	model.CreateAuditEntry(t.Context(), db, "study.rejected", "admin@test.com", "study", "s2", "", nil)
	model.CreateAuditEntry(t.Context(), db, "admin_user.created", "ops@test.com", "admin_user", "u1", "", nil)

	req := httptest.NewRequest("GET", "/api/audit", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 3, result.Total)
	assert.Len(t, result.Entries, 3)
}

func TestListAudit_ActionPrefixFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	model.CreateAuditEntry(t.Context(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(t.Context(), db, "study.rejected", "a", "study", "s2", "", nil)
	model.CreateAuditEntry(t.Context(), db, "admin_user.created", "a", "admin_user", "u1", "", nil)

	req := httptest.NewRequest("GET", "/api/audit?action=study", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 2, result.Total)
	assert.Len(t, result.Entries, 2)
	for _, e := range result.Entries {
		assert.Contains(t, e.Action, "study")
	}
}

func TestListAudit_ActorFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	model.CreateAuditEntry(t.Context(), db, "study.approved", "alice@test.com", "study", "s1", "", nil)
	model.CreateAuditEntry(t.Context(), db, "study.rejected", "bob@test.com", "study", "s2", "", nil)

	req := httptest.NewRequest("GET", "/api/audit?actor=alice%40test.com", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Entries, 1)
	assert.Equal(t, "alice@test.com", result.Entries[0].Actor)
}

func TestListAudit_Pagination(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	for i := 0; i < 5; i++ {
		model.CreateAuditEntry(t.Context(), db, "test.event", "a", "test", "t", "", nil)
	}

	req := httptest.NewRequest("GET", "/api/audit?limit=2&offset=0", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var page1 auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&page1))
	assert.Equal(t, 5, page1.Total)
	assert.Len(t, page1.Entries, 2)
	assert.Equal(t, 2, page1.Limit)
	assert.Equal(t, 0, page1.Offset)

	req2 := httptest.NewRequest("GET", "/api/audit?limit=2&offset=2", nil)
	rr2 := httptest.NewRecorder()
	srv.ListAudit(rr2, req2)

	var page2 auditPageResponse
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&page2))
	assert.Equal(t, 5, page2.Total)
	assert.Len(t, page2.Entries, 2)
	assert.Equal(t, 2, page2.Offset)

	// Pages must not overlap
	assert.NotEqual(t, page1.Entries[0].ID, page2.Entries[0].ID)
}

func TestListAudit_LimitCappedAt500(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/audit?limit=9999", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 500, result.Limit)
}

func TestListAudit_ResourceTypeFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	model.CreateAuditEntry(t.Context(), db, "study.approved", "a", "study", "s1", "", nil)
	model.CreateAuditEntry(t.Context(), db, "admin_user.created", "a", "admin_user", "u1", "", nil)

	req := httptest.NewRequest("GET", "/api/audit?resource_type=admin_user", nil)
	rr := httptest.NewRecorder()
	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "admin_user", result.Entries[0].ResourceType)
}

func TestListAudit_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-researcher@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/audit", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAudit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
}

func TestListAudit_ResearcherScopedToProjectStudyEvents(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	projA := testutil.SeedProject(t, db)
	projB, err := model.CreateProject(t.Context(), db, "Audit Scope B", "audit-scope-b", "")
	require.NoError(t, err)

	studyA := testutil.CreateTestStudy(t, db, projA.ID)
	studyB := testutil.CreateTestStudy(t, db, projB.ID)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-scoped@test.com", "researcher")

	require.NoError(t, model.CreateProjectMember(t.Context(), db, &model.ProjectMember{
		ProjectID:   projA.ID,
		AdminUserID: researcher.ID,
		Role:        "reviewer",
	}))

	require.NoError(t, model.CreateAuditEntry(t.Context(), db, "study.approved", "admin@test.com", "study", studyA.ID, "", nil))
	require.NoError(t, model.CreateAuditEntry(t.Context(), db, "study.rejected", "admin@test.com", "study", studyB.ID, "", nil))
	require.NoError(t, model.CreateAuditEntry(t.Context(), db, "admin_user.created", "ops@test.com", "admin_user", "x", "", nil))

	req := httptest.NewRequest("GET", "/api/audit?project_id="+projA.ID, nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 1, result.Total)
	require.Len(t, result.Entries, 1)
	assert.Equal(t, studyA.ID, result.Entries[0].ResourceID)
}

func TestListAudit_ResearcherWithProjectScopeWithoutMembershipDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-denied@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/audit?project_id="+proj.ID, nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAudit(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

// TestListAudit_ResearcherSelfQueryWithoutProjectScope verifies that a
// non-admin researcher can read their own audit history via
// ?actor=<their-email> without supplying a project_id. This unblocks the
// /profile/activity page for non-admin users.
func TestListAudit_ResearcherSelfQueryWithoutProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-self@test.com", "researcher")

	// Two entries by the researcher (visible) + one by someone else (not visible).
	require.NoError(t, model.CreateAuditEntry(t.Context(), db, "study.approved", researcher.Email, "study", "s1", "", nil))
	require.NoError(t, model.CreateAuditEntry(t.Context(), db, "study.note", researcher.Email, "study", "s2", "", nil))
	require.NoError(t, model.CreateAuditEntry(t.Context(), db, "study.rejected", "other@test.com", "study", "s3", "", nil))

	req := httptest.NewRequest("GET", "/api/audit?actor="+researcher.Email+"&limit=100", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAudit(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "self-query must bypass the project_id requirement")
	var result auditPageResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, 2, result.Total, "researcher sees their own 2 entries, not the third entry from another actor")
	for _, e := range result.Entries {
		assert.Equal(t, researcher.Email, e.Actor)
	}
}

// TestListAudit_ResearcherCrossUserQueryStillRequiresProjectScope verifies
// the bypass is strictly self-scoped — a researcher cannot use the same
// ?actor= filter to peek at a different user's history without project
// access (the project_id requirement still applies).
func TestListAudit_ResearcherCrossUserQueryStillRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "audit-self-strict@test.com", "researcher")

	req := httptest.NewRequest("GET", "/api/audit?actor=someone-else@test.com", nil)
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	rr := httptest.NewRecorder()

	srv.ListAudit(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries",
		"self-query bypass must NOT extend to querying other actors' history")
}
