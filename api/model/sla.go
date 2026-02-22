package model

import (
	"context"
	"database/sql"
)

// GetStuckStudies returns studies that are not in a terminal state (approved/rejected)
// and whose updated_at is older than olderThanMinutes. Used by the SLA endpoint and
// scheduler to identify studies that may be stalled in the pipeline.
// If projectID is non-empty, results are scoped to that project.
func GetStuckStudies(ctx context.Context, db *sql.DB, olderThanMinutes int, projectID string) ([]Study, error) {
	var rows *sql.Rows
	var err error

	base := `SELECT ` + studyColumns + `
		FROM studies
		WHERE status NOT IN ('approved', 'rejected')
		  AND updated_at < NOW() - ($1 * INTERVAL '1 minute')`

	if projectID != "" {
		rows, err = db.QueryContext(ctx, base+` AND project_id = $2 ORDER BY updated_at ASC LIMIT 200`, olderThanMinutes, projectID)
	} else {
		rows, err = db.QueryContext(ctx, base+` ORDER BY updated_at ASC LIMIT 200`, olderThanMinutes)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studies []Study
	for rows.Next() {
		var s Study
		if err := scanStudy(rows, &s); err != nil {
			return nil, err
		}
		studies = append(studies, s)
	}
	return studies, rows.Err()
}

// GetUnalertedStuckStudies returns stuck studies that have not been alerted within
// the cooldown window. Used by the SLA scheduler to determine which studies to email
// about without spamming operators on every hourly run.
func GetUnalertedStuckStudies(ctx context.Context, db *sql.DB, olderThanMinutes, cooldownHours int) ([]Study, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT `+studyColumns+`
		FROM studies s
		LEFT JOIN study_sla_alerts a ON a.study_id = s.id
		WHERE s.status NOT IN ('approved', 'rejected')
		  AND s.updated_at < NOW() - ($1 * INTERVAL '1 minute')
		  AND (a.last_alerted_at IS NULL OR a.last_alerted_at < NOW() - ($2 * INTERVAL '1 hour'))
		ORDER BY s.updated_at ASC
		LIMIT 200`,
		olderThanMinutes, cooldownHours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var studies []Study
	for rows.Next() {
		var s Study
		if err := scanStudy(rows, &s); err != nil {
			return nil, err
		}
		studies = append(studies, s)
	}
	return studies, rows.Err()
}

// MarkStudySLAAlerted upserts the last_alerted_at timestamp for a study, so the
// scheduler knows not to re-alert within the cooldown window.
func MarkStudySLAAlerted(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO study_sla_alerts (study_id)
		VALUES ($1)
		ON CONFLICT (study_id) DO UPDATE SET last_alerted_at = NOW()`,
		studyID)
	return err
}
