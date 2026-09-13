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

// ── Export tests ──────────────────────────────────────────────────────────────

func TestExportRoutingRules_Empty(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/routing-rules/export", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ExportRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "routing-rules.json")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, proj.ID, payload["project_id"])
	assert.Equal(t, float64(0), payload["count"])
}

func TestExportRoutingRules_WithRules(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Seed two project-scoped rules.
	r1 := testutil.CreateTestRoutingRuleForProject(t, db, "Defacing Rule", "require_defacing", proj.ID)
	r2 := testutil.CreateTestRoutingRuleForProject(t, db, "PHI Scan Rule", "require_phi_scan", proj.ID)
	_ = r1
	_ = r2

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/routing-rules/export", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ExportRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var payload struct {
		ProjectID string         `json:"project_id"`
		Count     int            `json:"count"`
		Rules     []map[string]any `json:"rules"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	assert.Equal(t, proj.ID, payload.ProjectID)
	assert.Equal(t, 2, payload.Count)
	assert.Len(t, payload.Rules, 2)

	// Audit entry should have been written.
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM audit_trail WHERE action = 'routing_rule.exported' AND resource_id = $1`, proj.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// ── Import tests ──────────────────────────────────────────────────────────────

func TestImportRoutingRules_ArrayFormat(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	payload := []map[string]any{
		{"name": "Head Defacing", "action": "require_defacing", "priority": 10, "enabled": true},
		{"name": "PHI Scan All",  "action": "require_phi_scan",  "priority": 20, "enabled": true},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/routing-rules/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ImportRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, float64(2), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])
}

func TestImportRoutingRules_ExportFormat(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	payload := map[string]any{
		"project_id": "some-other-project-id",
		"count":      1,
		"rules": []map[string]any{
			{"name": "Auto Approve MRI", "action": "auto_approve", "priority": 5, "enabled": true},
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/routing-rules/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ImportRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, float64(1), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])
}

func TestImportRoutingRules_SkipsDuplicates(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Pre-create a rule with the same name.
	_ = testutil.CreateTestRoutingRuleForProject(t, db, "Existing Rule", "require_qa", proj.ID)

	payload := []map[string]any{
		{"name": "Existing Rule", "action": "require_qa",       "priority": 10, "enabled": true}, // skipped
		{"name": "New Rule",      "action": "require_defacing", "priority": 20, "enabled": true}, // imported
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/routing-rules/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ImportRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, float64(1), result["imported"])
	assert.Equal(t, float64(1), result["skipped"])
}

func TestImportRoutingRules_StripsDestinationID(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Import a rule with a (fake) destination_id — it should be stripped.
	payload := []map[string]any{
		{
			"name":           "Route To Dest",
			"action":         "route_to",
			"priority":       10,
			"enabled":        true,
			"destination_id": "00000000-0000-0000-0000-000000000001",
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/routing-rules/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ImportRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, float64(1), result["imported"])

	// Verify the imported rule has no destination_id set.
	var destID *string
	err := db.QueryRow(`SELECT destination_id FROM routing_rules WHERE name = 'Route To Dest' AND project_id = $1`, proj.ID).Scan(&destID)
	require.NoError(t, err)
	assert.Nil(t, destID, "destination_id must be stripped on import")
}

func TestImportRoutingRules_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj.ID+"/routing-rules/import",
		bytes.NewReader([]byte(`not valid json`)))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.ImportRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid JSON")
}

func TestImportRoutingRules_ProjectNotFound(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	payload := []map[string]any{{"name": "Rule", "action": "require_qa"}}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/00000000-0000-0000-0000-000000000000/routing-rules/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	srv.ImportRoutingRules(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "project not found")
}

func TestExportImport_RoundTrip(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create two project-scoped rules in proj.
	testutil.CreateTestRoutingRuleForProject(t, db, "RT Rule 1", "require_defacing", proj.ID)
	testutil.CreateTestRoutingRuleForProject(t, db, "RT Rule 2", "auto_approve", proj.ID)

	// Export from proj.
	exportReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/routing-rules/export", nil)
	exportReq.SetPathValue("id", proj.ID)
	ew := httptest.NewRecorder()
	srv.ExportRoutingRules(ew, exportReq)
	require.Equal(t, http.StatusOK, ew.Code)
	exported := ew.Body.Bytes()

	// Create a second project and import into it.
	proj2 := testutil.CreateTestProject(t, db, "Import Target")

	importReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+proj2.ID+"/routing-rules/import", bytes.NewReader(exported))
	importReq.Header.Set("Content-Type", "application/json")
	importReq.SetPathValue("id", proj2.ID)
	iw := httptest.NewRecorder()
	srv.ImportRoutingRules(iw, importReq)

	require.Equal(t, http.StatusOK, iw.Code)
	var result map[string]any
	require.NoError(t, json.NewDecoder(iw.Body).Decode(&result))
	assert.Equal(t, float64(2), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])

	// Verify rules exist in proj2.
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM routing_rules WHERE project_id = $1`, proj2.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
