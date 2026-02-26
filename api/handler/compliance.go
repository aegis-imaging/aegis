package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// complianceReport is the full compliance summary for a project.
type complianceReport struct {
	ProjectID   string                   `json:"project_id"`
	GeneratedAt string                   `json:"generated_at"`
	PeriodDays  int                      `json:"period_days"`
	Studies     complianceStudies        `json:"studies"`
	PhiDetect   compliancePhiDetect      `json:"phi_detection"`
	Defacing    complianceDefacing       `json:"defacing"`
	Protocol    complianceProtocol       `json:"protocol_compliance"`
	Exports     complianceExports        `json:"exports"`
}

type complianceStudies struct {
	Total    int `json:"total"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
	Pending  int `json:"pending"`
}

type compliancePhiDetect struct {
	Scanned     int     `json:"scanned"`
	Flagged     int     `json:"flagged"`
	FlagRatePct float64 `json:"flag_rate_pct"`
}

type complianceDefacing struct {
	Required    int     `json:"required"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	AvgQAScore  float64 `json:"avg_qa_score"`
}

type complianceProtocol struct {
	Checked          int `json:"checked"`
	Compliant        int `json:"compliant"`
	MinorDeviations  int `json:"minor_deviations"`
	NonCompliant     int `json:"non_compliant"`
}

type complianceExports struct {
	SharesCreated    int `json:"shares_created"`
	SharesDownloaded int `json:"shares_downloaded"`
	TotalDownloads   int `json:"total_downloads"`
}

