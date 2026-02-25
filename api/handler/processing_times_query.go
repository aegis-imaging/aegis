package handler

import (
	"context"
	"database/sql"
	"time"
)

// queryProcessingTimes computes per-stage processing-time statistics by
// pairing `.triggered` audit events with their nearest subsequent `.complete`
// event on the same resource within a 2-hour window.
//
// When projectID is non-empty the results are restricted to studies in that
// project. All times are in seconds.
func queryProcessingTimes(ctx context.Context, db *sql.DB, since time.Time, projectID string) ([]StageTimingRow, error) {
	// Project scope clause — left-joined onto the studies table.
	projectJoin := ""
	projectWhere := ""
	if projectID != "" {
		projectJoin = `JOIN studies s ON s.id = t.resource_id`
		projectWhere = `AND s.project_id = $2`
	}

	// Dynamic $2 or nothing for the project filter.
	var args []any
	args = append(args, since)
	if projectID != "" {
		args = append(args, projectID)
	}

	q := `
WITH triggered AS (
  SELECT t.resource_id,
         t.created_at,
         regexp_replace(t.action, '\.triggered$', '') AS stage
  FROM   audit_trail t
  ` + projectJoin + `
  WHERE  t.action LIKE '%.triggered'
    AND  t.resource_type = 'study'
    AND  t.created_at >= $1
    ` + projectWhere + `
),
durations AS (
  SELECT  tr.stage,
          EXTRACT(EPOCH FROM (c.created_at - tr.created_at)) AS secs
  FROM    triggered tr
  JOIN LATERAL (
    SELECT created_at
    FROM   audit_trail c
    WHERE  c.resource_id  = tr.resource_id
      AND  c.resource_type = 'study'
      AND  c.action        = tr.stage || '.complete'
      AND  c.created_at   > tr.created_at
      AND  c.created_at   < tr.created_at + INTERVAL '2 hours'
    ORDER  BY c.created_at ASC
    LIMIT  1
  ) c ON true
  WHERE tr.stage IN (
    'deface', 'phi_scan', 'qc_check', 'bids_conversion',
    'classification', 'protocol_check', 'export'
  )
)
SELECT stage,
       COUNT(*)                                                           AS count,
       ROUND(AVG(secs)::numeric,                            1)           AS avg_seconds,
       ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP
             (ORDER BY secs)::numeric,                      1)           AS p95_seconds,
       ROUND(MIN(secs)::numeric,                            1)           AS min_seconds,
       ROUND(MAX(secs)::numeric,                            1)           AS max_seconds
FROM   durations
GROUP  BY stage
ORDER  BY avg_seconds DESC
`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StageTimingRow
	for rows.Next() {
		var r StageTimingRow
		if err := rows.Scan(
			&r.Stage, &r.Count, &r.AvgSeconds, &r.P95Seconds, &r.MinSeconds, &r.MaxSeconds,
		); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
