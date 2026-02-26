package model

import (
	"context"
	"database/sql"
	"time"
)

type StudyTransfer struct {
	ID            string    `json:"id"`
	StudyID       string    `json:"study_id"`
	FromProjectID string    `json:"from_project_id"`
	ToProjectID   string    `json:"to_project_id"`
	TransferredBy string    `json:"transferred_by"`
	Reason        string    `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
}

func CreateStudyTransfer(ctx context.Context, db *sql.DB, t *StudyTransfer) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO study_transfers (study_id, from_project_id, to_project_id, transferred_by, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		t.StudyID, t.FromProjectID, t.ToProjectID, t.TransferredBy, t.Reason).
		Scan(&t.ID, &t.CreatedAt)
}

func ListStudyTransfers(ctx context.Context, db *sql.DB, studyID string) ([]StudyTransfer, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, from_project_id, to_project_id, transferred_by, reason, created_at
		FROM study_transfers
		WHERE study_id = $1
		ORDER BY created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyTransfer
	for rows.Next() {
		var t StudyTransfer
		if err := rows.Scan(&t.ID, &t.StudyID, &t.FromProjectID, &t.ToProjectID, &t.TransferredBy, &t.Reason, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
