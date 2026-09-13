package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type statsResponse struct {
	StudyCounts  model.StudyStatusCounts `json:"study_counts"`
	ActiveShares int                     `json:"active_shares"`
	GeneratedAt  time.Time               `json:"generated_at"`
}

type institutionAttributionStatsResponse struct {
	Days              int                                      `json:"days"`
	TotalRestricted   int                                      `json:"total_restricted_studies"`
	UnattributedTotal int                                      `json:"unattributed_studies"`
	UnattributedPct   float64                                  `json:"unattributed_pct"`
	Projects          []model.InstitutionAttributionProjectRow `json:"projects"`
	GeneratedAt       time.Time                                `json:"generated_at"`
}

// GetStats returns a lightweight snapshot of study pipeline state and active
// share count. Used by the admin dashboard overview banner.
// Accepts optional ?project_id= to scope counts to a single project.
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil {
		projectID = access.ProjectID
		if access.IsSiteScoped() {
			institutionID = *access.InstitutionID
		}
	}

	var counts model.StudyStatusCounts
	var err error
	if access != nil {
		counts, err = model.GetStudyStatusCountsForScope(r.Context(), s.db, projectID, institutionID)
	} else {
		counts, err = model.GetStudyStatusCounts(r.Context(), s.db, projectID)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query study counts")
		return
	}

	var activeShares int
	if access != nil {
		activeShares, err = model.CountAllExportSharesForScope(r.Context(), s.db, model.ShareStatusActive, projectID, institutionID)
	} else {
		activeShares, err = model.CountAllExportShares(r.Context(), s.db, model.ShareStatusActive, projectID)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query share counts")
		return
	}

	s.writeJSON(w, http.StatusOK, statsResponse{
		StudyCounts:  counts,
		ActiveShares: activeShares,
		GeneratedAt:  time.Now().UTC(),
	})
}

// GetBreakdownStats returns study counts grouped by modality and body part.
// GET /api/stats/breakdown  — accepts optional ?project_id=
func (s *Server) GetBreakdownStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil {
		projectID = access.ProjectID
		if access.IsSiteScoped() {
			institutionID = *access.InstitutionID
		}
	}

	var rows []model.BreakdownRow
	var err error
	if access != nil {
		rows, err = model.GetStudyBreakdownForScope(r.Context(), s.db, projectID, institutionID)
	} else {
		rows, err = model.GetStudyBreakdown(r.Context(), s.db, projectID)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query breakdown stats")
		return
	}
	if rows == nil {
		rows = []model.BreakdownRow{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"breakdown":    rows,
		"generated_at": time.Now().UTC(),
	})
}

// GetStorageStats returns aggregate DICOM file counts derived from the studies table.
// GET /api/storage/stats  — accepts optional ?project_id=
func (s *Server) GetStorageStats(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil {
		projectID = access.ProjectID
		if access.IsSiteScoped() {
			institutionID = *access.InstitutionID
		}
	}

	var stats *model.StorageStats
	var err error
	if access != nil {
		stats, err = model.GetStorageStatsForScope(r.Context(), s.db, projectID, institutionID)
	} else {
		stats, err = model.GetStorageStats(r.Context(), s.db, projectID)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query storage stats")
		return
	}
	stats.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	s.writeJSON(w, http.StatusOK, stats)
}

// GetTimeline returns daily study ingestion counts.
// GET /api/stats/timeline  — accepts ?days=30 (default) and ?project_id=
func (s *Server) GetTimeline(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	days, _ := strconv.Atoi(q.Get("days"))
	projectID := q.Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil {
		projectID = access.ProjectID
		if access.IsSiteScoped() {
			institutionID = *access.InstitutionID
		}
	}

	var rows []model.TimelineDay
	var err error
	if access != nil {
		rows, err = model.GetStudyTimelineForScope(r.Context(), s.db, days, projectID, institutionID)
	} else {
		rows, err = model.GetStudyTimeline(r.Context(), s.db, days, projectID)
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query timeline")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"timeline":     rows,
		"generated_at": time.Now().UTC(),
	})
}

// GetInstitutionAttributionStats reports missing institution attribution
// (institution_id IS NULL) in restricted projects for the last N days.
// GET /api/stats/institution-attribution — accepts ?days=7 and optional ?project_id=
func (s *Server) GetInstitutionAttributionStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	days, _ := strconv.Atoi(q.Get("days"))
	projectID := q.Get("project_id")
	access, ok := s.requireResearcherProjectScope(w, r, projectID)
	if !ok {
		return
	}
	institutionID := ""
	if access != nil {
		projectID = access.ProjectID
		if access.IsSiteScoped() {
			institutionID = *access.InstitutionID
		}
	}

	stats, err := model.GetInstitutionAttributionStats(r.Context(), s.db, days, projectID, institutionID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query institution attribution stats")
		return
	}

	s.writeJSON(w, http.StatusOK, institutionAttributionStatsResponse{
		Days:              stats.Days,
		TotalRestricted:   stats.TotalRestricted,
		UnattributedTotal: stats.UnattributedTotal,
		UnattributedPct:   stats.UnattributedPct,
		Projects:          stats.Projects,
		GeneratedAt:       time.Now().UTC(),
	})
}
