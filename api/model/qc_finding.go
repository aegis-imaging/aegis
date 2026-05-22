package model

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// QCFinding is one discrete observation raised during human QC review.
type QCFinding struct {
	ID             string     `json:"id"`
	StudyID        string     `json:"study_id"`
	AnalystID      *string    `json:"analyst_id,omitempty"`
	AnalystEmail   string     `json:"analyst_email"`
	AnalystName    string     `json:"analyst_name,omitempty"`
	Category       string     `json:"category"`
	Severity       string     `json:"severity"`
	Body           string     `json:"body"`
	SeriesUID      string     `json:"series_uid,omitempty"`
	InstanceIndex  *int       `json:"instance_index,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     string     `json:"resolved_by,omitempty"`
	ResolutionNote string     `json:"resolution_note,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

var validQCCategories = map[string]bool{
	"phi_leak": true, "defacing": true, "motion": true, "protocol": true,
	"coverage": true, "metadata": true, "other": true,
}

var validQCSeverities = map[string]bool{
	"info": true, "minor": true, "major": true, "critical": true,
}

// IsValidQCCategory and IsValidQCSeverity are exported so handlers can
// validate inbound JSON without duplicating the lists.
func IsValidQCCategory(s string) bool { return validQCCategories[strings.ToLower(s)] }
func IsValidQCSeverity(s string) bool { return validQCSeverities[strings.ToLower(s)] }

func CreateQCFinding(ctx context.Context, db *sql.DB, f *QCFinding) error {
	var analystID any
	if f.AnalystID != nil && *f.AnalystID != "" {
		analystID = *f.AnalystID
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO qc_findings
			(study_id, analyst_id, analyst_email, category, severity, body,
			 series_uid, instance_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`,
		f.StudyID, analystID, f.AnalystEmail, f.Category, f.Severity, f.Body,
		f.SeriesUID, f.InstanceIndex,
	).Scan(&f.ID, &f.CreatedAt)
}

func ListQCFindingsByStudy(ctx context.Context, db *sql.DB, studyID string, openOnly bool) ([]QCFinding, error) {
	query := `
		SELECT f.id, f.study_id, f.analyst_id, f.analyst_email,
		       coalesce(u.name, ''),
		       f.category, f.severity, f.body, f.series_uid, f.instance_index,
		       f.resolved_at, f.resolved_by, f.resolution_note, f.created_at
		FROM qc_findings f
		LEFT JOIN admin_users u ON u.id = f.analyst_id
		WHERE f.study_id = $1`
	if openOnly {
		query += " AND f.resolved_at IS NULL"
	}
	query += " ORDER BY f.created_at DESC"

	rows, err := db.QueryContext(ctx, query, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]QCFinding, 0)
	for rows.Next() {
		var f QCFinding
		var analystID sql.NullString
		var resolvedAt sql.NullTime
		var instanceIdx sql.NullInt32
		if err := rows.Scan(&f.ID, &f.StudyID, &analystID, &f.AnalystEmail,
			&f.AnalystName,
			&f.Category, &f.Severity, &f.Body, &f.SeriesUID, &instanceIdx,
			&resolvedAt, &f.ResolvedBy, &f.ResolutionNote, &f.CreatedAt); err != nil {
			return nil, err
		}
		if analystID.Valid {
			a := analystID.String
			f.AnalystID = &a
		}
		if instanceIdx.Valid {
			n := int(instanceIdx.Int32)
			f.InstanceIndex = &n
		}
		if resolvedAt.Valid {
			t := resolvedAt.Time
			f.ResolvedAt = &t
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func ResolveQCFinding(ctx context.Context, db *sql.DB, findingID, resolvedBy, note string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE qc_findings
		SET resolved_at = now(), resolved_by = $1, resolution_note = $2
		WHERE id = $3`,
		resolvedBy, note, findingID)
	return err
}

func ReopenQCFinding(ctx context.Context, db *sql.DB, findingID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE qc_findings
		SET resolved_at = NULL, resolved_by = '', resolution_note = ''
		WHERE id = $1`, findingID)
	return err
}

func DeleteQCFinding(ctx context.Context, db *sql.DB, findingID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM qc_findings WHERE id = $1`, findingID)
	return err
}
