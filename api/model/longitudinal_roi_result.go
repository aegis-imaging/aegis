package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// LongitudinalROIResult stores a paired baseline/follow-up ROI measurement.
type LongitudinalROIResult struct {
	ID               string    `json:"id"`
	BaselineStudyID  string    `json:"baseline_study_id"`
	FollowupStudyID  string    `json:"followup_study_id"`
	Tool             string    `json:"tool"`
	AtlasName        string    `json:"atlas_name"`
	ROIName          string    `json:"roi_name"`
	BaselineValue    *float64  `json:"baseline_value,omitempty"`
	FollowupValue    *float64  `json:"followup_value,omitempty"`
	ChangeValue      *float64  `json:"change_value,omitempty"`
	ChangePercent    *float64  `json:"change_percent,omitempty"`
	AnnualizedChange *float64  `json:"annualized_change,omitempty"`
	MetricType       string    `json:"metric_type"`
	ScanIntervalDays int       `json:"scan_interval_days"`
	Hemisphere       string    `json:"hemisphere"`
	CreatedAt        time.Time `json:"created_at"`
}

// CreateLongitudinalROIResults bulk-inserts longitudinal ROI results.
func CreateLongitudinalROIResults(ctx context.Context, db *sql.DB, results []LongitudinalROIResult) error {
	if len(results) == 0 {
		return nil
	}
	const batchSize = 500
	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}
		if err := insertLongROIBatch(ctx, db, results[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func insertLongROIBatch(ctx context.Context, db *sql.DB, batch []LongitudinalROIResult) error {
	if len(batch) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO longitudinal_roi_results
		(baseline_study_id, followup_study_id, tool, atlas_name, roi_name,
		 baseline_value, followup_value, change_value, change_percent,
		 annualized_change, metric_type, scan_interval_days, hemisphere) VALUES `)

	args := make([]any, 0, len(batch)*13)
	for i, r := range batch {
		if i > 0 {
			b.WriteString(", ")
		}
		base := i * 13
		fmt.Fprintf(&b, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7,
			base+8, base+9, base+10, base+11, base+12, base+13)
		args = append(args, r.BaselineStudyID, r.FollowupStudyID, r.Tool, r.AtlasName,
			r.ROIName, r.BaselineValue, r.FollowupValue, r.ChangeValue,
			r.ChangePercent, r.AnnualizedChange, r.MetricType, r.ScanIntervalDays, r.Hemisphere)
	}

	_, err := db.ExecContext(ctx, b.String(), args...)
	return err
}

// GetLongitudinalROIResults returns longitudinal ROI results for a pair of studies.
func GetLongitudinalROIResults(ctx context.Context, db *sql.DB, baselineStudyID, followupStudyID string) ([]LongitudinalROIResult, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, baseline_study_id, followup_study_id, tool, atlas_name, roi_name,
			baseline_value, followup_value, change_value, change_percent,
			annualized_change, metric_type, scan_interval_days, hemisphere, created_at
		FROM longitudinal_roi_results
		WHERE baseline_study_id = $1 AND followup_study_id = $2
		ORDER BY roi_name`, baselineStudyID, followupStudyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LongitudinalROIResult
	for rows.Next() {
		var r LongitudinalROIResult
		if err := rows.Scan(&r.ID, &r.BaselineStudyID, &r.FollowupStudyID, &r.Tool,
			&r.AtlasName, &r.ROIName, &r.BaselineValue, &r.FollowupValue,
			&r.ChangeValue, &r.ChangePercent, &r.AnnualizedChange,
			&r.MetricType, &r.ScanIntervalDays, &r.Hemisphere, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetLongitudinalROIResultsByStudy returns all longitudinal results where the
// given study is either the baseline or the follow-up.
func GetLongitudinalROIResultsByStudy(ctx context.Context, db *sql.DB, studyID string) ([]LongitudinalROIResult, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, baseline_study_id, followup_study_id, tool, atlas_name, roi_name,
			baseline_value, followup_value, change_value, change_percent,
			annualized_change, metric_type, scan_interval_days, hemisphere, created_at
		FROM longitudinal_roi_results
		WHERE baseline_study_id = $1 OR followup_study_id = $1
		ORDER BY roi_name`, studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LongitudinalROIResult
	for rows.Next() {
		var r LongitudinalROIResult
		if err := rows.Scan(&r.ID, &r.BaselineStudyID, &r.FollowupStudyID, &r.Tool,
			&r.AtlasName, &r.ROIName, &r.BaselineValue, &r.FollowupValue,
			&r.ChangeValue, &r.ChangePercent, &r.AnnualizedChange,
			&r.MetricType, &r.ScanIntervalDays, &r.Hemisphere, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DeleteLongitudinalROIResults removes all longitudinal results for a follow-up study.
func DeleteLongitudinalROIResults(ctx context.Context, db *sql.DB, followupStudyID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM longitudinal_roi_results WHERE followup_study_id = $1`, followupStudyID)
	return err
}
