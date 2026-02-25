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

// simulateResp mirrors the simulateResponse struct for decoding test output.
type simulateResp struct {
	Input struct {
		Modality  string `json:"modality"`
		BodyPart  string `json:"body_part"`
		Source    string `json:"source"`
		ProjectID string `json:"project_id"`
	} `json:"input"`
	MatchedRules []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Action string `json:"action"`
	} `json:"matched_rules"`
	SkippedRules []struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	} `json:"skipped_rules"`
	ActionSummary struct {
		RequireDefacing       bool `json:"require_defacing"`
		RequirePhiScan        bool `json:"require_phi_scan"`
		RequireQcCheck        bool `json:"require_qc_check"`
		RequireBidsConversion bool `json:"require_bids_conversion"`
		RequireClassification bool `json:"require_classification"`
		RequireProtocolCheck  bool `json:"require_protocol_check"`
		RequireExport         bool `json:"require_export"`
		AutoApprove           bool `json:"auto_approve"`
		Reject                bool `json:"reject"`
	} `json:"action_summary"`
}

func TestSimulateRoutingRules_NoRules(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := bytes.NewBufferString(`{"modality":"MRI","body_part":"HEAD","source":"external"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.SimulateRoutingRules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp simulateResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp.MatchedRules)
	assert.Equal(t, "MRI", resp.Input.Modality)
	assert.Equal(t, "HEAD", resp.Input.BodyPart)
	assert.Equal(t, "external", resp.Input.Source)
}

func TestSimulateRoutingRules_MatchesAllModality(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Rule with no conditions (matches everything): action = require_defacing.
	testutil.CreateTestRoutingRule(t, db, "Deface all", "require_defacing")

	body := bytes.NewBufferString(`{"modality":"CT","body_part":"CHEST","source":"external"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.SimulateRoutingRules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp simulateResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.MatchedRules, 1)
	assert.Equal(t, "require_defacing", resp.MatchedRules[0].Action)
	assert.True(t, resp.ActionSummary.RequireDefacing)
}

func TestSimulateRoutingRules_ConditionFilter(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create a rule that only matches MRI + HEAD.
	modality := "MRI"
	bodyPart := "HEAD"
	_, err := db.Exec(`
		INSERT INTO routing_rules (name, description, priority, enabled, modality, body_part, action)
		VALUES ('MRI HEAD deface', '', 10, true, $1, $2, 'require_defacing')`,
		modality, bodyPart)
	require.NoError(t, err)

	// Create a rule that matches MRI only.
	_, err = db.Exec(`
		INSERT INTO routing_rules (name, description, priority, enabled, modality, action)
		VALUES ('MRI phi scan', '', 20, true, $1, 'require_phi_scan')`, modality)
	require.NoError(t, err)

	// Query 1: MRI HEAD should match both.
	body1 := bytes.NewBufferString(`{"modality":"MRI","body_part":"HEAD","source":"external"}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body1)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.SimulateRoutingRules(w1, req1)

	require.Equal(t, http.StatusOK, w1.Code)
	var resp1 simulateResp
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &resp1))
	assert.Len(t, resp1.MatchedRules, 2)
	assert.True(t, resp1.ActionSummary.RequireDefacing)
	assert.True(t, resp1.ActionSummary.RequirePhiScan)

	// Query 2: MRI CHEST should match only phi scan rule (HEAD condition fails for defacing rule).
	body2 := bytes.NewBufferString(`{"modality":"MRI","body_part":"CHEST","source":"external"}`)
	req2 := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body2)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.SimulateRoutingRules(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var resp2 simulateResp
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	assert.Len(t, resp2.MatchedRules, 1)
	assert.Equal(t, "require_phi_scan", resp2.MatchedRules[0].Action)
	assert.False(t, resp2.ActionSummary.RequireDefacing)
	assert.True(t, resp2.ActionSummary.RequirePhiScan)

	// Query 3: CT should match neither rule.
	body3 := bytes.NewBufferString(`{"modality":"CT","body_part":"HEAD","source":"external"}`)
	req3 := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body3)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	srv.SimulateRoutingRules(w3, req3)

	require.Equal(t, http.StatusOK, w3.Code)
	var resp3 simulateResp
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp3))
	assert.Empty(t, resp3.MatchedRules)
	assert.False(t, resp3.ActionSummary.RequireDefacing)
	assert.False(t, resp3.ActionSummary.RequirePhiScan)
}

func TestSimulateRoutingRules_RouteToResolvesDestName(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	dest := testutil.CreateTestDestination(t, db, "AWS Receiver")

	_, err := db.Exec(`
		INSERT INTO routing_rules (name, description, priority, enabled, action, destination_id)
		VALUES ('Route to AWS', '', 10, true, 'route_to', $1)`, dest.ID)
	require.NoError(t, err)

	body := bytes.NewBufferString(`{"modality":"MRI","body_part":"HEAD","source":"external"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.SimulateRoutingRules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		MatchedRules []struct {
			Action          string `json:"action"`
			DestinationID   string `json:"destination_id"`
			DestinationName string `json:"destination_name"`
		} `json:"matched_rules"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.MatchedRules, 1)
	assert.Equal(t, "route_to", resp.MatchedRules[0].Action)
	assert.Equal(t, dest.ID, resp.MatchedRules[0].DestinationID)
	assert.Equal(t, "AWS Receiver", resp.MatchedRules[0].DestinationName)
}

func TestSimulateRoutingRules_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/simulate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.SimulateRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
