package model

import (
	"context"
	"database/sql"
	"time"
)

type ReviewChecklistItem struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Label     string    `json:"label"`
	SortOrder int       `json:"sort_order"`
	Required  bool      `json:"required"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateReviewChecklistItem(ctx context.Context, db *sql.DB, item *ReviewChecklistItem) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO review_checklist_items (project_id, label, sort_order, required, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		item.ProjectID, item.Label, item.SortOrder, item.Required, item.Enabled).
		Scan(&item.ID, &item.CreatedAt)
}

func ListReviewChecklistItems(ctx context.Context, db *sql.DB, projectID string) ([]ReviewChecklistItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, label, sort_order, required, enabled, created_at
		FROM review_checklist_items
		WHERE project_id = $1
		ORDER BY sort_order ASC, created_at ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReviewChecklistItem
	for rows.Next() {
		var item ReviewChecklistItem
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Label, &item.SortOrder, &item.Required, &item.Enabled, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func UpdateReviewChecklistItem(ctx context.Context, db *sql.DB, id, label string, sortOrder int, required, enabled bool) error {
	_, err := db.ExecContext(ctx, `
		UPDATE review_checklist_items SET label = $1, sort_order = $2, required = $3, enabled = $4
		WHERE id = $5`, label, sortOrder, required, enabled, id)
	return err
}

func DeleteReviewChecklistItem(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM review_checklist_items WHERE id = $1`, id)
	return err
}

// StudyChecklistResponse tracks whether a checklist item has been checked for a study.
type StudyChecklistResponse struct {
	ID              string     `json:"id"`
	StudyID         string     `json:"study_id"`
	ChecklistItemID string     `json:"checklist_item_id"`
	Checked         bool       `json:"checked"`
	CheckedBy       string     `json:"checked_by"`
	CheckedAt       *time.Time `json:"checked_at,omitempty"`
}

func UpsertStudyChecklistResponse(ctx context.Context, db *sql.DB, studyID, itemID string, checked bool, checkedBy string) error {
	var checkedAt *time.Time
	if checked {
		now := time.Now().UTC()
		checkedAt = &now
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO study_checklist_responses (study_id, checklist_item_id, checked, checked_by, checked_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (study_id, checklist_item_id)
		DO UPDATE SET checked = $3, checked_by = $4, checked_at = $5`,
		studyID, itemID, checked, checkedBy, checkedAt)
	return err
}

func ListStudyChecklistResponses(ctx context.Context, db *sql.DB, studyID string) ([]StudyChecklistResponse, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, checklist_item_id, checked, checked_by, checked_at
		FROM study_checklist_responses
		WHERE study_id = $1
		ORDER BY checklist_item_id`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyChecklistResponse
	for rows.Next() {
		var r StudyChecklistResponse
		if err := rows.Scan(&r.ID, &r.StudyID, &r.ChecklistItemID, &r.Checked, &r.CheckedBy, &r.CheckedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
