package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Destinations ─────────────────────────────────────────────────────────────

func TestListDestinations_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/destinations", nil)
	rr := httptest.NewRecorder()

	srv.ListDestinations(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.Destination
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestListDestinations_ReturnsCreated(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	for _, name := range []string{"PACS East", "PACS West"} {
		body := map[string]any{
			"name":        name,
			"type":        "dicomweb",
			"dicomweb_url": "https://example.com/wado",
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/destinations", bytes.NewReader(b))
		rr := httptest.NewRecorder()
		srv.CreateDestination(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/destinations", nil)
	rr := httptest.NewRecorder()
	srv.ListDestinations(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.Destination
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestUpdateDestination_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "Original PACS")

	update := map[string]any{
		"name":         "Updated PACS",
		"type":         "dicomweb",
		"dicomweb_url": "https://updated.example.com/wado",
		"enabled":      true,
	}
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/api/destinations/"+dest.ID, bytes.NewReader(b))
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()

	srv.UpdateDestination(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.Destination
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "Updated PACS", result.Name)
	assert.Equal(t, "https://updated.example.com/wado", result.DicomwebURL)
}

func TestUpdateDestination_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	update := map[string]any{
		"name":         "X",
		"type":         "dicomweb",
		"dicomweb_url": "https://x.example.com",
	}
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/api/destinations/missing", bytes.NewReader(b))
	req.SetPathValue("id", "no-such-destination")
	rr := httptest.NewRecorder()

	srv.UpdateDestination(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "destination not found")
}

func TestDeleteDestination_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	dest := testutil.CreateTestDestination(t, db, "Delete Me PACS")

	req := httptest.NewRequest(http.MethodDelete, "/api/destinations/"+dest.ID, nil)
	req.SetPathValue("id", dest.ID)
	rr := httptest.NewRecorder()

	srv.DeleteDestination(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify gone from list
	req2 := httptest.NewRequest(http.MethodGet, "/api/destinations", nil)
	rr2 := httptest.NewRecorder()
	srv.ListDestinations(rr2, req2)
	var remaining []model.Destination
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&remaining))
	for _, d := range remaining {
		assert.NotEqual(t, dest.ID, d.ID)
	}
}

func TestDeleteDestination_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/destinations/missing", nil)
	req.SetPathValue("id", "no-such-destination")
	rr := httptest.NewRecorder()

	srv.DeleteDestination(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "destination not found")
}

// ─── Routing Rules ────────────────────────────────────────────────────────────

func TestListRoutingRules_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/routing-rules", nil)
	rr := httptest.NewRecorder()

	srv.ListRoutingRules(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.RoutingRule
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Empty(t, result)
}

func TestCreateRoutingRule_AutoApprove(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":   "Auto-approve all",
		"action": "auto_approve",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.CreateRoutingRule(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result model.RoutingRule
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "Auto-approve all", result.Name)
	assert.Equal(t, "auto_approve", result.Action)
	assert.Equal(t, 100, result.Priority) // default priority
	assert.True(t, result.Enabled)
}

func TestCreateRoutingRule_RequirePhiScan(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":     "PHI scan all external",
		"action":   "require_phi_scan",
		"priority": 10,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.CreateRoutingRule(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var result model.RoutingRule
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "require_phi_scan", result.Action)
	assert.Equal(t, 10, result.Priority)
}

func TestCreateRoutingRule_MissingName(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{"action": "auto_approve"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.CreateRoutingRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "name is required")
}

func TestCreateRoutingRule_InvalidAction(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":   "Bad rule",
		"action": "do_something_invalid",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.CreateRoutingRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid action")
}

func TestCreateRoutingRule_RouteToRequiresDestination(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := map[string]any{
		"name":   "Route without dest",
		"action": "route_to",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	srv.CreateRoutingRule(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "destination_id required")
}

func TestListRoutingRules_ReturnsCreated(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	for _, name := range []string{"Rule A", "Rule B"} {
		body := map[string]any{"name": name, "action": "require_qc_check"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/routing-rules", bytes.NewReader(b))
		rr := httptest.NewRecorder()
		srv.CreateRoutingRule(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/routing-rules", nil)
	rr := httptest.NewRecorder()
	srv.ListRoutingRules(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result []model.RoutingRule
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Len(t, result, 2)
}

func TestUpdateRoutingRule_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	rule := testutil.CreateTestRoutingRule(t, db, "Original Rule", "require_qc_check")

	update := map[string]any{
		"name":        "Updated Rule",
		"action":      "auto_approve",
		"priority":    50,
		"enabled":     false,
		"description": "Now auto-approves",
	}
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/api/routing-rules/"+rule.ID, bytes.NewReader(b))
	req.SetPathValue("id", rule.ID)
	rr := httptest.NewRecorder()

	srv.UpdateRoutingRule(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var result model.RoutingRule
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&result))
	assert.Equal(t, "Updated Rule", result.Name)
	assert.Equal(t, "auto_approve", result.Action)
	assert.Equal(t, 50, result.Priority)
	assert.False(t, result.Enabled)
}

func TestUpdateRoutingRule_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	update := map[string]any{"name": "X", "action": "auto_approve"}
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/api/routing-rules/missing", bytes.NewReader(b))
	req.SetPathValue("id", "no-such-rule")
	rr := httptest.NewRecorder()

	srv.UpdateRoutingRule(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "routing rule not found")
}

func TestDeleteRoutingRule_Handler(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	rule := testutil.CreateTestRoutingRule(t, db, "Delete Me Rule", "auto_approve")

	req := httptest.NewRequest(http.MethodDelete, "/api/routing-rules/"+rule.ID, nil)
	req.SetPathValue("id", rule.ID)
	rr := httptest.NewRecorder()

	srv.DeleteRoutingRule(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify gone
	req2 := httptest.NewRequest(http.MethodGet, "/api/routing-rules", nil)
	rr2 := httptest.NewRecorder()
	srv.ListRoutingRules(rr2, req2)
	var remaining []model.RoutingRule
	require.NoError(t, json.NewDecoder(rr2.Body).Decode(&remaining))
	for _, r := range remaining {
		assert.NotEqual(t, rule.ID, r.ID)
	}
}

func TestDeleteRoutingRule_NotFound(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodDelete, "/api/routing-rules/missing", nil)
	req.SetPathValue("id", "no-such-rule")
	rr := httptest.NewRecorder()

	srv.DeleteRoutingRule(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "routing rule not found")
}
