package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Server) ListProjects(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var (
		projects []model.Project
		err      error
	)
	switch {
	case user == nil:
		// Unauthenticated (public call from upload portal etc.) — return non-restricted projects only.
		projects, err = model.ListProjectsPublic(r.Context(), s.db)
	case user.Role == "researcher":
		// Researcher — return only projects they are a member of.
		projects, err = model.ListProjectsForResearcher(r.Context(), s.db, user.ID)
	default:
		// Platform admin / viewer — return all projects (unchanged behaviour).
		projects, err = model.ListProjects(r.Context(), s.db)
	}

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	if projects == nil {
		projects = []model.Project{}
	}
	s.writeJSON(w, http.StatusOK, projects)
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

func (s *Server) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Slug == "" {
		req.Slug = slugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}

	project, err := model.CreateProject(r.Context(), s.db, req.Name, req.Slug, req.Description)
	if err != nil {
		s.writeError(w, http.StatusConflict, "project slug already exists")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.created", actorEmail(r),
		"project", project.ID, clientIP(r), map[string]any{"name": project.Name, "slug": project.Slug})
	s.writeJSON(w, http.StatusCreated, project)
}

// GetProject returns a single project by ID.
// GET /api/projects/{id}
func (s *Server) GetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, ok := s.requireProjectReadAccess(w, r, id); !ok {
		return
	}
	project, err := model.GetProjectByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}
	s.writeJSON(w, http.StatusOK, project)
}

// SetProjectRetention sets or clears the retention policy for a project.
// PUT /api/projects/{id}/retention
// Body: {"retention_days": 90} or {"retention_days": null} to clear.
func (s *Server) SetProjectRetention(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		RetentionDays *int `json:"retention_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.RetentionDays != nil && *body.RetentionDays <= 0 {
		s.writeError(w, http.StatusBadRequest, "retention_days must be a positive integer or null")
		return
	}

	if err := model.UpdateProjectRetentionDays(r.Context(), s.db, id, body.RetentionDays); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update retention policy")
		return
	}
	detail := map[string]any{"retention_days": body.RetentionDays}
	model.CreateAuditEntry(r.Context(), s.db, "project.retention_updated", actorEmail(r),
		"project", id, clientIP(r), detail)

	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}

// SetProjectSLAThreshold sets or clears the per-project stuck-study threshold.
// PUT /api/projects/{id}/sla-threshold
// Body: {"stuck_threshold_minutes": 120} or {"stuck_threshold_minutes": null} to clear.
func (s *Server) SetProjectSLAThreshold(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		StuckThresholdMinutes *int `json:"stuck_threshold_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.StuckThresholdMinutes != nil && *body.StuckThresholdMinutes <= 0 {
		s.writeError(w, http.StatusBadRequest, "stuck_threshold_minutes must be a positive integer or null")
		return
	}

	if err := model.UpdateProjectSLAThreshold(r.Context(), s.db, id, body.StuckThresholdMinutes); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to update SLA threshold")
		return
	}
	detail := map[string]any{"stuck_threshold_minutes": body.StuckThresholdMinutes}
	model.CreateAuditEntry(r.Context(), s.db, "project.sla_threshold_updated", actorEmail(r),
		"project", id, clientIP(r), detail)

	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}

// ArchiveProject marks a project as archived.
// POST /api/projects/{id}/archive
func (s *Server) ArchiveProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := model.ArchiveProject(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to archive project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.archived", actorEmail(r), "project", id, clientIP(r), nil)
	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}

// RestoreProject unarchives a project.
// POST /api/projects/{id}/restore
func (s *Server) RestoreProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err := model.RestoreProject(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to restore project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.restored", actorEmail(r), "project", id, clientIP(r), nil)
	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}

