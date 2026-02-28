package handler

import (
	"context"
	"encoding/json"
	"log"
	"math"

	"github.com/aegis-imaging/aegis/api/model"
)

// storeROIResults parses analytics tool results and persists structured ROI data.
// Called after single-study analytics complete (FreeSurfer, Atlas ROI, etc.).
func (s *Server) storeROIResults(ctx context.Context, study *model.Study, results []analyticsToolResult) {
	for _, r := range results {
		if !r.Success || r.Metrics == nil {
			continue
		}
		switch r.Tool {
		case "freesurfer":
			s.storeFreeSurferROIs(ctx, study, r.Metrics)
		case "atlas_roi":
			s.storeAtlasROIResults(ctx, study, r.Metrics)
		case "synthseg":
			s.storeSynthSegROIs(ctx, study, r.Metrics)
		case "nnunet":
			s.storeGenericSegROIs(ctx, study, r.Metrics, "nnunet")
		case "totalsegmentator":
			s.storeGenericSegROIs(ctx, study, r.Metrics, "totalsegmentator")
		}
	}
}

// storeLongitudinalROIResults parses longitudinal analytics results and persists
// paired baseline/follow-up ROI data. Called after longitudinal analytics complete.
func (s *Server) storeLongitudinalROIResults(
	ctx context.Context,
	followup, baseline *model.Study,
	results []analyticsToolResult,
	scanIntervalDays float64,
) {
	for _, r := range results {
		if !r.Success || r.Metrics == nil {
			continue
		}
		switch r.Tool {
		case "tbm_syn":
			s.storeTBMSyNResults(ctx, followup, baseline, r.Metrics, scanIntervalDays)
		case "freesurfer_long":
			s.storeFreeSurferLongResults(ctx, followup, baseline, r.Metrics, scanIntervalDays)
		}
	}
}

// storeFreeSurferROIs extracts subcortical volumes and cortical parcellations
// from FreeSurfer recon-all metrics and stores them as ROI results.
func (s *Server) storeFreeSurferROIs(ctx context.Context, study *model.Study, metrics map[string]any) {
	// Delete existing results for idempotent re-processing.
	if err := model.DeleteROIResultsByStudy(ctx, s.db, study.ID); err != nil {
		log.Printf("roi-storage: delete existing ROIs for %s: %v", study.StudyInstanceUID, err)
		return
	}

	var rows []model.ROIResult

	// Subcortical volumes (aseg atlas)
	if sub, ok := metrics["subcortical_volumes"].(map[string]any); ok {
		for name, val := range sub {
			v := toFloat64(val)
			rows = append(rows, model.ROIResult{
				StudyID:    study.ID,
				Tool:       "freesurfer",
				AtlasName:  "aseg",
				ROIName:    name,
				MetricType: "volume_mm3",
				MetricValue: v,
				Hemisphere: "B",
				ScanType:   "T1w",
			})
		}
	}

	// Left hemisphere cortical parcellation (aparc atlas)
	if lh, ok := metrics["lh_cortical_parcellation"].(map[string]any); ok {
		for name, val := range lh {
			v := toFloat64(val)
			rows = append(rows, model.ROIResult{
				StudyID:    study.ID,
				Tool:       "freesurfer",
				AtlasName:  "aparc",
				ROIName:    name,
				MetricType: "volume_mm3",
				MetricValue: v,
				Hemisphere: "L",
				ScanType:   "T1w",
			})
		}
	}

	// Right hemisphere cortical parcellation (aparc atlas)
	if rh, ok := metrics["rh_cortical_parcellation"].(map[string]any); ok {
		for name, val := range rh {
			v := toFloat64(val)
			rows = append(rows, model.ROIResult{
				StudyID:    study.ID,
				Tool:       "freesurfer",
				AtlasName:  "aparc",
				ROIName:    name,
				MetricType: "volume_mm3",
				MetricValue: v,
				Hemisphere: "R",
				ScanType:   "T1w",
			})
		}
	}

	if len(rows) == 0 {
		return
	}

	if err := model.CreateROIResults(ctx, s.db, rows); err != nil {
		log.Printf("roi-storage: insert FreeSurfer ROIs for %s: %v", study.StudyInstanceUID, err)
		return
	}
	log.Printf("roi-storage: stored %d FreeSurfer ROI results for %s", len(rows), study.StudyInstanceUID)
}

