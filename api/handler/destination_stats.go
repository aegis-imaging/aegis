package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// destinationStatsResponse is the response payload for GetDestinationStats.
type destinationStatsResponse struct {
	DestinationID   string              `json:"destination_id"`
	PeriodDays      int                 `json:"period_days"`
	GeneratedAt     string              `json:"generated_at"`
	TotalAttempts   int                 `json:"total_attempts"`
	Successful      int                 `json:"successful"`
	Failed          int                 `json:"failed"`
	SuccessRate     float64             `json:"success_rate"` // 0.0–1.0
	RecentErrors    []routingErrorEntry `json:"recent_errors"`
	DailyBreakdown  []routingDayBucket  `json:"daily_breakdown"`
}

type routingErrorEntry struct {
	StudyID   string `json:"study_id"`
	Outcome   string `json:"outcome"`
	CreatedAt string `json:"created_at"`
}

type routingDayBucket struct {
	Day        string `json:"day"` // YYYY-MM-DD
	Attempts   int    `json:"attempts"`
	Successful int    `json:"successful"`
}

// routingOverviewResponse is the payload for GetRoutingStats.
type routingOverviewResponse struct {
	PeriodDays  int                     `json:"period_days"`
	GeneratedAt string                  `json:"generated_at"`
	Totals      routingTotals           `json:"totals"`
	ByDestination []destRoutingSummary  `json:"by_destination"`
}

type routingTotals struct {
	Attempts    int     `json:"attempts"`
	Successful  int     `json:"successful"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

type destRoutingSummary struct {
	DestinationID   string  `json:"destination_id"`
	DestinationName string  `json:"destination_name"`
	DestinationType string  `json:"destination_type"`
	Attempts        int     `json:"attempts"`
	Successful      int     `json:"successful"`
	Failed          int     `json:"failed"`
	SuccessRate     float64 `json:"success_rate"`
	LastAttemptAt   *string `json:"last_attempt_at"`
}

// GetDestinationStats returns routing success/failure statistics for a single destination.
//
// GET /api/destinations/{id}/stats?days=30
func (s *Server) GetDestinationStats(w http.ResponseWriter, r *http.Request) {
	destID := r.PathValue("id")
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().UTC().AddDate(0, 0, -days)

	// Total attempts and success count.
	var total, successful int
	err := s.db.QueryRowContext(r.Context(), `
		SELECT
		  COUNT(*)                                                          AS total,
		  COUNT(*) FILTER (WHERE outcome = 'forwarding dispatched')       AS successful
		FROM routing_log
		WHERE destination_id = $1
		  AND created_at >= $2
		  AND action = 'route_to'`,
		destID, since).Scan(&total, &successful)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing stats query failed")
		return
	}
	failed := total - successful
	successRate := 0.0
	if total > 0 {
		successRate = float64(successful) / float64(total)
	}

	// Recent error entries (last 10).
	errRows, err := s.db.QueryContext(r.Context(), `
		SELECT study_id, outcome, created_at
		FROM routing_log
		WHERE destination_id = $1
		  AND created_at >= $2
		  AND action = 'route_to'
		  AND outcome <> 'forwarding dispatched'
		ORDER BY created_at DESC
		LIMIT 10`,
		destID, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing error query failed")
		return
	}
	defer errRows.Close()

	recentErrors := []routingErrorEntry{}
	for errRows.Next() {
		var e routingErrorEntry
		var createdAt time.Time
		if err := errRows.Scan(&e.StudyID, &e.Outcome, &createdAt); err == nil {
			e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
			recentErrors = append(recentErrors, e)
		}
	}

	// Daily breakdown.
	dayRows, err := s.db.QueryContext(r.Context(), `
		SELECT
		  to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
		  COUNT(*)                                               AS attempts,
		  COUNT(*) FILTER (WHERE outcome = 'forwarding dispatched') AS successful
		FROM routing_log
		WHERE destination_id = $1
		  AND created_at >= $2
		  AND action = 'route_to'
		GROUP BY 1
		ORDER BY 1 ASC`,
		destID, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing daily query failed")
		return
	}
	defer dayRows.Close()

	daily := []routingDayBucket{}
	for dayRows.Next() {
		var b routingDayBucket
		if err := dayRows.Scan(&b.Day, &b.Attempts, &b.Successful); err == nil {
			daily = append(daily, b)
		}
	}

	s.writeJSON(w, http.StatusOK, destinationStatsResponse{
		DestinationID:  destID,
		PeriodDays:     days,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		TotalAttempts:  total,
		Successful:     successful,
		Failed:         failed,
		SuccessRate:    successRate,
		RecentErrors:   recentErrors,
		DailyBreakdown: daily,
	})
}

// GetRoutingStats returns aggregate routing stats across all destinations.
//
// GET /api/stats/routing?days=30
func (s *Server) GetRoutingStats(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}
	since := time.Now().UTC().AddDate(0, 0, -days)

	// Global totals.
	var totalAttempts, totalSuccessful int
	err := s.db.QueryRowContext(r.Context(), `
		SELECT
		  COUNT(*)                                                         AS total,
		  COUNT(*) FILTER (WHERE outcome = 'forwarding dispatched')       AS successful
		FROM routing_log
		WHERE destination_id IS NOT NULL
		  AND created_at >= $1
		  AND action = 'route_to'`,
		since).Scan(&totalAttempts, &totalSuccessful)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing stats query failed")
		return
	}
	totalFailed := totalAttempts - totalSuccessful
	totalSuccessRate := 0.0
	if totalAttempts > 0 {
		totalSuccessRate = float64(totalSuccessful) / float64(totalAttempts)
	}

	// Per-destination breakdown joined with destinations table for name/type.
	destRows, err := s.db.QueryContext(r.Context(), `
		SELECT
		  rl.destination_id,
		  d.name,
		  d.type,
		  COUNT(*)                                                          AS attempts,
		  COUNT(*) FILTER (WHERE rl.outcome = 'forwarding dispatched')     AS successful,
		  MAX(rl.created_at)                                               AS last_attempt_at
		FROM routing_log rl
		JOIN destinations d ON d.id = rl.destination_id
		WHERE rl.destination_id IS NOT NULL
		  AND rl.created_at >= $1
		  AND rl.action = 'route_to'
		GROUP BY rl.destination_id, d.name, d.type
		ORDER BY attempts DESC`,
		since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "routing destination query failed")
		return
	}
	defer destRows.Close()

	byDest := []destRoutingSummary{}
	for destRows.Next() {
		var row destRoutingSummary
		var lastAt sql.NullTime
		if err := destRows.Scan(&row.DestinationID, &row.DestinationName, &row.DestinationType,
			&row.Attempts, &row.Successful, &lastAt); err != nil {
			continue
		}
		row.Failed = row.Attempts - row.Successful
		if row.Attempts > 0 {
			row.SuccessRate = float64(row.Successful) / float64(row.Attempts)
		}
		if lastAt.Valid {
			ts := lastAt.Time.UTC().Format(time.RFC3339)
			row.LastAttemptAt = &ts
		}
		// Sanitize type field: trim whitespace, normalize to lowercase.
		row.DestinationType = strings.ToLower(strings.TrimSpace(row.DestinationType))
		byDest = append(byDest, row)
	}

	s.writeJSON(w, http.StatusOK, routingOverviewResponse{
		PeriodDays:    days,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Totals:        routingTotals{Attempts: totalAttempts, Successful: totalSuccessful, Failed: totalFailed, SuccessRate: totalSuccessRate},
		ByDestination: byDest,
	})
}
