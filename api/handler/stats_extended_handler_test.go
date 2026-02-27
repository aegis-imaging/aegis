package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis-imaging/aegis/api/model"
	"github.com/aegis-imaging/aegis/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── GetBreakdownStats ────────────────────────────────────────────────────────

func TestGetBreakdownStats_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/breakdown", nil)
	rr := httptest.NewRecorder()
	srv.GetBreakdownStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Breakdown []any `json:"breakdown"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Empty(t, resp.Breakdown)
}

func TestGetBreakdownStats_GroupsByModalityBodyPart(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Two studies with default MRI/HEAD, one study with CT/CHEST (set via direct DB).
	testutil.CreateTestStudy(t, db, proj.ID) // MRI, HEAD
	testutil.CreateTestStudy(t, db, proj.ID) // MRI, HEAD
	s3 := testutil.CreateTestStudy(t, db, proj.ID)
	_, err := db.ExecContext(context.Background(),
		`UPDATE studies SET modality='CT', body_part='CHEST' WHERE id=$1`, s3.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/breakdown", nil)
	rr := httptest.NewRecorder()
	srv.GetBreakdownStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Breakdown []struct {
			Modality string `json:"modality"`
			BodyPart string `json:"body_part"`
			Count    int    `json:"count"`
		} `json:"breakdown"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	require.Len(t, resp.Breakdown, 2)
	// Results ordered by count DESC: MRI/HEAD (2) first, CT/CHEST (1) second.
	assert.Equal(t, "MRI", resp.Breakdown[0].Modality)
	assert.Equal(t, "HEAD", resp.Breakdown[0].BodyPart)
	assert.Equal(t, 2, resp.Breakdown[0].Count)
	assert.Equal(t, "CT", resp.Breakdown[1].Modality)
	assert.Equal(t, "CHEST", resp.Breakdown[1].BodyPart)
	assert.Equal(t, 1, resp.Breakdown[1].Count)
}

func TestGetBreakdownStats_ProjectFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create a second project and studies in each.
	proj2, err := model.CreateProject(context.Background(), db, "Project Two", "project-two", "")
	require.NoError(t, err)

	testutil.CreateTestStudy(t, db, proj.ID)  // proj
	testutil.CreateTestStudy(t, db, proj2.ID) // proj2

	// Filter to proj2 only.
	req := httptest.NewRequest(http.MethodGet, "/api/stats/breakdown?project_id="+proj2.ID, nil)
	rr := httptest.NewRecorder()
	srv.GetBreakdownStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Breakdown []struct {
			Count int `json:"count"`
		} `json:"breakdown"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Breakdown, 1)
	assert.Equal(t, 1, resp.Breakdown[0].Count)
}

// ─── GetStorageStats ──────────────────────────────────────────────────────────

func TestGetStorageStats_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/storage/stats", nil)
	rr := httptest.NewRecorder()
	srv.GetStorageStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		RawFileCount   int    `json:"raw_file_count"`
		CleanFileCount int    `json:"clean_file_count"`
		TotalFileCount int    `json:"total_file_count"`
		TotalStudies   int    `json:"total_studies"`
		TotalSizeBytes int64  `json:"total_size_bytes"`
		GeneratedAt    string `json:"generated_at"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.RawFileCount)
	assert.Equal(t, 0, resp.CleanFileCount)
	assert.Equal(t, 0, resp.TotalFileCount)
	assert.Equal(t, 0, resp.TotalStudies)
	assert.Equal(t, int64(0), resp.TotalSizeBytes)
	assert.NotEmpty(t, resp.GeneratedAt)
}

