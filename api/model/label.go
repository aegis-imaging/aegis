package model

import (
	"context"
	"database/sql"
	"time"
)

// StudyLabel is a free-text tag applied to a study by an admin user.
type StudyLabel struct {
	ID        string    `json:"id"`
	StudyID   string    `json:"study_id"`
	Label     string    `json:"label"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// ListStudyLabels returns all labels for a study ordered by creation time.
func ListStudyLabels(ctx context.Context, db *sql.DB, studyID string) ([]StudyLabel, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, label, created_by, created_at
		FROM study_labels
		WHERE study_id = $1
		ORDER BY created_at ASC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyLabel
	for rows.Next() {
		var l StudyLabel
		if err := rows.Scan(&l.ID, &l.StudyID, &l.Label, &l.CreatedBy, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// AddStudyLabel inserts a new label for a study. Duplicate labels (case-insensitive) are ignored.
func AddStudyLabel(ctx context.Context, db *sql.DB, studyID, label, createdBy string) (*StudyLabel, error) {
	var l StudyLabel
	err := db.QueryRowContext(ctx, `
		INSERT INTO study_labels (study_id, label, created_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (study_id, lower(label)) DO NOTHING
		RETURNING id, study_id, label, created_by, created_at`,
		studyID, label, createdBy).
		Scan(&l.ID, &l.StudyID, &l.Label, &l.CreatedBy, &l.CreatedAt)
	if err == sql.ErrNoRows {
		// Conflict (duplicate label) — fetch the existing one.
		err = db.QueryRowContext(ctx, `
			SELECT id, study_id, label, created_by, created_at
			FROM study_labels WHERE study_id = $1 AND lower(label) = lower($2)`,
			studyID, label).
			Scan(&l.ID, &l.StudyID, &l.Label, &l.CreatedBy, &l.CreatedAt)
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// DeleteStudyLabel removes a label by ID from a study.
func DeleteStudyLabel(ctx context.Context, db *sql.DB, labelID, studyID string) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM study_labels WHERE id = $1 AND study_id = $2`, labelID, studyID)
	return err
}
