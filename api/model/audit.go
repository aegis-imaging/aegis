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
	Action       string // exact match on action
	ResourceType string // exact match on resource_type
	Actor        string // exact match on actor (user email)
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
