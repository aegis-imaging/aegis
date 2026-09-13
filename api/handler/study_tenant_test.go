package handler_test

import (
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

// Studies inherit tenant from their parent project — no schema change.
// These tests verify that ListStudies filters correctly and that
// cross-tenant GETs are denied via the access-helper layer.

func TestListStudies_WithTenantHeaderFiltersByProjectTenant(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenantA := &model.Tenant{Slug: "ten-a", Name: "Tenant A"}
	tenantB := &model.Tenant{Slug: "ten-b", Name: "Tenant B"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantA))
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantB))

	projA, err := model.CreateProjectForTenant(context.Background(), db, "PA", "pa-"+t.Name(), "", &tenantA.ID)
	require.NoError(t, err)
	projB, err := model.CreateProjectForTenant(context.Background(), db, "PB", "pb-"+t.Name(), "", &tenantB.ID)
	require.NoError(t, err)

	// 2 studies in tenant A's project, 1 in tenant B's.
	testutil.CreateTestStudy(t, db, projA.ID)
	testutil.CreateTestStudy(t, db, projA.ID)
	testutil.CreateTestStudy(t, db, projB.ID)

	req := httptest.NewRequest("GET", "/api/studies", nil)
	req.Header.Set(middleware.TenantHeader, "ten-a")
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListStudies)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Studies, 2)
	for _, st := range resp.Studies {
		assert.Equal(t, projA.ID, st.ProjectID,
			"every returned study must belong to a tenant-A project")
	}
}

func TestListStudies_NoTenantHeaderReturnsAll(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "ten-all", Name: "All"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))
	projT, err := model.CreateProjectForTenant(context.Background(), db, "PT", "pt-"+t.Name(), "", &tenant.ID)
	require.NoError(t, err)
	projU, err := model.CreateProject(context.Background(), db, "PU", "pu-"+t.Name(), "")
	require.NoError(t, err)

	testutil.CreateTestStudy(t, db, projT.ID)
	testutil.CreateTestStudy(t, db, projU.ID)

	req := httptest.NewRequest("GET", "/api/studies", nil)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.ListStudies)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Studies []model.Study `json:"studies"`
		Total   int           `json:"total"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	// Could be more if seeded; just confirm both of ours are visible.
	assert.GreaterOrEqual(t, resp.Total, 2)
}

func TestGetStudy_CrossTenantReturns404(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenantA := &model.Tenant{Slug: "study-a", Name: "A"}
	tenantB := &model.Tenant{Slug: "study-b", Name: "B"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantA))
	require.NoError(t, model.CreateTenant(context.Background(), db, tenantB))

	projA, err := model.CreateProjectForTenant(context.Background(), db, "OwnerProj", "op-"+t.Name(), "", &tenantA.ID)
	require.NoError(t, err)
	study := testutil.CreateTestStudy(t, db, projA.ID)

	// Try to GET tenant A's study from a request impersonating tenant B.
	req := httptest.NewRequest("GET", "/api/studies/"+study.ID, nil)
	req.Header.Set(middleware.TenantHeader, "study-b")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.GetStudy)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code,
		"cross-tenant GET must 404, not 200 or 403")
}

func TestGetStudy_OwnTenantReturns200(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "study-own", Name: "Owner"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))
	proj, err := model.CreateProjectForTenant(context.Background(), db, "P", "p-"+t.Name(), "", &tenant.ID)
	require.NoError(t, err)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID, nil)
	req.Header.Set(middleware.TenantHeader, "study-own")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.GetStudy)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var got model.Study
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	assert.Equal(t, study.ID, got.ID)
}

func TestGetStudyByUID_UntenantedStudyInvisibleToTenant(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	tenant := &model.Tenant{Slug: "ten-uid", Name: "T"}
	require.NoError(t, model.CreateTenant(context.Background(), db, tenant))

	// Untenanted project (legacy single-tenant data).
	proj, err := model.CreateProject(context.Background(), db, "Legacy", "legacy-uid-"+t.Name(), "")
	require.NoError(t, err)
	require.Nil(t, proj.TenantID)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/study-uid/"+study.StudyInstanceUID, nil)
	req.Header.Set(middleware.TenantHeader, "ten-uid")
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()
	middleware.ResolveTenant(db)(http.HandlerFunc(srv.GetStudyByUID)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code,
		"legacy NULL-tenant_id studies are invisible to a tenanted caller")
}
