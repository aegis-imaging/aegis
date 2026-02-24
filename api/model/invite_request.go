package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// InviteRequest is a prospective-user access request stored in the DB.
type InviteRequest struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Org          string     `json:"org"`
	Message      string     `json:"message"`
	Status       string     `json:"status"` // pending | approved | denied
	IP           *string    `json:"ip,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy   *string    `json:"reviewed_by,omitempty"`
	InviteCodeID *string    `json:"invite_code_id,omitempty"`
}

const inviteRequestColumns = `id, name, email, org, message, status, ip, created_at, reviewed_at, reviewed_by, invite_code_id`

func scanInviteRequest(row scannable, req *InviteRequest) error {
	return row.Scan(
		&req.ID, &req.Name, &req.Email, &req.Org, &req.Message,
		&req.Status, &req.IP, &req.CreatedAt, &req.ReviewedAt,
		&req.ReviewedBy, &req.InviteCodeID,
	)
}

// CreateInviteRequest stores a new pending access request.
func CreateInviteRequest(ctx context.Context, db *sql.DB, name, email, org, message, ip string) (*InviteRequest, error) {
	var r InviteRequest
	err := scanInviteRequest(
		db.QueryRowContext(ctx, `
			INSERT INTO invite_requests (name, email, org, message, ip)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING `+inviteRequestColumns,
			name, email, org, message, ip,
		), &r,
	)
	if err != nil {
		return nil, fmt.Errorf("create invite request: %w", err)
	}
	return &r, nil
}

// GetInviteRequest retrieves a single invite request by UUID.
func GetInviteRequest(ctx context.Context, db *sql.DB, id string) (*InviteRequest, error) {
	var r InviteRequest
	err := scanInviteRequest(
		db.QueryRowContext(ctx,
			`SELECT `+inviteRequestColumns+` FROM invite_requests WHERE id = $1`, id,
		), &r,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListInviteRequests returns invite requests ordered by creation date (newest first).
// Pass status="" to return all requests.
func ListInviteRequests(ctx context.Context, db *sql.DB, status string) ([]InviteRequest, error) {
	var query string
	var args []any
	if status != "" {
		query = `SELECT ` + inviteRequestColumns + ` FROM invite_requests WHERE status = $1 ORDER BY created_at DESC`
		args = []any{status}
	} else {
		query = `SELECT ` + inviteRequestColumns + ` FROM invite_requests ORDER BY created_at DESC`
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list invite requests: %w", err)
	}
	defer rows.Close()

	var out []InviteRequest
	for rows.Next() {
		var r InviteRequest
		if err := scanInviteRequest(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ApproveInviteRequest marks a request as approved, recording the invite code that was issued.
func ApproveInviteRequest(ctx context.Context, db *sql.DB, id, reviewedBy, inviteCodeID string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE invite_requests
		   SET status = 'approved', reviewed_at = NOW(), reviewed_by = $2, invite_code_id = $3
		 WHERE id = $1 AND status = 'pending'`,
		id, reviewedBy, inviteCodeID,
	)
	if err != nil {
		return fmt.Errorf("approve invite request: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DenyInviteRequest marks a request as denied.
func DenyInviteRequest(ctx context.Context, db *sql.DB, id, reviewedBy string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE invite_requests
		   SET status = 'denied', reviewed_at = NOW(), reviewed_by = $2
		 WHERE id = $1 AND status = 'pending'`,
		id, reviewedBy,
	)
	if err != nil {
		return fmt.Errorf("deny invite request: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
