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

func TestBulkCreateShares_EmptyBody(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateShares_MissingEmail(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]interface{}{
		"study_ids": []string{study.ID},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateShares_TooManyStudyIDs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	ids := make([]string, 201)
	for i := range ids {
		ids[i] = "00000000-0000-0000-0000-000000000001"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"study_ids":       ids,
		"recipient_email": "test@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateShares_InvalidMaxDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	zero := 0
	body, _ := json.Marshal(map[string]interface{}{
		"study_ids":       []string{study.ID},
		"recipient_email": "test@example.com",
		"max_downloads":   zero,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkCreateShares_NonApprovedStudy(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID) // status='received' by default

	body, _ := json.Marshal(map[string]interface{}{
		"study_ids":       []string{study.ID},
		"recipient_email": "test@example.com",
		"expiry_hours":    24,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	// Request succeeds but the study error is captured in the errors array
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Created int `json:"created"`
		Errors  []struct {
			StudyID string `json:"study_id"`
			Error   string `json:"error"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Created)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, study.ID, resp.Errors[0].StudyID)
}

func TestBulkCreateShares_ApprovedStudy(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]interface{}{
		"study_ids":       []string{study.ID},
		"recipient_email": "recipient@example.com",
		"expiry_hours":    48,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Created int `json:"created"`
		Errors  []interface{} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Created)
	assert.Empty(t, resp.Errors)
}

func TestBulkCreateShares_PartialSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	approved := testutil.CreateTestStudy(t, db, proj.ID)
	pending := testutil.CreateTestStudy(t, db, proj.ID)

	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, approved.ID)
	require.NoError(t, err)
	// pending study stays 'received'

	body, _ := json.Marshal(map[string]interface{}{
		"study_ids":       []string{approved.ID, pending.ID},
		"recipient_email": "bulk@example.com",
		"expiry_hours":    72,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Created int `json:"created"`
		Errors  []struct {
			StudyID string `json:"study_id"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Created)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, pending.ID, resp.Errors[0].StudyID)
}

func TestBulkCreateShares_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-share",
		bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkCreateShares(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
