package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetProjectDashboardSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/summary", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetProjectDashboardSummary(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "studies")
	assert.Contains(t, w.Body.String(), "breakdown")
	assert.Contains(t, w.Body.String(), "storage")
	assert.Contains(t, w.Body.String(), "milestones")
}
