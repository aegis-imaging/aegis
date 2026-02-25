package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

type destHealthSummary struct {
	DestinationID   string     `json:"destination_id"`
	DestinationName string     `json:"destination_name"`
	Type            string     `json:"type"`
	Enabled         bool       `json:"enabled"`
	TestCount       int        `json:"test_count"`
	SuccessCount    int        `json:"success_count"`
	FailureCount    int        `json:"failure_count"`
	SuccessRate     float64    `json:"success_rate"`  // 0.0–1.0; -1 = never tested
	LastTestedAt    *time.Time `json:"last_tested_at,omitempty"`
	LastSuccess     *time.Time `json:"last_success,omitempty"`
	LastFailure     *time.Time `json:"last_failure,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	LastLatencyMs   float64    `json:"last_latency_ms,omitempty"`
	// Status: "healthy" (last test passed) | "degraded" (mixed) | "failing" (last test failed) | "unknown" (never tested)
	Status string `json:"status"`
}

// GetDestinationHealth returns the last N connectivity test results for a single
// destination, aggregated into a health summary plus a recent test log.
//
// GET /api/destinations/{id}/health?limit=20
func (s *Server) GetDestinationHealth(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	dest, err := model.GetDestinationByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "destination not found")
		return
	}

	summary, tests, err := buildDestHealthSummary(r.Context(), s.db, dest, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "health query failed")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"summary":      summary,
		"recent_tests": tests,
	})
}

// GetAllDestinationsHealth returns a health summary for every destination.
//
// GET /api/destinations/health
func (s *Server) GetAllDestinationsHealth(w http.ResponseWriter, r *http.Request) {
	dests, err := model.ListDestinations(r.Context(), s.db)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list destinations")
		return
	}

	summaries := make([]destHealthSummary, 0, len(dests))
	for i := range dests {
		summary, _, qErr := buildDestHealthSummary(r.Context(), s.db, &dests[i], 0)
		if qErr != nil {
			continue
		}
		summaries = append(summaries, summary)
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"destinations": summaries,
		"total":        len(summaries),
	})
}

// buildDestHealthSummary queries the audit trail for destination.tested entries
// and builds a health summary. When limit > 0, also returns a recent test log.
func buildDestHealthSummary(ctx context.Context, db *sql.DB, dest *model.Destination, limit int) (destHealthSummary, []map[string]any, error) {
	summary := destHealthSummary{
		DestinationID:   dest.ID,
		DestinationName: dest.Name,
		Type:            dest.Type,
		Enabled:         dest.Enabled,
		SuccessRate:     -1, // unknown until we have data
		Status:          "unknown",
	}

	// ── Aggregate stats from audit_trail ─────────────────────────────────────
	row := db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*)                                              AS test_count,
		  COUNT(*) FILTER (WHERE (detail->>'success')::bool)   AS success_count,
		  MAX(created_at)                                       AS last_tested_at,
		  MAX(created_at) FILTER (WHERE (detail->>'success')::bool)  AS last_success,
		  MAX(created_at) FILTER (WHERE NOT (detail->>'success')::bool) AS last_failure
		FROM audit_trail
		WHERE action = 'destination.tested'
		  AND resource_id = $1`, dest.ID)

	var lastTestedAt, lastSuccess, lastFailure sql.NullTime
	if err := row.Scan(
		&summary.TestCount, &summary.SuccessCount,
		&lastTestedAt, &lastSuccess, &lastFailure,
	); err != nil {
		return summary, nil, err
	}
	summary.FailureCount = summary.TestCount - summary.SuccessCount

	if summary.TestCount > 0 {
		summary.SuccessRate = float64(summary.SuccessCount) / float64(summary.TestCount)
	}
	if lastTestedAt.Valid {
		summary.LastTestedAt = &lastTestedAt.Time
	}
	if lastSuccess.Valid {
		summary.LastSuccess = &lastSuccess.Time
	}
	if lastFailure.Valid {
		summary.LastFailure = &lastFailure.Time
	}

	// ── Derive status from most recent test ───────────────────────────────────
	if summary.TestCount == 0 {
		summary.Status = "unknown"
	} else if summary.SuccessRate >= 0.9 {
		summary.Status = "healthy"
	} else if summary.SuccessRate >= 0.5 {
		summary.Status = "degraded"
	} else {
		summary.Status = "failing"
	}

	// ── Recent test log (optional, only when limit > 0) ──────────────────────
	if limit <= 0 {
		return summary, nil, nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT created_at, actor, COALESCE(detail, '{}') as detail
		FROM audit_trail
		WHERE action = 'destination.tested'
		  AND resource_id = $1
		ORDER BY created_at DESC
		LIMIT $2`, dest.ID, limit)
	if err != nil {
		return summary, nil, err
	}
	defer rows.Close()

	var tests []map[string]any
	for rows.Next() {
		var createdAt time.Time
		var actor string
		var detailRaw []byte
		if err := rows.Scan(&createdAt, &actor, &detailRaw); err != nil {
			continue
		}
		var detail map[string]any
		_ = json.Unmarshal(detailRaw, &detail)
		entry := map[string]any{
			"tested_at": createdAt.UTC().Format(time.RFC3339),
			"actor":     actor,
		}
		if detail != nil {
			entry["success"] = detail["success"]
			if v, ok := detail["error"]; ok {
				entry["error"] = v
			}
			if v, ok := detail["latency_ms"]; ok {
				entry["latency_ms"] = v
			}
		}
		tests = append(tests, entry)

		// capture last error and latency from most recent test
		if len(tests) == 1 && detail != nil {
			if v, ok := detail["error"].(string); ok {
				summary.LastError = v
			}
			if v, ok := detail["latency_ms"].(float64); ok {
				summary.LastLatencyMs = v
			}
		}
	}
	return summary, tests, rows.Err()
}
