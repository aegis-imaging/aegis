package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/config"
	"github.com/aegis-imaging/aegis/api/handler"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/storage"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDiagnosticsServer(t *testing.T, db *sql.DB, dimseURL string) *handler.Server {
	t.Helper()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Port:             "0",
		StorageMode:      "local",
		LocalStorageDir:  tmpDir,
		APIBaseURL:       "http://localhost:8080",
		PipelineAuto:     false,
		AuthEnabled:      false,
		DevUserEmail:     "test@aegis.local",
		AllowedOrigins:   []string{"http://localhost:3000"},
		DimseReceiverURL: dimseURL,
	}

	store := storage.NewLocal(tmpDir, cfg.APIBaseURL)
	return handler.NewServer(db, store, cfg)
}

func TestGetStudyDiagnostics_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/missing/diagnostics", nil)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()

	srv.GetStudyDiagnostics(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "study not found")
}

func TestGetStudyDiagnostics_BuildsSummary(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	require.NoError(t, model.SetClassificationRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.UpdateClassificationStatus(context.Background(), db, study.ID, "failed"))
	require.NoError(t, model.SetPhiScanRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.SetQcRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.UpdateQcStatus(context.Background(), db, study.ID, "failed"))
	require.NoError(t, model.SetProtocolRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.SetExportRequired(context.Background(), db, study.ID, true))
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, study.ID, "approved"))

	require.NoError(t, model.CreateAuditEntry(
		context.Background(), db, "qc_check.failed", "system", "study", study.ID, "", map[string]any{"reason": "mock"},
	))
	rule := testutil.CreateTestRoutingRule(t, db, "Require QC", "require_qc_check")
	require.NoError(t, model.CreateRoutingLogEntry(context.Background(), db, &model.RoutingLogEntry{
		StudyID: study.ID,
		RuleID:  rule.ID,
		Action:  "require_qc_check",
		Outcome: "qc required",
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/diagnostics", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudyDiagnostics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var payload struct {
		Summary struct {
			Stuck              bool     `json:"stuck"`
			Blockers           []string `json:"blockers"`
			RecommendedActions []string `json:"recommended_actions"`
			LastAuditAction    string   `json:"last_audit_action"`
		} `json:"summary"`
		RecentAudit []model.AuditEntry      `json:"recent_audit"`
		RoutingLog  []model.RoutingLogEntry `json:"routing_log"`
		DimseRetry  struct {
			Available bool `json:"available"`
		} `json:"dimse_retry"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&payload))

	assert.True(t, payload.Summary.Stuck)
	blockers := strings.Join(payload.Summary.Blockers, " | ")
	assert.Contains(t, blockers, "Classification failed")
	assert.Contains(t, blockers, "PHI scan is pending")
	assert.Contains(t, blockers, "QC check failed")
	assert.Equal(t, "qc_check.failed", payload.Summary.LastAuditAction)
	assert.NotEmpty(t, payload.Summary.RecommendedActions)
	assert.False(t, payload.DimseRetry.Available)
	assert.NotEmpty(t, payload.RecentAudit)
	assert.NotEmpty(t, payload.RoutingLog)
}

func TestGetStudyDiagnostics_ReportsDimseSignals(t *testing.T) {
	db := testutil.TestDB(t)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The sidecar health loop probes /healthz on startup; ignore those requests.
		if r.URL.Path == "/healthz" || r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		assert.Equal(t, "/ingest/retry/details", r.URL.Path)
		assert.Equal(t, study.StudyInstanceUID, r.URL.Query().Get("study_instance_uid"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"ok",
			"ingest_retry":{
				"pending_total":1,
				"dead_letter_total":1,
				"pending_items":[{"study_instance_uid":"` + study.StudyInstanceUID + `"}],
				"dead_letter_items":[{"study_instance_uid":"` + study.StudyInstanceUID + `"}]
			}
		}`))
	}))
	defer upstream.Close()

	srv := testDiagnosticsServer(t, db, upstream.URL)
	req := httptest.NewRequest(http.MethodGet, "/api/studies/"+study.ID+"/diagnostics", nil)
	req.SetPathValue("id", study.ID)
	rr := httptest.NewRecorder()
	srv.GetStudyDiagnostics(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var payload struct {
		Summary struct {
			Blockers           []string `json:"blockers"`
			RecommendedActions []string `json:"recommended_actions"`
		} `json:"summary"`
		DimseRetry struct {
			Available       bool `json:"available"`
			PendingTotal    int  `json:"pending_total"`
			DeadLetterTotal int  `json:"dead_letter_total"`
		} `json:"dimse_retry"`
	}
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&payload))

	assert.True(t, payload.DimseRetry.Available)
	assert.Equal(t, 1, payload.DimseRetry.PendingTotal)
	assert.Equal(t, 1, payload.DimseRetry.DeadLetterTotal)
	assert.Contains(t, strings.Join(payload.Summary.Blockers, " | "), "DIMSE retry queue has 1 pending item")
	assert.Contains(t, strings.Join(payload.Summary.Blockers, " | "), "DIMSE dead-letter queue has 1 item")
	actions := strings.Join(payload.Summary.RecommendedActions, " | ")
	assert.Contains(t, actions, "/api/dimse/retry/process/"+study.StudyInstanceUID)
	assert.Contains(t, actions, "/api/dimse/retry/replay/"+study.StudyInstanceUID)
}
