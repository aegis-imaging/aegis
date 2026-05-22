package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

type createQCFindingReq struct {
	Category      string `json:"category"`
	Severity      string `json:"severity"`
	Body          string `json:"body"`
	SeriesUID     string `json:"series_uid"`
	InstanceIndex *int   `json:"instance_index,omitempty"`
}

func (s *Server) CreateQCFinding(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	if _, err := model.GetStudyByID(r.Context(), s.db, studyID); err != nil {
		s.writeError(w, http.StatusNotFound, "study not found")
		return
	}
	var req createQCFindingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cat := strings.ToLower(strings.TrimSpace(req.Category))
	sev := strings.ToLower(strings.TrimSpace(req.Severity))
	body := strings.TrimSpace(req.Body)
	if !model.IsValidQCCategory(cat) {
		s.writeError(w, http.StatusBadRequest, "category must be one of phi_leak/defacing/motion/protocol/coverage/metadata/other")
		return
	}
	if !model.IsValidQCSeverity(sev) {
		s.writeError(w, http.StatusBadRequest, "severity must be one of info/minor/major/critical")
		return
	}
	if body == "" {
		s.writeError(w, http.StatusBadRequest, "body is required")
		return
	}
	if len(body) > 4000 {
		s.writeError(w, http.StatusBadRequest, "body must be 4000 characters or fewer")
		return
	}

	finding := &model.QCFinding{
		StudyID:       studyID,
		AnalystEmail:  actorEmail(r),
		Category:      cat,
		Severity:      sev,
		Body:          body,
		SeriesUID:     req.SeriesUID,
		InstanceIndex: req.InstanceIndex,
	}
	if u := middleware.UserFromContext(r.Context()); u != nil && u.ID != "" {
		id := u.ID
		finding.AnalystID = &id
	}
	if err := model.CreateQCFinding(r.Context(), s.db, finding); err != nil {
		s.writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.finding_raised", actorEmail(r), "study", studyID, clientIP(r), map[string]any{
		"category": cat, "severity": sev,
	})
	s.writeJSON(w, http.StatusCreated, finding)
}

func (s *Server) ListQCFindings(w http.ResponseWriter, r *http.Request) {
	studyID := r.PathValue("id")
	openOnly := r.URL.Query().Get("open") == "true"
	findings, err := model.ListQCFindingsByStudy(r.Context(), s.db, studyID, openOnly)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"findings": findings, "total": len(findings)})
}

type resolveFindingReq struct {
	Note string `json:"note"`
}

func (s *Server) ResolveQCFinding(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("findingID")
	var req resolveFindingReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := model.ResolveQCFinding(r.Context(), s.db, findingID, actorEmail(r), strings.TrimSpace(req.Note)); err != nil {
		s.writeError(w, http.StatusInternalServerError, "resolve failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.finding_resolved", actorEmail(r), "qc_finding", findingID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (s *Server) ReopenQCFinding(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("findingID")
	if err := model.ReopenQCFinding(r.Context(), s.db, findingID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "reopen failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.finding_reopened", actorEmail(r), "qc_finding", findingID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "reopened"})
}

func (s *Server) DeleteQCFinding(w http.ResponseWriter, r *http.Request) {
	findingID := r.PathValue("findingID")
	if err := model.DeleteQCFinding(r.Context(), s.db, findingID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "qc.finding_deleted", actorEmail(r), "qc_finding", findingID, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
