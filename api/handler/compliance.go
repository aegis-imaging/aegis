package handler

import (
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
