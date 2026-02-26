package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type CustomFieldDefinition struct {
	ID        string          `json:"id"`
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	FieldType string          `json:"field_type"`
	Options   json.RawMessage `json:"options"`
	Required  bool            `json:"required"`
	CreatedAt time.Time       `json:"created_at"`
}

type StudyCustomFieldValue struct {
	ID        string    `json:"id"`
	StudyID   string    `json:"study_id"`
	FieldID   string    `json:"field_id"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CreateCustomFieldDefinition(ctx context.Context, db *sql.DB, d *CustomFieldDefinition) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO custom_field_definitions (project_id, name, field_type, options, required)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		d.ProjectID, d.Name, d.FieldType, d.Options, d.Required).
		Scan(&d.ID, &d.CreatedAt)
}

func ListCustomFieldDefinitions(ctx context.Context, db *sql.DB, projectID string) ([]CustomFieldDefinition, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, project_id, name, field_type, options, required, created_at
		FROM custom_field_definitions
		WHERE project_id = $1
		ORDER BY name ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomFieldDefinition
	for rows.Next() {
		var d CustomFieldDefinition
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Name, &d.FieldType, &d.Options, &d.Required, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func DeleteCustomFieldDefinition(ctx context.Context, db *sql.DB, id, projectID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM custom_field_definitions WHERE id = $1 AND project_id = $2`, id, projectID)
	return err
}

func UpsertStudyCustomFieldValue(ctx context.Context, db *sql.DB, studyID, fieldID, value string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO study_custom_field_values (study_id, field_id, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (study_id, field_id) DO UPDATE SET value = $3, updated_at = now()`,
		studyID, fieldID, value)
	return err
}

func ListStudyCustomFieldValues(ctx context.Context, db *sql.DB, studyID string) ([]StudyCustomFieldValue, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, field_id, value, updated_at
		FROM study_custom_field_values
		WHERE study_id = $1
		ORDER BY field_id`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudyCustomFieldValue
	for rows.Next() {
		var v StudyCustomFieldValue
		if err := rows.Scan(&v.ID, &v.StudyID, &v.FieldID, &v.Value, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
