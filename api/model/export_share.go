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

func GetExportShareByID(ctx context.Context, db *sql.DB, shareID string) (*ExportShare, error) {
	var s ExportShare
	err := scanShare(db.QueryRowContext(ctx,
		`SELECT`+shareColumns+` FROM export_shares WHERE id = $1`, shareID), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ExtendExportShare pushes the expiry forward by the given duration.
// Returns an error if the share is already revoked.
func ExtendExportShare(ctx context.Context, db *sql.DB, shareID string, newExpiry time.Time) error {
	res, err := db.ExecContext(ctx,
		`UPDATE export_shares SET expires_at = $1 WHERE id = $2 AND revoked_at IS NULL`,
		newExpiry, shareID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
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

// DownloadAnalytics holds aggregate download statistics across all export shares.
type DownloadAnalytics struct {
	TotalDownloads int              `json:"total_downloads"`
	Last30Days     []DailyDownloads `json:"last_30_days"`
	TopShares      []ShareDownloads `json:"top_shares"`
}

// DailyDownloads holds download count for a single UTC date.
type DailyDownloads struct {
	Date  string `json:"date"`  // YYYY-MM-DD
	Count int    `json:"count"`
}

// ShareDownloads holds aggregate download count for one export share.
type ShareDownloads struct {
	ShareID        string `json:"share_id"`
	RecipientEmail string `json:"recipient_email"`
	StudyID        string `json:"study_id"`
	DownloadCount  int    `json:"download_count"`
}

// GetExportDownloadAnalytics returns aggregate download analytics.
func GetExportDownloadAnalytics(ctx context.Context, db *sql.DB) (*DownloadAnalytics, error) {
	var total int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM export_downloads`).Scan(&total); err != nil {
		return nil, err
	}

	// Aggregate by UTC date for the last 30 days.
	rows, err := db.QueryContext(ctx, `
		SELECT to_char(accessed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, count(*)
		FROM export_downloads
		WHERE accessed_at >= now() - INTERVAL '30 days'
		GROUP BY day
		ORDER BY day`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var daily []DailyDownloads
	for rows.Next() {
		var d DailyDownloads
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		daily = append(daily, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if daily == nil {
		daily = []DailyDownloads{}
	}

	// Top 10 shares by download count.
	topRows, err := db.QueryContext(ctx, `
		SELECT es.id, es.recipient_email, es.study_id, count(ed.id) AS cnt
		FROM export_shares es
		JOIN export_downloads ed ON ed.share_id = es.id
		GROUP BY es.id, es.recipient_email, es.study_id
		ORDER BY cnt DESC
		LIMIT 10`)
	if err != nil {
		return nil, err
	}
	defer topRows.Close()

	var top []ShareDownloads
	for topRows.Next() {
		var s ShareDownloads
		if err := topRows.Scan(&s.ShareID, &s.RecipientEmail, &s.StudyID, &s.DownloadCount); err != nil {
			return nil, err
		}
		top = append(top, s)
	}
	if err := topRows.Err(); err != nil {
		return nil, err
	}
	if top == nil {
		top = []ShareDownloads{}
	}

	return &DownloadAnalytics{
		TotalDownloads: total,
		Last30Days:     daily,
		TopShares:      top,
	}, nil
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

func CountAllExportShares(ctx context.Context, db *sql.DB, status ShareStatusFilter, projectID ...string) (int, error) {
	proj := ""
	if len(projectID) > 0 {
		proj = projectID[0]
	}
	n := 1
	extraJoin := ""
	var args []any
	if proj != "" {
		extraJoin = ` JOIN studies s ON s.id = es.study_id AND s.project_id = $1`
		args = append(args, proj)
		n++
	}
	where, wargs := shareStatusWhere(status, n)
	args = append(args, wargs...)
	var cnt int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM export_shares es`+extraJoin+` WHERE 1=1`+where, args...).Scan(&cnt)
	return cnt, err
}

func ListAllExportShares(ctx context.Context, db *sql.DB, status ShareStatusFilter, limit, offset int, projectID ...string) ([]ExportShare, error) {
	proj := ""
	if len(projectID) > 0 {
		proj = projectID[0]
	}
	n := 1
	extraJoin := ""
	var args []any
	if proj != "" {
		extraJoin = ` JOIN studies s ON s.id = es.study_id AND s.project_id = $1`
		args = append(args, proj)
		n++
	}
	where, wargs := shareStatusWhere(status, n)
	args = append(args, wargs...)
	if len(wargs) > 0 {
		n++
	}

	query := `SELECT` + shareColumnsWithCount + ` FROM export_shares es` + extraJoin + ` WHERE 1=1` + where + ` ORDER BY es.created_at DESC`
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
