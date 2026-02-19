package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/msenjem/aegis/api/model"
	"github.com/msenjem/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateShare_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	model.UpdateStudyStatus(context.Background(), db, study.ID, "approved")

	body, _ := json.Marshal(map[string]any{
		"recipient_email": "recipient@test.com",
		"note":            "Please review",
		"expiry_hours":    72,
	})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/share", bytes.NewReader(body))
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.CreateShare(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result["token"])
	assert.NotEmpty(t, result["export_url"])
}

func TestCreateShare_NotApproved(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID) // status=received

	body, _ := json.Marshal(map[string]any{"recipient_email": "r@t.com"})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/share", bytes.NewReader(body))
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.CreateShare(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateShare_MissingRecipient(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	model.UpdateStudyStatus(context.Background(), db, study.ID, "approved")

	body, _ := json.Marshal(map[string]any{"note": "no email"})
	req := httptest.NewRequest("POST", "/api/studies/"+study.ID+"/share", bytes.NewReader(body))
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.CreateShare(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRedeemExport_InvalidToken(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest("GET", "/api/export/invalidtoken", nil)
	req.SetPathValue("token", "invalidtoken")
	rr := httptest.NewRecorder()
	srv.RedeemExport(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestListShares_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest("GET", "/api/studies/"+study.ID+"/shares", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.ListShares(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var shares []model.ExportShare
	json.NewDecoder(rr.Body).Decode(&shares)
	assert.NotNil(t, shares) // empty array, not null
}