// storeAtlasROIResults extracts per-ROI volumes and stats from Atlas ROI metrics.
func (s *Server) storeAtlasROIResults(ctx context.Context, study *model.Study, metrics map[string]any) {
	if err := model.DeleteROIResultsByStudy(ctx, s.db, study.ID); err != nil {
		log.Printf("roi-storage: delete existing ROIs for %s: %v", study.StudyInstanceUID, err)
		return
	}

	atlasName, _ := metrics["atlas"].(string)
	if atlasName == "" {
		atlasName = "unknown"
	}

	var rows []model.ROIResult

	// roi_volumes: map[roi_name] → volume_mm3
	if vols, ok := metrics["roi_volumes"].(map[string]any); ok {
		for name, val := range vols {
			rows = append(rows, model.ROIResult{
				StudyID:     study.ID,
				Tool:        "atlas_roi",
				AtlasName:   atlasName,
				ROIName:     name,
				MetricType:  "volume_mm3",
				MetricValue: toFloat64(val),
				Hemisphere:  hemisphereFromName(name),
				ScanType:    "T1w",
			})
		}
	}

	// roi_stats: []map with roi_name, mean, median, std, voxel_count
	if stats, ok := metrics["roi_stats"].([]any); ok {
		for _, item := range stats {
			stat, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := stat["roi_name"].(string)
			if name == "" {
				continue
			}
			hem := hemisphereFromName(name)

			for _, metricKey := range []string{"mean", "median", "std"} {
				if v, exists := stat[metricKey]; exists {
					rows = append(rows, model.ROIResult{
						StudyID:     study.ID,
						Tool:        "atlas_roi",
						AtlasName:   atlasName,
						ROIName:     name,
						MetricType:  metricKey,
						MetricValue: toFloat64(v),
						Hemisphere:  hem,
						ScanType:    "T1w",
					})
				}
			}
		}
	}

	if len(rows) == 0 {
		return
	}

	if err := model.CreateROIResults(ctx, s.db, rows); err != nil {
		log.Printf("roi-storage: insert Atlas ROI results for %s: %v", study.StudyInstanceUID, err)
		return
	}
	log.Printf("roi-storage: stored %d Atlas ROI results for %s", len(rows), study.StudyInstanceUID)
}

// storeSynthSegROIs extracts per-ROI volumes from SynthSeg metrics and stores
// them as ROI results. Additionally stores a QC composite score when available.
func (s *Server) storeSynthSegROIs(ctx context.Context, study *model.Study, metrics map[string]any) {
	if err := model.DeleteROIResultsByStudy(ctx, s.db, study.ID); err != nil {
		log.Printf("roi-storage: delete existing ROIs for %s: %v", study.StudyInstanceUID, err)
		return
	}

	atlasName, _ := metrics["atlas"].(string)
	if atlasName == "" {
		atlasName = "synthseg"
	}

	var rows []model.ROIResult

	// roi_volumes: map[roi_name] → volume_mm3
	if vols, ok := metrics["roi_volumes"].(map[string]any); ok {
		for name, val := range vols {
			rows = append(rows, model.ROIResult{
				StudyID:     study.ID,
				Tool:        "synthseg",
				AtlasName:   atlasName,
				ROIName:     name,
				MetricType:  "volume_mm3",
				MetricValue: toFloat64(val),
				Hemisphere:  synthsegHemisphere(name),
				ScanType:    "T1w",
			})
		}
	}

	if len(rows) == 0 {
		return
	}

	if err := model.CreateROIResults(ctx, s.db, rows); err != nil {
		log.Printf("roi-storage: insert SynthSeg ROIs for %s: %v", study.StudyInstanceUID, err)
		return
	}
	log.Printf("roi-storage: stored %d SynthSeg ROI results for %s", len(rows), study.StudyInstanceUID)

	// Store QC composite score if available.
	if qcScore, ok := metrics["qc_score"]; ok && qcScore != nil {
		meta, _ := json.Marshal(map[string]any{
			"parcellation": metrics["parcellation"],
			"roi_count":    metrics["roi_count"],
		})
		score := &model.CompositeScore{
			StudyID:    study.ID,
			Tool:       "synthseg",
			ScoreName:  "synthseg_qc",
			ScoreValue: toFloat64(qcScore),
			Metadata:   meta,
		}
		if err := model.DeleteCompositeScoresByStudy(ctx, s.db, study.ID); err != nil {
			log.Printf("roi-storage: delete existing composite scores for %s: %v", study.StudyInstanceUID, err)
		}
		if err := model.CreateCompositeScore(ctx, s.db, score); err != nil {
			log.Printf("roi-storage: insert SynthSeg QC score for %s: %v", study.StudyInstanceUID, err)
		}
	}
}