// GetProjectComplianceReport generates a compliance report for a project.
//
// GET /api/projects/{id}/compliance-report?days=30
func (s *Server) GetProjectComplianceReport(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project id")
		return
	}

	// Validate project exists.
	if _, err := model.GetProjectByID(r.Context(), s.db, projectID); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	since := time.Now().UTC().AddDate(0, 0, -days)
	report := complianceReport{
		ProjectID:   projectID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		PeriodDays:  days,
	}

	// ── Studies ──────────────────────────────────────────────────────────────
	row := s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*)                                              AS total,
			COUNT(*) FILTER (WHERE status = 'approved')          AS approved,
			COUNT(*) FILTER (WHERE status = 'rejected')          AS rejected,
			COUNT(*) FILTER (WHERE status NOT IN ('approved','rejected','expired')) AS pending
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since)
	row.Scan(
		&report.Studies.Total,
		&report.Studies.Approved,
		&report.Studies.Rejected,
		&report.Studies.Pending,
	) //nolint:errcheck

	// ── PHI detection ─────────────────────────────────────────────────────────
	row = s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE phi_scan_status NOT IN ('','pending')) AS scanned,
			COUNT(*) FILTER (WHERE phi_scan_status = 'flagged')           AS flagged
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since)
	row.Scan(&report.PhiDetect.Scanned, &report.PhiDetect.Flagged) //nolint:errcheck
	if report.PhiDetect.Scanned > 0 {
		report.PhiDetect.FlagRatePct = float64(report.PhiDetect.Flagged) / float64(report.PhiDetect.Scanned) * 100
	}

	// ── Defacing ─────────────────────────────────────────────────────────────
	row = s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE defacing_required)                          AS required,
			COUNT(*) FILTER (WHERE defacing_required AND status = 'defaced')   AS completed,
			COUNT(*) FILTER (WHERE defacing_required AND status = 'received')  AS failed,
			COALESCE(AVG(deface_qa_score) FILTER (WHERE deface_qa_score IS NOT NULL), 0) AS avg_qa
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since)
	row.Scan(
		&report.Defacing.Required,
		&report.Defacing.Completed,
		&report.Defacing.Failed,
		&report.Defacing.AvgQAScore,
	) //nolint:errcheck

	// ── Protocol compliance ───────────────────────────────────────────────────
	row = s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE protocol_required AND protocol_status NOT IN ('','pending','checking')) AS checked,
			COUNT(*) FILTER (WHERE protocol_status = 'compliant')                                         AS compliant,
			COUNT(*) FILTER (WHERE protocol_status = 'minor_deviations')                                  AS minor,
			COUNT(*) FILTER (WHERE protocol_status = 'non_compliant')                                     AS non_compliant
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since)
	row.Scan(
		&report.Protocol.Checked,
		&report.Protocol.Compliant,
		&report.Protocol.MinorDeviations,
		&report.Protocol.NonCompliant,
	) //nolint:errcheck

	// ── Export shares ─────────────────────────────────────────────────────────
	row = s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(DISTINCT es.id)                        AS shares_created,
			COUNT(DISTINCT es.id) FILTER (WHERE ed.id IS NOT NULL) AS shares_downloaded,
			COUNT(ed.id)                                 AS total_downloads
		FROM export_shares es
		JOIN studies st ON st.id = es.study_id
		LEFT JOIN export_downloads ed ON ed.share_id = es.id
		WHERE st.project_id = $1
		  AND es.created_at >= $2`, projectID, since)
	row.Scan(
		&report.Exports.SharesCreated,
		&report.Exports.SharesDownloaded,
		&report.Exports.TotalDownloads,
	) //nolint:errcheck

	s.writeJSON(w, http.StatusOK, report)
}

// ExportComplianceReportCSV streams the project compliance report as a CSV download.
//
// GET /api/projects/{id}/compliance-report.csv?days=30
// Columns: section, metric, value
func (s *Server) ExportComplianceReportCSV(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if projectID == "" {
		s.writeError(w, http.StatusBadRequest, "missing project id")
		return
	}

	proj, err := model.GetProjectByID(r.Context(), s.db, projectID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	since := time.Now().UTC().AddDate(0, 0, -days)
	report := complianceReport{
		ProjectID:   projectID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		PeriodDays:  days,
	}

	// Reuse the same queries as GetProjectComplianceReport.
	s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*)                                              AS total,
			COUNT(*) FILTER (WHERE status = 'approved')          AS approved,
			COUNT(*) FILTER (WHERE status = 'rejected')          AS rejected,
			COUNT(*) FILTER (WHERE status NOT IN ('approved','rejected','expired')) AS pending
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since).Scan( //nolint:errcheck
		&report.Studies.Total,
		&report.Studies.Approved,
		&report.Studies.Rejected,
		&report.Studies.Pending,
	)

	s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE phi_scan_status NOT IN ('','pending')) AS scanned,
			COUNT(*) FILTER (WHERE phi_scan_status = 'flagged')           AS flagged
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since).Scan( //nolint:errcheck
		&report.PhiDetect.Scanned, &report.PhiDetect.Flagged)
	if report.PhiDetect.Scanned > 0 {
		report.PhiDetect.FlagRatePct = float64(report.PhiDetect.Flagged) / float64(report.PhiDetect.Scanned) * 100
	}

	s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE defacing_required)                          AS required,
			COUNT(*) FILTER (WHERE defacing_required AND status = 'defaced')   AS completed,
			COUNT(*) FILTER (WHERE defacing_required AND status = 'received')  AS failed,
			COALESCE(AVG(deface_qa_score) FILTER (WHERE deface_qa_score IS NOT NULL), 0) AS avg_qa
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since).Scan( //nolint:errcheck
		&report.Defacing.Required,
		&report.Defacing.Completed,
		&report.Defacing.Failed,
		&report.Defacing.AvgQAScore,
	)

	s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE protocol_required AND protocol_status NOT IN ('','pending','checking')) AS checked,
			COUNT(*) FILTER (WHERE protocol_status = 'compliant')                                         AS compliant,
			COUNT(*) FILTER (WHERE protocol_status = 'minor_deviations')                                  AS minor,
			COUNT(*) FILTER (WHERE protocol_status = 'non_compliant')                                     AS non_compliant
		FROM studies
		WHERE project_id = $1
		  AND created_at >= $2`, projectID, since).Scan( //nolint:errcheck
		&report.Protocol.Checked,
		&report.Protocol.Compliant,
		&report.Protocol.MinorDeviations,
		&report.Protocol.NonCompliant,
	)

	s.db.QueryRowContext(r.Context(), `
		SELECT
			COUNT(DISTINCT es.id)                        AS shares_created,
			COUNT(DISTINCT es.id) FILTER (WHERE ed.id IS NOT NULL) AS shares_downloaded,
			COUNT(ed.id)                                 AS total_downloads
		FROM export_shares es
		JOIN studies st ON st.id = es.study_id
		LEFT JOIN export_downloads ed ON ed.share_id = es.id
		WHERE st.project_id = $1
		  AND es.created_at >= $2`, projectID, since).Scan( //nolint:errcheck
		&report.Exports.SharesCreated,
		&report.Exports.SharesDownloaded,
		&report.Exports.TotalDownloads,
	)

	filename := fmt.Sprintf("compliance-report-%s-%dd.csv", proj.Slug, days)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"section", "metric", "value"})

	// Metadata
	_ = cw.Write([]string{"meta", "project_id", report.ProjectID})
	_ = cw.Write([]string{"meta", "project_name", proj.Name})
	_ = cw.Write([]string{"meta", "generated_at", report.GeneratedAt})
	_ = cw.Write([]string{"meta", "period_days", strconv.Itoa(report.PeriodDays)})

	// Studies
	_ = cw.Write([]string{"studies", "total", strconv.Itoa(report.Studies.Total)})
	_ = cw.Write([]string{"studies", "approved", strconv.Itoa(report.Studies.Approved)})
	_ = cw.Write([]string{"studies", "rejected", strconv.Itoa(report.Studies.Rejected)})
	_ = cw.Write([]string{"studies", "pending", strconv.Itoa(report.Studies.Pending)})

	// PHI detection
	_ = cw.Write([]string{"phi_detection", "scanned", strconv.Itoa(report.PhiDetect.Scanned)})
	_ = cw.Write([]string{"phi_detection", "flagged", strconv.Itoa(report.PhiDetect.Flagged)})
	_ = cw.Write([]string{"phi_detection", "flag_rate_pct", fmt.Sprintf("%.2f", report.PhiDetect.FlagRatePct)})

	// Defacing
	_ = cw.Write([]string{"defacing", "required", strconv.Itoa(report.Defacing.Required)})
	_ = cw.Write([]string{"defacing", "completed", strconv.Itoa(report.Defacing.Completed)})
	_ = cw.Write([]string{"defacing", "failed", strconv.Itoa(report.Defacing.Failed)})
	_ = cw.Write([]string{"defacing", "avg_qa_score", fmt.Sprintf("%.4f", report.Defacing.AvgQAScore)})

	// Protocol compliance
	_ = cw.Write([]string{"protocol_compliance", "checked", strconv.Itoa(report.Protocol.Checked)})
	_ = cw.Write([]string{"protocol_compliance", "compliant", strconv.Itoa(report.Protocol.Compliant)})
	_ = cw.Write([]string{"protocol_compliance", "minor_deviations", strconv.Itoa(report.Protocol.MinorDeviations)})
	_ = cw.Write([]string{"protocol_compliance", "non_compliant", strconv.Itoa(report.Protocol.NonCompliant)})

	// Exports
	_ = cw.Write([]string{"exports", "shares_created", strconv.Itoa(report.Exports.SharesCreated)})
	_ = cw.Write([]string{"exports", "shares_downloaded", strconv.Itoa(report.Exports.SharesDownloaded)})
	_ = cw.Write([]string{"exports", "total_downloads", strconv.Itoa(report.Exports.TotalDownloads)})

	cw.Flush()
}
