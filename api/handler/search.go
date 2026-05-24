package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/aegis-imaging/aegis/api/middleware"
	"github.com/aegis-imaging/aegis/api/model"
	"github.com/lib/pq"
)

// SearchHit is one row in the global top-bar search dropdown.
//
// XNAT inspires the categorization (Project / Subject / Experiment) but the
// payload shape is our own: each hit has enough to render an entry and a
// link without a second round-trip.
type SearchHit struct {
	Kind      string `json:"kind"` // "project" | "subject" | "study"
	ProjectID string `json:"project_id,omitempty"`
	SubjectID string `json:"subject_id,omitempty"`
	StudyID   string `json:"study_id,omitempty"`
	Label     string `json:"label"`
	Secondary string `json:"secondary,omitempty"`
}

type searchResponse struct {
	Query    string      `json:"query"`
	Projects []SearchHit `json:"projects"`
	Subjects []SearchHit `json:"subjects"`
	Studies  []SearchHit `json:"studies"`
}

// Search GET /api/search?q=<term>
//
// Cross-entity search modeled on XNAT's "Standard Search": exact match on
// IDs / UIDs, partial (ILIKE) on free-text fields. Results are grouped by
// entity kind and scoped to the projects the caller has access to.
func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		s.writeJSON(w, http.StatusOK, searchResponse{Query: ""})
		return
	}

	const perSectionLimit = 10

	accessible, err := s.accessibleProjectIDs(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to resolve project access")
		return
	}
	// Unauthenticated callers and members of zero projects see nothing.
	if len(accessible) == 0 {
		s.writeJSON(w, http.StatusOK, searchResponse{Query: q})
		return
	}

	projHits, _ := s.searchProjects(r.Context(), q, accessible, perSectionLimit)
	subjHits, _ := s.searchSubjects(r.Context(), q, accessible, perSectionLimit)
	studyHits, _ := s.searchStudies(r.Context(), q, accessible, perSectionLimit)

	s.writeJSON(w, http.StatusOK, searchResponse{
		Query:    q,
		Projects: projHits,
		Subjects: subjHits,
		Studies:  studyHits,
	})
}

// accessibleProjectIDs returns the project IDs the caller can read.
//
//   - tenant-scoped request: all project IDs in the active tenant.
//   - researcher: member-only project IDs.
//   - platform admin / viewer / unauthenticated (dev with AUTH_ENABLED=false,
//     or tests calling the handler directly): all project IDs.
//
// The route is registered behind `auth(...)` so in production with auth
// enabled the unauthenticated path is unreachable. Treating `user == nil`
// as admin-equivalent matches the convention used by ListStudies and
// requireResearcherProjectScope elsewhere in the codebase.
func (s *Server) accessibleProjectIDs(r *http.Request) ([]string, error) {
	user := middleware.UserFromContext(r.Context())
	tenant := middleware.TenantFromContext(r.Context())

	var projects []model.Project
	var err error
	switch {
	case tenant != nil:
		projects, err = model.ListProjectsForTenant(r.Context(), s.db, tenant.ID)
	case user != nil && user.Role == "researcher":
		projects, err = model.ListProjectsForResearcher(r.Context(), s.db, user.ID)
	default:
		projects, err = model.ListProjects(r.Context(), s.db)
	}
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
	}
	return ids, nil
}

func (s *Server) searchProjects(ctx context.Context, q string, accessible []string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug
		FROM projects
		WHERE archived = false
		  AND id = ANY($1)
		  AND (
		      id::text = $2
		   OR slug = $2
		   OR name ILIKE '%' || $2 || '%'
		   OR slug ILIKE '%' || $2 || '%'
		  )
		ORDER BY (slug = $2) DESC, name
		LIMIT $3`, pq.Array(accessible), q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchHit
	for rows.Next() {
		var id, name, slug string
		if err := rows.Scan(&id, &name, &slug); err != nil {
			return nil, err
		}
		out = append(out, SearchHit{
			Kind:      "project",
			ProjectID: id,
			Label:     name,
			Secondary: slug,
		})
	}
	return out, rows.Err()
}

func (s *Server) searchSubjects(ctx context.Context, q string, accessible []string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT subject_id, project_id, count(*) AS study_count
		FROM studies
		WHERE deleted_at IS NULL
		  AND project_id = ANY($1)
		  AND subject_id IS NOT NULL
		  AND subject_id <> ''
		  AND (subject_id = $2 OR subject_id ILIKE '%' || $2 || '%')
		GROUP BY subject_id, project_id
		ORDER BY (subject_id = $2) DESC, subject_id
		LIMIT $3`, pq.Array(accessible), q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchHit
	for rows.Next() {
		var subjID, projID string
		var count int
		if err := rows.Scan(&subjID, &projID, &count); err != nil {
			return nil, err
		}
		secondary := strconv.Itoa(count) + " study"
		if count != 1 {
			secondary = strconv.Itoa(count) + " studies"
		}
		out = append(out, SearchHit{
			Kind:      "subject",
			ProjectID: projID,
			SubjectID: subjID,
			Label:     subjID,
			Secondary: secondary,
		})
	}
	return out, rows.Err()
}

func (s *Server) searchStudies(ctx context.Context, q string, accessible []string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, COALESCE(subject_id, ''), study_instance_uid,
		       COALESCE(study_description, ''), COALESCE(study_date, '')
		FROM studies
		WHERE deleted_at IS NULL
		  AND project_id = ANY($1)
		  AND (
		      study_instance_uid = $2
		   OR id::text = $2
		   OR study_description ILIKE '%' || $2 || '%'
		   OR label ILIKE '%' || $2 || '%'
		  )
		ORDER BY (study_instance_uid = $2) DESC, created_at DESC
		LIMIT $3`, pq.Array(accessible), q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchHit
	for rows.Next() {
		var id, projID, subjID, uid, desc, date string
		if err := rows.Scan(&id, &projID, &subjID, &uid, &desc, &date); err != nil {
			return nil, err
		}
		label := desc
		if label == "" {
			label = uid
		}
		secondary := date
		if secondary == "" {
			secondary = uid
		}
		out = append(out, SearchHit{
			Kind:      "study",
			ProjectID: projID,
			SubjectID: subjID,
			StudyID:   id,
			Label:     label,
			Secondary: secondary,
		})
	}
	return out, rows.Err()
}
