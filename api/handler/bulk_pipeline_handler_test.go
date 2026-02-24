package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkPipelineTrigger_InvalidStep(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"step":      "invalid_step",
		"study_ids": []string{"00000000-0000-0000-0000-000000000001"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBulkPipelineTrigger_EmptyStudyIDs(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"step":      "qc",
		"study_ids": []string{},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestBulkPipelineTrigger_StepNotRequired_CountsAsSkipped(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID) // qc_required=false by default

	body, _ := json.Marshal(map[string]any{
		"step":      "qc",
		"study_ids": []string{study.ID},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res struct {
		Triggered int `json:"triggered"`
		Skipped   int `json:"skipped"`
		Errors    []struct {
			StudyID string `json:"study_id"`
			Error   string `json:"error"`
		} `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	assert.Equal(t, 0, res.Triggered)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, res.Errors)
}

func TestBulkPipelineTrigger_InFlight_CountsAsSkipped(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Simulate PHI scan in-flight.
	_, err := db.ExecContext(context.Background(),
		`UPDATE studies SET phi_scan_required = true, phi_scan_status = 'scanning' WHERE id = $1`, study.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{
		"step":      "phi_scan",
		"study_ids": []string{study.ID},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res struct {
		Triggered int `json:"triggered"`
		Skipped   int `json:"skipped"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	assert.Equal(t, 0, res.Triggered)
	assert.Equal(t, 1, res.Skipped)
}

func TestBulkPipelineTrigger_TriggeredSuccessfully(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	s2 := testutil.CreateTestStudy(t, db, proj.ID)

	// Enable QC on both studies with a completed status.
	_, err := db.ExecContext(context.Background(),
		`UPDATE studies SET qc_required = true, qc_status = 'pass' WHERE id = $1 OR id = $2`, s1.ID, s2.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{
		"step":      "qc",
		"study_ids": []string{s1.ID, s2.ID},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res struct {
		Triggered int `json:"triggered"`
		Skipped   int `json:"skipped"`
		Errors    []any `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	assert.Equal(t, 2, res.Triggered)
	assert.Equal(t, 0, res.Skipped)
	assert.Empty(t, res.Errors)
}

func TestBulkPipelineTrigger_NotFound_ReturnsError(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]any{
		"step":      "qc",
		"study_ids": []string{"00000000-0000-0000-0000-000000000000"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res struct {
		Triggered int `json:"triggered"`
		Skipped   int `json:"skipped"`
		Errors    []struct {
			StudyID string `json:"study_id"`
			Error   string `json:"error"`
		} `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	assert.Equal(t, 0, res.Triggered)
	assert.Len(t, res.Errors, 1)
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", res.Errors[0].StudyID)
}

func TestBulkPipelineTrigger_MixedResults(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	triggered := testutil.CreateTestStudy(t, db, proj.ID)
	skipped := testutil.CreateTestStudy(t, db, proj.ID) // qc not required

	// Enable QC only on the first study.
	_, err := db.ExecContext(context.Background(),
		`UPDATE studies SET qc_required = true, qc_status = 'fail' WHERE id = $1`, triggered.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{
		"step":      "qc",
		"study_ids": []string{triggered.ID, skipped.ID},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/studies/bulk-pipeline-trigger", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.BulkPipelineTrigger(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res struct {
		Triggered int   `json:"triggered"`
		Skipped   int   `json:"skipped"`
		Errors    []any `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&res))
	assert.Equal(t, 1, res.Triggered)
	assert.Equal(t, 1, res.Skipped)
	assert.Empty(t, res.Errors)
}
