package model

import (
	"context"
	"database/sql"
	"time"
)

type ExportShare struct {
	ID             string     `json:"id"`
	StudyID        string     `json:"study_id"`
	RecipientEmail string     `json:"recipient_email"`
	Note           string     `json:"note,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedBy      string     `json:"created_by"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	// Token is populated only when a new share is created; never read from DB.
	Token string `json:"token,omitempty"`
}

const shareColumns = `
	id, study_id, recipient_email, note, expires_at, created_by, revoked_at, created_at`

func scanShare(row scannable, s *ExportShare) error {
	return row.Scan(&s.ID, &s.StudyID, &s.RecipientEmail, &s.Note,
		&s.ExpiresAt, &s.CreatedBy, &s.RevokedAt, &s.CreatedAt)
}

func CreateExportShare(ctx context.Context, db *sql.DB, studyID, tokenHash, recipientEmail, note, createdBy string, expiresAt time.Time) (*ExportShare, error) {
	var s ExportShare
	err := scanShare(db.QueryRowContext(ctx, `
		INSERT INTO export_shares (study_id, token_hash, recipient_email, note, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING`+shareColumns,
		studyID, tokenHash, recipientEmail, note, expiresAt, createdBy), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetExportShareByTokenHash(ctx context.Context, db *sql.DB, tokenHash string) (*ExportShare, error) {
	var s ExportShare
	err := scanShare(db.QueryRowContext(ctx,
		`SELECT`+shareColumns+` FROM export_shares WHERE token_hash = $1`, tokenHash), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ListExportSharesByStudy(ctx context.Context, db *sql.DB, studyID string) ([]ExportShare, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT`+shareColumns+` FROM export_shares WHERE study_id = $1 ORDER BY created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []ExportShare
	for rows.Next() {
		var s ExportShare
		if err := scanShare(rows, &s); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, rows.Err()
}

func RevokeExportShare(ctx context.Context, db *sql.DB, shareID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE export_shares SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, shareID)
	return err
}

func CreateExportDownload(ctx context.Context, db *sql.DB, shareID, ipAddress string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO export_downloads (share_id, client_ip) VALUES ($1, $2)`, shareID, ipAddress)
	return err
}
