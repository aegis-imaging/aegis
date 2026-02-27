package handler

import (
	"net/http"
	"strconv"
)

// GetRetentionPreview returns a dry-run preview of how many approved studies would
// be expired if the retention policy were set to the given number of days.
// It does NOT modify any data.
//
// GET /api/projects/{id}/retention-preview?days=90
//
// Also returns a study-age distribution: how many approved studies fall into each
// age bucket (0–7d, 8–30d, 31–90d, 91–180d, 181–365d, 365d+), which helps
// operators choose an appropriate retention period.
func (s *Server) GetRetentionPreview(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if _, ok := s.requireProjectReadAccess(w, r, projectID); !ok {
		return
	}

	days := 90
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 3650 {
			days = n
		}
	}

	// ── Count that would be expired ───────────────────────────────────────────
	var wouldExpireCount int
	err := s.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*)
		FROM studies
		WHERE project_id = $1
		  AND status = 'approved'
		  AND created_at < now() - ($2 * INTERVAL '1 day')`,
		projectID, days).Scan(&wouldExpireCount)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "retention preview query failed")
		return
	}

	// ── Age distribution of approved studies ─────────────────────────────────
	type ageBucket struct {
		Label       string `json:"label"`
		MinDays     int    `json:"min_days"`
		MaxDays     *int   `json:"max_days"` // nil = unbounded
		Count       int    `json:"count"`
		WouldExpire bool   `json:"would_expire"` // true if min_days >= days
	}

	bucketDefs := []struct {
		label   string
		minDays int
		maxDays *int
	}{
		{"0–7 days", 0, intPtr(7)},
		{"8–30 days", 8, intPtr(30)},
		{"31–90 days", 31, intPtr(90)},
		{"91–180 days", 91, intPtr(180)},
		{"181–365 days", 181, intPtr(365)},
		{"365+ days", 366, nil},
	}

	buckets := make([]ageBucket, len(bucketDefs))
	for i, def := range bucketDefs {
		b := ageBucket{
			Label:       def.label,
			MinDays:     def.minDays,
			MaxDays:     def.maxDays,
			WouldExpire: def.minDays >= days,
		}

		var q string
		var args []any
		if def.maxDays == nil {
			q = `SELECT COUNT(*) FROM studies
				 WHERE project_id = $1 AND status = 'approved'
				   AND created_at < now() - ($2 * INTERVAL '1 day')`
			args = []any{projectID, def.minDays}
		} else {
			q = `SELECT COUNT(*) FROM studies
				 WHERE project_id = $1 AND status = 'approved'
				   AND created_at >= now() - ($2 * INTERVAL '1 day')
				   AND created_at <  now() - ($3 * INTERVAL '1 day')`
			args = []any{projectID, *def.maxDays + 1, def.minDays}
		}
		if err := s.db.QueryRowContext(r.Context(), q, args...).Scan(&b.Count); err != nil {
			s.writeError(w, http.StatusInternalServerError, "age distribution query failed")
			return
		}
		buckets[i] = b
	}

	// ── Total approved studies for percentage calculation ─────────────────────
	var totalApproved int
	_ = s.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM studies WHERE project_id = $1 AND status = 'approved'`,
		projectID).Scan(&totalApproved)

	s.writeJSON(w, http.StatusOK, map[string]any{
		"project_id":         projectID,
		"preview_days":       days,
		"would_expire_count": wouldExpireCount,
		"total_approved":     totalApproved,
		"age_distribution":   buckets,
	})
}

func intPtr(n int) *int { return &n }
