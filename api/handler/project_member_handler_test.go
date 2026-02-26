package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withAdmin injects an admin AuthUser into the request context.
func withAdmin(r *http.Request, db *sql.DB, email string) *http.Request {
	u, err := model.GetAdminUserByEmail(context.Background(), db, email)
	if err != nil {
		// Fallback: create a synthetic admin user
		u = &model.AdminUser{ID: "00000000-0000-0000-0000-000000000001", Email: email, Role: "admin", Enabled: true}
	}
	ctx := context.WithValue(r.Context(), middleware.AuthUserContextKey(), &middleware.AuthUser{
		ID: u.ID, Email: u.Email, Role: u.Role,
	})
	return r.WithContext(ctx)
}

// ─── List project members ─────────────────────────────────────────────────────

func TestListProjectMembers_EmptyProject(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.CreateTestProject(t, db, "members-empty")
	admin := testutil.CreateTestAdminUser(t, db, "admin-lpm@test.com", "admin")

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/projects/%s/members", proj.ID), nil)
	req = withAdmin(req, db, admin.Email)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListProjectMembers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Members []model.ProjectMember `json:"members"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Empty(t, resp.Members)
}

// ─── Add project member ───────────────────────────────────────────────────────

func TestAddProjectMember_CoordinatorRole(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.CreateTestProject(t, db, "add-member-coord")
	admin := testutil.CreateTestAdminUser(t, db, "admin-apm@test.com", "admin")
	researcher := testutil.CreateTestAdminUser(t, db, "researcher-apm@test.com", "researcher")

	body, _ := json.Marshal(map[string]any{
		"admin_user_id": researcher.ID,
		"role":          "coordinator",
	})
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/projects/%s/members", proj.ID), bytes.NewReader(body))
	req = withAdmin(req, db, admin.Email)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.AddProjectMember(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var m model.ProjectMember
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&m))
	assert.Equal(t, "coordinator", m.Role)
	assert.Nil(t, m.InstitutionID)
}

func TestAddProjectMember_SiteRoleRequiresInstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.CreateTestProject(t, db, "add-member-site-noInst")
	admin := testutil.CreateTestAdminUser(t, db, "admin-site1@test.com", "admin")
	researcher := testutil.CreateTestAdminUser(t, db, "researcher-site1@test.com", "researcher")

	body, _ := json.Marshal(map[string]any{
		"admin_user_id": researcher.ID,
		"role":          "site_coordinator",
		// no institution_id — should fail
	})
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/projects/%s/members", proj.ID), bytes.NewReader(body))
	req = withAdmin(req, db, admin.Email)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.AddProjectMember(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAddProjectMember_SiteRoleWithInstitution(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.CreateTestProject(t, db, "add-member-site-ok")
	admin := testutil.CreateTestAdminUser(t, db, "admin-site2@test.com", "admin")
	researcher := testutil.CreateTestAdminUser(t, db, "researcher-site2@test.com", "researcher")
	inst := testutil.CreateTestInstitution(t, db, "Test Site Hospital")

	instID := inst.ID
	body, _ := json.Marshal(map[string]any{
		"admin_user_id":  researcher.ID,
		"role":           "site_coordinator",
		"institution_id": instID,
	})
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/projects/%s/members", proj.ID), bytes.NewReader(body))
	req = withAdmin(req, db, admin.Email)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.AddProjectMember(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var m model.ProjectMember
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&m))
	assert.Equal(t, "site_coordinator", m.Role)
	require.NotNil(t, m.InstitutionID)
	assert.Equal(t, instID, *m.InstitutionID)
}

// ─── Project member model CRUD ────────────────────────────────────────────────

func TestProjectMemberCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	proj := testutil.CreateTestProject(t, db, "member-crud")
	researcher := testutil.CreateTestAdminUser(t, db, "r-crud@test.com", "researcher")
	inst := testutil.CreateTestInstitution(t, db, "CRUD Institution")
	ctx := context.Background()

	// Create member
	m := &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_viewer",
		InstitutionID: &inst.ID,
		Notes:         "test member",
	}
	err := model.CreateProjectMember(ctx, db, m)
	require.NoError(t, err)
	assert.NotEmpty(t, m.ID)
	assert.Equal(t, "site_viewer", m.Role)
	assert.Equal(t, &inst.ID, m.InstitutionID)

	// Read
	got, err := model.GetProjectMember(ctx, db, m.ID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, got.ID)

	// Update
	m.Role = "site_coordinator"
	err = model.UpdateProjectMember(ctx, db, m)
	require.NoError(t, err)
	assert.Equal(t, "site_coordinator", m.Role)

	// List
	members, err := model.ListProjectMembers(ctx, db, proj.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)

	// Delete
	err = model.DeleteProjectMember(ctx, db, m.ID)
	require.NoError(t, err)
	members, err = model.ListProjectMembers(ctx, db, proj.ID)
	require.NoError(t, err)
	assert.Empty(t, members)
}

// ─── ListProjects membership filter ──────────────────────────────────────────

func TestListProjectsMembershipFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	// Create two projects; restrict project B
	projA := testutil.CreateTestProject(t, db, "proj-filter-a")
	projB := testutil.CreateTestProject(t, db, "proj-filter-b")
	_, err := db.ExecContext(ctx, `UPDATE projects SET restricted = true WHERE id = $1`, projB.ID)
	require.NoError(t, err)

	// Create researcher user and add only to projB
	researcher := testutil.CreateTestAdminUser(t, db, "r-filter@test.com", "researcher")
	err = model.CreateProjectMember(ctx, db, &model.ProjectMember{
		ProjectID:   projB.ID,
		AdminUserID: researcher.ID,
		Role:        "reviewer",
	})
	require.NoError(t, err)

	// Researcher should only see projB (the one they're a member of)
	projects, err := model.ListProjectsForResearcher(ctx, db, researcher.ID)
	require.NoError(t, err)
	ids := make([]string, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	assert.Contains(t, ids, projB.ID, "researcher should see projB (their project)")
	assert.NotContains(t, ids, projA.ID, "researcher should NOT see unrestricted projA they're not a member of")
}

func TestListProjectsPublic(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	projOpen := testutil.CreateTestProject(t, db, "proj-public-open")
	projRestricted := testutil.CreateTestProject(t, db, "proj-public-restricted")
	_, err := db.ExecContext(ctx, `UPDATE projects SET restricted = true WHERE id = $1`, projRestricted.ID)
	require.NoError(t, err)

	// Public list should include open project but NOT restricted
	projects, err := model.ListProjectsPublic(ctx, db)
	require.NoError(t, err)
	ids := make([]string, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	assert.Contains(t, ids, projOpen.ID, "public list should include non-restricted project")
	assert.NotContains(t, ids, projRestricted.ID, "public list should exclude restricted project")
}

// ─── Platform admin bypasses restriction ─────────────────────────────────────

func TestPlatformAdminBypassesRestriction(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	proj := testutil.CreateTestProject(t, db, "bypass-restricted")
	_, err := db.ExecContext(ctx, `UPDATE projects SET restricted = true WHERE id = $1`, proj.ID)
	require.NoError(t, err)

	// Platform admin (role=admin) — CanAccessProject should allow
	admin := &middleware.AuthUser{ID: "00000000-0000-0000-0000-000000000099", Email: "admin@test.com", Role: "admin"}
	_, allowed, err := middleware.CanAccessProject(ctx, db, admin, proj.ID, true)
	require.NoError(t, err)
	assert.True(t, allowed, "platform admin must bypass restriction")
}

func TestResearcherDeniedWithoutMembership(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	proj := testutil.CreateTestProject(t, db, "deny-no-membership")
	_, err := db.ExecContext(ctx, `UPDATE projects SET restricted = true WHERE id = $1`, proj.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "r-deny@test.com", "researcher")
	rUser := &middleware.AuthUser{ID: researcher.ID, Email: researcher.Email, Role: researcher.Role}

	_, allowed, err := middleware.CanAccessProject(ctx, db, rUser, proj.ID, true)
	require.NoError(t, err)
	assert.False(t, allowed, "researcher without membership must be denied")
}

func TestResearcherAllowedWithMembership(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	db := testutil.TestDB(t)
	ctx := context.Background()

	proj := testutil.CreateTestProject(t, db, "allow-with-membership")
	_, err := db.ExecContext(ctx, `UPDATE projects SET restricted = true WHERE id = $1`, proj.ID)
	require.NoError(t, err)

	researcher := testutil.CreateTestAdminUser(t, db, "r-allow@test.com", "researcher")
	err = model.CreateProjectMember(ctx, db, &model.ProjectMember{
		ProjectID:   proj.ID,
		AdminUserID: researcher.ID,
		Role:        "reviewer",
	})
	require.NoError(t, err)

	rUser := &middleware.AuthUser{ID: researcher.ID, Email: researcher.Email, Role: researcher.Role}
	access, allowed, err := middleware.CanAccessProject(ctx, db, rUser, proj.ID, true)
	require.NoError(t, err)
	assert.True(t, allowed, "researcher with membership must be allowed")
	require.NotNil(t, access, "access record must be returned")
	assert.Equal(t, "reviewer", access.Role)
}
