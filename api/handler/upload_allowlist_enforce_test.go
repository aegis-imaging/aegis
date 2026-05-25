package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnforceUploadMethod_DefaultAllows covers the default-on contract:
// an institution with no allowlist rows lets every method through. Upload
// init is the path most users hit, so we exercise it end-to-end through
// the handler rather than the helper directly.
func TestEnforceUploadMethod_DefaultAllows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "enforce-default")
	defaultProject, err := model.GetProjectBySlug(context.Background(), db, "default")
	require.NoError(t, err)
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db,
		&model.InstitutionProject{InstitutionID: inst.ID, ProjectID: defaultProject.ID, Role: "sender"}))

	body := `{
		"project_slug": "default",
		"file_count": 1,
		"institution_id": "` + inst.ID + `"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.UploadInit(w, req)

	require.NotEqual(t, http.StatusForbidden, w.Code, "default-on institution should NOT be blocked: %s", w.Body.String())
}

// TestEnforceUploadMethod_ExplicitDenyBlocksUpload verifies the
// admin-flipped row actually denies. PUT enabled=false for
// browser.web-upload on institution X, then watch a request attributed
// to X return 403.
func TestEnforceUploadMethod_ExplicitDenyBlocksUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	inst := testutil.CreateTestInstitution(t, db, "enforce-deny")
	defaultProject, err := model.GetProjectBySlug(context.Background(), db, "default")
	require.NoError(t, err)
	require.NoError(t, model.AddInstitutionToProject(context.Background(), db,
		&model.InstitutionProject{InstitutionID: inst.ID, ProjectID: defaultProject.ID, Role: "sender"}))

	// Deny browser.web-upload for this institution.
	err = model.UpsertUploadAllowlistRow(context.Background(), db, &model.InstitutionUploadAllowlistRow{
		InstitutionID: inst.ID,
		MethodID:      "browser.web-upload",
		Enabled:       false,
		Note:          "denied for the test",
		UpdatedBy:     "test@example.com",
	})
	require.NoError(t, err)

	body := `{
		"project_slug": "default",
		"file_count": 1,
		"institution_id": "` + inst.ID + `"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.UploadInit(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "explicit deny should produce 403: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "disabled for your institution")
}

// TestEnforceUploadMethod_NoInstitutionDefaultAllow guards the legacy
// path: a request that doesn't attribute any institution should pass
// through unchanged. The allowlist only constrains attributed flows.
func TestEnforceUploadMethod_NoInstitutionDefaultAllow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	body := `{"project_slug": "default", "file_count": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.UploadInit(w, req)

	require.NotEqual(t, http.StatusForbidden, w.Code, "unattributed upload should not 403: %s", w.Body.String())
}
