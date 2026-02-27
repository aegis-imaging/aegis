package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type AuditEntry struct {
	ID           string          `json:"id"`
	Action       string          `json:"action"`
	Actor        string          `json:"actor"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Detail       json.RawMessage `json:"detail,omitempty"`
	IPAddress    string          `json:"ip_address"`
	CreatedAt    time.Time       `json:"created_at"`
}

func CreateAuditEntry(ctx context.Context, db *sql.DB, action, actor, resourceType, resourceID, ipAddress string, detail any) error {
	var detailJSON []byte
	if detail != nil {
		var err error
		detailJSON, err = json.Marshal(detail)
		if err != nil {
			return err
		}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_trail (action, actor, resource_type, resource_id, detail, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		action, actor, resourceType, resourceID, detailJSON, ipAddress)
	return err
}

// AuditFilters holds optional filter values for ListAuditEntries / CountAuditEntries.
type AuditFilters struct {
	Action       string    // prefix match on action (e.g. "study" matches all study.* events)
	ResourceType string    // exact match on resource_type
	Actor        string    // exact match on actor (user email)
	Search       string    // free-text substring search across actor, action, resource_id
	DateFrom     time.Time // inclusive lower bound on created_at (zero = no bound)
	DateTo       time.Time // inclusive upper bound on created_at (zero = no bound)
}

func auditWhere(f AuditFilters) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	if f.Action != "" {
		clauses = append(clauses, fmt.Sprintf("action LIKE $%d || '%%'", n))
		args = append(args, f.Action)
		n++
	}
	if f.ResourceType != "" {
		clauses = append(clauses, fmt.Sprintf("resource_type = $%d", n))
		args = append(args, f.ResourceType)
		n++
	}
	if f.Actor != "" {
		clauses = append(clauses, fmt.Sprintf("actor = $%d", n))
		args = append(args, f.Actor)
		n++
	}
	if f.Search != "" {
		pat := "%" + f.Search + "%"
		clauses = append(clauses, fmt.Sprintf(
			"(actor ILIKE $%d OR action ILIKE $%d OR resource_id ILIKE $%d OR COALESCE(detail::text,'') ILIKE $%d)",
			n, n, n, n))
		args = append(args, pat)
		n++
	}
	if !f.DateFrom.IsZero() {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", n))
		args = append(args, f.DateFrom.UTC())
		n++
	}
	if !f.DateTo.IsZero() {
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", n))
		args = append(args, f.DateTo.UTC())
		n++
	}
	_ = n

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + clauses[0]
		for _, c := range clauses[1:] {
			where += " AND " + c
		}
	}
	return where, args
}

func CountAuditEntries(ctx context.Context, db *sql.DB, f AuditFilters) (int, error) {
	where, args := auditWhere(f)
	var n int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM audit_trail`+where, args...).Scan(&n)
	return n, err
}

func auditWhereStudyScope(f AuditFilters, projectID, institutionID string, argStart int) (string, []any) {
	clauses := []string{"resource_type = 'study'", fmt.Sprintf(`resource_id IN (SELECT id::text FROM studies WHERE project_id = $%d::uuid`, argStart)}
	args := []any{projectID}
	n := argStart + 1

	if institutionID != "" {
		clauses[1] += fmt.Sprintf(" AND institution_id = $%d::uuid", n)
		args = append(args, institutionID)
		n++
	}
	clauses[1] += ")"

	if f.Action != "" {
		clauses = append(clauses, fmt.Sprintf("action LIKE $%d || '%%'", n))
		args = append(args, f.Action)
		n++
	}
	if f.ResourceType != "" {
		clauses = append(clauses, fmt.Sprintf("resource_type = $%d", n))
		args = append(args, f.ResourceType)
		n++
	}
	if f.Actor != "" {
		clauses = append(clauses, fmt.Sprintf("actor = $%d", n))
		args = append(args, f.Actor)
		n++
	}
	if f.Search != "" {
		pat := "%" + f.Search + "%"
		clauses = append(clauses, fmt.Sprintf(
			"(actor ILIKE $%d OR action ILIKE $%d OR resource_id ILIKE $%d OR COALESCE(detail::text,'') ILIKE $%d)",
			n, n, n, n))
		args = append(args, pat)
		n++
	}
	if !f.DateFrom.IsZero() {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", n))
		args = append(args, f.DateFrom.UTC())
		n++
	}
	if !f.DateTo.IsZero() {
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", n))
		args = append(args, f.DateTo.UTC())
		n++
	}

	where := " WHERE " + clauses[0]
	for _, c := range clauses[1:] {
		where += " AND " + c
	}
	return where, args
}

