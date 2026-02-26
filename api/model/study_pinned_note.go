package model

import (
	"context"
	"database/sql"
	"time"
)

type StudyPinnedNote struct {
	ID        string    `json:"id"`
	StudyID   string    `json:"study_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CreateStudyPinnedNote(ctx context.Context, db *sql.DB, n *StudyPinnedNote) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO study_pinned_notes (study_id, author, body)
		VALUES ($1, $2, $3)
		RETURNING id, pinned, created_at, updated_at`,
		n.StudyID, n.Author, n.Body).
		Scan(&n.ID, &n.Pinned, &n.CreatedAt, &n.UpdatedAt)
}

func ListStudyPinnedNotes(ctx context.Context, db *sql.DB, studyID string) ([]StudyPinnedNote, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, author, body, pinned, created_at, updated_at
		FROM study_pinned_notes
		WHERE study_id = $1
		ORDER BY pinned DESC, created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyPinnedNote
	for rows.Next() {
		var n StudyPinnedNote
		if err := rows.Scan(&n.ID, &n.StudyID, &n.Author, &n.Body, &n.Pinned, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func UpdateStudyPinnedNote(ctx context.Context, db *sql.DB, id string, pinned bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE study_pinned_notes SET pinned = $1, updated_at = now() WHERE id = $2`, pinned, id)
	return err
}

func DeleteStudyPinnedNote(ctx context.Context, db *sql.DB, id, studyID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM study_pinned_notes WHERE id = $1 AND study_id = $2`, id, studyID)
	return err
}
