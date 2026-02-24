package model

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
)

// InviteCode is a unique per-person landing page access token.
type InviteCode struct {
	ID        string     `json:"id"`
	Code      string     `json:"code"`
	Label     string     `json:"label"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	UsedByIP  *string    `json:"used_by_ip,omitempty"`
}

// generateCode returns a random human-readable invite code in the form XXXX-XXXX-XXXX.
func generateCode() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	// base32 without padding, uppercase, trimmed to 12 chars → 3 groups of 4
	enc := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
	// enc is at least 16 chars; take first 12 → XXXXXXXXXXXX → XXXX-XXXX-XXXX
	raw := enc[:12]
	return raw[:4] + "-" + raw[4:8] + "-" + raw[8:12], nil
}

// CreateInviteCode generates a new unique invite code with the given label.
func CreateInviteCode(ctx context.Context, db *sql.DB, label string) (*InviteCode, error) {
	code, err := generateCode()
	if err != nil {
		return nil, err
	}
	ic := &InviteCode{
		Code:    code,
		Label:   label,
		Enabled: true,
	}
	err = db.QueryRowContext(ctx,
		`INSERT INTO invite_codes (code, label, enabled)
		 VALUES ($1, $2, TRUE)
		 RETURNING id, created_at`,
		code, label,
	).Scan(&ic.ID, &ic.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create invite code: %w", err)
	}
	return ic, nil
}

// ValidateAndRecordInviteCode checks whether a code is valid and enabled.
// On success it records the first-use timestamp and IP (subsequent calls are no-ops for used_at).
// Returns (true, nil) when admitted, (false, nil) when invalid/disabled, (false, err) on DB error.
func ValidateAndRecordInviteCode(ctx context.Context, db *sql.DB, code, ip string) (bool, error) {
	var id string
	var enabled bool
	err := db.QueryRowContext(ctx,
		`SELECT id, enabled FROM invite_codes WHERE code = $1`,
		code,
	).Scan(&id, &enabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("validate invite code: %w", err)
	}
	if !enabled {
		return false, nil
	}

	// Record first use (ignore error — admission still granted).
	_, _ = db.ExecContext(ctx,
		`UPDATE invite_codes
		    SET used_at = COALESCE(used_at, NOW()),
		        used_by_ip = COALESCE(used_by_ip, $2)
		  WHERE id = $1`,
		id, ip,
	)
	return true, nil
}

// ListInviteCodes returns all invite codes ordered by creation date (newest first).
func ListInviteCodes(ctx context.Context, db *sql.DB) ([]InviteCode, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, code, label, enabled, created_at, used_at, used_by_ip
		   FROM invite_codes
		  ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list invite codes: %w", err)
	}
	defer rows.Close()

	var codes []InviteCode
	for rows.Next() {
		var ic InviteCode
		if err := rows.Scan(&ic.ID, &ic.Code, &ic.Label, &ic.Enabled,
			&ic.CreatedAt, &ic.UsedAt, &ic.UsedByIP); err != nil {
			return nil, err
		}
		codes = append(codes, ic)
	}
	return codes, rows.Err()
}

// RevokeInviteCode disables an invite code by ID.
func RevokeInviteCode(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE invite_codes SET enabled = FALSE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("revoke invite code: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetInviteCode fetches a single invite code by ID.
func GetInviteCode(ctx context.Context, db *sql.DB, id string) (*InviteCode, error) {
	var ic InviteCode
	err := db.QueryRowContext(ctx,
		`SELECT id, code, label, enabled, created_at, used_at, used_by_ip
		   FROM invite_codes WHERE id = $1`, id).
		Scan(&ic.ID, &ic.Code, &ic.Label, &ic.Enabled,
			&ic.CreatedAt, &ic.UsedAt, &ic.UsedByIP)
	if err != nil {
		return nil, err
	}
	return &ic, nil
}

// DeleteInviteCode permanently removes an invite code by ID.
func DeleteInviteCode(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx,
		`DELETE FROM invite_codes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete invite code: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
