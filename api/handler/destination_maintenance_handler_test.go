package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestinationMaintenance_CreateListDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "maint-dest")

	starts := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	ends := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	body := `{"reason":"PACS upgrade","starts_at":"` + starts + `","ends_at":"` + ends + `"}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/maintenance", strings.NewReader(body))
	createReq.SetPathValue("id", dest.ID)
	createRR := httptest.NewRecorder()
	srv.CreateDestinationMaintenance(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code, createRR.Body.String())

	var window model.DestinationMaintenanceWindow
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&window))
	assert.Equal(t, "PACS upgrade", window.Reason)
	assert.NotEmpty(t, window.ID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/destinations/"+dest.ID+"/maintenance", nil)
	listReq.SetPathValue("id", dest.ID)
	listRR := httptest.NewRecorder()
	srv.ListDestinationMaintenance(listRR, listReq)
	require.Equal(t, http.StatusOK, listRR.Code)

	var windows []model.DestinationMaintenanceWindow
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&windows))
	require.Len(t, windows, 1)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/destinations/"+dest.ID+"/maintenance/"+window.ID, nil)
	delReq.SetPathValue("id", dest.ID)
	delReq.SetPathValue("windowID", window.ID)
	delRR := httptest.NewRecorder()
	srv.DeleteDestinationMaintenance(delRR, delReq)
	assert.Equal(t, http.StatusOK, delRR.Code)
}

func TestDestinationMaintenance_EndsBeforeStartsRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "maint-bad-window")

	starts := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	ends := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	body := `{"reason":"x","starts_at":"` + starts + `","ends_at":"` + ends + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/maintenance", strings.NewReader(body))
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.CreateDestinationMaintenance(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDestinationMaintenance_BadDateFormatRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "maint-bad-date")

	body := `{"reason":"x","starts_at":"not-a-date","ends_at":"also-not"}`
	req := httptest.NewRequest(http.MethodPost, "/api/destinations/"+dest.ID+"/maintenance", strings.NewReader(body))
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()
	srv.CreateDestinationMaintenance(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
