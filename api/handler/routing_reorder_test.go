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

func TestReorderRoutingRules_Basic(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	r1 := testutil.CreateTestRoutingRule(t, db, "Rule A", "require_qa")
	r2 := testutil.CreateTestRoutingRule(t, db, "Rule B", "require_defacing")

	// Swap priorities.
	payload := map[string]any{
		"rules": []map[string]any{
			{"id": r1.ID, "priority": 50},
			{"id": r2.ID, "priority": 5},
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/reorder", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ReorderRoutingRules(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Updated int `json:"updated"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, 2, resp.Updated)

	// Verify new priorities in DB.
	var p1, p2 int
	require.NoError(t, db.QueryRow(`SELECT priority FROM routing_rules WHERE id = $1`, r1.ID).Scan(&p1))
	require.NoError(t, db.QueryRow(`SELECT priority FROM routing_rules WHERE id = $1`, r2.ID).Scan(&p2))
	assert.Equal(t, 50, p1)
	assert.Equal(t, 5, p2)

	// Audit entry should have been written.
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM audit_trail WHERE action = 'routing_rule.reordered'`).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestReorderRoutingRules_EmptyArray(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	payload := map[string]any{"rules": []map[string]any{}}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/reorder", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ReorderRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "non-empty")
}

func TestReorderRoutingRules_DuplicateIDs(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	r := testutil.CreateTestRoutingRule(t, db, "Dup Rule", "require_qa")

	payload := map[string]any{
		"rules": []map[string]any{
			{"id": r.ID, "priority": 10},
			{"id": r.ID, "priority": 20}, // duplicate
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/reorder", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ReorderRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate")
}

func TestReorderRoutingRules_NotFound(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	payload := map[string]any{
		"rules": []map[string]any{
			{"id": "00000000-0000-0000-0000-000000000001", "priority": 10},
		},
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/reorder", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ReorderRoutingRules(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestReorderRoutingRules_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/routing-rules/reorder",
		bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ReorderRoutingRules(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