// storeGenericSegROIs extracts per-ROI volumes and intensity stats from
// segmentation-based backends (nnU-Net, TotalSegmentator). These share the
// same metrics structure: atlas, roi_volumes, roi_stats.
func (s *Server) storeGenericSegROIs(ctx context.Context, study *model.Study, metrics map[string]any, toolName string) {
	if err := model.DeleteROIResultsByStudy(ctx, s.db, study.ID); err != nil {
		log.Printf("roi-storage: delete existing ROIs for %s: %v", study.StudyInstanceUID, err)
		return
	}

	atlasName, _ := metrics["atlas"].(string)
	if atlasName == "" {
		atlasName = toolName
	}

	var rows []model.ROIResult

	// roi_volumes: map[roi_name] → volume_mm3
	if vols, ok := metrics["roi_volumes"].(map[string]any); ok {
		for name, val := range vols {
			rows = append(rows, model.ROIResult{
				StudyID:     study.ID,
				Tool:        toolName,
				AtlasName:   atlasName,
				ROIName:     name,
				MetricType:  "volume_mm3",
				MetricValue: toFloat64(val),
				Hemisphere:  hemisphereFromName(name),
				ScanType:    "T1w",
			})
		}
	}

	// roi_stats: []map with roi_name, mean, median, std, voxel_count
	if stats, ok := metrics["roi_stats"].([]any); ok {
		for _, item := range stats {
			stat, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := stat["roi_name"].(string)
			if name == "" {
				continue
			}
			hem := hemisphereFromName(name)

			for _, metricKey := range []string{"mean", "median", "std"} {
				if v, exists := stat[metricKey]; exists {
					rows = append(rows, model.ROIResult{
						StudyID:     study.ID,
						Tool:        toolName,
						AtlasName:   atlasName,
						ROIName:     name,
						MetricType:  metricKey,
						MetricValue: toFloat64(v),
						Hemisphere:  hem,
						ScanType:    "T1w",
					})
				}
			}
		}
	}

	if len(rows) == 0 {
		return
	}

	if err := model.CreateROIResults(ctx, s.db, rows); err != nil {
		log.Printf("roi-storage: insert %s ROIs for %s: %v", toolName, study.StudyInstanceUID, err)
		return
	}
	log.Printf("roi-storage: stored %d %s ROI results for %s", len(rows), toolName, study.StudyInstanceUID)
}

// synthsegHemisphere detects hemisphere from SynthSeg ROI names which use
// "Left-" / "Right-" or "ctx-lh-" / "ctx-rh-" prefixes.
func synthsegHemisphere(name string) string {
	if len(name) >= 5 {
		if name[:5] == "Left-" {
			return "L"
		}
		if name[:6] == "Right-" {
			return "R"
		}
	}
	if len(name) >= 6 {
		if name[:6] == "ctx-lh" {
			return "L"
		}
		if name[:6] == "ctx-rh" {
			return "R"
		}
	}
	return hemisphereFromName(name)
}

