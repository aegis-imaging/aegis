package handler_test

import (
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

func TestReactivateStudy_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Manually expire the study so ReactivateStudy has something to act on.
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "expired"))

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reactivate", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ReactivateStudy(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, "approved", resp["status"])

	// Verify the DB was updated.
	updated, err := model.GetStudyByID(context.Background(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", updated.Status)
}

func TestReactivateStudy_WrongStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID) // default status = "received"

	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/reactivate", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ReactivateStudy(rr, req)

	// "received" is not "expired", so 400.
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReactivateStudy_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("POST", "/api/studies/00000000-0000-0000-0000-000000000000/reactivate", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.ReactivateStudy(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
