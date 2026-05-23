package model

import (
	"context"
	"database/sql"
	"time"
)

type StudyTag struct {
	ID        string    `json:"id"`
	StudyID   string    `json:"study_id"`
	ProjectID string    `json:"project_id"`
	Tag       string    `json:"tag"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func AddStudyTag(ctx context.Context, db *sql.DB, studyID, projectID, tag, createdBy string) (*StudyTag, error) {
	var t StudyTag
	err := db.QueryRowContext(ctx, `
		INSERT INTO study_tags (study_id, project_id, tag, created_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (study_id, tag) DO UPDATE SET study_id = study_tags.study_id
		RETURNING id, study_id, project_id, tag, created_by, created_at`,
		studyID, projectID, tag, createdBy).
		Scan(&t.ID, &t.StudyID, &t.ProjectID, &t.Tag, &t.CreatedBy, &t.CreatedAt)
	return &t, err
}

func ListStudyTags(ctx context.Context, db *sql.DB, studyID string) ([]StudyTag, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, project_id, tag, created_by, created_at
		FROM study_tags
		WHERE study_id = $1
		ORDER BY tag ASC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyTag
	for rows.Next() {
		var t StudyTag
		if err := rows.Scan(&t.ID, &t.StudyID, &t.ProjectID, &t.Tag, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func DeleteStudyTag(ctx context.Context, db *sql.DB, id, studyID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM study_tags WHERE id = $1 AND study_id = $2`, id, studyID)
	return err
}

// ListProjectTagTaxonomy returns distinct tags used in a project with counts.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

func ListProjectTagTaxonomy(ctx context.Context, db *sql.DB, projectID string) ([]TagCount, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT tag, count(*) AS cnt
		FROM study_tags
		WHERE project_id = $1
		GROUP BY tag
		ORDER BY cnt DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagCount
	for rows.Next() {
		var t TagCount
		if err := rows.Scan(&t.Tag, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
