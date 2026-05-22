package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditAnnotation_CreateListDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Seed an audit entry so we have an audit_id to attach to.
	require.NoError(t, model.CreateAuditEntry(context.Background(), db,
		"study.created", "alice@test.com", "study", study.ID, "127.0.0.1", nil))
	entries, err := model.ListAuditEntries(context.Background(), db, model.AuditFilters{}, 1, 0)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "expected at least one audit entry seeded")
	auditID := entries[0].ID

	createReq := httptest.NewRequest(http.MethodPost, "/api/audit/"+auditID+"/annotations",
		strings.NewReader(`{"body":"reviewed by compliance"}`))
	createReq.SetPathValue("id", auditID)
	createRR := httptest.NewRecorder()
	srv.CreateAuditAnnotation(createRR, createReq)
	require.Equal(t, http.StatusCreated, createRR.Code, createRR.Body.String())

	var created model.AuditAnnotation
	require.NoError(t, json.NewDecoder(createRR.Body).Decode(&created))
	assert.Equal(t, "reviewed by compliance", created.Body)
	assert.NotEmpty(t, created.ID)

	listReq := httptest.NewRequest(http.MethodGet, "/api/audit/"+auditID+"/annotations", nil)
	listReq.SetPathValue("id", auditID)
	listRR := httptest.NewRecorder()
	srv.ListAuditAnnotations(listRR, listReq)
	assert.Equal(t, http.StatusOK, listRR.Code)

	var anns []model.AuditAnnotation
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&anns))
	require.Len(t, anns, 1)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/audit/"+auditID+"/annotations/"+created.ID, nil)
	delReq.SetPathValue("annotationID", created.ID)
	delRR := httptest.NewRecorder()
	srv.DeleteAuditAnnotation(delRR, delReq)
	assert.Equal(t, http.StatusOK, delRR.Code)
}

func TestAuditAnnotation_EmptyBodyRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/audit/abc/annotations", strings.NewReader(`{"body":"   "}`))
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	rr := httptest.NewRecorder()
	srv.CreateAuditAnnotation(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
