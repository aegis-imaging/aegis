package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpiringStudies_Empty(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	// Default project has no retention_days → no expiring studies.
	req := httptest.NewRequest(http.MethodGet, "/api/studies/expiring", nil)
	w := httptest.NewRecorder()
	srv.GetExpiringStudies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Studies   []any `json:"studies"`
		Total     int   `json:"total"`
		Days      int   `json:"days"`
		Truncated bool  `json:"truncated"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.Days)
	assert.Equal(t, 0, resp.Total)
	assert.False(t, resp.Truncated)
}

func TestGetExpiringStudies_WithRetentionPolicy(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Set study to approved and project retention to 1 day.
	// The study was created "now", so it expires in ~1 day → within the default 7-day window.
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE projects SET retention_days=1 WHERE id=$1`, proj.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/expiring?project_id="+proj.ID+"&days=7", nil)
	w := httptest.NewRecorder()
	srv.GetExpiringStudies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Studies []struct {
			ID              string `json:"id"`
			RetentionDays   int    `json:"retention_days"`
			ExpiresAt       string `json:"expires_at"`
			DaysUntilExpiry int    `json:"days_until_expiry"`
		} `json:"studies"`
		Total int `json:"total"`
		Days  int `json:"days"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.Days)
	require.Equal(t, 1, resp.Total)
	assert.Equal(t, study.ID, resp.Studies[0].ID)
	assert.Equal(t, 1, resp.Studies[0].RetentionDays)
	assert.NotEmpty(t, resp.Studies[0].ExpiresAt)
	assert.Equal(t, 0, resp.Studies[0].DaysUntilExpiry) // created now + 1 day = 0 full days remaining
}

func TestGetExpiringStudies_OutsideWindow(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)

	// Study approved, retention=365 days from now → expires in ~365 days.
	// With days=7 window, it should NOT appear.
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE projects SET retention_days=365 WHERE id=$1`, proj.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/studies/expiring?project_id="+proj.ID+"&days=7", nil)
	w := httptest.NewRecorder()
	srv.GetExpiringStudies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Total)
}
