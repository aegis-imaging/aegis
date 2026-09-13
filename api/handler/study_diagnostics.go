package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// isValidUUID returns true if s is a well-formed UUID (8-4-4-4-12 hex).
// Used to return 404 before hitting the DB when a non-UUID path value is given,
// avoiding a PostgreSQL "invalid input syntax for type uuid" error.
func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

type studyDiagnosticsSummary struct {
	Terminal           bool      `json:"terminal"`
	Stuck              bool      `json:"stuck"`
	Blockers           []string  `json:"blockers"`
	RecommendedActions []string  `json:"recommended_actions"`
	LastAuditAction    string    `json:"last_audit_action,omitempty"`
	LastAuditAt        time.Time `json:"last_audit_at,omitempty"`
}

type studyDiagnosticsDimseRetry struct {
	Available       bool             `json:"available"`
	PendingTotal    int              `json:"pending_total"`
	DeadLetterTotal int              `json:"dead_letter_total"`
	PendingItems    []map[string]any `json:"pending_items,omitempty"`
	DeadLetterItems []map[string]any `json:"dead_letter_items,omitempty"`
	Error           string           `json:"error,omitempty"`
}

type studyDiagnosticsResponse struct {
	Study       *model.Study               `json:"study"`
	Summary     studyDiagnosticsSummary    `json:"summary"`
	RecentAudit []model.AuditEntry         `json:"recent_audit"`
	RoutingLog  []model.RoutingLogEntry    `json:"routing_log"`
	DimseRetry  studyDiagnosticsDimseRetry `json:"dimse_retry"`
}

