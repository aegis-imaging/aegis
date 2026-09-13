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

func TestBulkToggleRoutingRules_Disable(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	rule := testutil.CreateTestRoutingRule(t, db, "disable-rule", "require_phi_scan")
	// Rule starts enabled (default from CreateTestRoutingRule)

	body, _ := json.Marshal(map[string]interface{}{
		"rule_ids": []string{rule.ID},
		"enabled":  false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/bulk-toggle", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.BulkToggleRoutingRules(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Updated int `json:"updated"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Updated)

	// Verify in DB
	var enabled bool
	err := db.QueryRow(`SELECT enabled FROM routing_rules WHERE id=$1`, rule.ID).Scan(&enabled)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestBulkToggleRoutingRules_MultipleRules(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	r1 := testutil.CreateTestRoutingRule(t, db, "multi-toggle-1", "require_qc_check")
	r2 := testutil.CreateTestRoutingRule(t, db, "multi-toggle-2", "require_phi_scan")
	r3 := testutil.CreateTestRoutingRule(t, db, "multi-toggle-3", "require_defacing")

	// Disable r1 and r2, leave r3 enabled
	body, _ := json.Marshal(map[string]interface{}{
		"rule_ids": []string{r1.ID, r2.ID},
		"enabled":  false,
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
	assert.Equal(t, 2, resp.Updated)
	assert.Len(t, resp.RuleIDs, 2)

	// Verify r3 is still enabled
	var enabled bool
	err := db.QueryRow(`SELECT enabled FROM routing_rules WHERE id=$1`, r3.ID).Scan(&enabled)
	require.NoError(t, err)
	assert.True(t, enabled)
}
