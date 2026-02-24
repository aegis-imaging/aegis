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

func TestBulkStudyAction_Approve(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]any{
		"action":    "approve",
		"study_ids": []string{s1.ID, s2.ID},
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkStudyAction(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Processed int `json:"processed"`
		Errors    []struct {
			StudyID string `json:"study_id"`
			Error   string `json:"error"`
		} `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Processed)
	assert.Empty(t, resp.Errors)
}

func TestBulkStudyAction_Reject(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]any{
		"action":    "reject",
		"study_ids": []string{study.ID},
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkStudyAction(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct{ Processed int `json:"processed"` }
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Processed)
}

func TestBulkStudyAction_InvalidAction(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	body, _ := json.Marshal(map[string]any{
		"action":    "archive",
		"study_ids": []string{study.ID},
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkStudyAction(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBulkStudyAction_EmptyStudyIDs(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"action":    "approve",
		"study_ids": []string{},
	})
	req := httptest.NewRequest("POST", "/api/studies/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkStudyAction(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBulkStudyAction_AlreadyApproved_PartialError(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	// Approve s1 first
	approveBody, _ := json.Marshal(map[string]any{"action": "approve", "study_ids": []string{s1.ID}})
	approveReq := httptest.NewRequest("POST", "/api/studies/bulk", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	srv.BulkStudyAction(httptest.NewRecorder(), approveReq)

	// Now try to approve both; s1 already approved so it should error
	body, _ := json.Marshal(map[string]any{"action": "approve", "study_ids": []string{s1.ID, s2.ID}})
	req := httptest.NewRequest("POST", "/api/studies/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkStudyAction(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Processed int `json:"processed"`
		Errors    []struct {
			StudyID string `json:"study_id"`
			Error   string `json:"error"`
		} `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, 1, resp.Processed)
	assert.Len(t, resp.Errors, 1)
	assert.Equal(t, s1.ID, resp.Errors[0].StudyID)
}
