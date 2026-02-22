package model

import (
	"context"
	"database/sql"
	"time"
)

// ProjectPhiConfig stores per-project PHI detection tuning overrides.
type ProjectPhiConfig struct {
	ProjectID           string    `json:"project_id"`
	ConfidenceThreshold float64   `json:"confidence_threshold"`
	MinTextLength       int       `json:"min_text_length"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// GetProjectPhiConfig returns the PHI config for a project, returning defaults
// if no override has been set.
func GetProjectPhiConfig(ctx context.Context, db *sql.DB, projectID string) (*ProjectPhiConfig, error) {
	cfg := &ProjectPhiConfig{
		ProjectID:           projectID,
		ConfidenceThreshold: 0.4,
		MinTextLength:       3,
		UpdatedAt:           time.Now().UTC(),
	}
	err := db.QueryRowContext(ctx, `
		SELECT confidence_threshold, min_text_length, updated_at
		FROM project_phi_config WHERE project_id = $1`, projectID).
		Scan(&cfg.ConfidenceThreshold, &cfg.MinTextLength, &cfg.UpdatedAt)
	if err == sql.ErrNoRows {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// UpsertProjectPhiConfig creates or replaces the PHI config for a project.
func UpsertProjectPhiConfig(ctx context.Context, db *sql.DB, projectID string, threshold float64, minLen int) (*ProjectPhiConfig, error) {
	cfg := &ProjectPhiConfig{ProjectID: projectID}
	err := db.QueryRowContext(ctx, `
		INSERT INTO project_phi_config (project_id, confidence_threshold, min_text_length)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id) DO UPDATE
		  SET confidence_threshold = EXCLUDED.confidence_threshold,
		      min_text_length      = EXCLUDED.min_text_length,
		      updated_at           = now()
		RETURNING confidence_threshold, min_text_length, updated_at`,
		projectID, threshold, minLen).
		Scan(&cfg.ConfidenceThreshold, &cfg.MinTextLength, &cfg.UpdatedAt)
	return cfg, err
}
