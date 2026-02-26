package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInstitutionBreakdown_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/00000000-0000-0000-0000-000000000000/institution-breakdown", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.GetInstitutionBreakdown(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetInstitutionBreakdown_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/institution-breakdown", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetInstitutionBreakdown(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp struct {
		ProjectID   string        `json:"project_id"`
		GeneratedAt string        `json:"generated_at"`
		Rows        []interface{} `json:"rows"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.NotEmpty(t, resp.GeneratedAt)
	assert.NotNil(t, resp.Rows)
	assert.Len(t, resp.Rows, 0)
}

func TestGetInstitutionBreakdown_WithStudies(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 2 studies with no institution
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/institution-breakdown", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetInstitutionBreakdown(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID string `json:"project_id"`
		Rows      []struct {
			InstitutionID   interface{} `json:"institution_id"`
			InstitutionName interface{} `json:"institution_name"`
			StudyCount      int         `json:"study_count"`
			Approved        int         `json:"approved"`
			Rejected        int         `json:"rejected"`
			Pending         int         `json:"pending"`
		} `json:"rows"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	// Both studies have no institution, so they group into one row with null institution_id
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, 2, resp.Rows[0].StudyCount)
}

func TestGetInstitutionBreakdown_ApprovedCounts(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, s1.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE studies SET status='rejected' WHERE id=$1`, s2.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/institution-breakdown", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetInstitutionBreakdown(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Rows []struct {
			StudyCount int `json:"study_count"`
			Approved   int `json:"approved"`
			Rejected   int `json:"rejected"`
		} `json:"rows"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Rows, 1)
	assert.Equal(t, 2, resp.Rows[0].StudyCount)
	assert.Equal(t, 1, resp.Rows[0].Approved)
	assert.Equal(t, 1, resp.Rows[0].Rejected)
}