// storeTBMSyNResults extracts per-ROI atrophy measurements from TBM-SyN metrics.
func (s *Server) storeTBMSyNResults(
	ctx context.Context,
	followup, baseline *model.Study,
	metrics map[string]any,
	scanIntervalDays float64,
) {
	// Delete existing longitudinal results for this follow-up (idempotent).
	if err := model.DeleteLongitudinalROIResults(ctx, s.db, followup.ID); err != nil {
		log.Printf("roi-storage: delete existing longitudinal ROIs for %s: %v", followup.StudyInstanceUID, err)
		return
	}

	var longRows []model.LongitudinalROIResult
	intervalDays := int(math.Round(scanIntervalDays))

	// roi_atrophy: []map with roi_name, mean, voxel_count, etc.
	if atrophy, ok := metrics["roi_atrophy"].([]any); ok {
		for _, item := range atrophy {
			stat, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, _ := stat["roi_name"].(string)
			if name == "" {
				continue
			}
			meanVal := toFloat64Ptr(stat["mean"])
			annualized := meanVal // TBM-SyN log-Jacobian is already annualized

			longRows = append(longRows, model.LongitudinalROIResult{
				BaselineStudyID:  baseline.ID,
				FollowupStudyID:  followup.ID,
				Tool:             "tbm_syn",
				AtlasName:        stringOrDefault(metrics["atlas"], "unknown"),
				ROIName:          name,
				ChangeValue:      meanVal,
				AnnualizedChange: annualized,
				MetricType:       "log_jacobian",
				ScanIntervalDays: intervalDays,
				Hemisphere:       hemisphereFromName(name),
			})
		}
	}

	if len(longRows) > 0 {
		if err := model.CreateLongitudinalROIResults(ctx, s.db, longRows); err != nil {
			log.Printf("roi-storage: insert TBM-SyN longitudinal ROIs for %s: %v", followup.StudyInstanceUID, err)
			return
		}
		log.Printf("roi-storage: stored %d TBM-SyN longitudinal ROI results for %s", len(longRows), followup.StudyInstanceUID)
	}

	// Store AD composite score
	if adMean, ok := metrics["ad_composite_mean"]; ok && adMean != nil {
		meta, _ := json.Marshal(map[string]any{
			"contributing_roi_count": metrics["ad_composite_roi_count"],
			"scan_interval_days":    scanIntervalDays,
		})
		score := &model.CompositeScore{
			StudyID:         followup.ID,
			BaselineStudyID: &baseline.ID,
			Tool:            "tbm_syn",
			ScoreName:       "ad_composite_mean",
			ScoreValue:      toFloat64(adMean),
			Metadata:        meta,
		}
		if err := model.DeleteCompositeScoresByStudy(ctx, s.db, followup.ID); err != nil {
			log.Printf("roi-storage: delete existing composite scores for %s: %v", followup.StudyInstanceUID, err)
		}
		if err := model.CreateCompositeScore(ctx, s.db, score); err != nil {
			log.Printf("roi-storage: insert AD composite score for %s: %v", followup.StudyInstanceUID, err)
		}
	}
}

