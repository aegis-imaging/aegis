package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkStatusUpdate_OK(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	body := `{"study_ids":["` + s1.ID + `","` + s2.ID + `"],"status":"approved"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-status", strings.NewReader(body))
	rr := httptest.NewRecorder()
	srv.BulkStatusUpdate(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["updated"])
	assert.Equal(t, float64(2), resp["total"])

	// Verify persistence.
	got1, err := model.GetStudyByID(context.Background(), db, s1.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", got1.Status)
}

func TestBulkStatusUpdate_InvalidStatusRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-status",
		strings.NewReader(`{"study_ids":["x"],"status":"shipped"}`))
	rr := httptest.NewRecorder()
	srv.BulkStatusUpdate(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBulkStatusUpdate_EmptyIDsRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-status",
		strings.NewReader(`{"study_ids":[],"status":"approved"}`))
	rr := httptest.NewRecorder()
	srv.BulkStatusUpdate(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBulkStatusUpdate_OverLimitRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ids := make([]string, 0, 201)
	for i := 0; i < 201; i++ {
		ids = append(ids, `"00000000-0000-0000-0000-000000000000"`)
	}
	body := `{"study_ids":[` + strings.Join(ids, ",") + `],"status":"approved"}`
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-status", strings.NewReader(body))
	rr := httptest.NewRecorder()
	srv.BulkStatusUpdate(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
