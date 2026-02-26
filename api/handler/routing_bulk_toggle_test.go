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

func TestBulkToggleRoutingRules_EmptyIDs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]interface{}{
		"rule_ids": []string{},
		"enabled":  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/bulk-toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkToggleRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkToggleRoutingRules_TooManyIDs(t *testing.T) {
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
		"rule_ids": ids,
		"enabled":  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/bulk-toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkToggleRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkToggleRoutingRules_InvalidUUID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]interface{}{
		"rule_ids": []string{"not-a-valid-uuid"},
		"enabled":  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/bulk-toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkToggleRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkToggleRoutingRules_UnknownIDsIgnored(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body, _ := json.Marshal(map[string]interface{}{
		"rule_ids": []string{"00000000-0000-0000-0000-000000000000"},
		"enabled":  false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/bulk-toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkToggleRoutingRules(w, req)

	// Unknown IDs are silently ignored — still 200 with updated=0
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Updated int      `json:"updated"`
		RuleIDs []string `json:"rule_ids"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Updated)
}

func TestBulkToggleRoutingRules_Enable(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rule := testutil.CreateTestRoutingRule(t, db, "toggle-rule", "require_qc_check")
	// Disable it first
	_, err := db.Exec(`UPDATE routing_rules SET enabled=false WHERE id=$1`, rule.ID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]interface{}{
		"rule_ids": []string{rule.ID},
		"enabled":  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/bulk-toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkToggleRoutingRules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Updated int      `json:"updated"`
		RuleIDs []string `json:"rule_ids"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Updated)
	require.Len(t, resp.RuleIDs, 1)
	assert.Equal(t, rule.ID, resp.RuleIDs[0])
}
