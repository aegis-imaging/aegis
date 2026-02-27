package handler_test

import (
	"bytes"
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

func TestPatchStudyFlag_SetFlag(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"flagged":true}`
	req := httptest.NewRequest("PATCH", "/api/studies/"+study.ID+"/flag", bytes.NewBufferString(body))
	req = withAdminUser(req, "admin-flag-set", "admin-flag-set@test.com")
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, true, resp["flagged"])
	assert.Equal(t, "ok", resp["status"])
}

func TestPatchStudyFlag_ClearFlag(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"flagged":false}`
	req := httptest.NewRequest("PATCH", "/api/studies/"+study.ID+"/flag", bytes.NewBufferString(body))
	req = withAdminUser(req, "admin-flag-clear", "admin-flag-clear@test.com")
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, false, resp["flagged"])
}

func TestPatchStudyFlag_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"flagged":true}`
	req := httptest.NewRequest("PATCH", "/api/studies/00000000-0000-0000-0000-000000000000/flag", bytes.NewBufferString(body))
	req = withAdminUser(req, "admin-flag-not-found", "admin-flag-not-found@test.com")
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPatchStudyFlag_InvalidBody(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("PATCH", "/api/studies/"+study.ID+"/flag", bytes.NewBufferString(`not json`))
	req = withAdminUser(req, "admin-flag-invalid", "admin-flag-invalid@test.com")
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPatchStudyFlag_ResearcherSiteCoordinator_OnSiteAllowed(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	inst := testutil.CreateTestInstitution(t, db, "flag-site-allowed")
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1 WHERE id = $2`, inst.ID, study.ID)
	require.NoError(t, err)
	researcher := testutil.CreateTestAdminUser(t, db, "flag-site-allowed@test.com", "researcher")

	err = model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_coordinator",
		InstitutionID: &inst.ID,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", "/api/studies/"+study.ID+"/flag", bytes.NewBufferString(`{"flagged":true}`))
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestPatchStudyFlag_ResearcherSiteCoordinator_OffSiteDeniedAsNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	instMember := testutil.CreateTestInstitution(t, db, "flag-site-member")
	instStudy := testutil.CreateTestInstitution(t, db, "flag-site-study")
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.ExecContext(context.Background(), `UPDATE studies SET institution_id = $1 WHERE id = $2`, instStudy.ID, study.ID)
	require.NoError(t, err)
	researcher := testutil.CreateTestAdminUser(t, db, "flag-site-denied@test.com", "researcher")

	err = model.CreateProjectMember(context.Background(), db, &model.ProjectMember{
		ProjectID:     proj.ID,
		AdminUserID:   researcher.ID,
		Role:          "site_coordinator",
		InstitutionID: &instMember.ID,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("PATCH", "/api/studies/"+study.ID+"/flag", bytes.NewBufferString(`{"flagged":true}`))
	req = withResearcherUser(req, researcher.ID, researcher.Email)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