func CountAuditEntriesForStudyScope(ctx context.Context, db *sql.DB, f AuditFilters, projectID, institutionID string) (int, error) {
	where, args := auditWhereStudyScope(f, projectID, institutionID, 1)
	var n int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM audit_trail`+where, args...).Scan(&n)
	return n, err
}

func ListAuditEntries(ctx context.Context, db *sql.DB, f AuditFilters, limit, offset int) ([]AuditEntry, error) {
	where, args := auditWhere(f)
	argN := len(args) + 1

	query := `SELECT id, action, actor, resource_type, resource_id, COALESCE(detail, 'null'), ip_address, created_at FROM audit_trail` + where + ` ORDER BY created_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, limit)
		argN++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argN)
		args = append(args, offset)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.Actor, &e.ResourceType, &e.ResourceID,
			&e.Detail, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func ListAuditEntriesForStudyScope(ctx context.Context, db *sql.DB, f AuditFilters, projectID, institutionID string, limit, offset int) ([]AuditEntry, error) {
	where, args := auditWhereStudyScope(f, projectID, institutionID, 1)
	argN := len(args) + 1

	query := `SELECT id, action, actor, resource_type, resource_id, COALESCE(detail, 'null'), ip_address, created_at FROM audit_trail` + where + ` ORDER BY created_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, limit)
		argN++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argN)
		args = append(args, offset)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.Actor, &e.ResourceType, &e.ResourceID,
			&e.Detail, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListAuditEntriesByActor returns the most recent audit entries for a specific actor (email).
func ListAuditEntriesByActor(ctx context.Context, db *sql.DB, actor string, limit int) ([]AuditEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, action, actor, resource_type, resource_id, COALESCE(detail, 'null'), ip_address, created_at
		  FROM audit_trail
		 WHERE actor = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, actor, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.Actor, &e.ResourceType, &e.ResourceID,
			&e.Detail, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListStudyNoteAuditEntries returns all study.note audit entries for a study,
// ordered newest-first. Notes are stored as audit_trail rows with action='study.note'.
func ListStudyNoteAuditEntries(ctx context.Context, db *sql.DB, studyID string) ([]AuditEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, action, actor, resource_type, resource_id, COALESCE(detail, 'null'), ip_address, created_at
		FROM audit_trail
		WHERE resource_id = $1
		  AND action = 'study.note'
		ORDER BY created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.Actor, &e.ResourceType, &e.ResourceID,
			&e.Detail, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ListAuditEntriesForStudy returns all audit entries for a specific study (by resource_id).
func ListAuditEntriesForStudy(ctx context.Context, db *sql.DB, studyID string) ([]AuditEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, action, actor, resource_type, resource_id, COALESCE(detail, 'null'), ip_address, created_at
		FROM audit_trail
		WHERE resource_id = $1
		ORDER BY created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Action, &e.Actor, &e.ResourceType, &e.ResourceID,
			&e.Detail, &e.IPAddress, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ActorSummary is one row in the recent-activity-by-actor query.
type ActorSummary struct {
	Actor       string    `json:"actor"`
	ActionCount int       `json:"action_count"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	LastAction  string    `json:"last_action"`
}

// GetActorSummary returns the top N actors by recency with action counts.
// Only the last 30 days are considered.
func GetActorSummary(ctx context.Context, db *sql.DB, limit int) ([]ActorSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	qrows, err := db.QueryContext(ctx, `
		SELECT actor,
		       count(*)                                         AS action_count,
		       max(created_at)                                  AS last_seen_at,
		       (SELECT action FROM audit_trail a2
		          WHERE a2.actor = a.actor
		          ORDER BY created_at DESC LIMIT 1)             AS last_action
		FROM audit_trail a
		WHERE created_at >= now() - INTERVAL '30 days'
		  AND actor <> ''
		GROUP BY actor
		ORDER BY max(created_at) DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer qrows.Close()
	var result []ActorSummary
	for qrows.Next() {
		var s ActorSummary
		if err := qrows.Scan(&s.Actor, &s.ActionCount, &s.LastSeenAt, &s.LastAction); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, qrows.Err()
}

func GetActorSummaryForStudyScope(ctx context.Context, db *sql.DB, projectID, institutionID string, limit int) ([]ActorSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	where, args := auditWhereStudyScope(AuditFilters{}, projectID, institutionID, 1)
	args = append(args, limit)

	query := `
		WITH scoped AS (
			SELECT actor, action, created_at
			FROM audit_trail` + where + `
		)
		SELECT s.actor,
		       count(*) AS action_count,
		       max(s.created_at) AS last_seen_at,
		       (
		         SELECT s2.action FROM scoped s2
		         WHERE s2.actor = s.actor
		         ORDER BY s2.created_at DESC
		         LIMIT 1
		       ) AS last_action
		FROM scoped s
		WHERE s.actor <> ''
		GROUP BY s.actor
		ORDER BY max(s.created_at) DESC
		LIMIT $` + fmt.Sprintf("%d", len(args))

	qrows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer qrows.Close()

	var result []ActorSummary
	for qrows.Next() {
		var s ActorSummary
		if err := qrows.Scan(&s.Actor, &s.ActionCount, &s.LastSeenAt, &s.LastAction); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, qrows.Err()
}
