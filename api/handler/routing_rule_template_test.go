package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRoutingRuleTemplateCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Create template
	body := `{"name":"Head MRI defacing","description":"Require defacing for head MRI","action":"require_defacing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rule-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.CreateRoutingRuleTemplate(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// List templates
	req = httptest.NewRequest(http.MethodGet, "/api/routing-rule-templates", nil)
	w = httptest.NewRecorder()
	srv.ListRoutingRuleTemplates(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Head MRI defacing")
}

func TestRoutingRuleTemplate_MissingName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"action":"require_defacing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rule-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.CreateRoutingRuleTemplate(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoutingRuleTemplate_MissingAction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"name":"bad template"}`
	req := httptest.NewRequest(http.MethodPost, "/api/routing-rule-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.CreateRoutingRuleTemplate(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