// storeFreeSurferLongResults extracts baseline/follow-up volumes and percent
// change from FreeSurfer longitudinal stream metrics.
func (s *Server) storeFreeSurferLongResults(
	ctx context.Context,
	followup, baseline *model.Study,
	metrics map[string]any,
	scanIntervalDays float64,
) {
	if err := model.DeleteLongitudinalROIResults(ctx, s.db, followup.ID); err != nil {
		log.Printf("roi-storage: delete existing longitudinal ROIs for %s: %v", followup.StudyInstanceUID, err)
		return
	}

	intervalDays := int(math.Round(scanIntervalDays))
	var longRows []model.LongitudinalROIResult

	// volume_change_percent: map[roi_name] → percent_change
	changeMap, _ := metrics["volume_change_percent"].(map[string]any)

	// baseline_subcortical_volumes, followup_subcortical_volumes
	blSub, _ := metrics["baseline_subcortical_volumes"].(map[string]any)
	fuSub, _ := metrics["followup_subcortical_volumes"].(map[string]any)
	longRows = appendFSLongROIs(longRows, baseline.ID, followup.ID, "aseg", "B",
		blSub, fuSub, changeMap, intervalDays, scanIntervalDays)

	// baseline_lh_cortical, followup_lh_cortical
	blLH, _ := metrics["baseline_lh_cortical"].(map[string]any)
	fuLH, _ := metrics["followup_lh_cortical"].(map[string]any)
	longRows = appendFSLongROIs(longRows, baseline.ID, followup.ID, "aparc", "L",
		blLH, fuLH, changeMap, intervalDays, scanIntervalDays)

	// baseline_rh_cortical, followup_rh_cortical
	blRH, _ := metrics["baseline_rh_cortical"].(map[string]any)
	fuRH, _ := metrics["followup_rh_cortical"].(map[string]any)
	longRows = appendFSLongROIs(longRows, baseline.ID, followup.ID, "aparc", "R",
		blRH, fuRH, changeMap, intervalDays, scanIntervalDays)

	if len(longRows) == 0 {
		return
	}

	if err := model.CreateLongitudinalROIResults(ctx, s.db, longRows); err != nil {
		log.Printf("roi-storage: insert FreeSurfer Long ROIs for %s: %v", followup.StudyInstanceUID, err)
		return
	}
	log.Printf("roi-storage: stored %d FreeSurfer Long ROI results for %s", len(longRows), followup.StudyInstanceUID)
}

// appendFSLongROIs builds longitudinal ROI rows from paired FreeSurfer volume maps.
func appendFSLongROIs(
	rows []model.LongitudinalROIResult,
	baselineID, followupID string,
	atlas, hemisphere string,
	blVols, fuVols, changeMap map[string]any,
	intervalDays int,
	scanIntervalDays float64,
) []model.LongitudinalROIResult {
	if blVols == nil && fuVols == nil {
		return rows
	}

	// Collect all ROI names from both timepoints.
	names := make(map[string]bool)
	for n := range blVols {
		names[n] = true
	}
	for n := range fuVols {
		names[n] = true
	}

	years := scanIntervalDays / 365.25

	for name := range names {
		r := model.LongitudinalROIResult{
			BaselineStudyID:  baselineID,
			FollowupStudyID:  followupID,
			Tool:             "freesurfer_long",
			AtlasName:        atlas,
			ROIName:          name,
			MetricType:       "volume_mm3",
			ScanIntervalDays: intervalDays,
			Hemisphere:       hemisphere,
		}

		if blVols != nil {
			r.BaselineValue = toFloat64Ptr(blVols[name])
		}
		if fuVols != nil {
			r.FollowupValue = toFloat64Ptr(fuVols[name])
		}

		// Change and percent from the precomputed map.
		if r.BaselineValue != nil && r.FollowupValue != nil {
			diff := *r.FollowupValue - *r.BaselineValue
			r.ChangeValue = &diff
		}
		if changeMap != nil {
			r.ChangePercent = toFloat64Ptr(changeMap[name])
		}

		// Annualized change = change / years
		if r.ChangeValue != nil && years > 0 {
			ann := *r.ChangeValue / years
			r.AnnualizedChange = &ann
		}

		rows = append(rows, r)
	}
	return rows
}

// toFloat64 converts an any value to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

// toFloat64Ptr converts an any value to *float64, returning nil for nil/missing values.
func toFloat64Ptr(v any) *float64 {
	if v == nil {
		return nil
	}
	f := toFloat64(v)
	return &f
}

// hemisphereFromName guesses hemisphere from an ROI name suffix.
func hemisphereFromName(name string) string {
	if len(name) < 2 {
		return "B"
	}
	suffix := name[len(name)-2:]
	switch suffix {
	case "_L":
		return "L"
	case "_R":
		return "R"
	}
	return "B"
}

// stringOrDefault extracts a string from a map value, with fallback.
func stringOrDefault(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}
