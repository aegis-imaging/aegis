package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerProtocolCheck_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/no.uid/protocol-check", nil)
	req.SetPathValue("studyUID", "no.uid")
	rr := httptest.NewRecorder()

	srv.TriggerProtocolCheck(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestTriggerProtocolCheck_NotRequired(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/protocol-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerProtocolCheck(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "does not require protocol check")
}

func TestTriggerProtocolCheck_AlreadyInProgress(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetProtocolRequired(t.Context(), db, study.ID, true))
	require.NoError(t, model.UpdateProtocolStatus(t.Context(), db, study.ID, "checking"))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/protocol-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerProtocolCheck(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "already in progress")
}

func TestTriggerProtocolCheck_NoTemplates(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetProtocolRequired(t.Context(), db, study.ID, true))
	// No protocol templates seeded for this project

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/protocol-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerProtocolCheck(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "no protocol templates configured")
}

func TestTriggerProtocolCheck_ServiceNotConfigured(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db) // no PROTOCOL_SERVICE_URL
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetProtocolRequired(t.Context(), db, study.ID, true))

	// Seed a protocol template so the templates check passes
	tmpl := &model.ProtocolTemplate{
		ProjectID: proj.ID,
		Name:      "T1w Template",
		Enabled:   true,
		Rules:     json.RawMessage(`[{"tag_keyword":"RepetitionTime","match_type":"numeric","target":2000}]`),
	}
	require.NoError(t, model.CreateProtocolTemplate(t.Context(), db, tmpl))

	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/protocol-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerProtocolCheck(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, "checking", body["status"])

	updated, err := model.GetStudyByID(t.Context(), db, study.ID)
	require.NoError(t, err)
	assert.Equal(t, "checking", updated.ProtocolStatus)
}

func TestTriggerProtocolCheck_WithMockService(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetProtocolRequired(t.Context(), db, study.ID, true))

	tmpl := &model.ProtocolTemplate{
		ProjectID: proj.ID,
		Name:      "Test Template",
		Enabled:   true,
		Rules:     json.RawMessage(`[{"tag_keyword":"RepetitionTime","match_type":"numeric","target":2000}]`),
	}
	require.NoError(t, model.CreateProtocolTemplate(t.Context(), db, tmpl))

	svc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/check", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"study_uid":          study.StudyInstanceUID,
			"status":             "complete",
			"overall_compliance": "compliant",
			"findings":           []any{},
			"tool_used":          "basic",
		})
	}))
	defer svc.Close()

	srv := sidecarServer(t, db, "protocol", svc.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/studies/"+study.StudyInstanceUID+"/protocol-check", nil)
	req.SetPathValue("studyUID", study.StudyInstanceUID)
	rr := httptest.NewRecorder()

	srv.TriggerProtocolCheck(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)

	// No DICOM files in test store → goroutine sets protocol_status to "failed"
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s, err := model.GetStudyByID(context.Background(), db, study.ID)
		if err == nil && s.ProtocolStatus == "failed" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("timed out waiting for protocol_status=failed")
}
