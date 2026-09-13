package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// ActivityEntry represents a unified timeline entry for a study.
type ActivityEntry struct {
	Type      string    `json:"type"` // "audit", "comment"
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Body      string    `json:"body,omitempty"`
	Metadata  any       `json:"metadata,omitempty"`
}

// GetStudyActivity GET /api/studies/{id}/activity
// Returns a unified timeline of audit entries and comments for a study.
func (s *Server) GetStudyActivity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	// Fetch audit entries for this study.
	audits, err := model.ListAuditEntriesForStudy(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "audit query failed")
		return
	}

	// Fetch comments for this study.
	comments, err := model.ListStudyComments(r.Context(), s.db, id)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "comments query failed")
		return
	}

	// Merge into unified timeline.
	var entries []ActivityEntry
	for _, a := range audits {
		entries = append(entries, ActivityEntry{
			Type:      "audit",
			Timestamp: a.CreatedAt,
			Actor:     a.Actor,
			Action:    a.Action,
			Metadata:  a.Detail,
		})
	}
	for _, c := range comments {
		entries = append(entries, ActivityEntry{
			Type:      "comment",
			Timestamp: c.CreatedAt,
			Actor:     c.Author,
			Action:    "comment",
			Body:      c.Body,
		})
	}

	// Sort by timestamp descending (newest first).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}
	if entries == nil {
		entries = []ActivityEntry{}
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"entries": entries,
		"total":   len(entries),
	})
}
