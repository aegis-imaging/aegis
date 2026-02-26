package model

import (
	"context"
	"database/sql"
	"time"
)

type StudyWatcher struct {
	ID        string    `json:"id"`
	StudyID   string    `json:"study_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func AddStudyWatcher(ctx context.Context, db *sql.DB, studyID, userID string) (*StudyWatcher, error) {
	var w StudyWatcher
	err := db.QueryRowContext(ctx, `
		INSERT INTO study_watchers (study_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (study_id, user_id) DO UPDATE SET study_id = study_watchers.study_id
		RETURNING id, study_id, user_id, created_at`,
		studyID, userID).
		Scan(&w.ID, &w.StudyID, &w.UserID, &w.CreatedAt)
	return &w, err
}

func ListStudyWatchers(ctx context.Context, db *sql.DB, studyID string) ([]StudyWatcher, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, user_id, created_at
		FROM study_watchers
		WHERE study_id = $1
		ORDER BY created_at ASC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyWatcher
	for rows.Next() {
		var w StudyWatcher
		if err := rows.Scan(&w.ID, &w.StudyID, &w.UserID, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func RemoveStudyWatcher(ctx context.Context, db *sql.DB, studyID, userID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM study_watchers WHERE study_id = $1 AND user_id = $2`, studyID, userID)
	return err
}

func GetStudyWatcherUserIDs(ctx context.Context, db *sql.DB, studyID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT user_id FROM study_watchers WHERE study_id = $1`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
