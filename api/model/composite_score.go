package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// CompositeScore stores a derived summary score from neuroimaging analytics.
type CompositeScore struct {
	ID              string          `json:"id"`
	StudyID         string          `json:"study_id"`
	BaselineStudyID *string         `json:"baseline_study_id,omitempty"`
	Tool            string          `json:"tool"`
	ScoreName       string          `json:"score_name"`
	ScoreValue      float64         `json:"score_value"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
}

// CreateCompositeScore inserts a single composite score record.
func CreateCompositeScore(ctx context.Context, db *sql.DB, s *CompositeScore) error {
	meta := s.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return db.QueryRowContext(ctx, `
		INSERT INTO analytics_composite_scores
			(study_id, baseline_study_id, tool, score_name, score_value, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		s.StudyID, s.BaselineStudyID, s.Tool, s.ScoreName, s.ScoreValue, meta).
		Scan(&s.ID, &s.CreatedAt)
}

// GetCompositeScoresByStudy returns all composite scores for a study.
func GetCompositeScoresByStudy(ctx context.Context, db *sql.DB, studyID string) ([]CompositeScore, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, baseline_study_id, tool, score_name, score_value, metadata, created_at
		FROM analytics_composite_scores
		WHERE study_id = $1
		ORDER BY score_name`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompositeScore
	for rows.Next() {
		var s CompositeScore
		if err := rows.Scan(&s.ID, &s.StudyID, &s.BaselineStudyID, &s.Tool,
			&s.ScoreName, &s.ScoreValue, &s.Metadata, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteCompositeScoresByStudy removes all composite scores for a study.
func DeleteCompositeScoresByStudy(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM analytics_composite_scores WHERE study_id = $1`, studyID)
	return err
}
