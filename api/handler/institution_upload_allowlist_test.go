package handler_test

import (
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

func TestUploadAllowlist_DefaultOn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "allowlist-default")

	req := httptest.NewRequest(http.MethodGet, "/api/institutions/"+inst.ID+"/upload-allowlist", nil)
	req.SetPathValue("id", inst.ID)
	w := httptest.NewRecorder()
	srv.ListInstitutionUploadAllowlist(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Entries []model.EffectiveAllowlistEntry `json:"entries"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Entries, len(model.KnownUploadMethods))
	for _, e := range resp.Entries {
		assert.True(t, e.Enabled, "method %s should default to enabled", e.Method.ID)
		assert.True(t, e.IsDefault, "method %s should be marked as default", e.Method.ID)
	}
}

func TestUploadAllowlist_DenyAndRevert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "allowlist-deny")

	// PUT a deny for browser.dimse-pull
	body := `{"enabled":false,"note":"site policy"}`
	req := httptest.NewRequest(http.MethodPut,
		"/api/institutions/"+inst.ID+"/upload-allowlist/browser.dimse-pull",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", inst.ID)
	req.SetPathValue("methodID", "browser.dimse-pull")
	w := httptest.NewRecorder()
	srv.PutInstitutionUploadAllowlist(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// IsUploadMethodAllowed should now return false for that method
	// and true for the others.
	allowed, err := model.IsUploadMethodAllowed(req.Context(), db, inst.ID, "browser.dimse-pull")
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = model.IsUploadMethodAllowed(req.Context(), db, inst.ID, "browser.web-upload")
	require.NoError(t, err)
	assert.True(t, allowed)

	// DELETE the row → reverts to default-on
	req = httptest.NewRequest(http.MethodDelete,
		"/api/institutions/"+inst.ID+"/upload-allowlist/browser.dimse-pull", nil)
	req.SetPathValue("id", inst.ID)
	req.SetPathValue("methodID", "browser.dimse-pull")
	w = httptest.NewRecorder()
	srv.DeleteInstitutionUploadAllowlist(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	allowed, err = model.IsUploadMethodAllowed(req.Context(), db, inst.ID, "browser.dimse-pull")
	require.NoError(t, err)
	assert.True(t, allowed, "should revert to default-on after delete")
}

func TestUploadAllowlist_UnknownMethodRejected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "allowlist-unknown")

	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPut,
		"/api/institutions/"+inst.ID+"/upload-allowlist/not-a-method",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", inst.ID)
	req.SetPathValue("methodID", "not-a-method")
	w := httptest.NewRecorder()
	srv.PutInstitutionUploadAllowlist(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// IsUploadMethodAllowed also rejects (rather than silently allowing)
	_, err := model.IsUploadMethodAllowed(req.Context(), db, inst.ID, "not-a-method")
	require.Error(t, err)
}

func TestUploadAllowlist_CrossInstitutionIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	instA := testutil.CreateTestInstitution(t, db, "allowlist-iso-a")
	instB := testutil.CreateTestInstitution(t, db, "allowlist-iso-b")

	// Deny dimse-pull on A only.
	body := `{"enabled":false}`
	req := httptest.NewRequest(http.MethodPut,
		"/api/institutions/"+instA.ID+"/upload-allowlist/browser.dimse-pull",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", instA.ID)
	req.SetPathValue("methodID", "browser.dimse-pull")
	w := httptest.NewRecorder()
	srv.PutInstitutionUploadAllowlist(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// B should remain default-on.
	allowed, err := model.IsUploadMethodAllowed(req.Context(), db, instB.ID, "browser.dimse-pull")
	require.NoError(t, err)
	assert.True(t, allowed, "deny on A must not affect B")
}
