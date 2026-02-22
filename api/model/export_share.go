package model

import (
	"context"
	"database/sql"
	"fmt"
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
	DownloadCount  int        `json:"download_count"`
	// Token is populated only when a new share is created; never read from DB.
	Token string `json:"token,omitempty"`
}

const shareColumns = `
	id, study_id, recipient_email, note, expires_at, created_by, revoked_at, created_at`

// shareColumnsWithCount extends shareColumns with a download_count subquery.
const shareColumnsWithCount = `
	es.id, es.study_id, es.recipient_email, es.note, es.expires_at, es.created_by, es.revoked_at, es.created_at,
	(SELECT count(*) FROM export_downloads ed WHERE ed.share_id = es.id)`

func scanShare(row scannable, s *ExportShare) error {
	return row.Scan(&s.ID, &s.StudyID, &s.RecipientEmail, &s.Note,
		&s.ExpiresAt, &s.CreatedBy, &s.RevokedAt, &s.CreatedAt)
}

func scanShareWithCount(row scannable, s *ExportShare) error {
	return row.Scan(&s.ID, &s.StudyID, &s.RecipientEmail, &s.Note,
		&s.ExpiresAt, &s.CreatedBy, &s.RevokedAt, &s.CreatedAt, &s.DownloadCount)
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
		`SELECT`+shareColumnsWithCount+` FROM export_shares es WHERE es.study_id = $1 ORDER BY es.created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []ExportShare
	for rows.Next() {
		var s ExportShare
		if err := scanShareWithCount(rows, &s); err != nil {
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

// ExportDownload is a single entry in the immutable download log for an export share.
type ExportDownload struct {
	ID         string    `json:"id"`
	ShareID    string    `json:"share_id"`
	ClientIP   string    `json:"client_ip"`
	AccessedAt time.Time `json:"accessed_at"`
}

// ListExportDownloadsByShare returns all download events for one share, newest first.
func ListExportDownloadsByShare(ctx context.Context, db *sql.DB, shareID string) ([]ExportDownload, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, share_id, client_ip, accessed_at FROM export_downloads WHERE share_id = $1 ORDER BY accessed_at DESC`,
		shareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []ExportDownload
	for rows.Next() {
		var d ExportDownload
		if err := rows.Scan(&d.ID, &d.ShareID, &d.ClientIP, &d.AccessedAt); err != nil {
			return nil, err
		}
		downloads = append(downloads, d)
	}
	return downloads, rows.Err()
}

// ShareStatusFilter constrains ListAllExportShares to a computed status bucket.
// "active" = not revoked AND expires_at > now()
// "expired" = not revoked AND expires_at <= now()
// "revoked" = revoked_at IS NOT NULL
// "" (empty) = no filter (all shares)
type ShareStatusFilter string

const (
	ShareStatusActive  ShareStatusFilter = "active"
	ShareStatusExpired ShareStatusFilter = "expired"
	ShareStatusRevoked ShareStatusFilter = "revoked"
)

func shareStatusWhere(status ShareStatusFilter, n int) (string, []any) {
	now := time.Now().UTC()
	switch status {
	case ShareStatusActive:
		return fmt.Sprintf(" AND es.revoked_at IS NULL AND es.expires_at > $%d", n), []any{now}
	case ShareStatusExpired:
		return fmt.Sprintf(" AND es.revoked_at IS NULL AND es.expires_at <= $%d", n), []any{now}
	case ShareStatusRevoked:
		return " AND es.revoked_at IS NOT NULL", nil
	default:
		return "", nil
	}
}

func CountAllExportShares(ctx context.Context, db *sql.DB, status ShareStatusFilter) (int, error) {
	where, args := shareStatusWhere(status, 1)
	var n int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM export_shares es WHERE 1=1`+where, args...).Scan(&n)
	return n, err
}

func ListAllExportShares(ctx context.Context, db *sql.DB, status ShareStatusFilter, limit, offset int) ([]ExportShare, error) {
	n := 1
	where, args := shareStatusWhere(status, n)
	if len(args) > 0 {
		n++
	}

	query := `SELECT` + shareColumnsWithCount + ` FROM export_shares es WHERE 1=1` + where + ` ORDER BY es.created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", n)
		args = append(args, limit)
		n++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", n)
		args = append(args, offset)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []ExportShare
	for rows.Next() {
		var s ExportShare
		if err := scanShareWithCount(rows, &s); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, rows.Err()
}