func (s *Server) GetStudyDiagnostics(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if !isValidUUID(studyID) {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	study, _, ok := s.requireStudyReadAccessByID(w, r, studyID)
	if !ok {
		return
	}

	entries, err := model.ListAuditEntriesForStudy(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []model.AuditEntry{}
	}
	const auditLimit = 20
	if len(entries) > auditLimit {
		entries = entries[:auditLimit]
	}

	routingEntries, err := model.ListRoutingLogForStudy(r.Context(), s.db, studyID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to read routing log")
		return
	}
	if routingEntries == nil {
		routingEntries = []model.RoutingLogEntry{}
	}

	dimseRetry := s.fetchStudyDimseRetryDiagnostics(r.Context(), study.StudyInstanceUID)
	summary := buildStudyDiagnosticsSummary(study, entries, dimseRetry)

	s.writeJSON(w, http.StatusOK, studyDiagnosticsResponse{
		Study:       study,
		Summary:     summary,
		RecentAudit: entries,
		RoutingLog:  routingEntries,
		DimseRetry:  dimseRetry,
	})
}

func buildStudyDiagnosticsSummary(
	study *model.Study,
	entries []model.AuditEntry,
	dimseRetry studyDiagnosticsDimseRetry,
) studyDiagnosticsSummary {
	summary := studyDiagnosticsSummary{
		Blockers:           []string{},
		RecommendedActions: []string{},
		Terminal:           study.Status == "approved" || study.Status == "rejected",
	}

	addBlocker := func(v string) {
		if v == "" {
			return
		}
		for _, existing := range summary.Blockers {
			if existing == v {
				return
			}
		}
		summary.Blockers = append(summary.Blockers, v)
	}
	addAction := func(v string) {
		if v == "" {
			return
		}
		for _, existing := range summary.RecommendedActions {
			if existing == v {
				return
			}
		}
		summary.RecommendedActions = append(summary.RecommendedActions, v)
	}

	if len(entries) > 0 {
		summary.LastAuditAction = entries[0].Action
		summary.LastAuditAt = entries[0].CreatedAt
	}

	if study.DefacingRequired {
		if study.Status == "received" {
			addBlocker("Defacing is required but study is still in received status")
			addAction(fmt.Sprintf("Trigger defacing: POST /api/deface/%s", study.StudyInstanceUID))
		}
		if study.Status == "defacing" {
			addBlocker("Defacing is currently in progress")
		}
	}

	evaluateRequiredStage(study.ClassificationRequired, study.ClassificationStatus,
		"Classification", fmt.Sprintf("POST /api/studies/%s/classify", study.StudyInstanceUID), addBlocker, addAction)
	evaluateRequiredStage(study.PhiScanRequired, study.PhiScanStatus,
		"PHI scan", fmt.Sprintf("POST /api/studies/%s/phi-scan", study.StudyInstanceUID), addBlocker, addAction)
	evaluateRequiredStage(study.PixelRedactionRequired, study.PixelRedactionStatus,
		"Pixel redaction", fmt.Sprintf("POST /api/studies/%s/pixel-redaction", study.StudyInstanceUID), addBlocker, addAction)
	evaluateRequiredStage(study.ProtocolRequired, study.ProtocolStatus,
		"Protocol check", fmt.Sprintf("POST /api/studies/%s/protocol-check", study.StudyInstanceUID), addBlocker, addAction)
	evaluateRequiredStage(study.QcRequired, study.QcStatus,
		"QC check", fmt.Sprintf("POST /api/studies/%s/qc-check", study.StudyInstanceUID), addBlocker, addAction)
	evaluateRequiredStage(study.BidsRequired, study.BidsStatus,
		"BIDS conversion", fmt.Sprintf("POST /api/studies/%s/bids-convert", study.StudyInstanceUID), addBlocker, addAction)
	evaluateRequiredStage(study.AnalyticsRequired, study.AnalyticsStatus,
		"Analytics", fmt.Sprintf("POST /api/studies/%s/analytics", study.StudyInstanceUID), addBlocker, addAction)

	if study.ExportRequired {
		if study.Status != "approved" {
			addBlocker("Export is required but study is not approved")
			addAction(fmt.Sprintf("Approve study first: POST /api/studies/%s/approve", study.ID))
		}
		evaluateRequiredStage(study.ExportRequired, study.ExportStatus,
			"Export forwarding", fmt.Sprintf("POST /api/studies/%s/trigger-export", study.StudyInstanceUID), addBlocker, addAction)
	}

	if dimseRetry.Available {
		if dimseRetry.PendingTotal > 0 {
			addBlocker(fmt.Sprintf("DIMSE retry queue has %d pending item(s) for this study", dimseRetry.PendingTotal))
			addAction(fmt.Sprintf("Process pending DIMSE retry: POST /api/dimse/retry/process/%s", study.StudyInstanceUID))
		}
		if dimseRetry.DeadLetterTotal > 0 {
			addBlocker(fmt.Sprintf("DIMSE dead-letter queue has %d item(s) for this study", dimseRetry.DeadLetterTotal))
			addAction(fmt.Sprintf("Replay DIMSE dead-letter study: POST /api/dimse/retry/replay/%s", study.StudyInstanceUID))
		}
	}

	if len(summary.Blockers) == 0 && !summary.Terminal {
		addAction("No explicit blockers detected; review recent audit and routing log entries")
	}
	summary.Stuck = len(summary.Blockers) > 0
	return summary
}

func evaluateRequiredStage(
	required bool,
	status string,
	name string,
	triggerEndpoint string,
	addBlocker func(string),
	addAction func(string),
) {
	if !required {
		return
	}

	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "" || normalized == "pending" {
		addBlocker(fmt.Sprintf("%s is pending", name))
		addAction(fmt.Sprintf("Trigger %s: %s", strings.ToLower(name), triggerEndpoint))
		return
	}

	if normalized == "failed" {
		addBlocker(fmt.Sprintf("%s failed", name))
		addAction(fmt.Sprintf("Retry %s: %s", strings.ToLower(name), triggerEndpoint))
		return
	}

	inProgress := map[string]bool{
		"scanning":    true,
		"checking":    true,
		"classifying": true,
		"converting":  true,
		"defacing":    true,
		"exporting":   true,
		"redacting":   true,
		"analyzing":   true,
	}
	if inProgress[normalized] {
		addBlocker(fmt.Sprintf("%s is in progress (%s)", name, normalized))
	}
}

func (s *Server) fetchStudyDimseRetryDiagnostics(ctx context.Context, studyInstanceUID string) studyDiagnosticsDimseRetry {
	baseURL := strings.TrimSpace(s.cfg.DimseReceiverURL)
	if baseURL == "" || studyInstanceUID == "" {
		return studyDiagnosticsDimseRetry{Available: false}
	}

	const timeout = 3 * time.Second
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	base, err := url.Parse(baseURL)
	if err != nil {
		return studyDiagnosticsDimseRetry{Available: false, Error: "invalid DIMSE receiver URL"}
	}
	target, err := base.Parse("/ingest/retry/details")
	if err != nil {
		return studyDiagnosticsDimseRetry{Available: false, Error: "failed to build DIMSE retry details URL"}
	}
	query := target.Query()
	query.Set("limit", "1")
	query.Set("study_instance_uid", studyInstanceUID)
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(dctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return studyDiagnosticsDimseRetry{Available: false, Error: "failed to construct DIMSE diagnostics request"}
	}
	req.Header.Set("Accept", "application/json")
	if key := strings.TrimSpace(s.cfg.DimseOperatorAPIKey); key != "" {
		req.Header.Set("X-AEGIS-Operator-Key", key)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return studyDiagnosticsDimseRetry{Available: false, Error: "failed to reach DIMSE receiver"}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		detail := strings.TrimSpace(string(body))
		if detail != "" {
			return studyDiagnosticsDimseRetry{
				Available: false,
				Error:     fmt.Sprintf("DIMSE receiver returned %d: %s", resp.StatusCode, detail),
			}
		}
		return studyDiagnosticsDimseRetry{
			Available: false,
			Error:     fmt.Sprintf("DIMSE receiver returned HTTP %d", resp.StatusCode),
		}
	}

	var payload struct {
		Status      string `json:"status"`
		IngestRetry struct {
			PendingTotal    int              `json:"pending_total"`
			DeadLetterTotal int              `json:"dead_letter_total"`
			PendingItems    []map[string]any `json:"pending_items"`
			DeadLetterItems []map[string]any `json:"dead_letter_items"`
		} `json:"ingest_retry"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return studyDiagnosticsDimseRetry{Available: false, Error: "failed to decode DIMSE retry payload"}
	}

	return studyDiagnosticsDimseRetry{
		Available:       true,
		PendingTotal:    payload.IngestRetry.PendingTotal,
		DeadLetterTotal: payload.IngestRetry.DeadLetterTotal,
		PendingItems:    payload.IngestRetry.PendingItems,
		DeadLetterItems: payload.IngestRetry.DeadLetterItems,
	}
}
