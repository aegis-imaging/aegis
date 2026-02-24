package model

import (
	"context"
	"database/sql"
	"time"
)

// StudySeries holds per-series DICOM metadata for a study.
type StudySeries struct {
	ID                string    `json:"id"`
	StudyID           string    `json:"study_id"`
	SeriesInstanceUID string    `json:"series_instance_uid"`
	SeriesDescription string    `json:"series_description,omitempty"`
	Modality          string    `json:"modality,omitempty"`
	BodyPart          string    `json:"body_part,omitempty"`
	InstanceCount     int       `json:"instance_count"`
	CreatedAt         time.Time `json:"created_at"`
}

// UpsertStudySeries inserts or ignores a series row (idempotent on duplicate series_instance_uid).
func UpsertStudySeries(ctx context.Context, db *sql.DB, s *StudySeries) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO study_series (study_id, series_instance_uid, series_description, modality, body_part, instance_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (study_id, series_instance_uid) DO UPDATE
		  SET series_description = EXCLUDED.series_description,
		      modality           = EXCLUDED.modality,
		      body_part          = EXCLUDED.body_part,
		      instance_count     = EXCLUDED.instance_count`,
		s.StudyID, s.SeriesInstanceUID, s.SeriesDescription, s.Modality, s.BodyPart, s.InstanceCount,
	)
	return err
}

// ListStudySeries returns all series for a study ordered by series_instance_uid.
func ListStudySeries(ctx context.Context, db *sql.DB, studyID string) ([]StudySeries, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, study_id, series_instance_uid, coalesce(series_description,''),
		       coalesce(modality,''), coalesce(body_part,''), instance_count, created_at
		FROM study_series
		WHERE study_id = $1
		ORDER BY series_instance_uid`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var series []StudySeries
	for rows.Next() {
		var s StudySeries
		if err := rows.Scan(&s.ID, &s.StudyID, &s.SeriesInstanceUID,
			&s.SeriesDescription, &s.Modality, &s.BodyPart, &s.InstanceCount, &s.CreatedAt); err != nil {
			return nil, err
		}
		series = append(series, s)
	}
	return series, rows.Err()
}
