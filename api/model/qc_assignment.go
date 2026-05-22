package model

import (
	"context"
	"database/sql"
	"time"
)

// QCAssignmentSummary is one row in the analyst triage queue — a study that's
// been assigned to (or is in-progress with) the requesting analyst, with
// just enough metadata to render the row without a follow-up fetch.
type QCAssignmentSummary struct {
	StudyID              string     `json:"study_id"`
	StudyInstanceUID     string     `json:"study_instance_uid"`
	Modality             string     `json:"modality"`
	BodyPart             string     `json:"body_part"`
	StudyDescription     string     `json:"study_description"`
	ProjectID            string     `json:"project_id"`
	ProjectName          string     `json:"project_name"`
	InstitutionID        *string    `json:"institution_id,omitempty"`
	InstitutionName      string     `json:"institution_name,omitempty"`
	Status               string     `json:"status"`
	QCStatus             string     `json:"qc_status"`
	QCAssignedToID       *string    `json:"qc_assigned_to_id,omitempty"`
	QCAssignedToEmail    string     `json:"qc_assigned_to_email,omitempty"`
	QCAssignedAt         *time.Time `json:"qc_assigned_at,omitempty"`
	QCReviewStartedAt    *time.Time `json:"qc_review_started_at,omitempty"`
	QCReviewEndedAt      *time.Time `json:"qc_review_ended_at,omitempty"`
	OpenFindingCount     int        `json:"open_finding_count"`
	HighestSeverityOpen  string     `json:"highest_severity_open,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// AssignQCReviewer sets qc_assigned_to + qc_assigned_at on a study.
// Clears any prior assignment.
func AssignQCReviewer(ctx context.Context, db *sql.DB, studyID, analystID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies
		SET qc_assigned_to = $1, qc_assigned_at = now(),
		    qc_review_started_at = NULL, qc_review_ended_at = NULL
		WHERE id = $2`, analystID, studyID)
	return err
}

// UnassignQCReviewer clears qc_assigned_to.
func UnassignQCReviewer(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies
		SET qc_assigned_to = NULL, qc_assigned_at = NULL,
		    qc_review_started_at = NULL, qc_review_ended_at = NULL
		WHERE id = $1`, studyID)
	return err
}

// StartQCReview stamps qc_review_started_at = now() so we can measure
// review duration. Idempotent — only sets it if NULL.
func StartQCReview(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies
		SET qc_review_started_at = COALESCE(qc_review_started_at, now())
		WHERE id = $1`, studyID)
	return err
}

