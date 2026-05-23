package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests wrap each handler call in ResolveTenant(db) so the
// tenant-resolution middleware runs end-to-end — mirroring what
// main.go does in production. testutil.TestServer doesn't apply the
// global middleware chain, so this is how a test gets a real tenant
// into request context.

func TestListProjects_NoTenantHeaderReturnsAllProjects(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create one untenanted project and one tenant-owned project directly via
	// the model. The tenanted project should not be hidden — the request has
	// no tenant context, so it sees everything (legacy single-tenant view).
	tenant := &model.Tenant{Slug: "acme", Name: "Acme"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	_, err := model.CreateProject(context.Background(), db, "Untenanted", "untenanted-"+t.Name(), "")
	require.NoError(t, err)
	_, err = model.CreateProjectForTenant(context.Background(), db, "Tenanted", "tenanted-"+t.Name(), "", &tenant.ID)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/projects", nil)
	rr := httptest.NewRecorder()

	// Run through the real middleware (no header → no tenant in context).
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListProjects)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var projects []model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&projects))
	// Expect at least the two we created (migrations may seed others).
	assert.GreaterOrEqual(t, len(projects), 2)
}

func TestListProjects_WithTenantHeaderFiltersToTenant(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenantA := &model.Tenant{Slug: "tenant-a", Name: "Tenant A"}
	tenantB := &model.Tenant{Slug: "tenant-b", Name: "Tenant B"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantA))
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantB))

	_, err := model.CreateProjectForTenant(context.Background(), db, "A1", "a1-"+t.Name(), "", &tenantA.ID)
	require.NoError(t, err)
	_, err = model.CreateProjectForTenant(context.Background(), db, "A2", "a2-"+t.Name(), "", &tenantA.ID)
	require.NoError(t, err)
	_, err = model.CreateProjectForTenant(context.Background(), db, "B1", "b1-"+t.Name(), "", &tenantB.ID)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/projects", nil)
	req.Header.Set(middleware.TenantHeader, "tenant-a")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListProjects)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var projects []model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&projects))
	require.Len(t, projects, 2)
	for _, p := range projects {
		require.NotNil(t, p.TenantID)
		assert.Equal(t, tenantA.ID, *p.TenantID,
			"every returned project must belong to the tenant in the header")
	}
}

func TestCreateProject_WithTenantHeaderSetsTenantID(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "create-tenant", Name: "Create Tenant"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	body, _ := json.Marshal(map[string]any{"name": "New Project", "slug": "new-project-" + t.Name()})
	req := httptest.NewRequest("POST", "/api/projects", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.TenantHeader, "create-tenant")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.CreateProject)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	var got model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.NotNil(t, got.TenantID)
	assert.Equal(t, tenant.ID, *got.TenantID,
		"a project created under a tenant header must carry that tenant_id")
}

func TestGetProject_CrossTenantAccessReturns404(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenantA := &model.Tenant{Slug: "owner", Name: "Owner"}
	tenantB := &model.Tenant{Slug: "intruder", Name: "Intruder"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantA))
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantB))

	project, err := model.CreateProjectForTenant(context.Background(), db, "Secret", "secret-"+t.Name(), "", &tenantA.ID)
	require.NoError(t, err)

	// Request via tenant B's header — must 404, not 200, and must not
	// leak the project body.
	req := httptest.NewRequest("GET", "/api/projects/"+project.ID, nil)
	req.Header.Set(middleware.TenantHeader, "intruder")
	req.SetPathValue("id", project.ID)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.GetProject)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetProject_OwnTenantAccessReturns200(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "owner-ok", Name: "Owner"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	project, err := model.CreateProjectForTenant(context.Background(), db, "Mine", "mine-"+t.Name(), "", &tenant.ID)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/projects/"+project.ID, nil)
	req.Header.Set(middleware.TenantHeader, "owner-ok")
	req.SetPathValue("id", project.ID)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.GetProject)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var got model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	assert.Equal(t, project.ID, got.ID)
}

func TestGetProject_UntenantedProjectInvisibleToTenantCaller(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "any-tenant", Name: "Any"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	// Untenanted project (legacy single-tenant deployment data).
	project, err := model.CreateProject(context.Background(), db, "Legacy", "legacy-"+t.Name(), "")
	require.NoError(t, err)
	require.Nil(t, project.TenantID)

	req := httptest.NewRequest("GET", "/api/projects/"+project.ID, nil)
	req.Header.Set(middleware.TenantHeader, "any-tenant")
	req.SetPathValue("id", project.ID)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.GetProject)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code,
		"a tenanted caller must not see untenanted (NULL tenant_id) projects")
}
