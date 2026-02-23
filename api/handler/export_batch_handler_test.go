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

func TestExportBatch_NoEligibleStudies(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest("POST", "/api/projects/"+proj.ID+"/export-batch", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.ExportBatch(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Dispatched int      `json:"dispatched"`
		StudyIDs   []string `json:"study_ids"`
		Message    string   `json:"message"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Dispatched)
	assert.Empty(t, resp.StudyIDs)
}

func TestExportBatch_StudiesNotApproved(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	testutil.CreateTestStudy(t, db, proj.ID) // received status, not approved

	req := httptest.NewRequest("POST", "/api/projects/"+proj.ID+"/export-batch", nil)
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.ExportBatch(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Dispatched int `json:"dispatched"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 0, resp.Dispatched, "unapproved studies should not be exported")
}