// CompleteQCReview stamps qc_review_ended_at = now().
func CompleteQCReview(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE studies
		SET qc_review_ended_at = now()
		WHERE id = $1`, studyID)
	return err
}

// QCTriageFilters narrows the analyst's triage queue.
type QCTriageFilters struct {
	AssignedToID string // empty = unassigned + everyone
	ProjectID    string
	StatusIn     []string // study.status filter
	OnlyOpen     bool     // exclude studies whose qc_review_ended_at is set
}

// ListTriageQueue returns the analyst's queue, optionally filtered.
func ListTriageQueue(ctx context.Context, db *sql.DB, f QCTriageFilters, limit int) ([]QCAssignmentSummary, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	args := []any{}
	cond := []string{}
	push := func(c string, v any) {
		args = append(args, v)
		cond = append(cond, replaceAt(c, len(args)))
	}
	if f.AssignedToID != "" {
		push("s.qc_assigned_to = $?", f.AssignedToID)
	}
	if f.ProjectID != "" {
		push("s.project_id = $?", f.ProjectID)
	}
	if len(f.StatusIn) > 0 {
		// Build IN(...) using positional placeholders.
		placeholders := []string{}
		for _, st := range f.StatusIn {
			args = append(args, st)
			placeholders = append(placeholders, "$"+itoa(len(args)))
		}
		cond = append(cond, "s.status IN ("+joinComma(placeholders)+")")
	}
	if f.OnlyOpen {
		cond = append(cond, "s.qc_review_ended_at IS NULL")
	}
	where := ""
	if len(cond) > 0 {
		where = " WHERE " + joinAnd(cond)
	}
	args = append(args, limit)

	query := `
		WITH finding_summary AS (
		    SELECT study_id,
		           count(*) FILTER (WHERE resolved_at IS NULL) AS open_count,
		           CASE
		             WHEN bool_or(severity = 'critical' AND resolved_at IS NULL) THEN 'critical'
		             WHEN bool_or(severity = 'major'    AND resolved_at IS NULL) THEN 'major'
		             WHEN bool_or(severity = 'minor'    AND resolved_at IS NULL) THEN 'minor'
		             WHEN bool_or(severity = 'info'     AND resolved_at IS NULL) THEN 'info'
		             ELSE ''
		           END AS highest_severity
		    FROM qc_findings
		    GROUP BY study_id
		)
		SELECT s.id, s.study_instance_uid, s.modality, s.body_part,
		       s.study_description, s.project_id, coalesce(p.name, ''),
		       s.institution_id, coalesce(i.name, ''),
		       s.status, s.qc_status,
		       s.qc_assigned_to, coalesce(u.email, ''),
		       s.qc_assigned_at, s.qc_review_started_at, s.qc_review_ended_at,
		       coalesce(fs.open_count, 0), coalesce(fs.highest_severity, ''),
		       s.created_at
		FROM studies s
		LEFT JOIN projects     p  ON p.id  = s.project_id
		LEFT JOIN institutions i  ON i.id  = s.institution_id
		LEFT JOIN admin_users  u  ON u.id  = s.qc_assigned_to
		LEFT JOIN finding_summary fs ON fs.study_id = s.id` + where + `
		ORDER BY
		    CASE coalesce(fs.highest_severity, '')
		        WHEN 'critical' THEN 0
		        WHEN 'major'    THEN 1
		        WHEN 'minor'    THEN 2
		        WHEN 'info'     THEN 3
		        ELSE                4
		    END,
		    s.qc_assigned_at NULLS LAST,
		    s.created_at DESC
		LIMIT $` + itoa(len(args))

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]QCAssignmentSummary, 0)
	for rows.Next() {
		var r QCAssignmentSummary
		var instID, assignedTo sql.NullString
		var assignedAt, startedAt, endedAt sql.NullTime
		if err := rows.Scan(&r.StudyID, &r.StudyInstanceUID, &r.Modality, &r.BodyPart,
			&r.StudyDescription, &r.ProjectID, &r.ProjectName,
			&instID, &r.InstitutionName,
			&r.Status, &r.QCStatus,
			&assignedTo, &r.QCAssignedToEmail,
			&assignedAt, &startedAt, &endedAt,
			&r.OpenFindingCount, &r.HighestSeverityOpen,
			&r.CreatedAt); err != nil {
			return nil, err
		}
		if instID.Valid {
			s := instID.String
			r.InstitutionID = &s
		}
		if assignedTo.Valid {
			s := assignedTo.String
			r.QCAssignedToID = &s
		}
		if assignedAt.Valid {
			t := assignedAt.Time
			r.QCAssignedAt = &t
		}
		if startedAt.Valid {
			t := startedAt.Time
			r.QCReviewStartedAt = &t
		}
		if endedAt.Valid {
			t := endedAt.Time
			r.QCReviewEndedAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AnalystThroughput summarises an analyst's QC review performance in a window.
type AnalystThroughput struct {
	AnalystID                string  `json:"analyst_id"`
	AnalystEmail             string  `json:"analyst_email"`
	AnalystName              string  `json:"analyst_name"`
	StudiesCompleted         int     `json:"studies_completed"`
	OpenAssignments          int     `json:"open_assignments"`
	FindingsRaised           int     `json:"findings_raised"`
	FindingsResolved         int     `json:"findings_resolved"`
	AvgReviewMinutes         float64 `json:"avg_review_minutes"`
	MedianReviewMinutes      float64 `json:"median_review_minutes"`
	StudiesPerHour           float64 `json:"studies_per_hour"`
}

// GetAnalystThroughput returns one row per analyst with stats over the
// specified lookback window (default 7 days).
func GetAnalystThroughput(ctx context.Context, db *sql.DB, days int) ([]AnalystThroughput, error) {
	if days <= 0 {
		days = 7
	}
	rows, err := db.QueryContext(ctx, `
		WITH completed AS (
		    SELECT s.qc_assigned_to AS analyst_id,
		           count(*) AS n,
		           avg(EXTRACT(EPOCH FROM (s.qc_review_ended_at - s.qc_review_started_at)) / 60.0) AS avg_minutes,
		           percentile_cont(0.5) WITHIN GROUP (
		             ORDER BY EXTRACT(EPOCH FROM (s.qc_review_ended_at - s.qc_review_started_at)) / 60.0
		           ) AS median_minutes
		    FROM studies s
		    WHERE s.qc_assigned_to IS NOT NULL
		      AND s.qc_review_ended_at IS NOT NULL
		      AND s.qc_review_started_at IS NOT NULL
		      AND s.qc_review_ended_at >= now() - ($1 * INTERVAL '1 day')
		    GROUP BY s.qc_assigned_to
		),
		open_n AS (
		    SELECT qc_assigned_to AS analyst_id, count(*) AS n
		    FROM studies
		    WHERE qc_assigned_to IS NOT NULL AND qc_review_ended_at IS NULL
		    GROUP BY qc_assigned_to
		),
		raised AS (
		    SELECT analyst_id, count(*) AS n
		    FROM qc_findings
		    WHERE analyst_id IS NOT NULL
		      AND created_at >= now() - ($1 * INTERVAL '1 day')
		    GROUP BY analyst_id
		),
		resolved AS (
		    SELECT analyst_id, count(*) AS n
		    FROM qc_findings
		    WHERE analyst_id IS NOT NULL
		      AND resolved_at IS NOT NULL
		      AND resolved_at >= now() - ($1 * INTERVAL '1 day')
		    GROUP BY analyst_id
		)
		SELECT u.id, u.email, u.name,
		       coalesce(c.n, 0)              AS studies_completed,
		       coalesce(o.n, 0)              AS open_assignments,
		       coalesce(r.n, 0)              AS findings_raised,
		       coalesce(rv.n, 0)             AS findings_resolved,
		       coalesce(c.avg_minutes, 0)    AS avg_minutes,
		       coalesce(c.median_minutes, 0) AS median_minutes,
		       CASE
		         WHEN coalesce(c.avg_minutes, 0) > 0
		         THEN 60.0 / c.avg_minutes
		         ELSE 0
		       END AS studies_per_hour
		FROM admin_users u
		LEFT JOIN completed c  ON c.analyst_id = u.id
		LEFT JOIN open_n    o  ON o.analyst_id = u.id
		LEFT JOIN raised    r  ON r.analyst_id = u.id
		LEFT JOIN resolved  rv ON rv.analyst_id = u.id
		WHERE u.enabled = TRUE
		  AND (coalesce(c.n, 0) > 0 OR coalesce(o.n, 0) > 0 OR coalesce(r.n, 0) > 0)
		ORDER BY studies_completed DESC, u.email ASC`,
		days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AnalystThroughput, 0)
	for rows.Next() {
		var r AnalystThroughput
		if err := rows.Scan(&r.AnalystID, &r.AnalystEmail, &r.AnalystName,
			&r.StudiesCompleted, &r.OpenAssignments,
			&r.FindingsRaised, &r.FindingsResolved,
			&r.AvgReviewMinutes, &r.MedianReviewMinutes, &r.StudiesPerHour); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── tiny helpers (no external dep) ─────────────────────────────────────────

func replaceAt(template string, n int) string {
	// Replace the first $? with $<n>.
	idx := indexOf(template, "$?")
	if idx < 0 {
		return template
	}
	return template[:idx] + "$" + itoa(n) + template[idx+2:]
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func joinComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += ", " + parts[i]
	}
	return out
}

func joinAnd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " AND " + parts[i]
	}
	return out
}
