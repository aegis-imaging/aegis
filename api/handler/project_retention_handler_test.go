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

func TestSetProjectRetention_Set(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]int{"retention_days": 90})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/retention", bytes.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectRetention(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	require.NotNil(t, p.RetentionDays)
	assert.Equal(t, 90, *p.RetentionDays)
}

func TestSetProjectRetention_Clear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	// First set a retention
	days := 30
	require.NoError(t, model.UpdateProjectRetentionDays(context.Background(), db, proj.ID, &days))

	// Now clear it with null
	body := []byte(`{"retention_days": null}`)
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/retention", bytes.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectRetention(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var p model.Project
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&p))
	assert.Nil(t, p.RetentionDays)
}

func TestSetProjectRetention_InvalidZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	proj := testutil.SeedProject(t, db)

	body, _ := json.Marshal(map[string]int{"retention_days": 0})
	req := httptest.NewRequest("PUT", "/api/projects/"+proj.ID+"/retention", bytes.NewReader(body))
	req.SetPathValue("id", proj.ID)
	rr := httptest.NewRecorder()
	srv.SetProjectRetention(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestSetProjectRetention_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]int{"retention_days": 30})
	req := httptest.NewRequest("PUT", "/api/projects/nonexistent/retention", bytes.NewReader(body))
	req.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()
	srv.SetProjectRetention(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