// CloneProject duplicates a project with all its settings (routing rules, anon profiles,
// protocol templates, PHI config, retention_days, stuck_threshold_minutes).
// Studies, audit entries, and invite codes are NOT cloned.
// POST /api/projects/{id}/clone
func (s *Server) CloneProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	src, err := model.GetProjectByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		req.Name = "Copy of " + src.Name
	}
	if req.Slug == "" {
		req.Slug = slugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}

	// Create the new project.
	dst, err := model.CreateProject(r.Context(), s.db, req.Name, req.Slug, src.Description)
	if err != nil {
		log.Printf("clone project %s create: %v", id, err)
		s.writeError(w, http.StatusConflict, "slug already exists or create failed")
		return
	}

	// Copy scalar settings.
	if src.RetentionDays != nil {
		_ = model.UpdateProjectRetentionDays(r.Context(), s.db, dst.ID, src.RetentionDays)
	}
	if src.StuckThresholdMinutes != nil {
		_ = model.UpdateProjectSLAThreshold(r.Context(), s.db, dst.ID, src.StuckThresholdMinutes)
	}

	// Clone anon profiles.
	profiles, _ := model.ListAnonProfilesByProject(r.Context(), s.db, src.ID)
	oldToNew := map[string]string{}
	for _, p := range profiles {
		np := model.AnonProfile{
			ProjectID:    dst.ID,
			Name:         p.Name,
			Description:  p.Description,
			RetainedTags: p.RetainedTags,
			Enabled:      p.Enabled,
		}
		if err := model.CreateAnonProfile(r.Context(), s.db, &np); err == nil {
			oldToNew[p.ID] = np.ID
		}
	}
	// Preserve the default anon profile pointer.
	if src.DefaultAnonProfileID != nil {
		if newID, ok := oldToNew[*src.DefaultAnonProfileID]; ok {
			_ = model.SetProjectDefaultAnonProfile(r.Context(), s.db, dst.ID, newID)
		}
	}

	// Clone project-scoped routing rules.
	rules, _ := model.ListRoutingRulesByProject(r.Context(), s.db, src.ID)
	for _, rule := range rules {
		nr := model.RoutingRule{
			Name:          rule.Name,
			Description:   rule.Description,
			Priority:      rule.Priority,
			Enabled:       rule.Enabled,
			ProjectID:     &dst.ID,
			Modality:      rule.Modality,
			BodyPart:      rule.BodyPart,
			Source:        rule.Source,
			Action:        rule.Action,
			DestinationID: rule.DestinationID,
		}
		_ = model.CreateRoutingRule(r.Context(), s.db, &nr)
	}

	// Clone protocol templates.
	templates, _ := model.ListProtocolTemplatesByProject(r.Context(), s.db, src.ID)
	for _, tmpl := range templates {
		nt := model.ProtocolTemplate{
			ProjectID:       dst.ID,
			Name:            tmpl.Name,
			Description:     tmpl.Description,
			Manufacturer:    tmpl.Manufacturer,
			Model:           tmpl.Model,
			SoftwareVersion: tmpl.SoftwareVersion,
			SequenceType:    tmpl.SequenceType,
			Rules:           tmpl.Rules,
			Enabled:         tmpl.Enabled,
		}
		_ = model.CreateProtocolTemplate(r.Context(), s.db, &nt)
	}

	// Clone PHI config (only if a row exists — skip on error or default).
	if phiCfg, err := model.GetProjectPhiConfig(r.Context(), s.db, src.ID); err == nil {
		_, _ = model.UpsertProjectPhiConfig(r.Context(), s.db, dst.ID, phiCfg.ConfidenceThreshold, phiCfg.MinTextLength)
	}

	// Re-fetch dst to include all copied settings.
	dst, _ = model.GetProjectByID(r.Context(), s.db, dst.ID)
	model.CreateAuditEntry(r.Context(), s.db, "project.cloned", actorEmail(r), "project", dst.ID, clientIP(r), map[string]any{
		"source_project_id":  src.ID,
		"source_name":        src.Name,
		"name":               dst.Name,
		"slug":               dst.Slug,
		"routing_rules":      len(rules),
		"anon_profiles":      len(profiles),
		"protocol_templates": len(templates),
	})
	s.writeJSON(w, http.StatusCreated, dst)
}

// UpdateProject updates a project's name, slug, and description.
// PUT /api/projects/{id}
func (s *Server) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Slug == "" {
		req.Slug = slugRe.ReplaceAllString(strings.ToLower(req.Name), "-")
		req.Slug = strings.Trim(req.Slug, "-")
	}

	project, err := model.UpdateProject(r.Context(), s.db, id, req.Name, req.Slug, req.Description)
	if err != nil {
		log.Printf("update project %s: %v", id, err)
		s.writeError(w, http.StatusConflict, "slug already exists or update failed")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "project.updated", actorEmail(r),
		"project", id, clientIP(r), map[string]any{"name": project.Name, "slug": project.Slug})
	s.writeJSON(w, http.StatusOK, project)
}

// SetProjectRestricted toggles the restricted flag on a project.
// PUT /api/projects/{id}/restricted
// Body: {"restricted": true}
func (s *Server) SetProjectRestricted(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// Verify project exists.
	if _, err := model.GetProjectByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "project not found")
		return
	}

	var body struct {
		Restricted bool `json:"restricted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := model.SetProjectRestricted(r.Context(), s.db, id, body.Restricted); err != nil {
		log.Printf("set project restricted %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}

	model.CreateAuditEntry(r.Context(), s.db, "project.restricted_updated", actorEmail(r),
		"project", id, clientIP(r), map[string]any{"restricted": body.Restricted})

	project, _ := model.GetProjectByID(r.Context(), s.db, id)
	s.writeJSON(w, http.StatusOK, project)
}
