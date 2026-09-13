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

func TestGetRetentionPreview_Empty(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/retention-preview", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetRetentionPreview(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		ProjectID        string `json:"project_id"`
		PreviewDays      int    `json:"preview_days"`
		WouldExpireCount int    `json:"would_expire_count"`
		TotalApproved    int    `json:"total_approved"`
		AgeDistribution  []struct {
			Label       string `json:"label"`
			MinDays     int    `json:"min_days"`
			Count       int    `json:"count"`
			WouldExpire bool   `json:"would_expire"`
		} `json:"age_distribution"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, proj.ID, resp.ProjectID)
	assert.Equal(t, 90, resp.PreviewDays) // default
	assert.Equal(t, 0, resp.WouldExpireCount)
	assert.Equal(t, 0, resp.TotalApproved)
	assert.Len(t, resp.AgeDistribution, 6) // 6 age buckets
}

func TestGetRetentionPreview_WithApprovedStudy(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.Exec(`UPDATE studies SET status='approved' WHERE id=$1`, study.ID)
	require.NoError(t, err)

	// Study was created ~now — should be in the 0-7 day bucket.
	// With preview_days=365, none would expire.
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/retention-preview?days=365", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetRetentionPreview(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		PreviewDays      int `json:"preview_days"`
		WouldExpireCount int `json:"would_expire_count"`
		TotalApproved    int `json:"total_approved"`
		AgeDistribution  []struct {
			Label       string `json:"label"`
			MinDays     int    `json:"min_days"`
			Count       int    `json:"count"`
			WouldExpire bool   `json:"would_expire"`
		} `json:"age_distribution"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 365, resp.PreviewDays)
	assert.Equal(t, 0, resp.WouldExpireCount) // none would expire at 365 days
	assert.Equal(t, 1, resp.TotalApproved)

	// The 0-7d bucket should have count=1 (study was just created).
	assert.Equal(t, "0–7 days", resp.AgeDistribution[0].Label)
	assert.Equal(t, 1, resp.AgeDistribution[0].Count)
	assert.False(t, resp.AgeDistribution[0].WouldExpire) // 0 < 365, so won't expire
}

func TestGetRetentionPreview_CustomDays(t *testing.T) {
	if testing.Short() {
		t.Log("integration test")
		t.Skip()
	}

	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	study := testutil.CreateTestStudy(t, db, proj.ID)
	// Set study to approved and back-date it by 100 days so it falls into the 91-180 bucket.
	_, err := db.Exec(
		`UPDATE studies SET status='approved', created_at=now()-interval'100 days' WHERE id=$1`,
		study.ID)
	require.NoError(t, err)

	// With days=90, the study (created 100 days ago) WOULD expire.
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+proj.ID+"/retention-preview?days=90", nil)
	req.SetPathValue("id", proj.ID)
	w := httptest.NewRecorder()
	srv.GetRetentionPreview(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		WouldExpireCount int `json:"would_expire_count"`
		TotalApproved    int `json:"total_approved"`
		AgeDistribution  []struct {
			Label       string `json:"label"`
			MinDays     int    `json:"min_days"`
			Count       int    `json:"count"`
			WouldExpire bool   `json:"would_expire"`
		} `json:"age_distribution"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.WouldExpireCount) // 100 days old > 90-day retention
	assert.Equal(t, 1, resp.TotalApproved)

	// The 91-180d bucket should have count=1 and WouldExpire=true.
	bucket91 := resp.AgeDistribution[3] // "91–180 days"
	assert.Equal(t, "91–180 days", bucket91.Label)
	assert.Equal(t, 1, bucket91.Count)
	assert.True(t, bucket91.WouldExpire) // 91 >= 90
}
