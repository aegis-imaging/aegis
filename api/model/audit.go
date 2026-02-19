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

func ListAuditEntries(ctx context.Context, db *sql.DB, action, resourceType string, limit int) ([]AuditEntry, error) {
	query := `SELECT id, action, actor, resource_type, resource_id, COALESCE(detail, 'null'), ip_address, created_at FROM audit_trail`
	var conditions []string
	var args []any
	argN := 1

	if action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argN))
		args = append(args, action)
		argN++
	}
	if resourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argN))
		args = append(args, resourceType)
		argN++
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, limit)
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
