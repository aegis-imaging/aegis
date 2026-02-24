package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.PatchStudyFlag(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
