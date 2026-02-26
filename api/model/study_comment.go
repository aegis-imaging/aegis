package model

import (
	"context"
	"database/sql"
	"time"
)

type StudyComment struct {
	ID        string     `json:"id"`
	StudyID   string     `json:"study_id"`
	Author    string     `json:"author"`
	Body      string     `json:"body"`
	ParentID  *string    `json:"parent_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func CreateStudyComment(ctx context.Context, db *sql.DB, c *StudyComment) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO study_comments (study_id, author, body, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		c.StudyID, c.Author, c.Body, c.ParentID).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func ListStudyComments(ctx context.Context, db *sql.DB, studyID string) ([]StudyComment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, author, body, parent_id, created_at, updated_at
		FROM study_comments
		WHERE study_id = $1
		ORDER BY created_at ASC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyComment
	for rows.Next() {
		var c StudyComment
		if err := rows.Scan(&c.ID, &c.StudyID, &c.Author, &c.Body, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func DeleteStudyComment(ctx context.Context, db *sql.DB, commentID, studyID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM study_comments WHERE id = $1 AND study_id = $2`, commentID, studyID)
	return err
}

func UpdateStudyComment(ctx context.Context, db *sql.DB, commentID, body string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE study_comments SET body = $1, updated_at = now() WHERE id = $2`, body, commentID)
	return err
}
