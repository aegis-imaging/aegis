package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAnonProfiles_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/anon-profiles", nil)
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.ListAnonProfiles(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.AnonProfile
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestCreateAnonProfile_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{
		"name":          "Brain MRI Profile",
		"description":   "Retains study date for longitudinal tracking",
		"retained_tags": json.RawMessage(`["StudyDate","PatientAge"]`),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/anon-profiles", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateAnonProfile(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result model.AnonProfile
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "Brain MRI Profile", result.Name)
	assert.Equal(t, proj.ID, result.ProjectID)
	assert.True(t, result.Enabled)
}

func TestCreateAnonProfile_MissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{"description": "no name provided"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/anon-profiles", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.CreateAnonProfile(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestCreateAnonProfile_ProjectNotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{"name": "My Profile"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/nonexistent/anon-profiles", bytes.NewReader(b))
	req.SetPathValue("projectID", "no-such-project")
	rr := httptest.NewRecorder()

	srv.CreateAnonProfile(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "project not found")
}

func TestListAnonProfiles_ReturnsCreated(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	for _, name := range []string{"Profile A", "Profile B"} {
		body := map[string]any{"name": name}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/anon-profiles", bytes.NewReader(b))
		req.SetPathValue("projectID", proj.ID)
		rr := httptest.NewRecorder()
		srv.CreateAnonProfile(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/anon-profiles", nil)
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()
	srv.ListAnonProfiles(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.AnonProfile
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestGetAnonProfile_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID:    proj.ID,
		Name:         "CT Head Profile",
		Description:  "For CT head studies",
		RetainedTags: json.RawMessage(`["StudyDate"]`),
		Enabled:      true,
	}
	require.NoError(t, model.CreateAnonProfile(t.Context(), db, p))

	req := httptest.NewRequest(http.MethodGet, "/api/anon-profiles/"+p.ID, nil)
	req.SetPathValue("id", p.ID)
	rr := httptest.NewRecorder()

	srv.GetAnonProfile(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.AnonProfile
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, "CT Head Profile", result.Name)
}

func TestGetAnonProfile_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/anon-profiles/missing", nil)
	req.SetPathValue("id", "no-such-profile")
	rr := httptest.NewRecorder()

	srv.GetAnonProfile(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "profile not found")
}

func TestUpdateAnonProfile_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID:    proj.ID,
		Name:         "Original",
		RetainedTags: json.RawMessage(`[]`),
		Enabled:      true,
	}
	require.NoError(t, model.CreateAnonProfile(t.Context(), db, p))

	update := map[string]any{
		"name":          "Updated Profile",
		"description":   "Updated description",
		"retained_tags": json.RawMessage(`["PatientAge","StudyDate"]`),
		"enabled":       false,
	}
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/api/anon-profiles/"+p.ID, bytes.NewReader(b))
	req.SetPathValue("id", p.ID)
	rr := httptest.NewRecorder()

	srv.UpdateAnonProfile(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.AnonProfile
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "Updated Profile", result.Name)
	assert.Equal(t, "Updated description", result.Description)
	assert.False(t, result.Enabled)
}

func TestUpdateAnonProfile_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{"name": "X"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/anon-profiles/missing", bytes.NewReader(b))
	req.SetPathValue("id", "no-such-profile")
	rr := httptest.NewRecorder()

	srv.UpdateAnonProfile(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "profile not found")
}

func TestDeleteAnonProfile_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID:    proj.ID,
		Name:         "To Delete",
		RetainedTags: json.RawMessage(`[]`),
		Enabled:      true,
	}
	require.NoError(t, model.CreateAnonProfile(t.Context(), db, p))

	req := httptest.NewRequest(http.MethodDelete, "/api/anon-profiles/"+p.ID, nil)
	req.SetPathValue("id", p.ID)
	rr := httptest.NewRecorder()

	srv.DeleteAnonProfile(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify deleted
	req2 := httptest.NewRequest(http.MethodGet, "/api/anon-profiles/"+p.ID, nil)
	req2.SetPathValue("id", p.ID)
	rr2 := httptest.NewRecorder()
	srv.GetAnonProfile(rr2, req2)
	assert.Equal(t, http.StatusNotFound, rr2.Code)
}

func TestDeleteAnonProfile_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/anon-profiles/missing", nil)
	req.SetPathValue("id", "no-such-profile")
	rr := httptest.NewRecorder()

	srv.DeleteAnonProfile(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "profile not found")
}

func TestSetDefaultAnonProfile_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID:    proj.ID,
		Name:         "Default Profile",
		RetainedTags: json.RawMessage(`[]`),
		Enabled:      true,
	}
	require.NoError(t, model.CreateAnonProfile(t.Context(), db, p))

	body := map[string]any{"profile_id": p.ID}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+proj.ID+"/default-anon-profile", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.SetDefaultAnonProfile(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestSetDefaultAnonProfile_ClearsDefault(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Clear with empty profile_id
	body := map[string]any{"profile_id": ""}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+proj.ID+"/default-anon-profile", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.SetDefaultAnonProfile(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestSetDefaultAnonProfile_ProfileNotInProject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	body := map[string]any{"profile_id": "00000000-0000-0000-0000-000000000000"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+proj.ID+"/default-anon-profile", bytes.NewReader(b))
	req.SetPathValue("projectID", proj.ID)
	rr := httptest.NewRecorder()

	srv.SetDefaultAnonProfile(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "profile not found in this project")
}

func TestGetDefaultAnonProfile_NoDefault(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.Slug+"/active-anon-profile", nil)
	req.SetPathValue("slug", proj.Slug)
	rr := httptest.NewRecorder()

	srv.GetDefaultAnonProfile(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
}

func TestGetDefaultAnonProfile_ReturnsDefault(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	p := &model.AnonProfile{
		ProjectID:    proj.ID,
		Name:         "Active Profile",
		RetainedTags: json.RawMessage(`["StudyDate"]`),
		Enabled:      true,
	}
	require.NoError(t, model.CreateAnonProfile(t.Context(), db, p))
	require.NoError(t, model.SetProjectDefaultAnonProfile(t.Context(), db, proj.ID, p.ID))

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.Slug+"/active-anon-profile", nil)
	req.SetPathValue("slug", proj.Slug)
	rr := httptest.NewRecorder()

	srv.GetDefaultAnonProfile(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.AnonProfile
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, p.ID, result.ID)
	assert.Equal(t, "Active Profile", result.Name)
}
