package model

import (
	"context"
	"database/sql"
	"time"
)

// AnalystQCRating is a human analyst quality assessment for a study.
type AnalystQCRating struct {
	ID              string    `json:"id"`
	StudyID         string    `json:"study_id"`
	AnalystID       string    `json:"analyst_id"`
	AnalystName     string    `json:"analyst_name,omitempty"` // populated by JOIN on reads
	AnalystEmail    string    `json:"analyst_email,omitempty"`
	RatingQuality   int16     `json:"rating_quality"`
	RatingMotion    int16     `json:"rating_motion"`
	RatingSNR       *int16    `json:"rating_snr,omitempty"`
	RatingCoverage  *int16    `json:"rating_coverage,omitempty"`
	RatingArtifacts *int16    `json:"rating_artifacts,omitempty"`
	RatingOverall   int16     `json:"rating_overall"`
	Comments        string    `json:"comments"`
	ReviewType      string    `json:"review_type"`
	CreatedAt       time.Time `json:"created_at"`
}

// UpsertAnalystQCRating creates or updates a QC rating (one per analyst per review_type per study).
func UpsertAnalystQCRating(ctx context.Context, db *sql.DB, r *AnalystQCRating) error {
	return db.QueryRowContext(ctx, `
		INSERT INTO analyst_qc_ratings
			(study_id, analyst_id, rating_quality, rating_motion, rating_snr,
			 rating_coverage, rating_artifacts, rating_overall, comments, review_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (study_id, analyst_id, review_type) DO UPDATE SET
			rating_quality   = EXCLUDED.rating_quality,
			rating_motion    = EXCLUDED.rating_motion,
			rating_snr       = EXCLUDED.rating_snr,
			rating_coverage  = EXCLUDED.rating_coverage,
			rating_artifacts = EXCLUDED.rating_artifacts,
			rating_overall   = EXCLUDED.rating_overall,
			comments         = EXCLUDED.comments
		RETURNING id, created_at`,
		r.StudyID, r.AnalystID, r.RatingQuality, r.RatingMotion, r.RatingSNR,
		r.RatingCoverage, r.RatingArtifacts, r.RatingOverall, r.Comments, r.ReviewType).
		Scan(&r.ID, &r.CreatedAt)
}

// GetQCRatingsByStudy returns all QC ratings for a study, including analyst names.
func GetQCRatingsByStudy(ctx context.Context, db *sql.DB, studyID string) ([]AnalystQCRating, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.id, r.study_id, r.analyst_id,
			COALESCE(u.name, ''), COALESCE(u.email, ''),
			r.rating_quality, r.rating_motion, r.rating_snr,
			r.rating_coverage, r.rating_artifacts, r.rating_overall,
			r.comments, r.review_type, r.created_at
		FROM analyst_qc_ratings r
		LEFT JOIN admin_users u ON u.id = r.analyst_id
		WHERE r.study_id = $1
		ORDER BY r.created_at DESC`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AnalystQCRating
	for rows.Next() {
		var r AnalystQCRating
		if err := rows.Scan(&r.ID, &r.StudyID, &r.AnalystID,
			&r.AnalystName, &r.AnalystEmail,
			&r.RatingQuality, &r.RatingMotion, &r.RatingSNR,
			&r.RatingCoverage, &r.RatingArtifacts, &r.RatingOverall,
			&r.Comments, &r.ReviewType, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetQCRatingByID returns a single QC rating by ID.
func GetQCRatingByID(ctx context.Context, db *sql.DB, id string) (*AnalystQCRating, error) {
	var r AnalystQCRating
	err := db.QueryRowContext(ctx, `
		SELECT id, study_id, analyst_id, rating_quality, rating_motion, rating_snr,
			rating_coverage, rating_artifacts, rating_overall, comments, review_type, created_at
		FROM analyst_qc_ratings WHERE id = $1`, id).
		Scan(&r.ID, &r.StudyID, &r.AnalystID, &r.RatingQuality, &r.RatingMotion, &r.RatingSNR,
			&r.RatingCoverage, &r.RatingArtifacts, &r.RatingOverall, &r.Comments, &r.ReviewType, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteAnalystQCRating removes a QC rating by ID.
func DeleteAnalystQCRating(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM analyst_qc_ratings WHERE id = $1`, id)
	return err
}