func TestGetStorageStats_CountsFilesAndSize(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// One raw study (default), one defaced/clean study.
	s1 := testutil.CreateTestStudy(t, db, proj.ID) // raw, 10 instances
	s2 := testutil.CreateTestStudy(t, db, proj.ID) // will be moved to clean
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, s2.ID, "defacing"))
	require.NoError(t, model.UpdateStudyDefaced(context.Background(), db, s2.ID))

	// Set size bytes on each study.
	require.NoError(t, model.UpdateStudySizeBytes(context.Background(), db, s1.ID, 1024*1024))   // 1 MB
	require.NoError(t, model.UpdateStudySizeBytes(context.Background(), db, s2.ID, 2*1024*1024)) // 2 MB

	req := httptest.NewRequest(http.MethodGet, "/api/storage/stats", nil)
	rr := httptest.NewRecorder()
	srv.GetStorageStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		RawFileCount   int   `json:"raw_file_count"`
		CleanFileCount int   `json:"clean_file_count"`
		TotalFileCount int   `json:"total_file_count"`
		TotalStudies   int   `json:"total_studies"`
		TotalSizeBytes int64 `json:"total_size_bytes"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 10, resp.RawFileCount)   // s1 has 10 instances in raw store
	assert.Equal(t, 10, resp.CleanFileCount) // s2 has 10 instances in clean store
	assert.Equal(t, 20, resp.TotalFileCount)
	assert.Equal(t, 2, resp.TotalStudies)
	assert.Equal(t, int64(3*1024*1024), resp.TotalSizeBytes) // 1 MB + 2 MB
}

func TestGetStorageStats_ProjectFilter(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	proj2, err := model.CreateProject(context.Background(), db, "Scoped Project", "scoped-project", "")
	require.NoError(t, err)

	testutil.CreateTestStudy(t, db, proj.ID)  // outside filter
	testutil.CreateTestStudy(t, db, proj2.ID) // inside filter

	req := httptest.NewRequest(http.MethodGet, "/api/storage/stats?project_id="+proj2.ID, nil)
	rr := httptest.NewRecorder()
	srv.GetStorageStats(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		TotalStudies int `json:"total_studies"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.TotalStudies)
}

// ─── GetTimeline ──────────────────────────────────────────────────────────────

func TestGetTimeline_Empty(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/timeline?days=7", nil)
	rr := httptest.NewRecorder()
	srv.GetTimeline(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Timeline    []any  `json:"timeline"`
		GeneratedAt string `json:"generated_at"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	// Timeline only returns rows for days that have studies — empty DB → empty slice.
	assert.Empty(t, resp.Timeline)
	assert.NotEmpty(t, resp.GeneratedAt)
}

func TestGetTimeline_StudiesCountInToday(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)

	// Create 2 received + 1 approved study today (3 total).
	s1 := testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)
	testutil.CreateTestStudy(t, db, proj.ID)
	require.NoError(t, model.UpdateStudyStatus(context.Background(), db, s1.ID, "approved"))

	req := httptest.NewRequest(http.MethodGet, "/api/stats/timeline?days=3", nil)
	rr := httptest.NewRecorder()
	srv.GetTimeline(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Timeline []struct {
			Date     string `json:"date"`
			Received int    `json:"received"`
			Approved int    `json:"approved"`
		} `json:"timeline"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	// Only days with studies are returned — all 3 created today → 1 row.
	require.Len(t, resp.Timeline, 1)
	today := resp.Timeline[0]
	assert.Equal(t, 3, today.Received) // all 3 created today (count(*) regardless of status)
	assert.Equal(t, 1, today.Approved)
}

func TestStatsExtended_ResearcherRequiresProjectScope(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "stats-ext-researcher@test.com", "researcher")

	tests := []struct {
		name string
		path string
		h    func(http.ResponseWriter, *http.Request)
	}{
		{name: "breakdown", path: "/api/stats/breakdown", h: srv.GetBreakdownStats},
		{name: "storage", path: "/api/storage/stats", h: srv.GetStorageStats},
		{name: "timeline", path: "/api/stats/timeline", h: srv.GetTimeline},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req = withResearcherUser(req, researcher.ID, researcher.Email)
			rr := httptest.NewRecorder()

			tc.h(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "project_id is required for researcher queries")
		})
	}
}

func TestStatsExtended_ResearcherWithProjectScopeWithoutMembershipDenied(t *testing.T) {
	db := testutil.TestDB(t)
	srv := testutil.TestServer(t, db)
	proj := testutil.SeedProject(t, db)
	researcher := testutil.CreateTestAdminUser(t, db, "stats-ext-denied@test.com", "researcher")

	tests := []struct {
		name string
		path string
		h    func(http.ResponseWriter, *http.Request)
	}{
		{name: "breakdown", path: "/api/stats/breakdown?project_id=" + proj.ID, h: srv.GetBreakdownStats},
		{name: "storage", path: "/api/storage/stats?project_id=" + proj.ID, h: srv.GetStorageStats},
		{name: "timeline", path: "/api/stats/timeline?project_id=" + proj.ID, h: srv.GetTimeline},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req = withResearcherUser(req, researcher.ID, researcher.Email)
			rr := httptest.NewRecorder()

			tc.h(rr, req)

			assert.Equal(t, http.StatusNotFound, rr.Code)
			assert.Contains(t, rr.Body.String(), "project not found")
		})
	}
}
