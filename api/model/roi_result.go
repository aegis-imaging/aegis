package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ROIResult stores a single ROI measurement from a neuroimaging analytics tool.
type ROIResult struct {
	ID           string    `json:"id"`
	StudyID      string    `json:"study_id"`
	Tool         string    `json:"tool"`
	AtlasName    string    `json:"atlas_name"`
	TemplateName string    `json:"template_name"`
	ROINumber    int       `json:"roi_number"`
	ROIName      string    `json:"roi_name"`
	MetricType   string    `json:"metric_type"`
	MetricValue  float64   `json:"metric_value"`
	Hemisphere   string    `json:"hemisphere"`
	ScanType     string    `json:"scan_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// ROIResultFilters are optional filters for querying ROI results.
type ROIResultFilters struct {
	Tool       string
	AtlasName  string
	MetricType string
	ROIName    string
	Hemisphere string
	Limit      int
}

// CreateROIResults bulk-inserts ROI results in batches of 500 rows.
func CreateROIResults(ctx context.Context, db *sql.DB, results []ROIResult) error {
	if len(results) == 0 {
		return nil
	}
	const batchSize = 500
	for i := 0; i < len(results); i += batchSize {
		end := i + batchSize
		if end > len(results) {
			end = len(results)
		}
		if err := insertROIBatch(ctx, db, results[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func insertROIBatch(ctx context.Context, db *sql.DB, batch []ROIResult) error {
	if len(batch) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO roi_results (study_id, tool, atlas_name, template_name, roi_number, roi_name, metric_type, metric_value, hemisphere, scan_type) VALUES `)

	args := make([]any, 0, len(batch)*10)
	for i, r := range batch {
		if i > 0 {
			b.WriteString(", ")
		}
		base := i * 10
		fmt.Fprintf(&b, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10)
		args = append(args, r.StudyID, r.Tool, r.AtlasName, r.TemplateName, r.ROINumber,
			r.ROIName, r.MetricType, r.MetricValue, r.Hemisphere, r.ScanType)
	}

	_, err := db.ExecContext(ctx, b.String(), args...)
	return err
}

// GetROIResultsByStudy returns ROI results for a study with optional filters.
func GetROIResultsByStudy(ctx context.Context, db *sql.DB, studyID string, f ROIResultFilters) ([]ROIResult, error) {
	var where []string
	var args []any
	idx := 1

	where = append(where, fmt.Sprintf("study_id = $%d", idx))
	args = append(args, studyID)
	idx++

	if f.Tool != "" {
		where = append(where, fmt.Sprintf("tool = $%d", idx))
		args = append(args, f.Tool)
		idx++
	}
	if f.AtlasName != "" {
		where = append(where, fmt.Sprintf("atlas_name = $%d", idx))
		args = append(args, f.AtlasName)
		idx++
	}
	if f.MetricType != "" {
		where = append(where, fmt.Sprintf("metric_type = $%d", idx))
		args = append(args, f.MetricType)
		idx++
	}
	if f.ROIName != "" {
		where = append(where, fmt.Sprintf("roi_name = $%d", idx))
		args = append(args, f.ROIName)
		idx++
	}
	if f.Hemisphere != "" {
		where = append(where, fmt.Sprintf("hemisphere = $%d", idx))
		args = append(args, f.Hemisphere)
		idx++
	}

	limit := 500
	if f.Limit > 0 && f.Limit <= 2000 {
		limit = f.Limit
	}

	q := fmt.Sprintf(`SELECT id, study_id, tool, atlas_name, template_name, roi_number,
		roi_name, metric_type, metric_value, hemisphere, scan_type, created_at
		FROM roi_results WHERE %s ORDER BY roi_name, metric_type LIMIT %d`,
		strings.Join(where, " AND "), limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ROIResult
	for rows.Next() {
		var r ROIResult
		if err := rows.Scan(&r.ID, &r.StudyID, &r.Tool, &r.AtlasName, &r.TemplateName,
			&r.ROINumber, &r.ROIName, &r.MetricType, &r.MetricValue,
			&r.Hemisphere, &r.ScanType, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ROIResultSummary provides aggregate counts about ROI results for a study.
type ROIResultSummary struct {
	StudyID     string   `json:"study_id"`
	TotalROIs   int      `json:"total_rois"`
	Tools       []string `json:"tools"`
	Atlases     []string `json:"atlases"`
	MetricTypes []string `json:"metric_types"`
}

// GetROIResultsSummary returns an aggregate summary of ROI results for a study.
func GetROIResultsSummary(ctx context.Context, db *sql.DB, studyID string) (*ROIResultSummary, error) {
	s := &ROIResultSummary{StudyID: studyID}

	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM roi_results WHERE study_id = $1`, studyID).Scan(&s.TotalROIs)
	if err != nil {
		return nil, err
	}
	if s.TotalROIs == 0 {
		return s, nil
	}

	s.Tools, _ = queryDistinct(ctx, db, "tool", studyID)
	s.Atlases, _ = queryDistinct(ctx, db, "atlas_name", studyID)
	s.MetricTypes, _ = queryDistinct(ctx, db, "metric_type", studyID)
	return s, nil
}

func queryDistinct(ctx context.Context, db *sql.DB, col, studyID string) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		fmt.Sprintf("SELECT DISTINCT %s FROM roi_results WHERE study_id = $1 AND %s != '' ORDER BY %s", col, col, col),
		studyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteROIResultsByStudy removes all ROI results for a study (for re-processing).
func DeleteROIResultsByStudy(ctx context.Context, db *sql.DB, studyID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM roi_results WHERE study_id = $1`, studyID)
	return err
}

// CountROIResultsByStudy returns the number of ROI result rows for a study.
func CountROIResultsByStudy(ctx context.Context, db *sql.DB, studyID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM roi_results WHERE study_id = $1`, studyID).Scan(&n)
	return n, err
}
