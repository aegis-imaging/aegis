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

func TestSoftDeleteStudy_Success(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodDelete, "/api/studies/"+study.ID+"/soft", nil)
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.SoftDeleteStudy(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "deleted", resp["status"])
	assert.Equal(t, study.ID, resp["id"])

	// Verify soft-deleted: study should have deleted_at set
	var deletedAt *string
	err := db.QueryRow(`SELECT deleted_at::text FROM studies WHERE id=$1`, study.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.NotNil(t, deletedAt)
}

func TestSoftDeleteStudy_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/studies/00000000-0000-0000-0000-000000000000/soft", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.SoftDeleteStudy(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRestoreStudy_Success(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Soft-delete first
	_, err := db.Exec(`UPDATE studies SET deleted_at=now() WHERE id=$1`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.ID+"/restore", nil)
	req.SetPathValue("id", study.ID)
	w := httptest.NewRecorder()
	srv.RestoreStudy(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "restored", resp["status"])

	// Verify restored
	var deletedAt *string
	err = db.QueryRow(`SELECT deleted_at::text FROM studies WHERE id=$1`, study.ID).Scan(&deletedAt)
	require.NoError(t, err)
	assert.Nil(t, deletedAt)
}

func TestRestoreStudy_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/00000000-0000-0000-0000-000000000000/restore", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.RestoreStudy(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListDeletedStudies_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/deleted", nil)
	w := httptest.NewRecorder()
	srv.ListDeletedStudies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp, "studies")
	assert.Contains(t, resp, "total")
}

func TestListDeletedStudies_ShowsSoftDeleted(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := db.Exec(`UPDATE studies SET deleted_at=now() WHERE id=$1`, study.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/deleted", nil)
	w := httptest.NewRecorder()
	srv.ListDeletedStudies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Studies []map[string]interface{} `json:"studies"`
		Total   int                      `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)
	assert.Len(t, resp.Studies, 1)
}
