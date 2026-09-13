package model

import (
	"context"
	"database/sql"
	"time"
)

type StudyApproval struct {
	ID         string    `json:"id"`
	StudyID    string    `json:"study_id"`
	ApprovedBy string    `json:"approved_by"`
	Action     string    `json:"action"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"created_at"`
}

func CreateStudyApproval(ctx context.Context, db *sql.DB, a *StudyApproval) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO study_approvals (study_id, approved_by, action, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		a.StudyID, a.ApprovedBy, a.Action, a.Reason).
		Scan(&a.ID, &a.CreatedAt)
}

func ListStudyApprovals(ctx context.Context, db *sql.DB, studyID string) ([]StudyApproval, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, approved_by, action, reason, created_at
		FROM study_approvals
		WHERE study_id = $1
		ORDER BY created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyApproval
	for rows.Next() {
		var a StudyApproval
		if err := rows.Scan(&a.ID, &a.StudyID, &a.ApprovedBy, &a.Action, &a.Reason, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
