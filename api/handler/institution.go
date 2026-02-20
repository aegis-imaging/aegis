package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/msenjem/aegis/api/model"
)

// ─── Institutions ─────────────────────────────────────────────────────────────

func (s *Server) ListInstitutions(w http.ResponseWriter, r *http.Request) {
	insts, err := model.ListInstitutions(r.Context(), s.db)
	if err != nil {
		log.Printf("list institutions: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to list institutions")
		return
	}
	if insts == nil {
		insts = []model.Institution{}
	}
	s.writeJSON(w, http.StatusOK, insts)
}

func (s *Server) CreateInstitution(w http.ResponseWriter, r *http.Request) {
	var inst model.Institution
	if err := json.NewDecoder(r.Body).Decode(&inst); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if inst.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	validTypes := map[string]bool{"sender": true, "receiver": true, "both": true}
	if !validTypes[inst.Type] {
		s.writeError(w, http.StatusBadRequest, "institution_type must be sender, receiver, or both")
		return
	}
	if inst.Slug == "" {
		inst.Slug = slugify(inst.Name)
	}
	if err := normalizeInstitutionNetworkIdentity(&inst); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	inst.Enabled = true

	if err := model.CreateInstitution(r.Context(), s.db, &inst); err != nil {
		log.Printf("create institution: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to create institution")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.created", actorEmail(r), "institution", inst.ID, clientIP(r), map[string]any{
		"name": inst.Name,
		"type": inst.Type,
	})
	s.writeJSON(w, http.StatusCreated, inst)
}

func (s *Server) GetInstitution(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := model.GetInstitutionByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}

	// Include linked projects
	links, err := model.ListProjectsForInstitution(r.Context(), s.db, id)
	if err != nil {
		log.Printf("list projects for institution %s: %v", id, err)
		links = []model.InstitutionProject{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"institution": inst,
		"projects":    links,
	})
}

func (s *Server) UpdateInstitution(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing, err := model.GetInstitutionByID(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}

	var update model.Institution
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if update.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	validTypes := map[string]bool{"sender": true, "receiver": true, "both": true}
	if !validTypes[update.Type] {
		s.writeError(w, http.StatusBadRequest, "institution_type must be sender, receiver, or both")
		return
	}
	if update.Slug == "" {
		update.Slug = slugify(update.Name)
	}
	if err := normalizeInstitutionNetworkIdentity(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	existing.Name = update.Name
	existing.Slug = update.Slug
	existing.Description = update.Description
	existing.Type = update.Type
	existing.ContactName = update.ContactName
	existing.ContactEmail = update.ContactEmail
	existing.IPRanges = update.IPRanges
	existing.AETitle = update.AETitle
	existing.Enabled = update.Enabled

	if err := model.UpdateInstitution(r.Context(), s.db, existing); err != nil {
		log.Printf("update institution %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to update institution")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.updated", actorEmail(r), "institution", id, clientIP(r), nil)
	s.writeJSON(w, http.StatusOK, existing)
}

func normalizeInstitutionNetworkIdentity(inst *model.Institution) error {
	inst.AETitle = normalizeAETitle(inst.AETitle)
	ranges, err := normalizeInstitutionIPRanges(inst.IPRanges)
	if err != nil {
		return err
	}
	inst.IPRanges = ranges
	return nil
}

func normalizeInstitutionIPRanges(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	parts := strings.Split(raw, ",")
	normalized := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			_, network, err := net.ParseCIDR(part)
			if err != nil {
				return "", fmt.Errorf("invalid ip_ranges value %q", part)
			}
			value := network.String()
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			normalized = append(normalized, value)
			continue
		}

		ip := net.ParseIP(part)
		if ip == nil {
			return "", fmt.Errorf("invalid ip_ranges value %q", part)
		}
		value := ip.String()
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return strings.Join(normalized, ","), nil
}

func (s *Server) DeleteInstitution(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetInstitutionByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}
	if err := model.DeleteInstitution(r.Context(), s.db, id); err != nil {
		log.Printf("delete institution %s: %v", id, err)
		s.writeError(w, http.StatusInternalServerError, "failed to delete institution")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.deleted", actorEmail(r), "institution", id, clientIP(r), nil)
	w.WriteHeader(http.StatusNoContent)
}

// ─── Institution-Project links ────────────────────────────────────────────────

type institutionProjectRequest struct {
	ProjectID string `json:"project_id"`
	Role      string `json:"role"`
}

func (s *Server) AddInstitutionProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetInstitutionByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}

	var req institutionProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	validRoles := map[string]bool{"sender": true, "receiver": true, "admin": true}
	if !validRoles[req.Role] {
		s.writeError(w, http.StatusBadRequest, "role must be sender, receiver, or admin")
		return
	}

	link := &model.InstitutionProject{
		InstitutionID: id,
		ProjectID:     req.ProjectID,
		Role:          req.Role,
	}
	if err := model.AddInstitutionToProject(r.Context(), s.db, link); err != nil {
		log.Printf("add institution %s to project %s: %v", id, req.ProjectID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to link institution to project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.project_linked", actorEmail(r), "institution", id, clientIP(r), map[string]any{
		"project_id": req.ProjectID,
		"role":       req.Role,
	})
	s.writeJSON(w, http.StatusCreated, link)
}

func (s *Server) RemoveInstitutionProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	projectID := r.PathValue("projectID")

	if err := model.RemoveInstitutionFromProject(r.Context(), s.db, id, projectID); err != nil {
		log.Printf("remove institution %s from project %s: %v", id, projectID, err)
		s.writeError(w, http.StatusInternalServerError, "failed to unlink institution from project")
		return
	}
	model.CreateAuditEntry(r.Context(), s.db, "institution.project_unlinked", actorEmail(r), "institution", id, clientIP(r), map[string]any{
		"project_id": projectID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ListInstitutionProjects(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := model.GetInstitutionByID(r.Context(), s.db, id); err != nil {
		s.writeError(w, http.StatusNotFound, "institution not found")
		return
	}
	links, err := model.ListProjectsForInstitution(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	if links == nil {
		links = []model.InstitutionProject{}
	}
	s.writeJSON(w, http.StatusOK, links)
}
